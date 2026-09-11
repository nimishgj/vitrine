package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/provider"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

type recBackend struct {
	spec sandbox.Spec
	cmd  []string
	ran  bool
}

func (b *recBackend) Name() string      { return "rec" }
func (b *recBackend) Available() error { return nil }
func (b *recBackend) Run(_ context.Context, s sandbox.Spec, c []string) (int, error) {
	b.spec, b.cmd, b.ran = s, c, true
	return 0, nil
}

func setupRun(t *testing.T) (*env, *recBackend, string) {
	t.Helper()
	e := testEnv(t)
	if err := newInitCmd(e).Execute(); err != nil {
		t.Fatal(err)
	}
	// A fake real binary on PATH outside the shim dir.
	realDir := filepath.Join(e.Home, "realbin")
	os.MkdirAll(realDir, 0o755)
	os.WriteFile(filepath.Join(realDir, "claude"), []byte("#!/bin/sh\n"), 0o755)
	e.HostEnv = []string{"PATH=" + e.Layout.Bin + ":" + realDir, "TERM=dumb"}
	b := &recBackend{}
	backendFactory = func(string, string) (sandbox.Backend, error) { return b, nil }
	providerLookup = func(string) (provider.Provider, bool) { return nil, false }
	repo := filepath.Join(e.Home, "repo")
	os.MkdirAll(repo, 0o755)
	e.Workdir = repo
	return e, b, repo
}

func TestRunWithExistingGrant(t *testing.T) {
	e, b, repo := setupRun(t)
	if err := addGrant(e, grants.Grant{Path: repo, Mode: grants.ModeWrite, Source: "cli"}); err != nil {
		t.Fatal(err)
	}
	code, err := runSession(e, "claude", []string{"--version"})
	if err != nil || code != 0 || !b.ran {
		t.Fatalf("%d %v ran=%v", code, err, b.ran)
	}
	if filepath.Base(b.cmd[0]) != "claude" || b.cmd[1] != "--version" {
		t.Fatalf("cmd %v", b.cmd)
	}
	if b.cmd[0] == filepath.Join(e.Layout.Bin, "claude") {
		t.Fatal("must resolve the real binary, not the shim")
	}
	evs, _ := e.Log.ReadAll()
	last := evs[len(evs)-1]
	if last.Type != audit.TypeSessionEnd {
		t.Fatalf("last event %s", last.Type)
	}
}

func TestRunPromptsAndRecordsGrant(t *testing.T) {
	e, b, repo := setupRun(t)
	e.Stdin = bytes.NewBufferString("r\n")
	e.IsTTY = true
	code, err := runSession(e, "claude", nil)
	if err != nil || code != 0 || !b.ran {
		t.Fatalf("%d %v", code, err)
	}
	s, _ := grants.Load(e.Layout.Grants)
	g, ok := s.Match(repo)
	if !ok || g.Mode != grants.ModeRead || g.Source != "prompt" {
		t.Fatalf("grant %+v %v", g, ok)
	}
	if has(b.spec.ReadWrite, repo) || !has(b.spec.ReadOnly, repo) {
		t.Fatal("read grant must be read-only in spec")
	}
	evs, _ := e.Log.ReadAll()
	var sawAdd bool
	for _, ev := range evs {
		if ev.Type == audit.TypeGrantAdd && ev.Data["path"] == repo {
			sawAdd = true
		}
	}
	if !sawAdd {
		t.Fatal("grant.add not logged")
	}
}

func TestRunNoGrantNonTTYFailsClosed(t *testing.T) {
	e, b, _ := setupRun(t)
	e.IsTTY = false
	_, err := runSession(e, "claude", nil)
	if err == nil || b.ran {
		t.Fatal("must fail closed without a grant on non-tty")
	}
}

func TestRunTamperedGrantNotHonoured(t *testing.T) {
	e, b, repo := setupRun(t)
	// Out-of-band: write a grant directly, never recorded in the chain.
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: repo, Mode: grants.ModeWrite})
	grants.Save(e.Layout.Grants, s)
	e.IsTTY = false
	_, err := runSession(e, "claude", nil)
	if err == nil || b.ran {
		t.Fatal("unrecognised grant must not be honoured")
	}
	evs, _ := e.Log.ReadAll()
	var tampered bool
	for _, ev := range evs {
		if ev.Type == audit.TypeTamper {
			tampered = true
		}
	}
	if !tampered {
		t.Fatal("tamper event missing")
	}
}

func has(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
