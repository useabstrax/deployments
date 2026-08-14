package abstrax

import (
	"context"
	"errors"
	"strings"
)

// Project returns project information via abstrax project inspect --json.
func (c *Client) Project(ctx context.Context, name string) (*ProjectResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("project name is required")
	}

	out, err := c.run(ctx, "project", "inspect", name, "--json")
	if err == nil {
		return parseProjectResponse(out.stdout)
	}

	var hostErr *HostCommandError
	if errors.As(err, &hostErr) {
		if resp, parseErr := parseProjectResponse(out.stdout); parseErr == nil {
			return resp, nil
		}
		return nil, hostFailureFromOutput(out, hostErr)
	}
	return nil, err
}
