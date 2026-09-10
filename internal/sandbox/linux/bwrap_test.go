package linux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

func TestArgsGolden(t *testing.T) {
	spec := sandbox.Spec{
		ReadOnly:  []string{"/etc/resolv.conf", "/home/u/ro", "/home/u/.vitrine/sessions/s/creds", "/usr"},
		ReadWrite: []string{"/home/u/.vitrine/agents/claude/home", "/home/u/.vitrine/sessions/s/scratch", "/home/u/repo"},
		Deny:      []string{"/home/u/repo/.git/config", "/home/u/repo/.git/hooks"},
		Env:       map[string]string{"HOME": "/home/u/.vitrine/agents/claude/home", "PATH": "/usr/bin", "TMPDIR": "/home/u/.vitrine/sessions/s/scratch"},
		Workdir:   "/home/u/repo",
		Home:      "/home/u/.vitrine/agents/claude/home",
		Scratch:   "/home/u/.vitrine/sessions/s/scratch",
	}
	got := strings.Join(Args(spec, "/home/u", 7), "\n") + "\n"
	golden := filepath.Join("..", "testdata", "bwrap_args.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.MkdirAll(filepath.Dir(golden), 0o755)
		os.WriteFile(golden, []byte(got), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run with UPDATE_GOLDEN=1 once)", err)
	}
	if got != string(want) {
		t.Fatalf("args differ from golden:\n%s", got)
	}
}

func TestArgsInvariants(t *testing.T) {
	spec := sandbox.Spec{ReadWrite: []string{"/home/u/repo"}, Deny: []string{"/home/u/repo/.git/hooks"}, Home: "/h", Scratch: "/s", Workdir: "/home/u/repo", Env: map[string]string{}}
	args := Args(spec, "/home/u", 3)
	joined := strings.Join(args, " ")
	for _, must := range []string{"--unshare-pid", "--unshare-ipc", "--unshare-uts", "--die-with-parent", "--clearenv", "--tmpfs /home/u", "--bind /home/u/repo /home/u/repo", "--ro-bind-try /home/u/repo/.git/hooks /home/u/repo/.git/hooks", "--seccomp 3", "--chdir /home/u/repo"} {
		if !strings.Contains(joined, must) {
			t.Errorf("missing %q in %s", must, joined)
		}
	}
	if strings.Contains(joined, "--new-session") || strings.Contains(joined, "--unshare-net") {
		t.Error("must not use --new-session or --unshare-net")
	}
	// Denies must come after the bind they override.
	bi := strings.Index(joined, "--bind /home/u/repo ")
	di := strings.Index(joined, "--ro-bind-try /home/u/repo/.git/hooks")
	if di < bi {
		t.Error("deny must follow the write bind")
	}
}
