package letterbox

import (
	"encoding/json"
	"errors"
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
