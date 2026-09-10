package grants

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMatchNearestAncestor(t *testing.T) {
	s := &Store{Version: 1}
	s.Add(Grant{Path: "/home/u/repo", Mode: ModeWrite})
	s.Add(Grant{Path: "/home/u/repo/vendor", Mode: ModeRead})

	g, ok := s.Match("/home/u/repo/src/main.go")
	if !ok || g.Mode != ModeWrite {
		t.Fatalf("src: %+v %v", g, ok)
	}
	g, ok = s.Match("/home/u/repo/vendor/x/y.go")
	if !ok || g.Mode != ModeRead {
		t.Fatalf("vendor: %+v %v", g, ok)
	}
	if _, ok := s.Match("/home/u/other"); ok {
		t.Fatal("other should not match")
	}
	if _, ok := s.Match("/home/u/repository"); ok {
		t.Fatal("prefix without separator must not match")
	}
	if g, ok := s.Match("/home/u/repo"); !ok || g.Path != "/home/u/repo" {
		t.Fatal("exact path should match")
	}
}

func TestAddReplacesAndRemove(t *testing.T) {
	s := &Store{Version: 1}
	s.Add(Grant{Path: "/a", Mode: ModeRead})
	s.Add(Grant{Path: "/a", Mode: ModeWrite})
	if len(s.Grants) != 1 || s.Grants[0].Mode != ModeWrite {
		t.Fatalf("%+v", s.Grants)
	}
	if !s.Remove("/a") || s.Remove("/a") {
		t.Fatal("remove semantics")
	}
}

func TestLoadSaveRoundTripAndHash(t *testing.T) {
	p := filepath.Join(t.TempDir(), "grants.json")
	s, err := Load(p)
	if err != nil || len(s.Grants) != 0 {
		t.Fatalf("empty load: %v %+v", err, s)
	}
	if h, _ := FileHash(p); h != "" {
		t.Fatal("missing file hash should be empty")
	}
	s.Add(Grant{Path: "/x", Mode: ModeWrite, CreatedAt: time.Now().UTC(), Source: "cli"})
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", st.Mode().Perm())
	}
	s2, _ := Load(p)
	if len(s2.Grants) != 1 || s2.Grants[0].Path != "/x" || s2.Version != 1 {
		t.Fatalf("%+v", s2)
	}
	h1, _ := FileHash(p)
	s2.Add(Grant{Path: "/y", Mode: ModeRead})
	Save(p, s2)
	h2, _ := FileHash(p)
	if h1 == "" || h1 == h2 {
		t.Fatal("hash must change when content changes")
	}
}

func TestRefuse(t *testing.T) {
	home := "/home/u"
	roots := []string{"/usr", "/bin"}
	bad := []string{"/", "/usr", "/usr/lib", home, "/home/u/.ssh", "/home/u/.kube/x", "/home/u/.vitrine", "/home/u/.vitrine/bin", "/home/u/.npmrc"}
	for _, p := range bad {
		if Refuse(p, home, roots) == nil {
			t.Errorf("expected refusal for %s", p)
		}
	}
	good := []string{"/home/u/repo", "/home/u/repo/.git", "/home/u/projects/.hidden", "/srv/data"}
	for _, p := range good {
		if err := Refuse(p, home, roots); err != nil {
			t.Errorf("unexpected refusal for %s: %v", p, err)
		}
	}
}

func TestCanonicalResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	os.Mkdir(real, 0o755)
	link := filepath.Join(dir, "link")
	os.Symlink(real, link)
	got, err := Canonical(link)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(real)
	if got != want {
		t.Fatalf("%q != %q", got, want)
	}
	missing := filepath.Join(dir, "nope")
	if got, err := Canonical(missing); err != nil || got != missing {
		t.Fatalf("missing path: %q %v", got, err)
	}
}
