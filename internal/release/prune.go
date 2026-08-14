package release

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Prune deletes old releases, keeping keep newest by directory name (timestamp id).
// Never deletes the release that current points at.
func Prune(projectPath string, keep int) ([]string, error) {
	if keep < 1 {
		return nil, fmt.Errorf("keep must be >= 1")
	}
	releasesDir := filepath.Join(projectPath, "releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	currentTarget, _ := os.Readlink(filepath.Join(projectPath, "current"))
	currentID := ""
	if currentTarget != "" {
		currentID = filepath.Base(currentTarget)
	}

	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		ids = append(ids, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids))) // newest timestamp first

	var kept []string
	var removed []string
	for _, id := range ids {
		if id == currentID || len(kept) < keep {
			kept = append(kept, id)
			continue
		}
		path := filepath.Join(releasesDir, id)
		if err := os.RemoveAll(path); err != nil {
			return removed, fmt.Errorf("pruning release %s: %w", id, err)
		}
		removed = append(removed, id)
	}
	return removed, nil
}

// ListReleases returns release entries sorted newest-first.
func ListReleases(projectPath string) ([]ListEntry, error) {
	releasesDir := filepath.Join(projectPath, "releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	currentTarget, _ := os.Readlink(filepath.Join(projectPath, "current"))
	currentID := ""
	if currentTarget != "" {
		currentID = filepath.Base(currentTarget)
	}

	var out []ListEntry
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(releasesDir, e.Name())
		entry := ListEntry{
			ID:      e.Name(),
			Current: e.Name() == currentID,
			Path:    path,
		}
		if meta, err := ReadMeta(path); err == nil && meta != nil {
			entry.Ref = meta.Ref
			entry.SHA = meta.SHA
			entry.ShortSHA = meta.ShortSHA
			entry.CreatedAt = meta.CreatedAt
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// PreviousReleaseID returns the release immediately older than current, or "".
func PreviousReleaseID(projectPath string) (string, error) {
	list, err := ListReleases(projectPath)
	if err != nil {
		return "", err
	}
	var foundCurrent bool
	for _, e := range list {
		if e.Current {
			foundCurrent = true
			continue
		}
		if foundCurrent {
			return e.ID, nil
		}
	}
	// If current is not first (shouldn't happen with sort), pick first non-current.
	for _, e := range list {
		if !e.Current {
			return e.ID, nil
		}
	}
	return "", fmt.Errorf("no previous release available")
}
