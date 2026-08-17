package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/key"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/provider"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/runtimephp"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newSetupCmd() *cobra.Command {
	var (
		repository    string
		branch        string
		preset        string
		publicDir     string
		keep          int
		noFirstDeploy bool
	)

	cmd := &cobra.Command{
		Use:   "setup <project>",
		Short: "Guided one-shot deploy setup (init + configure + key + optional first deploy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			p := printer()
			action := "deploy.setup"

			client, err := abstrax.New()
			if err != nil {
				return err
			}
			resp, err := client.Project(cmd.Context(), projectName)
			if err != nil {
				return err
			}
			proj := resp.Project

			interactive := isTTY() && !globals.Yes

			if repository == "" && interactive {
				repository, err = prompt("Git repository (git@github.com:org/repo.git)", "")
				if err != nil {
					return err
				}
			}
			if repository == "" {
				return fmt.Errorf("--repository is required (or run interactively on a TTY)")
			}

			prov, err := provider.Lookup("github")
			if err != nil {
				return err
			}
			repository, err = prov.NormalizeRepository(repository)
			if err != nil {
				return err
			}

			if branch == "" {
				if interactive {
					branch, err = prompt("Branch", "main")
					if err != nil {
						return err
					}
				} else {
					branch = "main"
				}
			}
			if preset == "" {
				def := inferPreset(proj.Runtime.Type)
				if interactive {
					preset, err = prompt(fmt.Sprintf("Preset (%s)", strings.Join(presets.All(), "/")), def)
					if err != nil {
						return err
					}
				} else {
					preset = def
				}
			}
			if publicDir == "" && interactive {
				publicDir, err = prompt("Public directory", "public")
				if err != nil {
					return err
				}
			}
			if keep == 0 {
				keep = 5
			}

			if globals.DryRun {
				p.DryRun("would init, configure, create deploy key, and optionally deploy %s", proj.Name)
				p.Print(output.Success(action, "Dry run complete.", map[string]string{
					"repository": repository,
					"branch":     branch,
					"preset":     preset,
				}))
				return nil
			}

			p.Progress(action, "init", "Initializing deploy layout")
			if err := layout.EnsureScaffold(proj.Path); err != nil {
				return err
			}

			cfg := config.Default(proj.Name)
			cfg.Repository = repository
			cfg.Branch = branch
			cfg.KeepReleases = keep
			if err := presets.Apply(&cfg, preset); err != nil {
				return err
			}
			if publicDir != "" {
				cfg.PublicDir = publicDir
			}

			if err := config.Save(proj.Path, &cfg); err != nil {
				return err
			}
			if err := presets.EnsureSharedScaffold(proj.Path, &cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)
			_ = userx.ChownPath(layout.ForProject(proj.Path).Releases, proj.User)
			_ = userx.ChownPath(layout.ForProject(proj.Path).Shared, proj.User)

			docRoot := layout.DocumentRootRel(cfg.PublicDir)
			p.Progress(action, "web_root", "Setting public-dir to "+docRoot)
			if err := client.SetProjectPublicDir(cmd.Context(), proj.Name, docRoot); err != nil {
				p.Warn("could not update project public-dir: %v", err)
			}

			home, err := userx.LookupHome(proj.User)
			if err != nil {
				return err
			}
			keyPath := key.DefaultPath(home, proj.Name)
			p.Progress(action, "key", "Creating deploy key")
			if _, err := os.Stat(keyPath); err == nil {
				p.Info("Deploy key already exists at %s", keyPath)
			} else {
				if err := key.Generate(keyPath, "abstrax-deploy:"+proj.Name); err != nil {
					return err
				}
			}
			_ = userx.ChownPath(keyPath, proj.User)
			_ = userx.ChownPath(keyPath+".pub", proj.User)
			knownHosts := filepath.Join(home, ".ssh", "known_hosts")
			if err := key.EnsureGitHubKnownHosts(knownHosts); err != nil {
				p.Warn("known_hosts: %v", err)
			}
			cfg.DeployKey = keyPath
			if err := config.Save(proj.Path, &cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)

			pub, _ := key.PublicKey(keyPath)
			if !globals.JSON && !globals.JSONStream {
				p.Line("%s", key.GitHubDeployKeyInstructions(cfg.Repository, pub))
			}

			doDeploy := false
			switch {
			case noFirstDeploy:
				doDeploy = false
			case interactive:
				ok, _ := confirm("Run first deploy now? (add the GitHub deploy key first)")
				doDeploy = ok
			case globals.Yes:
				doDeploy = true
			default:
				p.Info("Skipping first deploy (pass --yes to deploy now, or use abstrax deploy now).")
			}

			var deployResult *release.DeployResult
			if doDeploy {
				phpCLI := ""
				if proj.Runtime.Type == "php" {
					phpCLI = runtimephp.ResolveCLI(proj.Runtime.Version)
				}
				p.Progress(action, "deploy", "Running first deploy")
				deployResult, err = release.Deploy(cmd.Context(), release.DeployOptions{
					ProjectPath: proj.Path,
					Config:      &cfg,
					RunAsUser:   userx.RunAs(proj.User),
					Owner:       proj.User,
					CLIPHP:      phpCLI,
					KnownHosts:  knownHosts,
					Progress:    p,
					Action:      action,
				})
				if err != nil {
					return fmt.Errorf("setup succeeded but first deploy failed: %w", err)
				}
			}

			data := map[string]interface{}{
				"config": cfg,
				"deploy": deployResult,
			}
			p.Print(output.Success(action, fmt.Sprintf("Deploy setup complete for %s.", proj.Name), data))
			return nil
		},
	}

	cmd.Flags().StringVar(&repository, "repository", "", "Git repository URL")
	cmd.Flags().StringVar(&branch, "branch", "", "Default branch")
	cmd.Flags().StringVar(&preset, "preset", "", "Preset: laravel, node, ruby, static, none")
	cmd.Flags().StringVar(&publicDir, "public-dir", "", "Public directory inside each release")
	cmd.Flags().IntVar(&keep, "keep", 0, "Number of releases to keep")
	cmd.Flags().BoolVar(&noFirstDeploy, "no-first-deploy", false, "Skip the optional first deploy")
	return cmd
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func prompt(label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}
