package letterbox

type Event interface {
	Name() string
	Version() int
}
