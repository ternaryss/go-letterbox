package letterbox

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type InboxOption func(*inboxConfig)

type inboxConfig struct {
	scheduleMode  scheduleMode
	cron          string
	interval      time.Duration
	receivedLimit int
	consumer      consumer
}

func defaultInboxConfig() inboxConfig {
	return inboxConfig{
		scheduleMode:  modeCron,
		cron:          "* * * * *",
		interval:      0,
		receivedLimit: 0,
		consumer:      nil,
	}
}

func (c inboxConfig) validate() error {
	switch c.scheduleMode {
	case modeCron:
		cronFields := strings.Fields(c.cron)

		if len(cronFields) != 5 {
			return errors.New("invalid inbox cron expression")
		}

	case modeInterval:
		if c.interval <= 0 {
			return errors.New("invalid inbox interval")
		}

	default:
		return errors.New("invalid inbox schedule mode")
	}

	if c.receivedLimit < 0 {
		return errors.New("invalid inbox received limit")
	}

	if c.consumer == nil {
		return errors.New("inbox consumer is nil")
	}

	return nil
}

func WithInboxCron(expression string) InboxOption {
	return func(config *inboxConfig) {
		config.scheduleMode = modeCron
		config.cron = expression
		config.interval = 0
	}
}

func WithInboxInterval(interval time.Duration) InboxOption {
	return func(config *inboxConfig) {
		config.scheduleMode = modeInterval
		config.interval = interval
		config.cron = ""
	}
}

func WithInboxReceivedLimit(limit int) InboxOption {
	return func(config *inboxConfig) {
		config.receivedLimit = limit
	}
}

func WithInboxHttpConsumer(mux *http.ServeMux) InboxOption {
	return func(config *inboxConfig) {
		config.consumer = newHttpConsumer(mux)
	}
}
