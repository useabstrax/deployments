package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <project>",
		Short: "Cancel a stuck or long-running deploy",
		Long: `Stop an in-progress deploy now for the project.

Sends SIGTERM to the deploy process group (hooks included), removes the
unfinished release directory, and clears the deploy lock. The live current
symlink is never changed.

If nothing is running, reports that and exits 0 (stale locks are cleaned up).
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			p := printer()
			action := "deploy.cancel"

			client, err := abstrax.New()
			if err != nil {
				return err
			}
			resp, err := client.Project(cmd.Context(), projectName)
			if err != nil {
				return err
			}
			proj := resp.Project

			if !config.Exists(proj.Path) {
				return fmt.Errorf("deploy.json not found; run abstrax deploy init %s first", projectName)
			}

			if globals.DryRun {
				st, err := release.ReadLock(proj.Path)
				if err != nil {
					return err
				}
				if st == nil || !release.ProcessAlive(st.PID) {
					p.DryRun("no deploy in progress for %s", proj.Name)
					p.Print(output.Success(action, "No deploy in progress.", map[string]interface{}{
						"cancelled": false,
					}))
					return nil
				}
				p.DryRun("would cancel deploy pid=%d release=%s step=%s", st.PID, st.ReleaseID, st.Step)
				p.Print(output.Success(action, "Dry run complete.", map[string]interface{}{
					"cancelled":  true,
					"release_id": st.ReleaseID,
					"step":       st.Step,
					"pid":        st.PID,
				}))
				return nil
			}

			result, err := release.CancelDeploy(release.CancelOptions{ProjectPath: proj.Path})
			if err != nil {
				return err
			}

			summary := result.Message
			if summary == "" {
				if result.Cancelled {
					summary = "Deploy cancelled."
				} else {
					summary = "No deploy in progress."
				}
			}
			p.Print(output.Success(action, summary, result))
			return nil
		},
	}
	return cmd
}
