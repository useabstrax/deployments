package plugin_test

import (
	"errors"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, plugin.ExitSuccess},
		{"binary not found", abstrax.ErrBinaryNotFound, plugin.ExitBinaryNotFound},
		{"unsupported api", abstrax.ErrUnsupportedAPIVersion, plugin.ExitUnsupportedVersion},
		{"malformed json", abstrax.ErrMalformedJSON, plugin.ExitMalformedJSON},
		{"host failure", &abstrax.HostCommandError{ExitCode: 1, Stderr: "fail"}, plugin.ExitHostFailure},
		{"internal", errors.New("boom"), plugin.ExitInternal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := plugin.ExitCodeFor(tc.err); got != tc.want {
				t.Fatalf("ExitCodeFor() = %d, want %d", got, tc.want)
			}
		})
	}
}
