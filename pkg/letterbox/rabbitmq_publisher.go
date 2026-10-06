package letterbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMqPublisher struct {
	url        string
	exchange   string
	timeout    time.Duration
	mu         sync.Mutex
	connection *amqp.Connection
	channel    *amqp.Channel
	returns    <-chan amqp.Return
	confirms   <-chan amqp.Confirmation
}

func newRabbitMqPublisher(url, exchange string, timeout time.Duration) (*rabbitMqPublisher, error) {
	if url == "" {
		return nil, errors.New("rabbitmq url is empty")
	}

	if exchange == "" {
		return nil, errors.New("rabbitmq exchange is empty")
	}

	if timeout <= 0 {
		return nil, errors.New("invalid rabbitmq timeout")
	}

	connection, err := amqp.Dial(url)

	if err != nil {
		return nil, err
	}

	channel, err := connection.Channel()

	if err != nil {
		if closeErr := connection.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}

		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		closeErr := errors.Join(channel.Close(), connection.Close())

		if closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}

		return nil, err
	}

	return &rabbitMqPublisher{
		url:        url,
		exchange:   exchange,
		timeout:    timeout,
		connection: connection,
		channel:    channel,
		returns:    channel.NotifyReturn(make(chan amqp.Return, 1)),
		confirms:   channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

func (p *rabbitMqPublisher) publish(envelope Envelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel == nil {
		return errors.New("rabbitmq channel is nil")
	}

	body, err := json.Marshal(envelope)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		envelope.Type,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    envelope.Id,
			Type:         envelope.Type,
			Timestamp:    envelope.OccurredAt,
			Body:         body,
		},
	); err != nil {
		p.closeResources()
		return err
	}

	var returned *amqp.Return

	for {
		select {
		case ret, ok := <-p.returns:
			if !ok {
				p.closeResources()
				return errors.New("rabbitmq returns channel closed")
			}

			if ret.MessageId != envelope.Id {
				slog.Warn(
					"[LETTERBOX] RabbitMQ returned message with unexpected id",
					"expectedId", envelope.Id, "actualId", ret.MessageId,
					"exchange", ret.Exchange, "routingKey", ret.RoutingKey,
					"replyCode", ret.ReplyCode, "replyText", ret.ReplyText,
				)
				continue
			}

			returned = &ret

		case confirmation, ok := <-p.confirms:
			if !ok {
				p.closeResources()
				return errors.New("rabbitmq confirms channel closed")
			}

			if !confirmation.Ack {
				return fmt.Errorf("rabbitmq publish was negatively acknowledged: id=%q", envelope.Id)
			}

			if ret := p.drainReturn(envelope.Id); ret != nil {
				returned = ret
			}

			if returned != nil {
				return fmt.Errorf(
					"rabbitmq message unroutable: id=%q exchange=%q routingKey=%q replyCode=%d replyText=%q",
					envelope.Id, returned.Exchange, returned.RoutingKey, returned.ReplyCode, returned.ReplyText,
				)
			}

			return nil

		case <-ctx.Done():
			p.closeResources()
			return ctx.Err()
		}
	}
}

func (p *rabbitMqPublisher) close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closeResources()
}

func (p *rabbitMqPublisher) closeResources() error {
	var err error

	if p.channel != nil {
		err = errors.Join(err, p.channel.Close())
		p.channel = nil
	}

	if p.connection != nil {
		err = errors.Join(err, p.connection.Close())
		p.connection = nil
	}

	p.returns = nil
	p.confirms = nil

	return err
}

func (p *rabbitMqPublisher) drainReturn(id string) *amqp.Return {
	select {
	case ret, ok := <-p.returns:
		if !ok {
			return nil
		}

		if ret.MessageId != id {
			slog.Warn(
				"[LETTERBOX] RabbitMQ returned message with unexpected id",
				"expectedId", id, "actualId", ret.MessageId,
				"exchange", ret.Exchange, "routingKey", ret.RoutingKey,
				"replyCode", ret.ReplyCode, "replyText", ret.ReplyText,
			)

			return nil
		}

		return &ret

	default:
		return nil
	}
}
