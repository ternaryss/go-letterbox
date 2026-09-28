package letterbox

type InboxStore interface {
	Save(message Message) (bool, error)
	UpdateStatus(key MessageKey, status Status) error
}

type OutboxStore interface {
	Save(message Message) error
	UpdateStatus(key MessageKey, status Status) error
}
