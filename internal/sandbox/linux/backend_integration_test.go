//go:build linux && integration

package linux

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nimishgj/vitrine/internal/probe"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

func TestProbeUnderBwrap(t *testing.T) {
	home, _ := os.UserHomeDir()
	// Must live under home so the tmpfs shadowing of home is what hides it.
	root := filepath.Join(home, ".vitrine-itest")
	os.RemoveAll(root)
	os.MkdirAll(root, 0o700)
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
	agentHome := filepath.Join(root, "ah")
	scratch := filepath.Join(root, "sc")
	os.MkdirAll(agentHome, 0o700)
	os.MkdirAll(scratch, 0o700)

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

	bin := buildVitrine(t)
	os.Setenv("VITRINE_CANARY", "1")
	spec := sandbox.BuildSpec(sandbox.BuildInput{
		Roots:       sandbox.ExistingRoots(sandbox.SystemRoots("linux", nil)),
		ReadGrants:  []string{rg},
		WriteGrants: []string{wg},
		Home:        agentHome, Scratch: scratch, Workdir: wg,
		Env: map[string]string{"HOME": agentHome, "TMPDIR": scratch, "PATH": "/usr/bin:/bin"},
	})
	// The probe binary itself must be visible: bind the build dir read-only.
	spec.ReadOnly = append(spec.ReadOnly, filepath.Dir(bin))

	b := New(home)
	b.VitrineBin = bin
	if err := b.Available(); err != nil {
		t.Skip("backend unavailable:", err)
	}
	out := filepath.Join(scratch, "out.json")
	cmd := []string{bin, "probe",
		"--cred-path", cred, "--home", home, "--state-file", state,
		"--outside", filepath.Join(root, "outside.txt"),
		"--write-grant", wg, "--read-grant", rg, "--agent-home", agentHome,
		"--scratch", scratch, "--symlink", filepath.Join(wg, "escape"),
		"--net", ln.Addr().String(), "--out", out,
	}
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
