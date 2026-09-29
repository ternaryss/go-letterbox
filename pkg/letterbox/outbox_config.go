package letterbox

import (
	"errors"
	"strings"
)

type OutboxOption func(*outboxConfig)

type outboxConfig struct {
	cron         string
	pendingLimit int
	publisher    publisher
}

func defaultOutboxConfig() outboxConfig {
	return outboxConfig{
		cron:         "* * * * *",
		pendingLimit: 0,
		publisher:    newConsolePublisher(),
	}
}

func (c outboxConfig) validate() error {
	cronFields := strings.Fields(c.cron)

	if len(cronFields) != 5 {
		return errors.New("invalid cron expression")
	}

	if c.pendingLimit < 0 {
		return errors.New("invalid pending limit")
	}

	if c.publisher == nil {
		return errors.New("publisher is nil")
	}

	return nil
}

func WithCron(expression string) OutboxOption {
	return func(config *outboxConfig) {
		config.cron = expression
	}
}

func WithPendingLimit(limit int) OutboxOption {
	return func(config *outboxConfig) {
		config.pendingLimit = limit
	}
}

func WithConsolePublisher() OutboxOption {
	return func(config *outboxConfig) {
		config.publisher = newConsolePublisher()
	}
}
