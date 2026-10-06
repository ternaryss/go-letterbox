package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
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

type InboxRepo struct {
	mu       sync.Mutex
	messages map[letterbox.MessageKey]letterbox.Message
}

func NewInboxRepo() *InboxRepo {
	return &InboxRepo{messages: make(map[letterbox.MessageKey]letterbox.Message)}
}

func (r *InboxRepo) Save(message letterbox.Message) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := letterbox.MessageKey{Id: message.Envelope.Id, Sender: message.Envelope.Sender}

	if _, exists := r.messages[key]; exists {
		return false, nil
	}

	r.messages[key] = message

	return true, nil
}

func (r *InboxRepo) UpdateStatus(key letterbox.MessageKey, status letterbox.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	message, exists := r.messages[key]

	if !exists {
		return fmt.Errorf("message %q from sender %q not found", key.Id, key.Sender)
	}

	message.Status = status
	r.messages[key] = message

	return nil
}

func (r *InboxRepo) Received(limit int) ([]letterbox.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	messages := make([]letterbox.Message, 0)

	for _, message := range r.messages {
		if message.Status != letterbox.StatusReceived {
			continue
		}

		messages = append(messages, message)

		if limit > 0 && len(messages) >= limit {
			break
		}
	}

	return messages, nil
}

func main() {
	mux := http.NewServeMux()
	storage := NewInboxRepo()
	inbox, err := letterbox.NewInbox(
		storage,
		letterbox.WithInboxInterval(time.Second),
		letterbox.WithInboxHttpConsumer(mux),
	)

	if err != nil {
		slog.Error("Failed to initialize inbox", "err", err)
		os.Exit(1)
	}

	if err := letterbox.Subscribe(inbox, func(event HelloEvent) error {
		slog.Info("Hello event handled", "message", event.Message)
		return nil
	}); err != nil {
		slog.Error("Failed to subscribe handler", "err", err)
		os.Exit(1)
	}

	inbox.Start()
	defer func() {
		if err := inbox.Stop(); err != nil {
			slog.Error("Failed to stop inbox", "err", err)
		}
	}()

	slog.Info("Starting HTTP inbox consumer", "addr", ":8080", "path", "POST /events")

	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		slog.Error("HTTP server failed", "err", err)
		os.Exit(1)
	}
}
