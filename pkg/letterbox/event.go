package letterbox

type Event interface {
	Type() string
	Version() int
}
