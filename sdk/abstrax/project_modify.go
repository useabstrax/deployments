package abstrax

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SetProjectPublicDir updates the project public directory via abstrax project modify --json.
// Use values like "current/public" for zero-downtime deploys.
func (c *Client) SetProjectPublicDir(ctx context.Context, project, publicDir string) error {
	project = strings.TrimSpace(project)
	publicDir = strings.TrimSpace(publicDir)
	if project == "" {
		return errors.New("project name is required")
	}
	if publicDir == "" {
		return errors.New("public dir is required")
	}

	out, err := c.run(ctx, "project", "modify", project, "--public-dir="+publicDir, "--json")
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
		return nil
	}

	var hostErr *HostCommandError
	if errors.As(err, &hostErr) {
		return hostFailureFromOutput(out, hostErr)
	}
	return err
}

// ResolveCLIPHP is a helper note: PHP CLI resolution is handled by the plugin,
// not Abstrax. Kept here so callers remember not to invent private Abstrax APIs.
func ResolveCLIPHPHint(runtimeType, version string) string {
	if strings.ToLower(runtimeType) != "php" {
		return ""
	}
	return fmt.Sprintf("php CLI for version %s", version)
}
