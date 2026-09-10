//go:build linux

package linux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

// Backend is the native Linux sandbox.
type Backend struct {
	EngineerHome string
	VitrineBin   string // absolute path of the vitrine binary, for the inner launcher
}

// New returns a Backend for the engineer's home directory.
func New(engineerHome string) *Backend {
	self, _ := os.Executable()
	return &Backend{EngineerHome: engineerHome, VitrineBin: self}
}

func (b *Backend) Name() string { return "native-linux" }

// Available checks bwrap and unprivileged user namespaces.
func (b *Backend) Available() error {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return errors.New("bubblewrap (bwrap) not found; install the bubblewrap package")
	}
	if v, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone"); err == nil && strings.TrimSpace(string(v)) == "0" {
		return errors.New("unprivileged user namespaces disabled; run: sudo sysctl kernel.unprivileged_userns_clone=1")
	}
	if v, err := os.ReadFile("/proc/sys/kernel/apparmor_restrict_unprivileged_userns"); err == nil && strings.TrimSpace(string(v)) == "1" {
		// Ubuntu 24.04+. bwrap ships an AppArmor profile on newer releases; if
		// the probe below fails, the one-line fix is printed.
		if err := exec.Command("bwrap", "--ro-bind", "/", "/", "true").Run(); err != nil {
			return errors.New("AppArmor restricts unprivileged user namespaces; run: sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0 (or install a bwrap AppArmor profile)")
		}
	}
	return nil
}

// Run launches cmd under bwrap. The inner process is `vitrine _inner`, which
// applies Landlock and execs the real command.
func (b *Backend) Run(ctx context.Context, spec sandbox.Spec, cmd []string) (int, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return 1, err
	}
	if _, err := w.Write(SeccompProgram()); err != nil {
		return 1, err
	}
	w.Close()
	defer r.Close()

	specR, specW, err := os.Pipe()
	if err != nil {
		return 1, err
	}
	go func() {
		_ = sandbox.WriteSpec(specW, spec)
		specW.Close()
	}()
	defer specR.Close()

	// ExtraFiles[0] is fd 3 (seccomp program), ExtraFiles[1] is fd 4 (spec).
	args := Args(spec, b.EngineerHome, 3)
	inner := []string{b.VitrineBin, "_inner", "--spec-fd", "4", "--"}
	full := append(append(args, "--"), append(inner, cmd...)...)
	c := exec.CommandContext(ctx, "bwrap", full...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.ExtraFiles = []*os.File{r, specR}
	c.Env = []string{}
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode(), nil
		}
		return 1, fmt.Errorf("bwrap: %w", err)
	}
	return 0, nil
}
