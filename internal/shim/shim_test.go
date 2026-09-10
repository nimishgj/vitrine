package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "vitrine")
	os.WriteFile(bin, []byte("x"), 0o755)
	binDir := filepath.Join(dir, "bin")
	if err := Install(binDir, bin, []string{"claude", "codex"}); err != nil {
		t.Fatal(err)
	}
	if err := Install(binDir, bin, []string{"claude", "codex"}); err != nil {
		t.Fatal("second install must succeed")
	}
	for _, n := range []string{"claude", "codex"} {
		target, err := os.Readlink(filepath.Join(binDir, n))
		if err != nil || target != bin {
			t.Fatalf("%s -> %q %v", n, target, err)
		}
	}
	// A stale link pointing elsewhere is replaced.
	os.Remove(filepath.Join(binDir, "claude"))
	os.Symlink("/nonexistent", filepath.Join(binDir, "claude"))
	Install(binDir, bin, []string{"claude"})
	if target, _ := os.Readlink(filepath.Join(binDir, "claude")); target != bin {
		t.Fatal("stale link not replaced")
	}
}

func TestStripFromPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	in := strings.Join([]string{"/a", "/home/u/.vitrine/bin", "/b", "/home/u/.vitrine/bin/"}, sep)
	got := StripFromPath(in, "/home/u/.vitrine/bin")
	if got != strings.Join([]string{"/a", "/b"}, sep) {
		t.Fatalf("%q", got)
	}
}
