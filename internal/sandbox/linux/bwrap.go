// Package linux implements the native Linux backend using bubblewrap.
package linux

import (
	"strconv"
	"strings"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

// Args builds the bwrap argument list for spec. Order matters: later mounts
// override earlier ones, so denies follow the binds they narrow.
func Args(spec sandbox.Spec, engineerHome string, seccompFD int) []string {
	a := []string{
		"--unshare-pid", "--unshare-ipc", "--unshare-uts", "--die-with-parent",
		"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
		"--tmpfs", engineerHome,
	}
	for _, p := range spec.ReadOnly {
		a = append(a, "--ro-bind-try", p, p)
	}
	for _, p := range spec.ReadWrite {
		a = append(a, "--bind", p, p)
	}
	for _, p := range spec.Deny {
		// Re-bind the path read-only on top of the write bind. --ro-bind-try
		// tolerates repos without .git/hooks. Writes then fail with EROFS.
		a = append(a, "--ro-bind-try", p, p)
	}
	a = append(a, "--clearenv")
	for _, kv := range sandbox.EnvSlice(spec.Env) {
		k, v, _ := strings.Cut(kv, "=")
		a = append(a, "--setenv", k, v)
	}
	a = append(a, "--chdir", spec.Workdir)
	if seccompFD >= 0 {
		a = append(a, "--seccomp", strconv.Itoa(seccompFD))
	}
	return a
}
