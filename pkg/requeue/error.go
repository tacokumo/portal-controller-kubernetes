package requeue

import (
	"github.com/cockroachdb/errors"
)

func NewError(message string) error {
	return &Error{
		Message: message,
	}
}

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func IsRequeueError(err error) bool {
	var requeueErr *Error
	return errors.As(err, &requeueErr)
}

var _ error = &Error{}
