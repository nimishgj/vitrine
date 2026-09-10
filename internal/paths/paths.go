// Package paths defines the on-disk layout of Vitrine's state directory.
package paths

import (
	"os"
	"path/filepath"
)

// Layout is the set of state file locations for one engineer.
type Layout struct {
	Root      string // ~/.vitrine
	Config    string // config.toml
	Grants    string // grants.json
	Audit     string // audit.jsonl
	AuditHead string // audit.head
	Bin       string // shim directory
	Agents    string // per-agent state homes (Linux)
}

// Resolve returns the layout rooted at home/.vitrine.
func Resolve(home string) Layout {
	root := filepath.Join(home, ".vitrine")
	return Layout{
		Root:      root,
		Config:    filepath.Join(root, "config.toml"),
		Grants:    filepath.Join(root, "grants.json"),
		Audit:     filepath.Join(root, "audit.jsonl"),
		AuditHead: filepath.Join(root, "audit.head"),
		Bin:       filepath.Join(root, "bin"),
		Agents:    filepath.Join(root, "agents"),
	}
}

// Default resolves the layout for the current user.
func Default() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, err
	}
	return Resolve(home), nil
}

// AgentHome is the state home for an agent profile under the Linux layout.
// macOS uses sysuser.AgentHome instead (owned by the vitrine user).
func AgentHome(l Layout, name string) string {
	return filepath.Join(l.Agents, name, "home")
}

// EnsureRoot creates the root and bin/agents directories with 0700.
func EnsureRoot(l Layout) error {
	for _, d := range []string{l.Root, l.Bin, l.Agents} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}
