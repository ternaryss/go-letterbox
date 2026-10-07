package letterbox

import (
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type inboxRetentionWorker struct {
	storage   InboxStore
	enabled   bool
	age       time.Duration
	scheduler gocron.Scheduler
}

func newInboxRetentionWorker(storage InboxStore, config inboxConfig) (*inboxRetentionWorker, error) {
	scheduler, err := gocron.NewScheduler()

	if err != nil {
		return nil, err
	}

	worker := &inboxRetentionWorker{
		storage:   storage,
		enabled:   config.retentionEnabled,
		age:       config.retentionAge,
		scheduler: scheduler,
	}

	if !worker.enabled {
		return worker, nil
	}

	job := gocron.CronJob(config.retentionCron, false)
	task := gocron.NewTask(worker.process)

	if _, err := scheduler.NewJob(job, task); err != nil {
		return nil, err
	}

	return worker, nil
}

func (w *inboxRetentionWorker) process() {
	if !w.enabled {
		return
	}

	olderThan := time.Now().UTC().Add(-w.age)
	deleted, err := w.storage.DeleteExpired(olderThan)

	if err != nil {
		slog.Error("[LETTERBOX] Failed to delete expired inbox messages", "err", err)
		return
	}

	slog.Info("[LETTERBOX] Expired inbox messages deleted", "count", deleted)
}
