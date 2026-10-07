package letterbox

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMqConsumer struct {
	url        string
	queue      string
	mu         sync.Mutex
	connection *amqp.Connection
	channel    *amqp.Channel
}

func newRabbitMqConsumer(url, queue string) (*rabbitMqConsumer, error) {
	if url == "" {
		return nil, errors.New("rabbitmq url is empty")
	}

	if queue == "" {
		return nil, errors.New("rabbitmq queue is empty")
	}

	connection, err := amqp.Dial(url)

	if err != nil {
		return nil, err
	}

	channel, err := connection.Channel()

	if err != nil {
		closeErr := connection.Close()

		if closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}

		return nil, err
	}

	if err := channel.Qos(1, 0, false); err != nil {
		closeErr := errors.Join(channel.Close(), connection.Close())

		if closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}

		return nil, err
	}

	return &rabbitMqConsumer{
		url:        url,
		queue:      queue,
		connection: connection,
		channel:    channel,
	}, nil
}

func (c *rabbitMqConsumer) register(inbox *Inbox) error {
	if inbox == nil {
		return errors.New("inbox is nil")
	}

	if c.channel == nil {
		return errors.New("rabbitmq channel is nil")
	}

	deliveries, err := c.channel.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	go c.consume(inbox, deliveries)

	return nil
}

func (c *rabbitMqConsumer) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error

	if c.channel != nil {
		err = errors.Join(err, c.channel.Close())
		c.channel = nil
	}

	if c.connection != nil {
		err = errors.Join(err, c.connection.Close())
		c.connection = nil
	}

	return err
}

func (c *rabbitMqConsumer) consume(inbox *Inbox, deliveries <-chan amqp.Delivery) {
	for delivery := range deliveries {
		var envelope Envelope

		if err := json.Unmarshal(delivery.Body, &envelope); err != nil {
			slog.Error("[LETTERBOX] Invalid RabbitMQ envelope", "err", err)
			c.reject(delivery, false)
			continue
		}

		if err := envelope.validate(); err != nil {
			slog.Error("[LETTERBOX] Invalid RabbitMq envelope", "err", err)
			c.reject(delivery, false)
			continue
		}

		saved, err := inbox.Receive(envelope)

		if err != nil {
			slog.Error(
				"[LETTERBOX] Failed to receive RabbitMQ envelope",
				"id", envelope.Id, "sender", envelope.Sender, "err", err,
			)
			c.nack(delivery, true)
			continue
		}

		if !saved {
			slog.Info("[LETTERBOX] RabbitMQ envelope already received", "id", envelope.Id, "sender", envelope.Sender)
		}

		c.ack(delivery)
	}
}

func (c *rabbitMqConsumer) ack(delivery amqp.Delivery) {
	if err := delivery.Ack(false); err != nil {
		slog.Error("[LETTERBOX] Failed to ack RabbitMQ delivery", "err", err)
	}
}

func (c *rabbitMqConsumer) nack(delivery amqp.Delivery, requeue bool) {
	if err := delivery.Nack(false, requeue); err != nil {
		slog.Error("[LETTERBOX] Failed to nack RabbitMQ delivery", "requeue", requeue, "err", err)
	}
}

func (c *rabbitMqConsumer) reject(delivery amqp.Delivery, requeue bool) {
	if err := delivery.Reject(requeue); err != nil {
		slog.Error("[LETTERBOX] Failed to reject RabbitMQ delivery", "requeue", requeue, "err", err)
	}
}
