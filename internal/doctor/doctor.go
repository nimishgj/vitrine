// Package doctor proves the sandbox works by running the probe inside it.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/probe"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

// Deps are the collaborators doctor needs.
type Deps struct {
	Home        string
	Engineer    string
	GOOS        string
	VitrineBin  string
	Backend     sandbox.Backend
	AgentHome   func(profile string) string
	ApplyGrant  func(grants.Grant) error
	RemoveGrant func(grants.Grant) error
}

// Run builds a throwaway tree, grants it, runs the probe in the backend, and
// returns the parsed checks. backendErr is non-nil when the backend is unavailable.
func Run(ctx context.Context, d Deps) ([]probe.Check, error, error) {
	if err := d.Backend.Available(); err != nil {
		return nil, err, nil
	}
	l := paths.Resolve(d.Home)
	root := filepath.Join(l.Root, "doctor")
	os.RemoveAll(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(root)

	mk := func(rel, content string, perm os.FileMode) string {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), perm)
		return p
	}
	cred := mk("secrets/kubeconfig", "not-a-real-secret", 0o600)
	wg := filepath.Join(root, "write-grant")
	os.MkdirAll(filepath.Join(wg, ".git", "hooks"), 0o755)
	os.WriteFile(filepath.Join(wg, ".git", "config"), []byte("[core]\n"), 0o644)
	os.Symlink(cred, filepath.Join(wg, "escape"))
	rg := filepath.Join(root, "read-grant")
	mk("read-grant/readme", "hello", 0o644)

	wGrant := grants.Grant{Path: wg, Mode: grants.ModeWrite, CreatedAt: time.Now().UTC(), Source: "doctor"}
	rGrant := grants.Grant{Path: rg, Mode: grants.ModeRead, CreatedAt: time.Now().UTC(), Source: "doctor"}
	for _, g := range []grants.Grant{wGrant, rGrant} {
		if err := d.ApplyGrant(g); err != nil {
			return nil, nil, err
		}
	}
	defer d.RemoveGrant(rGrant)
	defer d.RemoveGrant(wGrant)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
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
	defer os.Unsetenv("VITRINE_CANARY")

	agentHome := d.AgentHome("doctor")
	os.MkdirAll(agentHome, 0o700)
	scratch := filepath.Join(root, "scratch")
	os.MkdirAll(scratch, 0o700)
	env := sandbox.BuildEnv(sandbox.EnvInput{HostEnv: os.Environ(), Home: agentHome, Scratch: scratch})
	spec := sandbox.BuildSpec(sandbox.BuildInput{
		Roots:       sandbox.ExistingRoots(sandbox.SystemRoots(d.GOOS, nil)),
		ReadGrants:  []string{rg},
		WriteGrants: []string{wg},
		Home:        agentHome, Scratch: scratch, Workdir: wg, Env: env,
	})
	spec.ReadOnly = append(spec.ReadOnly, filepath.Dir(d.VitrineBin))

	out := filepath.Join(wg, "probe.json")
	cmd := []string{d.VitrineBin, "probe",
		"--cred-path", cred, "--home", d.Home, "--state-file", l.Grants,
		"--outside", filepath.Join(root, "outside.txt"),
		"--write-grant", wg, "--read-grant", rg,
		// agent home and scratch are taken from $HOME and $TMPDIR inside
		// the sandbox, because backends may relocate them.
		"--symlink", filepath.Join(wg, "escape"),
		"--canary-env", "VITRINE_CANARY", "--net", ln.Addr().String(),
		"--out", out,
	}
	if _, err := d.Backend.Run(ctx, spec, cmd); err != nil {
		return nil, nil, fmt.Errorf("run probe: %w", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil, nil, fmt.Errorf("probe produced no output (is %s readable inside the sandbox?)", d.VitrineBin)
	}
	var cs []probe.Check
	if err := json.Unmarshal(raw, &cs); err != nil {
		return nil, nil, fmt.Errorf("parse probe output: %w: %s", err, strings.TrimSpace(string(raw)))
	}
	return cs, nil, nil
}
