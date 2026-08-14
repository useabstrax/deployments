package plugin

import "errors"

var (
	// ErrInternal indicates an unexpected plugin failure.
	ErrInternal = errors.New("internal plugin error")
)
