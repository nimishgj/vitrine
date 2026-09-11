package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/profile"
	"github.com/nimishgj/vitrine/internal/provider"
	"github.com/nimishgj/vitrine/internal/provider/fake"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

type fakeBackend struct {
	availErr error
	gotSpec  sandbox.Spec
	gotCmd   []string
	exit     int
	ran      bool
}

func (b *fakeBackend) Name() string      { return "fake" }
func (b *fakeBackend) Available() error { return b.availErr }
func (b *fakeBackend) Run(_ context.Context, s sandbox.Spec, cmd []string) (int, error) {
	b.gotSpec, b.gotCmd, b.ran = s, cmd, true
	// Prove credential files exist while the sandbox runs.
	if tok := s.Env["FAKE_TOKEN_FILE"]; tok != "" {
		if _, err := os.Stat(tok); err != nil {
			return 99, nil
		}
	}
	return b.exit, nil
}

func deps(t *testing.T, b sandbox.Backend, fp *fake.Provider) (Deps, string) {
	t.Helper()
	home := t.TempDir()
	l := paths.Resolve(home)
	paths.EnsureRoot(l)
	cfg := config.Default()
	cfg.AddTarget(config.Target{Name: "t1", Kind: "fake"})
	return Deps{
		Layout:  l,
		Log:     audit.Open(l.Audit, l.AuditHead),
		Config:  cfg,
		Grants:  &grants.Store{Version: 1},
		Backend: b,
		Providers: func(kind string) (provider.Provider, bool) {
			if kind == "fake" {
				return fp, true
			}
			return nil, false
		},
		HostEnv:   []string{"PATH=/usr/bin", "TERM=dumb", "SECRET=x"},
		GOOS:      "linux",
		Engineer:  "u",
		AgentHome: func(name string) string { return paths.AgentHome(l, name) },
		Now:       func() time.Time { return time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC) },
		NewID:     func() string { return "sess1" },
	}, home
}

func TestRunHappyPath(t *testing.T) {
	var cleaned []string
	fp := &fake.Provider{Cleaned: &cleaned}
	b := &fakeBackend{exit: 3}
	d, home := deps(t, b, fp)
	repo := filepath.Join(home, "repo")
	os.MkdirAll(repo, 0o755)
	p, _ := profile.Lookup("claude", nil)
	code, err := Run(context.Background(), d, Request{
		Profile: p, RealBinary: "/opt/claude", Args: []string{"--foo"}, Workdir: repo,
		Grant: grants.Grant{Path: repo, Mode: grants.ModeWrite},
	})
	if err != nil || code != 3 {
		t.Fatalf("code %d err %v", code, err)
	}
	if !b.ran || b.gotCmd[0] != "/opt/claude" || b.gotCmd[1] != "--foo" {
		t.Fatalf("cmd %v", b.gotCmd)
	}
	s := b.gotSpec
	if s.Workdir != repo || s.Home != paths.AgentHome(d.Layout, "claude") {
		t.Fatalf("spec %+v", s)
	}
	if _, ok := s.Env["SECRET"]; ok {
		t.Fatal("host secret leaked")
	}
	if !has(s.ReadWrite, repo) || !has(s.Deny, filepath.Join(repo, ".git", "hooks")) {
		t.Fatalf("grants not applied: %+v", s)
	}
	if len(cleaned) != 1 {
		t.Fatal("provider cleanup not called")
	}
	if _, err := os.Stat(filepath.Dir(s.Env["FAKE_TOKEN_FILE"])); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("credential dir must be removed after session")
	}
	if _, err := os.Stat(s.Scratch); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("scratch must be removed after session")
	}
	if _, err := os.Stat(s.Home); err != nil {
		t.Fatal("agent home must be created and persist")
	}
	evs, _ := d.Log.ReadAll()
	if len(evs) != 2 || evs[0].Type != audit.TypeSessionStart || evs[1].Type != audit.TypeSessionEnd {
		t.Fatalf("events %+v", evs)
	}
	start := evs[0].Data
	if start["session"] != "sess1" || start["agent"] != "claude" || start["grant_mode"] != "write" {
		t.Fatalf("start data %+v", start)
	}
	ids := start["identities"].(map[string]any)
	if ids["t1"] != "fake:sess1" {
		t.Fatalf("identities %+v", ids)
	}
	if evs[1].Data["exit_code"].(float64) != 3 {
		t.Fatalf("end data %+v", evs[1].Data)
	}
}

func TestRunReadGrantIsReadOnly(t *testing.T) {
	fp := &fake.Provider{}
	b := &fakeBackend{}
	d, home := deps(t, b, fp)
	repo := filepath.Join(home, "repo")
	os.MkdirAll(repo, 0o755)
	Run(context.Background(), d, Request{Profile: profile.Generic("/x/tool"), RealBinary: "/x/tool", Workdir: repo,
		Grant: grants.Grant{Path: repo, Mode: grants.ModeRead}})
	if has(b.gotSpec.ReadWrite, repo) || !has(b.gotSpec.ReadOnly, repo) {
		t.Fatalf("%+v", b.gotSpec)
	}
	if b.gotSpec.Home != paths.AgentHome(d.Layout, "tool") {
		t.Fatal("generic profile home keyed by basename")
	}
}

// On macOS the agent home lives under a directory owned by the vitrine user,
// so the shim (running as the engineer) cannot create it. The backend does.
func TestRunToleratesUnwritableAgentHomeParent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	fp := &fake.Provider{}
	b := &fakeBackend{}
	d, home := deps(t, b, fp)
	locked := filepath.Join(home, "locked")
	os.MkdirAll(locked, 0o700)
	os.Chmod(locked, 0o500)
	t.Cleanup(func() { os.Chmod(locked, 0o700) })
	d.AgentHome = func(name string) string { return filepath.Join(locked, name, "home") }
	repo := filepath.Join(home, "repo")
	os.MkdirAll(repo, 0o755)
	code, err := Run(context.Background(), d, Request{Profile: profile.Generic("/x"), RealBinary: "/x", Workdir: repo,
		Grant: grants.Grant{Path: repo, Mode: grants.ModeRead}})
	if err != nil || code != 0 || !b.ran {
		t.Fatalf("code %d err %v ran %v", code, err, b.ran)
	}
	if b.gotSpec.Home != filepath.Join(locked, "x", "home") {
		t.Fatalf("home %q", b.gotSpec.Home)
	}
}

func TestRunMintFailureAbortsBeforeBackend(t *testing.T) {
	fp := &fake.Provider{MintErr: errors.New("no creds")}
	b := &fakeBackend{}
	d, home := deps(t, b, fp)
	_, err := Run(context.Background(), d, Request{Profile: profile.Generic("/x"), RealBinary: "/x", Workdir: home,
		Grant: grants.Grant{Path: home, Mode: grants.ModeRead}})
	if !errors.Is(err, ErrMint) || b.ran {
		t.Fatalf("err %v ran %v", err, b.ran)
	}
	evs, _ := d.Log.ReadAll()
	if len(evs) != 0 {
		t.Fatal("aborted session must not log start")
	}
}

func TestRunBackendUnavailable(t *testing.T) {
	fp := &fake.Provider{}
	b := &fakeBackend{availErr: errors.New("no bwrap")}
	d, home := deps(t, b, fp)
	_, err := Run(context.Background(), d, Request{Profile: profile.Generic("/x"), RealBinary: "/x", Workdir: home,
		Grant: grants.Grant{Path: home, Mode: grants.ModeRead}})
	if !errors.Is(err, ErrBackendUnavailable) || b.ran {
		t.Fatal(err)
	}
}

func TestRunUnknownProviderKind(t *testing.T) {
	fp := &fake.Provider{}
	b := &fakeBackend{}
	d, home := deps(t, b, fp)
	d.Config.AddTarget(config.Target{Name: "t2", Kind: "nope"})
	_, err := Run(context.Background(), d, Request{Profile: profile.Generic("/x"), RealBinary: "/x", Workdir: home,
		Grant: grants.Grant{Path: home, Mode: grants.ModeRead}})
	if err == nil || b.ran {
		t.Fatal("unknown provider kind must abort")
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
