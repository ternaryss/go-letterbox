package letterbox

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type InboxOption func(*inboxConfig) error

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

func WithInboxCron(expression string) InboxOption {
	return func(config *inboxConfig) error {
		if len(strings.Fields(expression)) != 5 {
			return errors.New("invalid inbox cron expression")
		}

		config.scheduleMode = modeCron
		config.cron = expression
		config.interval = 0

		return nil
	}
}

func WithInboxInterval(interval time.Duration) InboxOption {
	return func(config *inboxConfig) error {
		if interval <= 0 {
			return errors.New("invalid inbox interval")
		}

		config.scheduleMode = modeInterval
		config.interval = interval
		config.cron = ""

		return nil
	}
}

func WithInboxReceivedLimit(limit int) InboxOption {
	return func(config *inboxConfig) error {
		if limit < 0 {
			return errors.New("invalid inbox received limit")
		}

		config.receivedLimit = limit

		return nil
	}
}

func WithInboxHttpConsumer(mux *http.ServeMux) InboxOption {
	return func(config *inboxConfig) error {
		config.consumer = newHttpConsumer(mux)

		return nil
	}
}

func WithInboxRabbitMqConsumer(url, queue string) InboxOption {
	return func(config *inboxConfig) error {
		consumer, err := newRabbitMqConsumer(url, queue)

		if err != nil {
			return err
		}

		config.consumer = consumer

		return nil
	}
}
