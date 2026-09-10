package profile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuiltinAndLookup(t *testing.T) {
	p, ok := Lookup("claude", nil)
	if !ok || p.Binary != "claude" || len(p.StateDirs) != 2 {
		t.Fatalf("%+v %v", p, ok)
	}
	p, ok = Lookup("claude", map[string]string{"claude": "claude-nightly"})
	if !ok || p.Binary != "claude-nightly" {
		t.Fatalf("override: %+v", p)
	}
	if _, ok := Lookup("nope", nil); ok {
		t.Fatal("unknown profile")
	}
	g := Generic("/usr/local/bin/mytool")
	if g.Name != "mytool" || !g.Generic || g.Binary != "/usr/local/bin/mytool" {
		t.Fatalf("%+v", g)
	}
}

func TestFromArgv0(t *testing.T) {
	if n, ok := FromArgv0("/home/u/.vitrine/bin/codex"); !ok || n != "codex" {
		t.Fatal(n, ok)
	}
	if _, ok := FromArgv0("/usr/local/bin/vitrine"); ok {
		t.Fatal("vitrine itself is not a shim")
	}
}

func TestFindRealBinarySkipsShimDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	shimDir := t.TempDir()
	realDir := t.TempDir()
	os.WriteFile(filepath.Join(shimDir, "claude"), []byte("#!/bin/sh\n"), 0o755)
	os.WriteFile(filepath.Join(realDir, "claude"), []byte("#!/bin/sh\n"), 0o755)
	pathEnv := strings.Join([]string{shimDir, realDir}, string(os.PathListSeparator))
	got, err := FindRealBinary("claude", pathEnv, shimDir)
	if err != nil || got != filepath.Join(realDir, "claude") {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := FindRealBinary("missing", pathEnv, shimDir); err == nil {
		t.Fatal("expected not found")
	}
}
