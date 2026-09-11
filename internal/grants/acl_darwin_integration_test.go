//go:build darwin && integration

package grants

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

func aces(t *testing.T, p string) string {
	t.Helper()
	out, _ := exec.Command("ls", "-lde", p).CombinedOutput()
	return string(out)
}

func TestApplyAndRemoveACL(t *testing.T) {
	if exec.Command("id", "-u", sysuser.Name).Run() != nil {
		t.Skip("vitrine user missing; run sudo vitrine init")
	}
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".vitrine-acltest")
	os.RemoveAll(base)
	repo := filepath.Join(base, "proj", "repo")
	os.MkdirAll(filepath.Join(repo, "sub"), 0o755)
	defer os.RemoveAll(base)
	me, _ := exec.Command("id", "-un").Output()
	engineer := strings.TrimSpace(string(me))
	// The engineer's real grants on this machine share the home ancestor;
	// treat them as remaining so the test never strips an entry they need.
	l, _ := paths.Default()
	real, _ := Load(l.Grants)
	remaining := func(extra ...Grant) []Grant { return append(append([]Grant{}, real.Grants...), extra...) }

	g := Grant{Path: repo, Mode: ModeWrite}
	if err := ApplyACL(g, home, engineer); err != nil {
		t.Fatal(err)
	}
	for _, anc := range []string{home, base, filepath.Join(base, "proj")} {
		if !strings.Contains(aces(t, anc), "user:vitrine allow search") {
			t.Errorf("ancestor %s lacks search ACE:\n%s", anc, aces(t, anc))
		}
	}
	got := aces(t, repo)
	if !strings.Contains(got, "user:vitrine allow") || !strings.Contains(got, "add_file") || !strings.Contains(got, "directory_inherit") {
		t.Errorf("repo ACE wrong:\n%s", got)
	}
	if !strings.Contains(got, "user:"+engineer+" allow") {
		t.Errorf("engineer ACE missing:\n%s", got)
	}
	if !strings.Contains(aces(t, filepath.Join(repo, "sub")), "user:vitrine allow") {
		t.Error("recursive ACE missing on sub")
	}

	// A second grant that shares the home ancestor must keep home traversable
	// after the first grant is removed.
	other := filepath.Join(base, "other")
	os.MkdirAll(other, 0o755)
	og := Grant{Path: other, Mode: ModeRead}
	if err := ApplyACL(og, home, engineer); err != nil {
		t.Fatal(err)
	}
	if err := RemoveACL(g, home, remaining(og)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(aces(t, repo), "user:vitrine") {
		t.Errorf("ACE not removed:\n%s", aces(t, repo))
	}
	for _, keep := range []string{home, base} {
		if !strings.Contains(aces(t, keep), "user:vitrine allow search") {
			t.Errorf("ancestor %s lost its search ACE while another grant still needs it:\n%s", keep, aces(t, keep))
		}
	}
	if strings.Contains(aces(t, filepath.Join(base, "proj")), "user:vitrine") {
		t.Error("ancestor unique to the removed grant should lose its search ACE")
	}
	if err := RemoveACL(og, home, remaining()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(aces(t, base), "user:vitrine") {
		t.Error("ancestor search ACE not removed when no grants remain")
	}
}
