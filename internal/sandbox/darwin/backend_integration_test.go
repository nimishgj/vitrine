//go:build darwin && integration

package darwin

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
	"github.com/nimishgj/vitrine/internal/sysuser"
)

func ace(user, perms string) string { return "user:" + user + " allow " + perms }

// Requires: sudo vitrine init already run on this machine (creates the user,
// sudoers rule and /var/lib/vitrine/sessions).
func TestProbeUnderSeatbelt(t *testing.T) {
	home, _ := os.UserHomeDir()
	bin := buildVitrine(t)
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

	// The vitrine user needs ACLs to traverse into root and use the grants.
	// This mirrors what grants.ApplyACL does for real grants.
	const readPerms = "read,readattr,readextattr,readsecurity,list,search,execute"
	const writePerms = readPerms + ",write,append,delete,add_file,add_subdirectory,delete_child,writeattr,writeextattr"
	for _, p := range []string{home, root} {
		if out, err := exec.Command("chmod", "+a", ace(sysuser.Name, "search"), p).CombinedOutput(); err != nil {
			t.Fatalf("chmod +a on %s: %v %s", p, err, out)
		}
	}
	defer exec.Command("chmod", "-a", ace(sysuser.Name, "search"), home).Run()
	exec.Command("chmod", "-R", "+a", ace(sysuser.Name, readPerms+",file_inherit,directory_inherit"), rg).Run()
	exec.Command("chmod", "-R", "+a", ace(sysuser.Name, writePerms+",file_inherit,directory_inherit"), wg).Run()

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

	// The vitrine user must be able to execute the test binary.
	exec.Command("chmod", "+a", ace(sysuser.Name, "read,execute,search"), filepath.Dir(bin)).Run()
	exec.Command("chmod", "+a", ace(sysuser.Name, "read,execute"), bin).Run()
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
