//go:build linux

package linux

import (
	"errors"

	"github.com/landlock-lsm/go-landlock/landlock"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

// ErrLandlockUnavailable means the kernel has no Landlock support.
var ErrLandlockUnavailable = errors.New("landlock unavailable on this kernel")

// ApplyLandlock restricts the current process (and children) so writes are
// only possible under ReadWrite paths and reads under ReadOnly paths.
// It is a second layer beneath the mount namespace.
func ApplyLandlock(spec sandbox.Spec) error {
	var rules []landlock.Rule
	for _, p := range spec.ReadOnly {
		rules = append(rules, landlock.RODirs(p).IgnoreIfMissing())
	}
	for _, p := range spec.ReadWrite {
		rules = append(rules, landlock.RWDirs(p).IgnoreIfMissing())
	}
	// /proc, /dev, /tmp are fresh mounts inside the namespace.
	rules = append(rules, landlock.RWDirs("/tmp", "/dev").IgnoreIfMissing(), landlock.RODirs("/proc").IgnoreIfMissing())
	err := landlock.V3.BestEffort().RestrictPaths(rules...)
	if err != nil {
		return errors.Join(ErrLandlockUnavailable, err)
	}
	return nil
}
