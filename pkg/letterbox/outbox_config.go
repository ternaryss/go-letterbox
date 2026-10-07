package letterbox

import (
	"errors"
	"strings"
	"time"
)

type OutboxOption func(*outboxConfig) error

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

func WithOutboxCron(expression string) OutboxOption {
	return func(config *outboxConfig) error {
		if len(strings.Fields(expression)) != 5 {
			return errors.New("invalid outbox cron expression")
		}

		config.scheduleMode = modeCron
		config.cron = expression
		config.interval = 0

		return nil
	}
}

func WithOutboxInterval(interval time.Duration) OutboxOption {
	return func(config *outboxConfig) error {
		if interval <= 0 {
			return errors.New("invalid outbox interval")
		}

		config.scheduleMode = modeInterval
		config.interval = interval
		config.cron = ""

		return nil
	}
}

func WithOutboxPendingLimit(limit int) OutboxOption {
	return func(config *outboxConfig) error {
		if limit < 0 {
			return errors.New("invalid outbox pending limit")
		}

		config.pendingLimit = limit

		return nil
	}
}

func WithOutboxConsolePublisher() OutboxOption {
	return func(config *outboxConfig) error {
		config.publisher = newConsolePublisher()

		return nil
	}
}

func WithOutboxRabbitMqPublisher(url, exchange string, timeout time.Duration) OutboxOption {
	return func(config *outboxConfig) error {
		publisher, err := newRabbitMqPublisher(url, exchange, timeout)

		if err != nil {
			return err
		}

		config.publisher = publisher

		return nil
	}
}
