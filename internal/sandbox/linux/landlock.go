//go:build linux

package linux

import (
	"errors"
	"os"

	"github.com/landlock-lsm/go-landlock/landlock"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

// ErrLandlockUnavailable means the kernel has no Landlock support.
var ErrLandlockUnavailable = errors.New("landlock unavailable on this kernel")

// ApplyLandlock restricts the current process (and children) so writes are
// only possible under ReadWrite paths and reads under ReadOnly paths.
// It is a second layer beneath the mount namespace.
func ApplyLandlock(spec sandbox.Spec) error {
	roDirs, roFiles := classifyPaths(spec.ReadOnly, os.Stat)
	rwDirs, rwFiles := classifyPaths(spec.ReadWrite, os.Stat)
	// /proc, /dev, /tmp are fresh mounts inside the namespace.
	roDirs = append(roDirs, "/proc")
	rwDirs = append(rwDirs, "/tmp", "/dev")
	rules := []landlock.Rule{
		landlock.RODirs(roDirs...).IgnoreIfMissing(),
		landlock.ROFiles(roFiles...).IgnoreIfMissing(),
		landlock.RWDirs(rwDirs...).IgnoreIfMissing(),
		landlock.RWFiles(rwFiles...).IgnoreIfMissing(),
	}
	if err := landlock.V3.BestEffort().RestrictPaths(rules...); err != nil {
		return errors.Join(ErrLandlockUnavailable, err)
	}
	return nil
}
