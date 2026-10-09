package tsukuyomi

import "errors"

var (
	ErrStateNotFound     = errors.New("state not found")
	ErrInvalidTransition = errors.New("invalid transition: states must be of the same type")
)
