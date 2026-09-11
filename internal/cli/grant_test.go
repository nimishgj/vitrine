package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/grants"
)

func TestGrantRevokeList(t *testing.T) {
	e := testEnv(t)
	newInitCmd(e).Execute()
	repo := filepath.Join(e.Home, "repo")
	os.MkdirAll(repo, 0o755)

	c := newGrantCmd(e)
	c.SetArgs([]string{repo, "--read-only"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	rows, _ := listGrants(e)
	if len(rows) != 1 || rows[0].Grant.Mode != grants.ModeRead || !rows[0].Recognised || rows[0].Grant.Source != "cli" {
		t.Fatalf("%+v", rows)
	}
	// Refusal.
	c = newGrantCmd(e)
	c.SetArgs([]string{filepath.Join(e.Home, ".ssh")})
	if err := c.Execute(); err == nil {
		t.Fatal("must refuse ~/.ssh")
	}
	// Revoke.
	r := newRevokeCmd(e)
	r.SetArgs([]string{repo})
	if err := r.Execute(); err != nil {
		t.Fatal(err)
	}
	rows, _ = listGrants(e)
	if len(rows) != 0 {
		t.Fatalf("%+v", rows)
	}
	evs, _ := e.Log.ReadAll()
	last := evs[len(evs)-1]
	if last.Type != audit.TypeGrantRevoke || last.Data["path"] != repo {
		t.Fatalf("%+v", last)
	}
	r = newRevokeCmd(e)
	r.SetArgs(nil)
	if err := r.Execute(); err == nil {
		t.Fatal("revoke requires a path")
	}
}

func TestGrantsAccept(t *testing.T) {
	e := testEnv(t)
	newInitCmd(e).Execute()
	repo := filepath.Join(e.Home, "repo")
	os.MkdirAll(repo, 0o755)
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: repo, Mode: grants.ModeWrite, Source: "manual"})
	grants.Save(e.Layout.Grants, s)

	rows, _ := listGrants(e)
	if len(rows) != 1 || rows[0].Recognised {
		t.Fatalf("out-of-band grant must show unrecognised: %+v", rows)
	}
	out := &bytes.Buffer{}
	e.Stdout = out
	l := newGrantsCmd(e)
	l.SetArgs(nil)
	l.Execute()
	if !strings.Contains(out.String(), "UNRECOGNISED") {
		t.Fatalf("list output: %s", out.String())
	}
	if err := acceptGrants(e); err != nil {
		t.Fatal(err)
	}
	rows, _ = listGrants(e)
	if !rows[0].Recognised {
		t.Fatal("accept must make it recognised")
	}
	evs, _ := e.Log.ReadAll()
	last := evs[len(evs)-1]
	if last.Type != audit.TypeGrantAccept {
		t.Fatalf("%+v", last)
	}
	// Accepting a refused path must fail and not be honoured.
	s.Add(grants.Grant{Path: filepath.Join(e.Home, ".aws"), Mode: grants.ModeRead})
	grants.Save(e.Layout.Grants, s)
	if err := acceptGrants(e); err == nil {
		t.Fatal("accept must apply the refusal list")
	}
}
