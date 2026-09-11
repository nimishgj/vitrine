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

func TestInstallRoot(t *testing.T) {
	cases := map[string]string{
		"/Users/u/.local/share/claude/versions/2.1.268":                 "/Users/u/.local/share/claude/versions",
		"/opt/homebrew/lib/node_modules/@openai/codex/bin/codex.js":     "/opt/homebrew/lib/node_modules",
		"/home/u/.npm-global/lib/node_modules/pi/node_modules/x/bin/pi": "/home/u/.npm-global/lib/node_modules",
		"/usr/local/bin/tool": "/usr/local/bin",
	}
	for in, want := range cases {
		if got := InstallRoot(in); got != want {
			t.Errorf("InstallRoot(%s) = %s, want %s", in, got, want)
		}
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
