package letterbox

type consumer interface {
	register(inbox *Inbox) error
	close() error
}
