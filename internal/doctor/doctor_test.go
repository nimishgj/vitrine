package doctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/probe"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

// echoBackend runs the probe command unsandboxed, just to test plumbing.
type echoBackend struct{ spec sandbox.Spec }

func (b *echoBackend) Name() string      { return "echo" }
func (b *echoBackend) Available() error { return nil }
func (b *echoBackend) Run(_ context.Context, s sandbox.Spec, cmd []string) (int, error) {
	b.spec = s
	// Simulate the probe: write a canned result to the output path named in cmd.
	out := cmd[len(cmd)-1]
	cs := []probe.Check{{ID: 5, Name: "write inside write grant", Want: "allowed", Got: "allowed", OK: true}}
	bts, _ := json.Marshal(cs)
	return 0, os.WriteFile(out, bts, 0o600)
}

func TestRunPlumbing(t *testing.T) {
	home := t.TempDir()
	b := &echoBackend{}
	var applied, removed []grants.Grant
	cs, berr, err := Run(context.Background(), Deps{
		Home: home, Engineer: "u", GOOS: "linux", VitrineBin: "/bin/true", Backend: b,
		AgentHome:   func(n string) string { return filepath.Join(home, "ah", n) },
		ApplyGrant:  func(g grants.Grant) error { applied = append(applied, g); return nil },
		RemoveGrant: func(g grants.Grant) error { removed = append(removed, g); return nil },
	})
	if err != nil || berr != nil {
		t.Fatal(err, berr)
	}
	if len(cs) != 1 || !cs[0].OK {
		t.Fatalf("%+v", cs)
	}
	if len(applied) != 2 || len(removed) != 2 {
		t.Fatalf("grants applied=%d removed=%d", len(applied), len(removed))
	}
	if _, err := os.Stat(filepath.Join(home, ".vitrine", "doctor")); !os.IsNotExist(err) {
		t.Fatal("doctor tree must be cleaned up")
	}
	joined := strings.Join(b.spec.ReadWrite, " ")
	if !strings.Contains(joined, "doctor") {
		t.Fatalf("write grant not in spec: %v", b.spec.ReadWrite)
	}
	if os.Getenv("VITRINE_CANARY") != "" {
		t.Fatal("canary must be unset after run")
	}
}
