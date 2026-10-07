package letterbox

import "errors"

type connectionError struct {
	err error
}

func newConnectionError(err error) error {
	return &connectionError{err: err}
}

func (e *connectionError) Error() string {
	return e.err.Error()
}

func isConnectionError(err error) bool {
	var connectionErr *connectionError

	return errors.As(err, &connectionErr)
}
