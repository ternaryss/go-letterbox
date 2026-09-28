package letterbox

type Status string

const (
	StatusEmitted  Status = "EMITTED"
	StatusAcked    Status = "ACKED"
	StatusReceived Status = "RECEIVED"
	StatusExecuted Status = "EXECUTED"
	StatusError    Status = "ERROR"
)

type Message struct {
	Envelope Envelope
	Status   Status
}

type MessageKey struct {
	Id     string
	Sender string
}
