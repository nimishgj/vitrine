package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/profile"
	"github.com/nimishgj/vitrine/internal/shim"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

func newInitCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up Vitrine for this user (run with sudo on macOS the first time)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if e.GOOS == "darwin" && os.Geteuid() == 0 {
				return initPrivileged(e)
			}
			return initUnprivileged(e)
		},
	}
}

// initPrivileged runs as root on macOS: create the vitrine user, then re-run
// the unprivileged half as the invoking engineer.
func initPrivileged(e *env) error {
	engineer := os.Getenv("SUDO_USER")
	if engineer == "" {
		return fmt.Errorf("run as: sudo vitrine init (SUDO_USER not set)")
	}
	if err := sysuser.Ensure(e.Exe, []string{engineer}); err != nil {
		return err
	}
	fmt.Fprintf(e.Stdout, "created user %q and sudoers rule at %s\n", sysuser.Name, sysuser.SudoersPath)
	c := exec.Command("sudo", "-n", "-u", engineer, "-H", e.Exe, "init")
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, e.Stdout, e.Stderr
	return c.Run()
}

func initUnprivileged(e *env) error {
	if err := paths.EnsureRoot(e.Layout); err != nil {
		return err
	}
	if err := shim.Install(e.Layout.Bin, e.Exe, profile.Names()); err != nil {
		return err
	}
	if _, err := os.Stat(e.Layout.Config); os.IsNotExist(err) {
		if err := config.Save(e.Layout.Config, config.Default(), e.Log); err != nil {
			return err
		}
	}
	if _, err := e.Log.Append(audit.TypeInit, map[string]any{"platform": e.GOOS, "version": Version, "exe": e.Exe}); err != nil {
		return err
	}
	fmt.Fprintf(e.Stdout, "vitrine is set up in %s\n\nAdd the shim directory to the front of your PATH:\n\n  export PATH=%q:$PATH\n\nThen `claude`, `codex`, and `pi` run inside Vitrine automatically.\n", e.Layout.Root, e.Layout.Bin)
	if e.GOOS == "darwin" && !sysuser.Exists() {
		fmt.Fprintf(e.Stdout, "\nThe %q user is not created yet. Run once:\n\n  sudo vitrine init\n", sysuser.Name)
	}
	return nil
}
