package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/key"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newKeyCmd() *cobra.Command {
	var (
		show        bool
		fingerprint bool
		rotate      bool
	)

	cmd := &cobra.Command{
		Use:   "key <project>",
		Short: "Create or manage the project deploy key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			p := printer()
			action := "deploy.key"

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
			cfg, err := config.Load(proj.Path)
			if err != nil {
				return err
			}

			home, err := userx.LookupHome(proj.User)
			if err != nil {
				return err
			}
			keyPath := cfg.DeployKey
			if keyPath == "" {
				keyPath = key.DefaultPath(home, proj.Name)
			}

			if show || fingerprint {
				if _, err := os.Stat(keyPath); err != nil {
					return fmt.Errorf("deploy key not found at %s", keyPath)
				}
				if fingerprint {
					fp, err := key.Fingerprint(keyPath)
					if err != nil {
						return err
					}
					if globals.JSON || globals.JSONStream {
						p.Print(output.Success(action, fp, map[string]string{"fingerprint": fp, "path": keyPath}))
					} else {
						p.Line("%s", fp)
					}
					return nil
				}
				pub, err := key.PublicKey(keyPath)
				if err != nil {
					return err
				}
				if globals.JSON || globals.JSONStream {
					p.Print(output.Success(action, "Public key", map[string]string{"public_key": pub, "path": keyPath}))
				} else {
					p.Line("%s", pub)
				}
				return nil
			}

			if err := userx.RequireRoot(); err != nil {
				return err
			}

			if rotate && !globals.Yes {
				return fmt.Errorf("--rotate requires --yes")
			}

			if globals.DryRun {
				p.DryRun("would generate deploy key at %s", keyPath)
				p.Print(output.Success(action, "Dry run complete.", map[string]string{"path": keyPath}))
				return nil
			}

			comment := fmt.Sprintf("abstrax-deploy:%s", proj.Name)
			if rotate {
				if err := key.Rotate(keyPath, comment); err != nil {
					return err
				}
			} else {
				if err := key.Generate(keyPath, comment); err != nil {
					return err
				}
			}
			_ = userx.ChownPath(filepath.Dir(keyPath), proj.User)
			_ = userx.ChownPath(keyPath, proj.User)
			_ = userx.ChownPath(keyPath+".pub", proj.User)

			sshDir := filepath.Dir(keyPath)
			knownHosts := filepath.Join(sshDir, "known_hosts")
			if err := key.EnsureGitHubKnownHosts(knownHosts); err != nil {
				p.Warn("could not update known_hosts: %v", err)
			} else {
				_ = userx.ChownPath(knownHosts, proj.User)
			}

			cfg.DeployKey = keyPath
			if err := config.Save(proj.Path, cfg); err != nil {
				return err
			}

			pub, err := key.PublicKey(keyPath)
			if err != nil {
				return err
			}
			help := key.GitHubDeployKeyInstructions(cfg.Repository, pub)
			if !globals.JSON && !globals.JSONStream {
				p.Line("%s", help)
			}
			p.Print(output.Success(action, fmt.Sprintf("Deploy key ready at %s.", keyPath), map[string]string{
				"path":       keyPath,
				"public_key": pub,
			}))
			return nil
		},
	}

	cmd.Flags().BoolVar(&show, "show", false, "Print the public key")
	cmd.Flags().BoolVar(&fingerprint, "fingerprint", false, "Print the key fingerprint")
	cmd.Flags().BoolVar(&rotate, "rotate", false, "Replace the deploy key (requires --yes)")
	return cmd
}
