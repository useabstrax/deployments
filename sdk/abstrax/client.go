package abstrax

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const envBinary = "ABSTRAX_BINARY"

// Client executes typed Abstrax CLI commands.
type Client struct {
	Binary string
}

// New creates a Client using ABSTRAX_BINARY or PATH lookup.
func New() (*Client, error) {
	path, err := resolveBinary("")
	return &Client{Binary: path}, err
}

// NewWithBinary creates a Client with an explicit Abstrax binary path.
func NewWithBinary(path string) (*Client, error) {
	resolved, err := resolveBinary(path)
	return &Client{Binary: resolved}, err
}

func resolveBinary(explicit string) (string, error) {
	if explicit != "" {
		if err := validateExecutable(explicit); err != nil {
			return "", err
		}
		return explicit, nil
	}
	if env := strings.TrimSpace(os.Getenv(envBinary)); env != "" {
		if err := validateExecutable(env); err != nil {
			return "", err
		}
		return env, nil
	}
	path, err := exec.LookPath("abstrax")
	if err != nil {
		return "", fmt.Errorf("%w: set ABSTRAX_BINARY or install abstrax on PATH", ErrBinaryNotFound)
	}
	return path, nil
}

func validateExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %q", ErrBinaryNotFound, path)
		}
		return fmt.Errorf("%w: %q: %v", ErrBinaryNotFound, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: %q is a directory", ErrBinaryNotFound, path)
	}
	return nil
}

type runOutput struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

func (c *Client) run(ctx context.Context, args ...string) (*runOutput, error) {
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := &runOutput{
		stdout: stdout.Bytes(),
		stderr: stderr.Bytes(),
	}
	if err == nil {
		return out, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.exitCode = exitErr.ExitCode()
		return out, &HostCommandError{
			ExitCode: out.exitCode,
			Stderr:   strings.TrimSpace(stderr.String()),
			Err:      err,
		}
	}
	return out, fmt.Errorf("running abstrax: %w", err)
}

func parseProjectResponse(data []byte) (*ProjectResponse, error) {
	var resp ProjectResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if resp.APIVersion == "" {
		return nil, fmt.Errorf("%w: missing api_version", ErrMalformedJSON)
	}
	if resp.APIVersion != SupportedProjectAPIVersion {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrUnsupportedAPIVersion, resp.APIVersion, SupportedProjectAPIVersion)
	}
	return &resp, nil
}

func parseResult(data []byte) (*Result, error) {
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	return &result, nil
}

func hostFailureFromOutput(out *runOutput, hostErr *HostCommandError) error {
	if result, err := parseResult(out.stdout); err == nil && result.Status == "error" {
		msg := result.Message
		if msg == "" {
			msg = hostErr.Stderr
		}
		if strings.Contains(strings.ToLower(msg), "not found") {
			return fmt.Errorf("%w: %s", ErrProjectNotFound, msg)
		}
		return &HostCommandError{
			ExitCode: hostErr.ExitCode,
			Stderr:   msg,
			Err:      hostErr.Err,
		}
	}
	if strings.Contains(strings.ToLower(hostErr.Stderr), "not found") {
		return fmt.Errorf("%w: %s", ErrProjectNotFound, hostErr.Stderr)
	}
	return hostErr
}

func (c *Client) runServiceAction(ctx context.Context, action, project, service string) error {
	if strings.TrimSpace(project) == "" || strings.TrimSpace(service) == "" {
		return fmt.Errorf("project and service names are required")
	}

	out, err := c.run(ctx, "project", "service", action, project, service, "--json")
	if err == nil {
		result, parseErr := parseResult(out.stdout)
		if parseErr != nil {
			return parseErr
		}
		if result.Status != "success" {
			msg := result.Message
			if msg == "" {
				msg = result.Summary
			}
			return &HostCommandError{ExitCode: 1, Stderr: msg}
		}
		expected := actionProjectServiceRestart
		if action == "reload" {
			expected = actionProjectServiceReload
		}
		if result.Action != "" && result.Action != expected {
			return fmt.Errorf("%w: unexpected action %q", ErrMalformedJSON, result.Action)
		}
		return nil
	}

	var hostErr *HostCommandError
	if errors.As(err, &hostErr) {
		return hostFailureFromOutput(out, hostErr)
	}
	return err
}
