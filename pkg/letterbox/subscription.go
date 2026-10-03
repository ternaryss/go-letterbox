package letterbox

import "errors"

func Subscribe[T Event](inbox *Inbox, handler Handler[T]) error {
	if inbox == nil {
		return errors.New("inbox is nil")
	}

	if handler == nil {
		return errors.New("handler is nil")
	}

	var event T

	if event.Type() == "" {
		return errors.New("event type is empty")
	}

	if event.Version() <= 0 {
		return errors.New("event version must be positive")
	}

	key := newEventKey(event.Type(), event.Version())
	wrapped := func(envelope Envelope) error {
		event, err := decode[T](envelope)

		if err != nil {
			return err
		}

		return handler(event)
	}

	return inbox.dispatcher.register(key, wrapped)
}
