package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimishgj/vitrine/internal/grants"
)

func recordGrants(t *testing.T, l *Log, path string, s *grants.Store) {
	t.Helper()
	if err := grants.Save(path, s); err != nil {
		t.Fatal(err)
	}
	h, _ := grants.FileHash(path)
	if _, err := l.Append(TypeGrantAdd, map[string]any{"grants_hash": h, "grants": GrantSnapshot(s)}); err != nil {
		t.Fatal(err)
	}
}

func TestExpectedHashesAndRecognised(t *testing.T) {
	l := newLog(t)
	gp := filepath.Join(filepath.Dir(l.Path), "grants.json")
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: "/a", Mode: grants.ModeWrite})
	recordGrants(t, l, gp, s)
	l.Append(TypeConfigWrite, map[string]any{"config_hash": "cfg1"})

	evs, _ := l.ReadAll()
	gh, ch := ExpectedHashes(evs)
	want, _ := grants.FileHash(gp)
	if gh != want || ch != "cfg1" {
		t.Fatalf("hashes %q %q", gh, ch)
	}
	rec := RecognisedGrants(evs)
	if len(rec) != 1 || rec[0].Path != "/a" || rec[0].Mode != grants.ModeWrite {
		t.Fatalf("recognised %+v", rec)
	}
}

func TestCheckGrantsClean(t *testing.T) {
	l := newLog(t)
	gp := filepath.Join(filepath.Dir(l.Path), "grants.json")
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: "/a", Mode: grants.ModeWrite})
	recordGrants(t, l, gp, s)

	cur, _ := grants.Load(gp)
	hon, tampered, err := CheckGrants(l, gp, cur)
	if err != nil || tampered || len(hon.Grants) != 1 {
		t.Fatalf("%v %v %+v", err, tampered, hon)
	}
	evs, _ := l.ReadAll()
	if len(evs) != 1 {
		t.Fatal("clean check must not append events")
	}
}

func TestCheckGrantsOutOfBandAddNotHonoured(t *testing.T) {
	l := newLog(t)
	gp := filepath.Join(filepath.Dir(l.Path), "grants.json")
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: "/a", Mode: grants.ModeWrite})
	recordGrants(t, l, gp, s)

	// Out-of-band edit: add /b, downgrade /a to read.
	edited := &grants.Store{Version: 1}
	edited.Add(grants.Grant{Path: "/a", Mode: grants.ModeRead})
	edited.Add(grants.Grant{Path: "/b", Mode: grants.ModeWrite})
	grants.Save(gp, edited)

	cur, _ := grants.Load(gp)
	hon, tampered, err := CheckGrants(l, gp, cur)
	if err != nil || !tampered {
		t.Fatalf("%v %v", err, tampered)
	}
	if len(hon.Grants) != 1 || hon.Grants[0].Path != "/a" || hon.Grants[0].Mode != grants.ModeRead {
		t.Fatalf("honoured must be intersection with current (narrower) mode: %+v", hon.Grants)
	}
	evs, _ := l.ReadAll()
	last := evs[len(evs)-1]
	if last.Type != TypeTamper || last.Data["file"] != "grants" {
		t.Fatalf("tamper event: %+v", last)
	}
	diff := last.Data["diff"].(map[string]any)
	if len(diff["added"].([]any)) != 1 || len(diff["changed"].([]any)) != 1 {
		t.Fatalf("diff: %+v", diff)
	}
}

func TestCheckGrantsMissingFileAfterRecord(t *testing.T) {
	l := newLog(t)
	gp := filepath.Join(filepath.Dir(l.Path), "grants.json")
	s := &grants.Store{Version: 1}
	s.Add(grants.Grant{Path: "/a", Mode: grants.ModeWrite})
	recordGrants(t, l, gp, s)
	os.Remove(gp)
	cur, _ := grants.Load(gp)
	hon, tampered, _ := CheckGrants(l, gp, cur)
	if !tampered || len(hon.Grants) != 0 {
		t.Fatalf("%v %+v", tampered, hon)
	}
}

func TestCheckConfig(t *testing.T) {
	l := newLog(t)
	cp := filepath.Join(filepath.Dir(l.Path), "config.toml")
	os.WriteFile(cp, []byte("version = 1\n"), 0o600)
	h, _ := grants.FileHash(cp)
	l.Append(TypeConfigWrite, map[string]any{"config_hash": h})
	if tampered, _ := CheckConfig(l, cp); tampered {
		t.Fatal("clean config flagged")
	}
	os.WriteFile(cp, []byte("version = 1\nbackend = \"x\"\n"), 0o600)
	if tampered, _ := CheckConfig(l, cp); !tampered {
		t.Fatal("edited config not flagged")
	}
}

func TestCheckGrantsNeverRecordedIsNotTamper(t *testing.T) {
	l := newLog(t)
	gp := filepath.Join(filepath.Dir(l.Path), "grants.json")
	cur, _ := grants.Load(gp)
	_, tampered, err := CheckGrants(l, gp, cur)
	if err != nil || tampered {
		t.Fatal("fresh install with no grants must be clean")
	}
}
