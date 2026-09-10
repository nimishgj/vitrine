package probe

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunClassifiesResults(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs non-root POSIX permissions")
	}
	dir := t.TempDir()
	// Restore permissions so TempDir cleanup can remove everything.
	t.Cleanup(func() {
		filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err == nil {
				os.Chmod(p, 0o700)
			}
			return nil
		})
	})
	mk := func(rel string, perm os.FileMode) string {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), perm)
		return p
	}
	cred := mk("home/.kube/config", 0o000)
	state := mk("home/.vitrine/grants.json", 0o000)
	outsideDir := filepath.Join(dir, "outside")
	os.MkdirAll(outsideDir, 0o500)
	wg := filepath.Join(dir, "wg")
	os.MkdirAll(filepath.Join(wg, ".git", "hooks"), 0o755)
	os.Chmod(filepath.Join(wg, ".git", "hooks"), 0o500)
	os.WriteFile(filepath.Join(wg, ".git", "config"), []byte(""), 0o400)
	rg := filepath.Join(dir, "rg")
	os.MkdirAll(rg, 0o755)
	mk("rg/readme", 0o644)
	os.Chmod(rg, 0o500)
	agentHome := filepath.Join(dir, "ah")
	os.MkdirAll(agentHome, 0o700)
	scratch := filepath.Join(dir, "sc")
	os.MkdirAll(scratch, 0o700)
	secret := mk("secret/file", 0o000)
	os.Symlink(secret, filepath.Join(wg, "escape"))
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	os.Unsetenv("VITRINE_CANARY")

	cs := Run(Params{
		CredPath: cred, HomeDir: filepath.Join(dir, "home"), StateFile: state,
		OutsidePath: filepath.Join(outsideDir, "x"), WriteGrant: wg, ReadGrant: rg,
		AgentHome: agentHome, Scratch: scratch, SymlinkInGrant: filepath.Join(wg, "escape"),
		CanaryEnv: "VITRINE_CANARY", NetAddr: ln.Addr().String(),
	})
	if len(cs) < 14 {
		t.Fatalf("got %d checks", len(cs))
	}
	for _, c := range cs {
		if c.ID == 2 || c.ID == 13 {
			continue // home listing is readable here; TIOCSTI not asserted unsandboxed
		}
		if !c.OK {
			t.Errorf("check %d %s: want %s got %s (%s)", c.ID, c.Name, c.Want, c.Got, c.Detail)
		}
	}
	if !Passed(cs[:1]) {
		t.Fatal("Passed on a passing slice")
	}
	if Format(cs) == "" {
		t.Fatal("format empty")
	}
}
