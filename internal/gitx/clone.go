package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RefKind classifies a deploy --ref value.
type RefKind int

const (
	RefBranch RefKind = iota
	RefTag
	RefSHA
)

var shaRE = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)

// ClassifyRef guesses whether ref is a SHA, tag, or branch.
// Tags are only detected when prefixed with "tags/" or "refs/tags/".
// Otherwise non-SHA refs are treated as branch names for shallow clone.
func ClassifyRef(ref string) RefKind {
	ref = strings.TrimSpace(ref)
	if shaRE.MatchString(ref) {
		return RefSHA
	}
	if strings.HasPrefix(ref, "refs/tags/") || strings.HasPrefix(ref, "tags/") {
		return RefTag
	}
	return RefBranch
}

// NormalizeTag strips refs/tags/ or tags/ prefix.
func NormalizeTag(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "refs/tags/")
	ref = strings.TrimPrefix(ref, "tags/")
	return ref
}

// CloneOptions controls a shallow clone into an empty release directory.
type CloneOptions struct {
	Repository string
	Ref        string
	Dest       string
	SSHKey     string
	KnownHosts string // path to known_hosts file (optional)
	Env        []string
}

// CloneResult is captured after a successful checkout.
type CloneResult struct {
	SHA      string
	ShortSHA string
	Ref      string
}

// SSHCommand builds GIT_SSH_COMMAND for a deploy key.
func SSHCommand(keyPath, knownHosts string) string {
	args := []string{"ssh", "-i", keyPath, "-o", "IdentitiesOnly=yes"}
	if knownHosts != "" {
		args = append(args, "-o", "UserKnownHostsFile="+knownHosts, "-o", "StrictHostKeyChecking=yes")
	} else {
		args = append(args, "-o", "StrictHostKeyChecking=accept-new")
	}
	return strings.Join(args, " ")
}

// CloneFresh performs a shallow clone of ref into dest, returning commit metadata.
// Dest must not exist or must be an empty directory.
func CloneFresh(ctx context.Context, opts CloneOptions) (*CloneResult, error) {
	if opts.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if opts.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}
	if opts.Dest == "" {
		return nil, fmt.Errorf("destination is required")
	}
	if opts.SSHKey == "" {
		return nil, fmt.Errorf("deploy key is required")
	}

	if err := prepareDest(opts.Dest); err != nil {
		return nil, err
	}

	sshCmd := SSHCommand(opts.SSHKey, opts.KnownHosts)
	env := append(os.Environ(), opts.Env...)
	env = append(env, "GIT_SSH_COMMAND="+sshCmd)

	kind := ClassifyRef(opts.Ref)
	var err error
	switch kind {
	case RefBranch:
		err = runGit(ctx, env, opts.Dest, "clone", "--depth", "1", "--branch", opts.Ref, opts.Repository, opts.Dest)
		if err != nil {
			// Fallback: init + fetch deeper
			_ = os.RemoveAll(opts.Dest)
			if mkErr := os.MkdirAll(opts.Dest, 0o755); mkErr != nil {
				return nil, mkErr
			}
			err = cloneViaFetch(ctx, env, opts, opts.Ref, false)
		}
	case RefTag:
		tag := NormalizeTag(opts.Ref)
		err = cloneTag(ctx, env, opts, tag)
	case RefSHA:
		err = cloneViaFetch(ctx, env, opts, opts.Ref, true)
	}
	if err != nil {
		return nil, err
	}

	sha, err := revParse(ctx, env, opts.Dest, "HEAD")
	if err != nil {
		return nil, err
	}
	short := sha
	if len(short) > 7 {
		short = short[:7]
	}
	return &CloneResult{SHA: sha, ShortSHA: short, Ref: opts.Ref}, nil
}

// RemoveGitDir deletes the release .git directory after metadata is recorded.
func RemoveGitDir(releasePath string) error {
	gitDir := filepath.Join(releasePath, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("removing .git: %w", err)
	}
	return nil
}

func prepareDest(dest string) error {
	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dest, 0o755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("destination %s exists and is not a directory", dest)
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination %s is not empty", dest)
	}
	return nil
}

func cloneTag(ctx context.Context, env []string, opts CloneOptions, tag string) error {
	// Prefer single-branch shallow clone by tag name.
	err := runGit(ctx, env, opts.Dest, "clone", "--depth", "1", "--branch", tag, opts.Repository, opts.Dest)
	if err == nil {
		return nil
	}
	_ = os.RemoveAll(opts.Dest)
	if mkErr := os.MkdirAll(opts.Dest, 0o755); mkErr != nil {
		return mkErr
	}
	return cloneViaFetch(ctx, env, opts, tag, false)
}

func cloneViaFetch(ctx context.Context, env []string, opts CloneOptions, ref string, isSHA bool) error {
	if err := runGit(ctx, env, opts.Dest, "init", opts.Dest); err != nil {
		return err
	}
	if err := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "remote", "add", "origin", opts.Repository); err != nil {
		return err
	}
	fetchArgs := []string{"-C", opts.Dest, "fetch", "--depth", "1", "origin", ref}
	if err := runGit(ctx, env, opts.Dest, fetchArgs...); err != nil {
		// Deeper fallback for SHAs that are not tip-of-branch.
		if isSHA {
			if err2 := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "fetch", "--depth", "50", "origin", ref); err2 != nil {
				if err3 := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "fetch", "origin", ref); err3 != nil {
					return fmt.Errorf("fetching ref %s: %w", ref, err)
				}
			}
		} else {
			if err2 := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "fetch", "origin", "tag", ref, "--depth", "1"); err2 != nil {
				if err3 := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "fetch", "origin", ref); err3 != nil {
					return fmt.Errorf("fetching ref %s: %w", ref, err)
				}
			}
		}
	}
	// Checkout FETCH_HEAD or the ref
	if err := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "checkout", "--force", "FETCH_HEAD"); err != nil {
		if err2 := runGit(ctx, env, opts.Dest, "-C", opts.Dest, "checkout", "--force", ref); err2 != nil {
			return fmt.Errorf("checking out %s: %w", ref, err2)
		}
	}
	return nil
}

func revParse(ctx context.Context, env []string, dir, rev string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", rev)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runGit(ctx context.Context, env []string, workDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env
	// workDir is only used when not already set via -C in args; set Dir to parent for clone into dest
	if workDir != "" {
		cmd.Dir = filepath.Dir(workDir)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
