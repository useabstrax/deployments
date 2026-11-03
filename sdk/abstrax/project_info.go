package abstrax

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// projectInfoEnvelope is the Abstrax CLI JSON result for project info.
type projectInfoEnvelope struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Summary string `json:"summary,omitempty"`
	Data    struct {
		PublicDir string `json:"public_dir,omitempty"`
	} `json:"data"`
}

// ProjectPublicDir returns the project's current public_dir via abstrax project info --json.
func (c *Client) ProjectPublicDir(ctx context.Context, project string) (string, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return "", errors.New("project name is required")
	}

	out, err := c.run(ctx, "project", "info", project, "--json")
	if err == nil {
		return parseProjectPublicDir(out.stdout)
	}

	var hostErr *HostCommandError
	if errors.As(err, &hostErr) {
		if dir, parseErr := parseProjectPublicDir(out.stdout); parseErr == nil {
			return dir, nil
		}
		return "", hostFailureFromOutput(out, hostErr)
	}
	return "", err
}

func parseProjectPublicDir(data []byte) (string, error) {
	var env projectInfoEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "", fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if env.Status != "" && env.Status != "success" {
		msg := env.Message
		if msg == "" {
			msg = env.Summary
		}
		if msg == "" {
			msg = "project info failed"
		}
		return "", &HostCommandError{ExitCode: 1, Stderr: msg}
	}
	return strings.TrimSpace(env.Data.PublicDir), nil
}
