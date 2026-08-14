package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
)

// Meta is stored as releases/{id}/.abstrax-release.json before .git is removed.
type Meta struct {
	ID        string    `json:"id"`
	Ref       string    `json:"ref"`
	SHA       string    `json:"sha,omitempty"`
	ShortSHA  string    `json:"short_sha,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// WriteMeta writes release metadata next to the release tree.
func WriteMeta(releasePath string, meta Meta) error {
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(releasePath, layout.ReleaseMeta)
	return os.WriteFile(path, data, 0o644)
}

// ReadMeta loads release metadata if present.
func ReadMeta(releasePath string) (*Meta, error) {
	path := filepath.Join(releasePath, layout.ReleaseMeta)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid release metadata: %w", err)
	}
	return &meta, nil
}

// ListEntry is one release for list/status output.
type ListEntry struct {
	ID        string    `json:"id"`
	Current   bool      `json:"current"`
	Ref       string    `json:"ref,omitempty"`
	SHA       string    `json:"sha,omitempty"`
	ShortSHA  string    `json:"short_sha,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Path      string    `json:"path"`
}
