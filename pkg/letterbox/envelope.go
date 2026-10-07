package letterbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	Id         string          `json:"id"`
	Type       string          `json:"type"`
	Version    int             `json:"version"`
	Sender     string          `json:"sender"`
	OccurredAt time.Time       `json:"occurredAt"`
	Content    json.RawMessage `json:"content"`
}

func encode[T Event](event T, sender string) (Envelope, error) {
	if event.Type() == "" {
		return Envelope{}, errors.New("event type is empty")
	}

	if event.Version() <= 0 {
		return Envelope{}, errors.New("event version must be positive")
	}

	if sender == "" {
		return Envelope{}, errors.New("sender is empty")
	}

	content, err := json.Marshal(event)

	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		Id:         uuid.NewString(),
		Type:       event.Type(),
		Version:    event.Version(),
		Sender:     sender,
		OccurredAt: time.Now().UTC(),
		Content:    content,
	}, nil
}

func decode[T Event](envelope Envelope) (T, error) {
	var event T

	if envelope.Type == "" {
		return event, errors.New("envelope type is empty")
	}

	if envelope.Version <= 0 {
		return event, errors.New("envelope version must be positive")
	}

	if len(envelope.Content) == 0 {
		return event, errors.New("envelope content is empty")
	}

	if err := json.Unmarshal(envelope.Content, &event); err != nil {
		return event, err
	}

	if event.Type() == "" {
		return event, errors.New("event type is empty")
	}

	if event.Version() <= 0 {
		return event, errors.New("event version must be positive")
	}

	if event.Type() != envelope.Type {
		return event, fmt.Errorf("event type mismatch: envelope %q, event %q", envelope.Type, event.Type())
	}

	if event.Version() != envelope.Version {
		return event, fmt.Errorf("event version mismatch: envelope %d, event %d", envelope.Version, event.Version())
	}

	return event, nil
}

func (e *Envelope) validate() error {
	if e.Id == "" {
		return errors.New("envelope id is empty")
	}

	if e.Type == "" {
		return errors.New("envelope type is empty")
	}

	if e.Version <= 0 {
		return errors.New("envelope version must be positive")
	}

	if e.Sender == "" {
		return errors.New("envelope sender is empty")
	}

	if e.OccurredAt.IsZero() {
		return errors.New("envelope occurred at is empty")
	}

	if len(e.Content) == 0 {
		return errors.New("envelope content is empty")
	}

	if string(e.Content) == "null" {
		return errors.New("envelope content is null")
	}

	return nil
}
