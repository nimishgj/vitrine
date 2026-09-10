// Package sandbox defines the platform-neutral sandbox spec and backend interface.
package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"sort"
)

// Spec is everything a backend needs to isolate one process.
type Spec struct {
	ReadOnly  []string          // visible read-only
	ReadWrite []string          // visible read-write
	Deny      []string          // denied even if under a ReadWrite path
	Env       map[string]string // complete environment
	Workdir   string
	Home      string // agent state home; $HOME inside
	Scratch   string // session tmp; $TMPDIR inside
}

// Backend isolates a process according to a Spec.
type Backend interface {
	Name() string
	Available() error
	Run(ctx context.Context, spec Spec, cmd []string) (exitCode int, err error)
}

// SystemRoots returns the read-only system paths for goos, or override when
// non-empty.
func SystemRoots(goos string, override []string) []string {
	if len(override) > 0 {
		return append([]string(nil), override...)
	}
	switch goos {
	case "darwin":
		return []string{
			"/usr", "/bin", "/sbin", "/System", "/Library", "/opt/homebrew", "/opt/local",
			"/private/etc", "/private/var/db/timezone",
			"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom", "/dev/tty",
		}
	default:
		return []string{
			"/usr", "/bin", "/sbin", "/lib", "/lib64",
			"/etc/resolv.conf", "/etc/hosts", "/etc/ssl", "/etc/ca-certificates",
			"/etc/passwd", "/etc/group", "/etc/localtime", "/etc/nsswitch.conf",
			"/nix", "/opt",
		}
	}
}

// ExistingRoots drops roots that are absent on this machine.
func ExistingRoots(roots []string) []string {
	var out []string
	for _, r := range roots {
		if _, err := os.Lstat(r); err == nil {
			out = append(out, r)
		}
	}
	return out
}

// BuildInput is the material BuildSpec assembles.
type BuildInput struct {
	Roots       []string
	ReadGrants  []string
	WriteGrants []string
	CredDir     string
	Home        string
	Scratch     string
	Workdir     string
	Env         map[string]string
}

// BuildSpec assembles a Spec, adding the git denies for every write grant.
func BuildSpec(in BuildInput) Spec {
	ro := append([]string{}, in.Roots...)
	ro = append(ro, in.ReadGrants...)
	if in.CredDir != "" {
		ro = append(ro, in.CredDir)
	}
	rw := append([]string{}, in.WriteGrants...)
	rw = append(rw, in.Home, in.Scratch)
	var deny []string
	for _, w := range dedup(in.WriteGrants) {
		deny = append(deny, filepath.Join(w, ".git", "config"), filepath.Join(w, ".git", "hooks"))
	}
	return Spec{
		ReadOnly:  dedup(ro),
		ReadWrite: dedup(rw),
		Deny:      dedup(deny),
		Env:       in.Env,
		Workdir:   in.Workdir,
		Home:      in.Home,
		Scratch:   in.Scratch,
	}
}

func dedup(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
