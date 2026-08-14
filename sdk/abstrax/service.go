package abstrax

import "context"

// RestartProjectService restarts a project-owned service.
func (c *Client) RestartProjectService(ctx context.Context, project, service string) error {
	return c.runServiceAction(ctx, "restart", project, service)
}

// ReloadProjectService reloads a project-owned service.
func (c *Client) ReloadProjectService(ctx context.Context, project, service string) error {
	return c.runServiceAction(ctx, "reload", project, service)
}
