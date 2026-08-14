package provider

import (
	"fmt"
	"net/url"
	"strings"
)

// Provider describes a git host seam for future hosts beyond GitHub.
type Provider interface {
	Name() string
	Host() string
	NormalizeRepository(raw string) (string, error)
	KnownHostsHost() string
}

// GitHub is the v1 provider.
type GitHub struct{}

func (GitHub) Name() string           { return "github" }
func (GitHub) Host() string           { return "github.com" }
func (GitHub) KnownHostsHost() string { return "github.com" }

func (GitHub) NormalizeRepository(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("repository is required")
	}
	// git@github.com:org/repo.git
	if strings.HasPrefix(raw, "git@") {
		return raw, nil
	}
	// https://github.com/org/repo(.git)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid repository URL: %w", err)
		}
		host := strings.ToLower(u.Host)
		if host != "github.com" && host != "www.github.com" {
			return "", fmt.Errorf("v1 only supports github.com repositories, got host %q", u.Host)
		}
		path := strings.Trim(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return "", fmt.Errorf("invalid GitHub repository path %q", u.Path)
		}
		return fmt.Sprintf("git@github.com:%s/%s.git", parts[0], parts[1]), nil
	}
	// org/repo shorthand
	if strings.Count(raw, "/") == 1 && !strings.Contains(raw, ":") {
		raw = strings.TrimSuffix(raw, ".git")
		parts := strings.Split(raw, "/")
		return fmt.Sprintf("git@github.com:%s/%s.git", parts[0], parts[1]), nil
	}
	return raw, nil
}

// Lookup returns a provider by name.
func Lookup(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "github":
		return GitHub{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}
