package darwin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

func sampleSpec() sandbox.Spec {
	return sandbox.Spec{
		ReadOnly:  []string{"/System", "/Users/u/ro", "/Users/u/.vitrine/sessions/s/creds", "/usr"},
		ReadWrite: []string{"/Users/u/repo", "/var/lib/vitrine/agents/claude/home", "/var/lib/vitrine/sessions/s/scratch"},
		Deny:      []string{"/Users/u/repo/.git/config", "/Users/u/repo/.git/hooks"},
		Env:       map[string]string{"HOME": "/var/lib/vitrine/agents/claude/home"},
		Workdir:   "/Users/u/repo",
		Home:      "/var/lib/vitrine/agents/claude/home",
		Scratch:   "/var/lib/vitrine/sessions/s/scratch",
	}
}

func TestProfileGolden(t *testing.T) {
	got := Profile(sampleSpec())
	golden := filepath.Join("..", "testdata", "seatbelt.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.MkdirAll(filepath.Dir(golden), 0o755)
		os.WriteFile(golden, []byte(got), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run with UPDATE_GOLDEN=1 once)", err)
	}
	if got != string(want) {
		t.Fatalf("profile differs from golden:\n%s", got)
	}
}

func TestProfileInvariants(t *testing.T) {
	p := Profile(sampleSpec())
	if !strings.HasPrefix(p, "(version 1)\n(deny default)\n") {
		t.Fatal("must start with deny default")
	}
	allowRW := strings.Index(p, `(allow file-read* file-write* (subpath "/Users/u/repo")`)
	denyHooks := strings.Index(p, `(deny file-write* (subpath "/Users/u/repo/.git/hooks")`)
	denyCfg := strings.Index(p, `(deny file-write* (literal "/Users/u/repo/.git/config")`)
	if allowRW < 0 || denyHooks < 0 || denyCfg < 0 {
		t.Fatalf("missing rules in:\n%s", p)
	}
	if denyHooks < allowRW || denyCfg < allowRW {
		t.Fatal("denies must follow allows (last match wins)")
	}
	if !strings.Contains(p, `(allow file-read* (subpath "/Users/u/ro"))`) {
		t.Fatal("read grant missing")
	}
	if strings.Contains(p, `file-write* (subpath "/Users/u/ro")`) {
		t.Fatal("read grant must not be writable")
	}
	if !strings.Contains(p, "(allow network*)") || !strings.Contains(p, "(allow process-exec*") {
		t.Fatal("process/network rules")
	}
	// Process startup aborts without the first two (libSystem init); agents
	// hardcode /tmp paths so shared temp must be writable.
	for _, must := range []string{`(allow file-read-data (literal "/"))`, `(literal "/dev/dtracehelper")`, `file-write* (subpath "/private/tmp")`} {
		if !strings.Contains(p, must) {
			t.Fatalf("missing startup rule %s", must)
		}
	}
	for _, svc := range MachServices() {
		if !strings.Contains(p, `(global-name "`+svc+`")`) {
			t.Fatalf("mach service %s missing", svc)
		}
	}
}

func TestProfileResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	os.MkdirAll(real, 0o755)
	link := filepath.Join(dir, "link")
	os.Symlink(real, link)
	want, _ := filepath.EvalSymlinks(real)

	s := sampleSpec()
	s.ReadWrite = []string{link}
	s.Deny = []string{filepath.Join(link, "missing", ".git", "hooks")} // does not exist yet
	p := Profile(s)
	if !strings.Contains(p, `(allow file-read* file-write* (subpath "`+want+`"))`) {
		t.Fatalf("symlinked write path not resolved:\n%s", p)
	}
	if !strings.Contains(p, `(deny file-write* (subpath "`+filepath.Join(want, "missing", ".git", "hooks")+`"))`) {
		t.Fatalf("missing path not resolved through ancestor:\n%s", p)
	}
}

func TestProfileEscapesQuotes(t *testing.T) {
	s := sampleSpec()
	s.ReadWrite = []string{`/Users/u/we"ird`}
	p := Profile(s)
	if !strings.Contains(p, `(subpath "/Users/u/we\"ird")`) {
		t.Fatalf("quote not escaped:\n%s", p)
	}
}
