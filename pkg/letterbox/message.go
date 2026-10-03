package letterbox

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusEmitted  Status = "EMITTED"
	StatusReceived Status = "RECEIVED"
	StatusExecuted Status = "EXECUTED"
	StatusError    Status = "ERROR"
)

type Message struct {
	Envelope Envelope
	Status   Status
}

func newMessage(envelope Envelope, status Status) Message {
	return Message{Envelope: envelope, Status: status}
}

type MessageKey struct {
	Id     string
	Sender string
}

func newMessageKey(id, sender string) MessageKey {
	return MessageKey{Id: id, Sender: sender}
}
