//go:build darwin && integration

package darwin

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/probe"
	"github.com/nimishgj/vitrine/internal/sandbox"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

// Requires: sudo vitrine init already run on this machine (creates the user,
// sudoers rule and /var/lib/vitrine/sessions). The sudoers rule names one
// binary, so VITRINE_BIN must point at that installed binary; otherwise the
// test builds a fresh one, which the rule will not accept, and skips.
func TestProbeUnderSeatbelt(t *testing.T) {
	home, _ := os.UserHomeDir()
	bin := os.Getenv("VITRINE_BIN")
	if bin == "" {
		bin = buildVitrine(t)
	}
	b := New(home)
	b.VitrineBin = bin
	if err := b.Available(); err != nil {
		t.Skip("backend unavailable:", err)
	}
	root := filepath.Join(home, ".vitrine-itest")
	os.RemoveAll(root)
	os.MkdirAll(root, 0o755)
	defer os.RemoveAll(root)

	mk := func(rel, content string) string {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}
	cred := mk("secrets/kube", "secret")
	state := mk("state/grants.json", "{}")
	wg := filepath.Join(root, "wg")
	os.MkdirAll(filepath.Join(wg, ".git", "hooks"), 0o755)
	os.WriteFile(filepath.Join(wg, ".git", "config"), []byte(""), 0o644)
	os.Symlink(cred, filepath.Join(wg, "escape"))
	rg := filepath.Join(root, "rg")
	mk("rg/readme", "hi")

	// Use the real grant ACL code so the vitrine user can traverse and use
	// the grants and the engineer can read what the agent writes. Cleanup
	// passes the engineer's real grants as remaining so shared ancestor
	// entries (the home directory) survive.
	engineer, _ := exec.Command("id", "-un").Output()
	eng := strings.TrimSpace(string(engineer))
	l, _ := paths.Default()
	realGrants, _ := grants.Load(l.Grants)
	wGrant := grants.Grant{Path: wg, Mode: grants.ModeWrite}
	rGrant := grants.Grant{Path: rg, Mode: grants.ModeRead}
	for _, g := range []grants.Grant{wGrant, rGrant} {
		if err := grants.ApplyACL(g, home, eng); err != nil {
			t.Fatal(err)
		}
	}
	defer grants.RemoveACL(rGrant, home, realGrants.Grants)
	defer grants.RemoveACL(wGrant, home, append(realGrants.Grants, rGrant))

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

	os.Setenv("VITRINE_CANARY", "1")

	agentHome := sysuser.AgentHome("itest")
	sessScratch := filepath.Join(root, "scratch")
	spec := sandbox.BuildSpec(sandbox.BuildInput{
		Roots:       sandbox.ExistingRoots(sandbox.SystemRoots("darwin", nil)),
		ReadGrants:  []string{rg},
		WriteGrants: []string{wg},
		Home:        agentHome, Scratch: sessScratch, Workdir: wg,
		Env: map[string]string{"HOME": agentHome, "TMPDIR": sessScratch, "PATH": "/usr/bin:/bin"},
	})
	spec.ReadOnly = append(spec.ReadOnly, filepath.Dir(bin))

	out := filepath.Join(wg, "out.json")
	cmd := []string{bin, "probe",
		"--cred-path", cred, "--home", home, "--state-file", state,
		"--outside", filepath.Join(root, "outside.txt"),
		"--write-grant", wg, "--read-grant", rg,
		"--agent-home", agentHome, "--scratch", "$TMPDIR",
		"--symlink", filepath.Join(wg, "escape"),
		"--net", ln.Addr().String(), "--out", out,
	}
	// $TMPDIR is relocated by the backend; resolve it through a shell.
	cmd = []string{"/bin/sh", "-c", shellJoin(cmd)}
	if _, err := b.Run(context.Background(), spec, cmd); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("no probe output: %v", err)
	}
	var cs []probe.Check
	if err := json.Unmarshal(raw, &cs); err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	for _, c := range cs {
		if !c.OK {
			t.Errorf("check %d %s: want %s got %s (%s)", c.ID, c.Name, c.Want, c.Got, c.Detail)
		}
	}
	t.Log("\n" + probe.Format(cs))
}

func shellJoin(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += " "
		}
		if a == "$TMPDIR" {
			s += a
			continue
		}
		s += "'" + a + "'"
	}
	return s
}

func buildVitrine(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "vitrine")
	c := exec.Command("go", "build", "-o", out, "../../../cmd/vitrine")
	if b, err := c.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return out
}
