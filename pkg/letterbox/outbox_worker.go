package letterbox

import (
	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

type outboxWorker struct {
	config    outboxConfig
	storage   OutboxStore
	publisher publisher
	scheduler gocron.Scheduler
}

func newOutboxWorker(storage OutboxStore, config outboxConfig) (*outboxWorker, error) {
	scheduler, err := gocron.NewScheduler()

	if err != nil {
		return nil, err
	}

	worker := &outboxWorker{config: config, storage: storage, publisher: config.publisher, scheduler: scheduler}
	job := gocron.CronJob(config.cron, false)
	task := gocron.NewTask(worker.process)

	if _, err := scheduler.NewJob(job, task); err != nil {
		return nil, err
	}

	return worker, nil
}

func (w *outboxWorker) process() error {
	messages, err := w.storage.Pending(w.config.pendingLimit)

	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := w.publisher.publish(message.Envelope); err != nil {
			slog.Error("[LETTERBOX] Failed to publish pending message", "id", message.Envelope.Id, "err", err)

			if err := w.storage.UpdateStatus(message.Envelope.Id, StatusError); err != nil {
				slog.Error(
					"[LETTERBOX] Failed to update message status", "id", message.Envelope.Id, "status", StatusError, "err", err,
				)
				continue
			}

			continue
		}

		if err := w.storage.UpdateStatus(message.Envelope.Id, StatusEmitted); err != nil {
			slog.Error(
				"[LETTERBOX] Failed to update message status", "id", message.Envelope.Id, "status", StatusEmitted, "err", err,
			)
		}
	}

	return nil
}
