//go:build darwin

package darwin

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nimishgj/vitrine/internal/sandbox"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

// Launch runs as the vitrine user: reads the spec, writes the profile next to
// it, and execs sandbox-exec with the explicit environment.
func Launch(specPath string, cmd []string) error {
	if !isVitrineUser() {
		return errors.New("launch must run as the vitrine user")
	}
	f, err := os.Open(specPath)
	if err != nil {
		return err
	}
	spec, err := sandbox.ReadSpec(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	if err := os.MkdirAll(spec.Home, 0o700); err != nil {
		return err
	}
	profilePath := filepath.Join(filepath.Dir(specPath), "profile.sb")
	if err := os.WriteFile(profilePath, []byte(Profile(spec)), 0o600); err != nil {
		return err
	}
	if err := os.Chdir(spec.Workdir); err != nil {
		return fmt.Errorf("chdir %s: %w", spec.Workdir, err)
	}
	argv := append([]string{"/usr/bin/sandbox-exec", "-f", profilePath}, cmd...)
	return syscall.Exec(argv[0], argv, sandbox.EnvSlice(spec.Env))
}

func isVitrineUser() bool {
	u, err := user.LookupId(strconv.Itoa(os.Getuid()))
	return err == nil && u.Username == sysuser.Name
}
