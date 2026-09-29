package letterbox

import "log/slog"

type publisher interface {
	publish(envelope Envelope) error
}

type consolePublisher struct{}

func newConsolePublisher() *consolePublisher {
	return &consolePublisher{}
}

func (p *consolePublisher) publish(envelope Envelope) error {
	slog.Info(
		"Event published", "id", envelope.Id, "type", envelope.Type, "version", envelope.Version,
		"sender", envelope.Sender, "occurredAt", envelope.OccurredAt, "content", string(envelope.Content),
	)

	return nil
}
