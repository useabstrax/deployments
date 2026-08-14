package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LinkShared creates symlinks from release paths into project shared/.
// For each shared entry "foo/bar":
//   - ensures shared/foo/bar exists (directory, or empty file for dotted leaf files like .env)
//   - replaces release/foo/bar with a symlink to shared/foo/bar
func LinkShared(projectPath, releasePath string, shared []string) error {
	sharedRoot := filepath.Join(projectPath, "shared")
	for _, entry := range shared {
		entry = strings.TrimSpace(entry)
		entry = strings.TrimPrefix(entry, "./")
		if entry == "" || entry == "." || strings.Contains(entry, "..") {
			return fmt.Errorf("invalid shared path %q", entry)
		}
		if filepath.IsAbs(entry) {
			return fmt.Errorf("shared path must be relative: %q", entry)
		}

		sharedTarget := filepath.Join(sharedRoot, entry)
		releaseLink := filepath.Join(releasePath, entry)

		if err := ensureSharedTarget(sharedTarget, entry); err != nil {
			return err
		}

		// Remove existing file/dir/symlink in the release so we can link.
		if err := os.RemoveAll(releaseLink); err != nil {
			return fmt.Errorf("preparing shared link %s: %w", entry, err)
		}
		if err := os.MkdirAll(filepath.Dir(releaseLink), 0o755); err != nil {
			return err
		}

		rel, err := filepath.Rel(filepath.Dir(releaseLink), sharedTarget)
		if err != nil {
			rel = sharedTarget
		}
		if err := os.Symlink(rel, releaseLink); err != nil {
			return fmt.Errorf("linking shared %s: %w", entry, err)
		}
	}
	return nil
}

func ensureSharedTarget(sharedTarget, entry string) error {
	if _, err := os.Lstat(sharedTarget); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(sharedTarget), 0o755); err != nil {
		return err
	}

	base := filepath.Base(entry)
	// Treat leaf names that look like files (contain a dot) as files; else directories.
	if strings.Contains(base, ".") {
		f, err := os.OpenFile(sharedTarget, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		return f.Close()
	}
	return os.MkdirAll(sharedTarget, 0o755)
}
