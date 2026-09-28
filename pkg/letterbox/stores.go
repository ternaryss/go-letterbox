package letterbox

type InboxStore interface {
	Save(message Message) (bool, error)
	UpdateStatus(key MessageKey, status Status) error
}
