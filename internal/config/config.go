// Package config reads and writes ~/.vitrine/config.toml.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/grants"
)

// Target is a remote system a provider can mint credentials for.
type Target struct {
	Name     string            `toml:"name"`
	Kind     string            `toml:"kind"`
	Settings map[string]string `toml:"settings,omitempty"`
}

// ProfileOverride lets config change an agent profile's binary name.
type ProfileOverride struct {
	Binary string `toml:"binary,omitempty"`
}

// Config is the whole file.
type Config struct {
	Version     int                        `toml:"version"`
	Backend     string                     `toml:"backend"`
	SystemRoots []string                   `toml:"system_roots,omitempty"`
	Targets     []Target                   `toml:"targets,omitempty"`
	Profiles    map[string]ProfileOverride `toml:"profiles,omitempty"`
}

// Default is the config of a fresh install.
func Default() *Config {
	return &Config{Version: 1, Backend: "native"}
}

// Load parses the file, returning Default() if it does not exist.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	c := Default()
	if err := toml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Backend == "" {
		c.Backend = "native"
	}
	return c, nil
}

// Save writes the file with 0600 and records its hash in the audit log.
func Save(path string, c *Config, l *audit.Log) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	h, err := grants.FileHash(path)
	if err != nil {
		return err
	}
	_, err = l.Append(audit.TypeConfigWrite, map[string]any{"config_hash": h})
	return err
}

// AddTarget appends t; names must be unique.
func (c *Config) AddTarget(t Target) error {
	if t.Name == "" || t.Kind == "" {
		return errors.New("target needs a name and a kind")
	}
	for _, x := range c.Targets {
		if x.Name == t.Name {
			return fmt.Errorf("target %q already exists", t.Name)
		}
	}
	c.Targets = append(c.Targets, t)
	return nil
}

// RemoveTarget deletes the named target.
func (c *Config) RemoveTarget(name string) bool {
	for i, x := range c.Targets {
		if x.Name == name {
			c.Targets = append(c.Targets[:i], c.Targets[i+1:]...)
			return true
		}
	}
	return false
}
