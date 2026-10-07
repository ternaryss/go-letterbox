package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ternaryss/go-letterbox/pkg/letterbox"
)

type HelloEvent struct {
	Message string `json:"message"`
}

func (e HelloEvent) Type() string {
	return "hello"
}

func (e HelloEvent) Version() int {
	return 1
}

type OutboxRepo struct {
	messages map[string]letterbox.Message
}

func NewOutboxRepo() *OutboxRepo {
	return &OutboxRepo{messages: make(map[string]letterbox.Message)}
}

func (r *OutboxRepo) Save(message letterbox.Message) error {
	id := message.Envelope.Id

	if _, exists := r.messages[id]; exists {
		return fmt.Errorf("%q message duplicate", id)
	}

	r.messages[id] = message

	return nil
}

func (r *OutboxRepo) UpdateStatus(id string, status letterbox.Status) error {
	message, exists := r.messages[id]

	if !exists {
		return fmt.Errorf("message %q not found", id)
	}

	message.Status = status
	r.messages[id] = message

	return nil
}

func (r *OutboxRepo) Pending(limit int) ([]letterbox.Message, error) {
	messages := make([]letterbox.Message, 0)

	for _, message := range r.messages {
		if message.Status != letterbox.StatusPending {
			continue
		}

		messages = append(messages, message)

		if limit > 0 && len(messages) >= limit {
			break
		}
	}

	return messages, nil
}

func (r *OutboxRepo) DeleteExpired(olderThan time.Time) (int, error) {
	return 0, nil
}

func main() {
	storage := NewOutboxRepo()
	outbox, err := letterbox.NewOutbox(
		"example-app",
		storage,
		letterbox.WithOutboxRabbitMqPublisher(
			"amqp://admin:admin@rabbitmq:5672/",
			"letterbox.events",
			5*time.Second,
		),
	)

	if err != nil {
		slog.Error("Failed to initialize outbox", "err", err)
		os.Exit(1)
	}

	if err := outbox.Publish(HelloEvent{Message: "Hello World!"}); err != nil {
		slog.Error("Failed to publish message", "err", err)
		os.Exit(1)
	}

	if err := outbox.Flush(); err != nil {
		slog.Error("Flushing failed", "err", err)
		os.Exit(1)
	}
}
