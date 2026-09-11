package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

type unavailBackend struct{}

func (unavailBackend) Name() string                                             { return "native-test" }
func (unavailBackend) Available() error                                         { return errors.New("bwrap missing") }
func (unavailBackend) Run(context.Context, sandbox.Spec, []string) (int, error) { return 1, nil }

func TestStatusOutput(t *testing.T) {
	e := testEnv(t)
	newInitCmd(e).Execute()
	backendFactory = func(string, string) (sandbox.Backend, error) { return unavailBackend{}, nil }
	out := &bytes.Buffer{}
	e.Stdout = out
	if err := newStatusCmd(e).Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"backend: native-test", "available: no (bwrap missing)", "enforcement: unmanaged", "grants: none", "targets: none", "doctor: never run"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestAuditListAndVerify(t *testing.T) {
	e := testEnv(t)
	newInitCmd(e).Execute()
	out := &bytes.Buffer{}
	e.Stdout = out
	c := newAuditCmd(e)
	c.SetArgs(nil)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), audit.TypeInit) {
		t.Fatalf("list: %s", out.String())
	}
	out.Reset()
	c = newAuditCmd(e)
	c.SetArgs([]string{"--json"})
	c.Execute()
	if !strings.HasPrefix(out.String(), "{\"seq\":1") {
		t.Fatalf("json: %s", out.String())
	}
	v := newAuditCmd(e)
	v.SetArgs([]string{"verify"})
	if err := v.Execute(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(e.Layout.Audit)
	os.WriteFile(e.Layout.Audit, bytes.Replace(raw, []byte(`"init"`), []byte(`"inix"`), 1), 0o600)
	v = newAuditCmd(e)
	v.SetArgs([]string{"verify"})
	err := v.Execute()
	var ee exitError
	if !errors.As(err, &ee) || ee.code != 3 {
		t.Fatalf("verify on tampered log: %v", err)
	}
}
