package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/key"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newRemoveCmd() *cobra.Command {
	var (
		removeShared     bool
		removeKey        bool
		restorePublicDir string
	)

	cmd := &cobra.Command{
		Use:   "remove <project>",
		Short: "Remove deploy layout and restore the project to a normal Abstrax project",
		Long: `Tear down per-project deploy state created by setup/init.

Removes deploy.json, the current symlink, and releases/. Shared data and the
local deploy key are kept unless --remove-shared / --remove-key are passed.

When setup/init stored previous_public_dir (or --restore-public-dir is set),
the project public directory is restored before the layout is deleted.

Removing the deploy key from GitHub/GitLab remains Console-owned.
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			p := printer()
			action := "deploy.remove"

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
				return fmt.Errorf("deploy.json not found; nothing to remove for %s", projectName)
			}
			cfg, err := config.Load(proj.Path)
			if err != nil {
				return err
			}

			if !globals.Yes && !globals.DryRun {
				msg := fmt.Sprintf("Remove deploy layout for %s? (releases, current, deploy.json)", proj.Name)
				if removeShared {
					msg += " Also deletes shared/."
				}
				if removeKey {
					msg += " Also deletes the local deploy key."
				}
				ok, err := confirm(msg)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted")
				}
			}

			paths := layout.ForProject(proj.Path)
			home, homeErr := userx.LookupHome(proj.User)
			keyPath := cfg.DeployKey
			if keyPath == "" && homeErr == nil {
				keyPath = key.DefaultPath(home, proj.Name)
			}

			restoreTo := strings.TrimSpace(restorePublicDir)
			if restoreTo == "" {
				restoreTo = strings.TrimSpace(cfg.PreviousPublicDir)
			}

			if globals.DryRun {
				if restoreTo != "" {
					p.DryRun("would restore project public-dir to %s", restoreTo)
				} else {
					p.DryRun("would skip public-dir restore (no previous_public_dir; pass --restore-public-dir)")
				}
				p.DryRun("would remove %s", paths.Current)
				p.DryRun("would remove %s", filepath.Join(proj.Path, "current.tmp"))
				p.DryRun("would remove %s", paths.Releases)
				p.DryRun("would remove %s", config.Path(proj.Path))
				if removeShared {
					p.DryRun("would remove %s", paths.Shared)
				} else {
					p.DryRun("would keep %s", paths.Shared)
				}
				if removeKey && keyPath != "" {
					p.DryRun("would remove deploy key %s (+ .pub)", keyPath)
				} else {
					p.DryRun("would keep local deploy key")
				}
				p.Print(output.Success(action, "Dry run complete.", map[string]interface{}{
					"project":            proj.Name,
					"restore_public_dir": restoreTo,
					"remove_shared":      removeShared,
					"remove_key":         removeKey,
					"key_path":           keyPath,
				}))
				return nil
			}

			if restoreTo != "" {
				p.Progress(action, "web_root", "Restoring public-dir to "+restoreTo)
				if err := client.SetProjectPublicDir(cmd.Context(), proj.Name, restoreTo); err != nil {
					return fmt.Errorf("restore public-dir: %w", err)
				}
			} else {
				p.Warn("no previous_public_dir stored; pass --restore-public-dir to restore the project web root")
			}

			p.Progress(action, "layout", "Removing deploy layout")
			removed, kept, err := removeDeployArtifacts(proj.Path, keyPath, removeShared, removeKey)
			if err != nil {
				return err
			}

			data := map[string]interface{}{
				"project":            proj.Name,
				"removed":            removed,
				"kept":               kept,
				"restore_public_dir": restoreTo,
				"remove_shared":      removeShared,
				"remove_key":         removeKey,
			}

			p.Print(output.Success(action, fmt.Sprintf("Deploy removed for %s.", proj.Name), data))
			return nil
		},
	}

	cmd.Flags().BoolVar(&removeShared, "remove-shared", false, "Also delete shared/ (env, uploads, etc.)")
	cmd.Flags().BoolVar(&removeKey, "remove-key", false, "Also delete the local deploy key pair (not the GitHub/GitLab deploy key)")
	cmd.Flags().StringVar(&restorePublicDir, "restore-public-dir", "", "Public directory to restore (defaults to previous_public_dir from setup/init)")
	return cmd
}

// removeDeployArtifacts deletes deploy layout paths. current is removed via
// os.Remove so the symlink is not followed into a release directory.
func removeDeployArtifacts(projectPath, keyPath string, removeShared, removeKey bool) (removed, kept []string, err error) {
	paths := layout.ForProject(projectPath)

	for _, path := range []string{
		paths.Current,
		filepath.Join(projectPath, "current.tmp"),
		paths.Releases,
		config.Path(projectPath),
	} {
		existed := pathExists(path)
		if err := removePath(path); err != nil {
			return removed, kept, err
		}
		if existed {
			removed = append(removed, path)
		}
	}

	if removeShared {
		existed := pathExists(paths.Shared)
		if err := removePath(paths.Shared); err != nil {
			return removed, kept, err
		}
		if existed {
			removed = append(removed, paths.Shared)
		}
	} else if pathExists(paths.Shared) {
		kept = append(kept, paths.Shared)
	}

	if removeKey {
		if keyPath == "" {
			return removed, kept, nil
		}
		if err := validateDeployKeyPath(keyPath); err != nil {
			return removed, kept, err
		}
		for _, path := range []string{keyPath, keyPath + ".pub"} {
			existed := pathExists(path)
			if err := removePath(path); err != nil {
				return removed, kept, err
			}
			if existed {
				removed = append(removed, path)
			}
		}
	} else if keyPath != "" && pathExists(keyPath) {
		kept = append(kept, keyPath)
	}

	return removed, kept, nil
}

func removePath(path string) error {
	if path == "" {
		return nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// validateDeployKeyPath refuses to delete paths that are not Abstrax deploy keys.
func validateDeployKeyPath(keyPath string) error {
	base := filepath.Base(keyPath)
	if !strings.HasPrefix(base, "abstrax_deploy_") {
		return fmt.Errorf("refusing to remove key %s: expected abstrax_deploy_* filename", keyPath)
	}
	dir := filepath.Base(filepath.Dir(keyPath))
	if dir != ".ssh" {
		return fmt.Errorf("refusing to remove key %s: expected path under .ssh/", keyPath)
	}
	return nil
}

// isDeployPublicDir reports whether publicDir already points through current/.
func isDeployPublicDir(publicDir string) bool {
	publicDir = filepath.ToSlash(strings.TrimSpace(publicDir))
	return publicDir == layout.CurrentLink || strings.HasPrefix(publicDir, layout.CurrentLink+"/")
}

// capturePreviousPublicDir stores the pre-deploy public dir on cfg when missing.
func capturePreviousPublicDir(cfg *config.Config, current string) {
	if cfg == nil || cfg.PreviousPublicDir != "" {
		return
	}
	current = strings.TrimSpace(current)
	if current == "" || isDeployPublicDir(current) {
		return
	}
	cfg.PreviousPublicDir = current
}
