package letterbox

type InboxStore interface {
	Save(message Message) (bool, error)
	UpdateStatus(key MessageKey, status Status) error
	Received(limit int) ([]Message, error)
}

type OutboxStore interface {
	Save(message Message) error
	UpdateStatus(id string, status Status) error
	Pending(limit int) ([]Message, error)
}
