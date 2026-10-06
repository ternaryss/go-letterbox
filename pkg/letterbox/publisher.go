package letterbox

type publisher interface {
	publish(envelope Envelope) error
	close() error
}
