package abstrax

import "errors"

const (
	// SupportedProjectAPIVersion is the project inspect API version this SDK supports.
	SupportedProjectAPIVersion = "v1"

	actionProjectServiceRestart = "project.service.restart"
	actionProjectServiceReload  = "project.service.reload"
)

var (
	// ErrBinaryNotFound indicates the Abstrax binary could not be located.
	ErrBinaryNotFound = errors.New("abstrax binary not found")

	// ErrUnsupportedAPIVersion indicates an unsupported project inspect API version.
	ErrUnsupportedAPIVersion = errors.New("unsupported Abstrax API version")

	// ErrMalformedJSON indicates Abstrax returned invalid JSON.
	ErrMalformedJSON = errors.New("malformed JSON from Abstrax")

	// ErrProjectNotFound indicates the requested project does not exist.
	ErrProjectNotFound = errors.New("project not found")
)

// HostCommandError indicates the Abstrax host command exited with an error.
type HostCommandError struct {
	ExitCode int
	Stderr   string
	Err      error
}

func (e *HostCommandError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "abstrax command failed"
}

func (e *HostCommandError) Unwrap() error {
	return e.Err
}
