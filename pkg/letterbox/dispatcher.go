package letterbox

import (
	"errors"
	"fmt"
)

type Handler[T Event] func(event T) error

type eventKey struct {
	typee   string
	version int
}

func newEventKey(typee string, version int) eventKey {
	return eventKey{typee: typee, version: version}
}

type dispatcher struct {
	handlers map[eventKey]func(Envelope) error
}

func newDispatcher() *dispatcher {
	return &dispatcher{handlers: make(map[eventKey]func(Envelope) error)}
}

func (d *dispatcher) register(key eventKey, handler func(Envelope) error) error {
	if key.typee == "" {
		return errors.New("event type is empty")
	}

	if key.version <= 0 {
		return errors.New("event version must be positive")
	}

	if handler == nil {
		return errors.New("handler is nil")
	}

	if _, exists := d.handlers[key]; exists {
		return fmt.Errorf("handler already registered for event %q version %d", key.typee, key.version)
	}

	d.handlers[key] = handler

	return nil
}

func (d *dispatcher) dispatch(envelope Envelope) error {
	key := newEventKey(envelope.Type, envelope.Version)
	handler, exists := d.handlers[key]

	if !exists {
		return fmt.Errorf("handler not registered for event %q version %d", envelope.Type, envelope.Version)
	}

	return handler(envelope)
}
