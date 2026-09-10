// Package grants stores the engineer's explicit filesystem grants.
package grants

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Mode is read or write. Write implies read.
type Mode string

const (
	ModeRead  Mode = "read"
	ModeWrite Mode = "write"
)

// Grant is one explicit allowance on a canonical absolute path.
type Grant struct {
	Path      string    `json:"path"`
	Mode      Mode      `json:"mode"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"` // "prompt" or "cli"
}

// Store is the on-disk grants file.
type Store struct {
	Version int     `json:"version"`
	Grants  []Grant `json:"grants"`
}

// Load reads the store; a missing file yields an empty version-1 store.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Store{Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return &s, nil
}

// Save writes the store with mode 0600, sorted by path for stable hashing.
func Save(path string, s *Store) error {
	sort.Slice(s.Grants, func(i, j int) bool { return s.Grants[i].Path < s.Grants[j].Path })
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// FileHash returns the sha256 hex of the file, or "" if it does not exist.
func FileHash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Match returns the grant whose path is the nearest ancestor of (or equal to) p.
func (s *Store) Match(p string) (Grant, bool) {
	var best Grant
	found := false
	for _, g := range s.Grants {
		if !isUnder(p, g.Path) {
			continue
		}
		if !found || len(g.Path) > len(best.Path) {
			best, found = g, true
		}
	}
	return best, found
}

// Add inserts g, replacing any grant with the same path.
func (s *Store) Add(g Grant) error {
	if !filepath.IsAbs(g.Path) {
		return errors.New("grant path must be absolute")
	}
	if g.Mode != ModeRead && g.Mode != ModeWrite {
		return errors.New("grant mode must be read or write")
	}
	s.Remove(g.Path)
	s.Grants = append(s.Grants, g)
	return nil
}

// Remove deletes the grant at exactly path. Returns whether one existed.
func (s *Store) Remove(path string) bool {
	for i, g := range s.Grants {
		if g.Path == path {
			s.Grants = append(s.Grants[:i], s.Grants[i+1:]...)
			return true
		}
	}
	return false
}

// Canonical makes p absolute and resolves symlinks. A path that does not
// exist is returned cleaned and absolute without symlink resolution.
func Canonical(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	real, err := filepath.EvalSymlinks(abs)
	if errors.Is(err, os.ErrNotExist) {
		return abs, nil
	}
	if err != nil {
		return "", err
	}
	return real, nil
}

// isUnder reports whether p equals base or is inside it (separator-aware).
func isUnder(p, base string) bool {
	if p == base {
		return true
	}
	if base == string(filepath.Separator) {
		return true
	}
	return strings.HasPrefix(p, base+string(filepath.Separator))
}
