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

	publisher := &rabbitMqPublisher{
		url:      url,
		exchange: exchange,
		timeout:  timeout,
	}

	if err := publisher.connect(); err != nil {
		return nil, err
	}

	return publisher, nil
}

func (p *rabbitMqPublisher) publish(envelope Envelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureConnected(); err != nil {
		return newConnectionError(err)
	}

	if p.channel == nil {
		return newConnectionError(errors.New("rabbitmq channel is nil"))
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
		closeErr := p.closeResources()
		return newConnectionError(errors.Join(err, closeErr))
	}

	var returned *amqp.Return

	for {
		select {
		case ret, ok := <-p.returns:
			if !ok {
				closeErr := p.closeResources()
				return newConnectionError(errors.Join(errors.New("rabbitmq returns channel closed"), closeErr))
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
				closeErr := p.closeResources()
				return newConnectionError(errors.Join(errors.New("rabbitmq confirms channel closed"), closeErr))
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
			closeErr := p.closeResources()
			return newConnectionError(errors.Join(ctx.Err(), closeErr))
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

func (p *rabbitMqPublisher) connect() error {
	connection, err := amqp.Dial(p.url)

	if err != nil {
		return err
	}

	channel, err := connection.Channel()

	if err != nil {
		closeErr := connection.Close()

		if closeErr != nil {
			return errors.Join(err, closeErr)
		}

		return err
	}

	if err := channel.Confirm(false); err != nil {
		closeErr := errors.Join(channel.Close(), connection.Close())

		if closeErr != nil {
			return errors.Join(err, closeErr)
		}

		return err
	}

	p.connection = connection
	p.channel = channel
	p.returns = channel.NotifyReturn(make(chan amqp.Return, 1))
	p.confirms = channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	return nil
}

func (p *rabbitMqPublisher) ensureConnected() error {
	if p.connection != nil && !p.connection.IsClosed() && p.channel != nil {
		return nil
	}

	if err := p.closeResources(); err != nil {
		return err
	}

	return p.connect()
}
