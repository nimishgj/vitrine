package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/paths"
)

func testEnv(t *testing.T) *env {
	t.Helper()
	// Resolve symlinks (macOS /var -> /private/var) so canonical paths match.
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	l := paths.Resolve(home)
	exe := filepath.Join(home, "vitrine")
	os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755)
	return &env{
		Layout: l, Log: audit.Open(l.Audit, l.AuditHead), Home: home, Engineer: "tester",
		Stdin: bytes.NewReader(nil), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
		GOOS: "linux", Exe: exe, HostEnv: []string{"PATH=/usr/bin"},
	}
}

func TestInitUnprivileged(t *testing.T) {
	e := testEnv(t)
	cmd := newInitCmd(e)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"claude", "codex", "pi"} {
		target, err := os.Readlink(filepath.Join(e.Layout.Bin, n))
		if err != nil || target != e.Exe {
			t.Fatalf("shim %s: %q %v", n, target, err)
		}
	}
	if _, err := os.Stat(e.Layout.Config); err != nil {
		t.Fatal("config not written")
	}
	evs, _ := e.Log.ReadAll()
	if len(evs) != 2 || evs[0].Type != audit.TypeConfigWrite || evs[1].Type != audit.TypeInit {
		t.Fatalf("events %+v", evs)
	}
	out := e.Stdout.(*bytes.Buffer).String()
	if !bytes.Contains([]byte(out), []byte(e.Layout.Bin)) {
		t.Fatalf("PATH hint missing: %s", out)
	}
	// Second run is idempotent and does not rewrite config.
	if err := newInitCmd(e).Execute(); err != nil {
		t.Fatal(err)
	}
	evs, _ = e.Log.ReadAll()
	if len(evs) != 3 || evs[2].Type != audit.TypeInit {
		t.Fatalf("second init events %+v", evs)
	}
}
