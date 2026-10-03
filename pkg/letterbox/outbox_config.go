package letterbox

import (
	"errors"
	"strings"
	"time"
)

type OutboxOption func(*outboxConfig)

type outboxConfig struct {
	scheduleMode scheduleMode
	cron         string
	interval     time.Duration
	pendingLimit int
	publisher    publisher
}

func defaultOutboxConfig() outboxConfig {
	return outboxConfig{
		scheduleMode: modeCron,
		cron:         "* * * * *",
		interval:     0,
		pendingLimit: 0,
		publisher:    nil,
	}
}

func (c outboxConfig) validate() error {
	switch c.scheduleMode {
	case modeCron:
		cronFields := strings.Fields(c.cron)

		if len(cronFields) != 5 {
			return errors.New("invalid outbox cron expression")
		}

	case modeInterval:
		if c.interval <= 0 {
			return errors.New("invalid outbox interval")
		}

	default:
		return errors.New("invalid outbox schedule mode")
	}

	if c.pendingLimit < 0 {
		return errors.New("invalid outbox pending limit")
	}

	if c.publisher == nil {
		return errors.New("outbox publisher is nil")
	}

	return nil
}

func WithOutboxCron(expression string) OutboxOption {
	return func(config *outboxConfig) {
		config.scheduleMode = modeCron
		config.cron = expression
		config.interval = 0
	}
}

func WithOutboxInterval(interval time.Duration) OutboxOption {
	return func(config *outboxConfig) {
		config.scheduleMode = modeInterval
		config.interval = interval
		config.cron = ""
	}
}

func WithOutboxPendingLimit(limit int) OutboxOption {
	return func(config *outboxConfig) {
		config.pendingLimit = limit
	}
}

func WithOutboxConsolePublisher() OutboxOption {
	return func(config *outboxConfig) {
		config.publisher = newConsolePublisher()
	}
}
