package letterbox

import (
	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

type inboxWorker struct {
	config     inboxConfig
	storage    InboxStore
	dispatcher *dispatcher
	consumer   consumer
	scheduler  gocron.Scheduler
}

func newInboxWorker(storage InboxStore, dispatcher *dispatcher, config inboxConfig) (*inboxWorker, error) {
	scheduler, err := gocron.NewScheduler()

	if err != nil {
		return nil, err
	}

	worker := &inboxWorker{
		config: config, storage: storage, dispatcher: dispatcher, consumer: config.consumer, scheduler: scheduler,
	}
	var job gocron.JobDefinition

	switch config.scheduleMode {
	case modeCron:
		job = gocron.CronJob(config.cron, false)

	case modeInterval:
		job = gocron.DurationJob(config.interval)
	}

	task := gocron.NewTask(worker.process)

	if _, err := scheduler.NewJob(job, task); err != nil {
		return nil, err
	}

	return worker, nil
}

func (w *inboxWorker) process() error {
	messages, err := w.storage.Received(w.config.receivedLimit)

	if err != nil {
		return err
	}

	for _, message := range messages {
		key := newMessageKey(message.Envelope.Id, message.Envelope.Sender)

		if err := w.dispatcher.dispatch(message.Envelope); err != nil {
			slog.Error(
				"[LETTERBOX] Failed to dispatch received message",
				"id", message.Envelope.Id, "sender", message.Envelope.Sender, "err", err,
			)

			if err := w.storage.UpdateStatus(key, StatusError); err != nil {
				slog.Error(
					"[LETTERBOX] Failed to update message status",
					"id", message.Envelope.Id, "sender", message.Envelope.Sender, "status", StatusError, "err", err,
				)
			}

			continue
		}

		if err := w.storage.UpdateStatus(key, StatusExecuted); err != nil {
			slog.Error(
				"[LETTERBOX] Failed to update message status",
				"id", message.Envelope.Id, "sender", message.Envelope.Sender, "status", StatusExecuted, "err", err,
			)
		}
	}

	return nil
}
