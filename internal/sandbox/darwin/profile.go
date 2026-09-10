package darwin

import (
	"strings"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

func q(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

// Profile renders the SBPL profile for spec. Later rules win, so denies are
// emitted after the allows they narrow.
func Profile(spec sandbox.Spec) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	b.WriteString("(allow process-exec* process-fork signal sysctl-read)\n")
	b.WriteString("(allow file-read-metadata)\n") // stat() anywhere; content still denied
	b.WriteString("(allow file-ioctl (literal \"/dev/tty\"))\n")
	b.WriteString("(allow file-read* file-write* (literal \"/dev/null\") (literal \"/dev/tty\") (regex #\"^/dev/ttys[0-9]+$\"))\n")
	b.WriteString("(allow ipc-posix-shm)\n(allow mach-lookup\n")
	for _, s := range MachServices() {
		b.WriteString("  (global-name " + q(s) + ")\n")
	}
	b.WriteString(")\n")
	for _, p := range spec.ReadOnly {
		b.WriteString("(allow file-read* (subpath " + q(p) + "))\n")
	}
	for _, p := range spec.ReadWrite {
		b.WriteString("(allow file-read* file-write* (subpath " + q(p) + "))\n")
	}
	for _, p := range spec.Deny {
		if strings.HasSuffix(p, "/.git/config") {
			b.WriteString("(deny file-write* (literal " + q(p) + "))\n")
		} else {
			b.WriteString("(deny file-write* (subpath " + q(p) + "))\n")
		}
	}
	b.WriteString("(allow network*)\n")
	return b.String()
}
