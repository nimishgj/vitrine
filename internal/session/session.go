// Package session orchestrates one sandboxed agent launch.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/profile"
	"github.com/nimishgj/vitrine/internal/provider"
	"github.com/nimishgj/vitrine/internal/sandbox"
)

var (
	// ErrMint wraps a provider failure. The session never starts.
	ErrMint = errors.New("credential minting failed")
	// ErrBackendUnavailable means the sandbox cannot be established.
	ErrBackendUnavailable = errors.New("sandbox backend unavailable")
)

// Deps are the collaborators Run needs; the CLI wires real ones, tests wire fakes.
type Deps struct {
	Layout    paths.Layout
	Log       *audit.Log
	Config    *config.Config
	Grants    *grants.Store
	Backend   sandbox.Backend
	Providers func(kind string) (provider.Provider, bool)
	HostEnv   []string
	GOOS      string
	Engineer  string
	AgentHome func(name string) string
	Now       func() time.Time
	NewID     func() string
}

// Request describes what to launch.
type Request struct {
	Profile    profile.Profile
	RealBinary string
	Args       []string
	Workdir    string
	Grant      grants.Grant // already matched and honoured by the caller
}

type minted struct {
	target config.Target
	prov   provider.Provider
	cred   provider.Credential
}

// Run mints credentials, builds the sandbox spec, logs the start, runs the
// backend, cleans up, and logs the end.
func Run(ctx context.Context, d Deps, r Request) (int, error) {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.NewID == nil {
		d.NewID = newID
	}
	if err := d.Backend.Available(); err != nil {
		return 1, fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	id := d.NewID()

	// Session directories (credentials + scratch) live under the state root
	// so they inherit its 0700 and are never inside a grant.
	sessDir := filepath.Join(d.Layout.Root, "sessions", id)
	credDir := filepath.Join(sessDir, "creds")
	scratch := filepath.Join(sessDir, "scratch")
	for _, p := range []string{credDir, scratch} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			return 1, err
		}
	}
	defer os.RemoveAll(sessDir)

	// Mint.
	info := provider.SessionInfo{ID: id, Agent: r.Profile.Name, Workdir: r.Workdir, Engineer: d.Engineer}
	var ms []minted
	credEnv := map[string]string{}
	identities := map[string]any{}
	for _, t := range d.Config.Targets {
		p, ok := d.Providers(t.Kind)
		if !ok {
			return 1, fmt.Errorf("%w: target %q has unknown kind %q", ErrMint, t.Name, t.Kind)
		}
		c, err := p.Mint(ctx, t, info)
		if err != nil {
			cleanupAll(ctx, ms)
			return 1, fmt.Errorf("%w: target %q: %v", ErrMint, t.Name, err)
		}
		ms = append(ms, minted{t, p, c})
		if err := placeFiles(credDir, c.Files); err != nil {
			cleanupAll(ctx, ms)
			return 1, err
		}
		for k, v := range c.Env {
			credEnv[k] = strings.ReplaceAll(v, "$CRED_DIR", credDir)
		}
		identities[t.Name] = c.Identity
	}
	defer cleanupAll(ctx, ms)

	// Spec.
	home := d.AgentHome(r.Profile.Name)
	if err := os.MkdirAll(home, 0o700); err != nil {
		return 1, err
	}
	gitName, gitEmail := sandbox.GitIdentity(func(args ...string) (string, error) {
		out, err := exec.Command("git", args...).Output()
		return string(out), err
	})
	env := sandbox.BuildEnv(sandbox.EnvInput{
		HostEnv: d.HostEnv, ShimDir: d.Layout.Bin, Home: home, Scratch: scratch,
		Extra: credEnv, GitName: gitName, GitEmail: gitEmail,
	})
	var ro, rw []string
	if r.Grant.Mode == grants.ModeWrite {
		rw = append(rw, r.Grant.Path)
	} else {
		ro = append(ro, r.Grant.Path)
	}
	spec := sandbox.BuildSpec(sandbox.BuildInput{
		Roots:       sandbox.ExistingRoots(sandbox.SystemRoots(d.GOOS, d.Config.SystemRoots)),
		ReadGrants:  ro,
		WriteGrants: rw,
		CredDir:     credDir,
		Home:        home,
		Scratch:     scratch,
		Workdir:     r.Workdir,
		Env:         env,
	})

	// Audit start.
	started := d.Now()
	if _, err := d.Log.Append(audit.TypeSessionStart, map[string]any{
		"session": id, "agent": r.Profile.Name, "binary": r.RealBinary, "cwd": r.Workdir,
		"grant_path": r.Grant.Path, "grant_mode": string(r.Grant.Mode),
		"identities": identities, "backend": d.Backend.Name(),
	}); err != nil {
		return 1, fmt.Errorf("audit log unwritable, refusing to launch: %w", err)
	}

	// Run.
	cmd := append([]string{r.RealBinary}, r.Args...)
	code, runErr := d.Backend.Run(ctx, spec, cmd)

	// Cleanup happens via defers; log end now.
	data := map[string]any{"session": id, "exit_code": code, "duration_ms": d.Now().Sub(started).Milliseconds()}
	if runErr != nil {
		data["error"] = runErr.Error()
	}
	if _, err := d.Log.Append(audit.TypeSessionEnd, data); err != nil {
		return code, err
	}
	return code, runErr
}

func placeFiles(credDir string, files map[string][]byte) error {
	for rel, content := range files {
		if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
			return fmt.Errorf("provider returned unsafe credential path %q", rel)
		}
		p := filepath.Join(credDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(p, content, 0o400); err != nil {
			return err
		}
	}
	return nil
}

func cleanupAll(ctx context.Context, ms []minted) {
	for i := len(ms) - 1; i >= 0; i-- {
		_ = ms[i].prov.Cleanup(ctx, ms[i].cred)
	}
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}
