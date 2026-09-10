//go:build darwin

package darwin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nimishgj/vitrine/internal/sandbox"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

// Backend is the native macOS sandbox: separate user + Seatbelt.
type Backend struct {
	EngineerHome string
	VitrineBin   string
}

// New returns a Backend for the engineer's home directory.
func New(engineerHome string) *Backend {
	self, _ := os.Executable()
	if real, err := filepath.EvalSymlinks(self); err == nil {
		self = real
	}
	return &Backend{EngineerHome: engineerHome, VitrineBin: self}
}

func (b *Backend) Name() string { return "native-darwin" }

// Available checks sandbox-exec, the vitrine user, and the sudoers rule.
func (b *Backend) Available() error {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return errors.New("sandbox-exec not found")
	}
	if err := exec.Command("id", "-u", sysuser.Name).Run(); err != nil {
		return fmt.Errorf("user %q does not exist; run: sudo vitrine init", sysuser.Name)
	}
	if err := exec.Command("sudo", "-n", "-u", sysuser.Name, b.VitrineBin, "launch", "--check").Run(); err != nil {
		return fmt.Errorf("sudo rule for %q missing or stale; run: sudo vitrine init", sysuser.Name)
	}
	return nil
}

// Run writes the spec where the vitrine user can read it, then launches
// `vitrine launch` as that user via sudo. Credential and scratch dirs are
// re-parented under /var/lib/vitrine/sessions so the vitrine user owns them.
func (b *Backend) Run(ctx context.Context, spec sandbox.Spec, cmd []string) (int, error) {
	sessDir, err := os.MkdirTemp(sysuser.SessionsDir(), "s-")
	if err != nil {
		return 1, fmt.Errorf("create session dir under %s (run: sudo vitrine init): %w", sysuser.SessionsDir(), err)
	}
	defer os.RemoveAll(sessDir)

	spec, err = relocate(spec, sessDir)
	if err != nil {
		return 1, err
	}
	specPath := filepath.Join(sessDir, "spec.json")
	f, err := os.OpenFile(specPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return 1, err
	}
	if err := sandbox.WriteSpec(f, spec); err != nil {
		f.Close()
		return 1, err
	}
	f.Close()
	if err := b.adopt(sessDir); err != nil {
		return 1, err
	}

	args := append([]string{"-n", "-u", sysuser.Name, b.VitrineBin, "launch", specPath, "--"}, cmd...)
	c := exec.CommandContext(ctx, "sudo", args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode(), nil
		}
		return 1, fmt.Errorf("sudo launch: %w", err)
	}
	return 0, nil
}

// isSessionCreds reports whether p looks like <...>/sessions/<id>/creds.
func isSessionCreds(p string) bool {
	return filepath.Base(p) == "creds" && filepath.Base(filepath.Dir(filepath.Dir(p))) == "sessions"
}

// relocate copies the credential directory and repoints scratch so both live
// under sessDir; the originals under ~/.vitrine stay unreadable by the agent.
func relocate(spec sandbox.Spec, sessDir string) (sandbox.Spec, error) {
	newScratch := filepath.Join(sessDir, "scratch")
	if err := os.MkdirAll(newScratch, 0o700); err != nil {
		return spec, err
	}
	newCreds := filepath.Join(sessDir, "creds")
	oldCreds := ""
	var ro []string
	for _, p := range spec.ReadOnly {
		if isSessionCreds(p) {
			if err := copyTree(p, newCreds); err != nil {
				return spec, err
			}
			oldCreds = p
			ro = append(ro, newCreds)
			continue
		}
		ro = append(ro, p)
	}
	var rw []string
	for _, p := range spec.ReadWrite {
		if p == spec.Scratch {
			rw = append(rw, newScratch)
			continue
		}
		rw = append(rw, p)
	}
	env := map[string]string{}
	for k, v := range spec.Env {
		v = strings.ReplaceAll(v, spec.Scratch, newScratch)
		if oldCreds != "" {
			v = strings.ReplaceAll(v, oldCreds, newCreds)
		}
		env[k] = v
	}
	spec.ReadOnly, spec.ReadWrite, spec.Env, spec.Scratch = ro, rw, env, newScratch
	return spec, nil
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o400)
	})
}

// adopt hands sessDir to the vitrine user. The engineer cannot chown to
// another user without root, so this goes through the same sudo rule using
// the hidden `launch --adopt` mode, which chowns its argument to itself.
func (b *Backend) adopt(dir string) error {
	c := exec.Command("sudo", "-n", "-u", sysuser.Name, b.VitrineBin, "launch", "--adopt", dir)
	c.Stderr = os.Stderr
	return c.Run()
}
