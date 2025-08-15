package serviceerrors

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrContextCanceled  = errors.New("context canceled")
	ErrDeadlineExceeded = errors.New("deadline exceeded")
)
