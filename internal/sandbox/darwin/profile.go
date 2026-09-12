package darwin

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

func q(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

// realPath resolves symlinks in p. Seatbelt matches rules against real
// paths, and on macOS /var, /tmp and /etc are symlinks into /private, so a
// rule written with the symlinked path never matches. Paths that do not
// exist yet are resolved through their nearest existing ancestor.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	var tail []string
	cur := filepath.Clean(p)
	for cur != string(filepath.Separator) {
		parent, base := filepath.Split(cur)
		parent = filepath.Clean(parent)
		if _, err := os.Lstat(cur); err == nil {
			break
		}
		tail = append([]string{base}, tail...)
		cur = parent
	}
	r, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return p
	}
	return filepath.Join(append([]string{r}, tail...)...)
}

// Profile renders the SBPL profile for spec. Later rules win, so denies are
// emitted after the allows they narrow.
func Profile(spec sandbox.Spec) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	b.WriteString("(allow process-exec* process-fork signal sysctl-read)\n")
	b.WriteString("(allow file-read-metadata)\n") // stat() anywhere; content still denied
	// Every process reads the root directory and touches /dev/dtracehelper
	// during libSystem initialisation; without these it aborts before main.
	b.WriteString("(allow file-read-data (literal \"/\"))\n")
	b.WriteString("(allow file-read* file-write* file-ioctl (literal \"/dev/dtracehelper\"))\n")
	// Interactive agents put the terminal into raw mode and query its size
	// (TIOCSETA, TIOCGWINSZ) on the pty device itself, not on /dev/tty.
	b.WriteString("(allow file-ioctl (literal \"/dev/tty\") (regex #\"^/dev/ttys[0-9]+$\"))\n")
	// Shared temp locations. Agents hardcode /tmp/<name>-<uid> regardless of
	// TMPDIR, and macOS creates a per-user tree under /var/folders. Both are
	// protected by ordinary per-user permissions, so the engineer's own temp
	// files stay unreadable. Linux gets the same via a private tmpfs /tmp.
	b.WriteString("(allow file-read* file-write* (subpath \"/private/tmp\") (subpath \"/private/var/folders\"))\n")
	b.WriteString("(allow file-read* file-write* (literal \"/dev/null\") (literal \"/dev/tty\") (regex #\"^/dev/ttys[0-9]+$\"))\n")
	b.WriteString("(allow ipc-posix-shm)\n(allow mach-lookup\n")
	for _, s := range MachServices() {
		b.WriteString("  (global-name " + q(s) + ")\n")
	}
	b.WriteString(")\n")
	for _, p := range spec.ReadOnly {
		b.WriteString("(allow file-read* (subpath " + q(realPath(p)) + "))\n")
	}
	for _, p := range spec.ReadWrite {
		b.WriteString("(allow file-read* file-write* (subpath " + q(realPath(p)) + "))\n")
	}
	for _, p := range spec.Deny {
		p = realPath(p)
		if strings.HasSuffix(p, "/.git/config") {
			b.WriteString("(deny file-write* (literal " + q(p) + "))\n")
		} else {
			b.WriteString("(deny file-write* (subpath " + q(p) + "))\n")
		}
	}
	b.WriteString("(allow network*)\n")
	return b.String()
}
