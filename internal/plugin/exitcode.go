package plugin

import (
	"errors"
	"os"

	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

const (
	// ExitSuccess indicates the command completed successfully.
	ExitSuccess = 0
	// ExitInternal indicates an unexpected plugin failure.
	ExitInternal = 1
	// ExitUsage indicates invalid arguments or command usage.
	ExitUsage = 2
	// ExitBinaryNotFound indicates the Abstrax binary could not be located.
	ExitBinaryNotFound = 3
	// ExitHostFailure indicates the Abstrax host command failed.
	ExitHostFailure = 4
	// ExitUnsupportedVersion indicates an unsupported API or protocol version.
	ExitUnsupportedVersion = 5
	// ExitMalformedJSON indicates malformed JSON from Abstrax.
	ExitMalformedJSON = 6
)

// ExitCodeFor maps an error to a documented plugin exit code.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitSuccess
	}
	switch {
	case errors.Is(err, abstrax.ErrBinaryNotFound):
		return ExitBinaryNotFound
	case errors.Is(err, abstrax.ErrUnsupportedAPIVersion):
		return ExitUnsupportedVersion
	case errors.Is(err, abstrax.ErrMalformedJSON):
		return ExitMalformedJSON
	}
	var hostErr *abstrax.HostCommandError
	if errors.As(err, &hostErr) {
		return ExitHostFailure
	}
	return ExitInternal
}

// Exit prints err to stderr when non-nil and exits with the mapped code.
func Exit(err error) {
	if err != nil {
		os.Exit(ExitCodeFor(err))
	}
	os.Exit(ExitSuccess)
}
