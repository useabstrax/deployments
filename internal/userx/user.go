package userx

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// RequireRoot returns an error when not running as root.
func RequireRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command requires root (try sudo)")
	}
	return nil
}

// LookupHome returns the home directory for username.
func LookupHome(username string) (string, error) {
	if username == "" || username == "www-data" || username == "nginx" || username == "apache" {
		// Shared ownership: use /var/www or fall back
		if st, err := os.Stat("/var/www"); err == nil && st.IsDir() {
			return "/var/www", nil
		}
		return "/var/www", nil
	}
	u, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("looking up user %q: %w", username, err)
	}
	return u.HomeDir, nil
}

// ChownPath recursively sets ownership to username when running as root.
func ChownPath(path, username string) error {
	if os.Geteuid() != 0 || username == "" {
		return nil
	}
	u, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(p, uid, gid)
	})
}

// RunAs builds an optional runuser wrapper. Empty when not root or shared user.
func RunAs(username string) string {
	if os.Geteuid() != 0 {
		return ""
	}
	if username == "" || username == "www-data" || username == "nginx" || username == "apache" {
		return ""
	}
	return username
}

// EnsureSSHDir creates ~/.ssh for the project user.
func EnsureSSHDir(home, username string) (string, error) {
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", err
	}
	_ = ChownPath(sshDir, username)
	return sshDir, nil
}

// Which reports whether a binary exists on PATH.
func Which(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// IsSharedOwner reports whether the project user is a shared web user.
func IsSharedOwner(username string) bool {
	u := strings.ToLower(strings.TrimSpace(username))
	return u == "" || u == "www-data" || u == "nginx" || u == "apache"
}
