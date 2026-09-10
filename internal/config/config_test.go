package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/grants"
)

func TestDefaultWhenMissing(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil || c.Version != 1 || c.Backend != "native" {
		t.Fatalf("%v %+v", err, c)
	}
}

func TestSaveLogsHashAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	l := audit.Open(filepath.Join(dir, "a.jsonl"), filepath.Join(dir, "a.head"))
	c := Default()
	if err := c.AddTarget(Target{Name: "prod", Kind: "k8s", Settings: map[string]string{"context": "prod-eu"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddTarget(Target{Name: "prod", Kind: "k8s"}); err == nil {
		t.Fatal("duplicate target name must fail")
	}
	c.Profiles = map[string]ProfileOverride{"claude": {Binary: "claude-nightly"}}
	if err := Save(p, c, l); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", st.Mode().Perm())
	}
	evs, _ := l.ReadAll()
	h, _ := grants.FileHash(p)
	if len(evs) != 1 || evs[0].Type != audit.TypeConfigWrite || evs[0].Data["config_hash"] != h {
		t.Fatalf("%+v", evs)
	}
	c2, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Targets) != 1 || c2.Targets[0].Settings["context"] != "prod-eu" || c2.Profiles["claude"].Binary != "claude-nightly" {
		t.Fatalf("%+v", c2)
	}
	if !c2.RemoveTarget("prod") || c2.RemoveTarget("prod") {
		t.Fatal("remove semantics")
	}
}
