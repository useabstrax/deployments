package key

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultPath returns the conventional deploy key path for a project user.
func DefaultPath(homeDir, projectName string) string {
	safe := strings.ReplaceAll(projectName, ".", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	return filepath.Join(homeDir, ".ssh", "abstrax_deploy_"+safe)
}

// EnsureDir creates the .ssh directory with 0700 if needed.
func EnsureDir(keyPath string) error {
	dir := filepath.Dir(keyPath)
	return os.MkdirAll(dir, 0o700)
}

// Generate creates an ed25519 key pair at keyPath (private) and keyPath.pub.
func Generate(keyPath, comment string) error {
	if err := EnsureDir(keyPath); err != nil {
		return err
	}
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("deploy key already exists at %s (use --rotate to replace)", keyPath)
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", comment)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	_ = os.Chmod(keyPath, 0o600)
	_ = os.Chmod(keyPath+".pub", 0o644)
	return nil
}

// Rotate removes existing key pair and generates a new one.
func Rotate(keyPath, comment string) error {
	_ = os.Remove(keyPath)
	_ = os.Remove(keyPath + ".pub")
	return Generate(keyPath, comment)
}

// PublicKey reads the .pub file contents.
func PublicKey(keyPath string) (string, error) {
	data, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Fingerprint returns the SHA256 fingerprint of the public key.
func Fingerprint(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath+".pub", "-E", "sha256")
	out, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			return fields[1], nil
		}
		return strings.TrimSpace(string(out)), nil
	}
	// Fallback: hash raw key material
	pub, err2 := PublicKey(keyPath)
	if err2 != nil {
		return "", err
	}
	parts := strings.Fields(pub)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid public key")
	}
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

// EnsureGitHubKnownHosts appends github.com host keys to known_hosts if missing.
func EnsureGitHubKnownHosts(knownHostsPath string) error {
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0o700); err != nil {
		return err
	}
	existing, _ := os.ReadFile(knownHostsPath)
	if bytes.Contains(existing, []byte("github.com")) {
		return nil
	}
	cmd := exec.Command("ssh-keyscan", "-t", "rsa,ecdsa,ed25519", "github.com")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan github.com: %w", err)
	}
	f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		if _, err := f.Write([]byte("\n")); err != nil {
			return err
		}
	}
	_, err = f.Write(out)
	return err
}

// GitHubDeployKeyInstructions returns human help for adding the key on GitHub.
func GitHubDeployKeyInstructions(repo, pubkey string) string {
	var b strings.Builder
	b.WriteString("Add this public key as a GitHub Deploy Key (read-only is enough for deploys):\n\n")
	b.WriteString(pubkey)
	b.WriteString("\n\n")
	if repo != "" {
		// Best-effort URL hint
		b.WriteString("GitHub → repository Settings → Deploy keys → Add deploy key\n")
		b.WriteString("Repository: ")
		b.WriteString(repo)
		b.WriteString("\n")
	} else {
		b.WriteString("GitHub → repository Settings → Deploy keys → Add deploy key\n")
	}
	return b.String()
}
