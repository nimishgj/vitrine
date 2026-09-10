// Package profile describes the agents Vitrine knows how to launch.
package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile is what Vitrine needs to know about one agent.
type Profile struct {
	Name      string   // "claude", "codex", "pi", or basename for generic
	Binary    string   // name to look up on PATH, or absolute path for generic
	StateDirs []string // relative to the sandbox $HOME; persist across sessions
	Generic   bool
}

// Builtin returns the first-class agent profiles.
func Builtin() []Profile {
	return []Profile{
		{Name: "claude", Binary: "claude", StateDirs: []string{".claude", ".claude.json"}},
		{Name: "codex", Binary: "codex", StateDirs: []string{".codex"}},
		{Name: "pi", Binary: "pi", StateDirs: []string{".pi"}},
	}
}

// Names lists builtin profile names in order.
func Names() []string {
	var out []string
	for _, p := range Builtin() {
		out = append(out, p.Name)
	}
	return out
}

// Lookup finds a builtin profile, applying a binary override if present.
func Lookup(name string, overrides map[string]string) (Profile, bool) {
	for _, p := range Builtin() {
		if p.Name == name {
			if b, ok := overrides[name]; ok && b != "" {
				p.Binary = b
			}
			return p, true
		}
	}
	return Profile{}, false
}

// Generic builds a profile for an arbitrary command.
func Generic(cmd string) Profile {
	return Profile{Name: filepath.Base(cmd), Binary: cmd, Generic: true}
}

// FromArgv0 reports whether argv0's basename names a builtin profile.
func FromArgv0(argv0 string) (string, bool) {
	base := filepath.Base(argv0)
	for _, p := range Builtin() {
		if p.Name == base {
			return p.Name, true
		}
	}
	return "", false
}

// FindRealBinary searches pathEnv for an executable named name, skipping skipDir.
func FindRealBinary(name string, pathEnv string, skipDir string) (string, error) {
	if strings.Contains(name, string(filepath.Separator)) {
		if isExec(name) {
			return name, nil
		}
		return "", fmt.Errorf("%s is not executable", name)
	}
	skip := filepath.Clean(skipDir)
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" || filepath.Clean(dir) == skip {
			continue
		}
		cand := filepath.Join(dir, name)
		if isExec(cand) {
			return cand, nil
		}
	}
	return "", errors.New("agent binary " + name + " not found on PATH (outside the vitrine shim directory)")
}

func isExec(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode().Perm()&0o111 != 0
}
