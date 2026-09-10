package sandbox

import (
	"reflect"
	"testing"
)

func TestSystemRootsPerOS(t *testing.T) {
	d := SystemRoots("darwin", nil)
	l := SystemRoots("linux", nil)
	if !contains(d, "/System") || contains(l, "/System") {
		t.Fatal("darwin roots")
	}
	if !contains(l, "/etc/resolv.conf") || contains(d, "/etc/resolv.conf") {
		t.Fatal("linux roots")
	}
	if got := SystemRoots("linux", []string{"/custom"}); !reflect.DeepEqual(got, []string{"/custom"}) {
		t.Fatal("override must replace")
	}
}

func TestBuildSpecAddsGitDenies(t *testing.T) {
	s := BuildSpec(BuildInput{
		Roots:       []string{"/usr", "/bin"},
		ReadGrants:  []string{"/home/u/ro"},
		WriteGrants: []string{"/home/u/repo", "/home/u/repo"},
		CredDir:     "/tmp/sess/creds",
		Home:        "/home/u/.vitrine/agents/claude/home",
		Scratch:     "/tmp/sess/scratch",
		Workdir:     "/home/u/repo",
		Env:         map[string]string{"A": "1"},
	})
	wantRO := []string{"/bin", "/home/u/ro", "/tmp/sess/creds", "/usr"}
	if !reflect.DeepEqual(s.ReadOnly, wantRO) {
		t.Fatalf("ro %v", s.ReadOnly)
	}
	wantRW := []string{"/home/u/.vitrine/agents/claude/home", "/home/u/repo", "/tmp/sess/scratch"}
	if !reflect.DeepEqual(s.ReadWrite, wantRW) {
		t.Fatalf("rw %v", s.ReadWrite)
	}
	wantDeny := []string{"/home/u/repo/.git/config", "/home/u/repo/.git/hooks"}
	if !reflect.DeepEqual(s.Deny, wantDeny) {
		t.Fatalf("deny %v", s.Deny)
	}
	if s.Env["A"] != "1" || s.Workdir != "/home/u/repo" {
		t.Fatal("env/workdir")
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
