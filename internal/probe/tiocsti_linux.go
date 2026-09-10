package probe

import (
	"os"

	"golang.org/x/sys/unix"
)

// tryTIOCSTI attempts to inject a harmless byte into the controlling terminal.
// Under Vitrine's seccomp filter this must fail with EPERM.
func tryTIOCSTI() error {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return err // no controlling tty counts as denied
	}
	defer f.Close()
	b := []byte{0}
	return unix.IoctlSetPointerInt(int(f.Fd()), unix.TIOCSTI, int(b[0]))
}
