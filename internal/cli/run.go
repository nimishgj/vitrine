package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimishgj/vitrine/internal/audit"
	"github.com/nimishgj/vitrine/internal/config"
	"github.com/nimishgj/vitrine/internal/grants"
	"github.com/nimishgj/vitrine/internal/paths"
	"github.com/nimishgj/vitrine/internal/profile"
	"github.com/nimishgj/vitrine/internal/provider"
	"github.com/nimishgj/vitrine/internal/sandbox"
	"github.com/nimishgj/vitrine/internal/sandbox/pick"
	"github.com/nimishgj/vitrine/internal/session"
	"github.com/nimishgj/vitrine/internal/sysuser"
)

// Swappable for tests.
var (
	backendFactory = pick.Default
	providerLookup = provider.Lookup
)

func newRunCmd(e *env) *cobra.Command {
	var agent string
	cmd := &cobra.Command{
		Use:   "run [--agent NAME] -- CMD [ARGS...]",
		Short: "Run any command inside the Vitrine sandbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := agent
			if name == "" {
				name = profile.Generic(args[0]).Name
			}
			code, err := runSessionCmd(e, name, args[0], args[1:])
			if err != nil {
				return err
			}
			if code != 0 {
				return exitError{code: code}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "", "agent profile to use (claude, codex, pi)")
	return cmd
}

// runSession is the shim path: profileName is a builtin, binary resolved from PATH.
func runSession(e *env, profileName string, args []string) (int, error) {
	return runSessionCmd(e, profileName, "", args)
}

// runSessionCmd performs the integrity check, profile resolution and grant
// lookup, then hands off to session.Run. explicitBinary is set by
// `vitrine run`; empty means resolve from the profile.
func runSessionCmd(e *env, profileName, explicitBinary string, args []string) (int, error) {
	cfg, err := config.Load(e.Layout.Config)
	if err != nil {
		return 1, err
	}
	// Integrity.
	if tampered, err := audit.CheckConfig(e.Log, e.Layout.Config); err != nil {
		return 1, err
	} else if tampered {
		fmt.Fprintln(e.Stderr, "vitrine: warning: config.toml was modified outside vitrine (logged)")
	}
	current, err := grants.Load(e.Layout.Grants)
	if err != nil {
		return 1, err
	}
	honoured, tampered, err := audit.CheckGrants(e.Log, e.Layout.Grants, current)
	if err != nil {
		return 1, err
	}
	if tampered {
		fmt.Fprintln(e.Stderr, "vitrine: warning: grants.json was modified outside vitrine; unrecognised grants are ignored (run: vitrine grants accept)")
	}

	// Profile.
	overrides := map[string]string{}
	for k, v := range cfg.Profiles {
		overrides[k] = v.Binary
	}
	var prof profile.Profile
	var binary string
	if explicitBinary != "" {
		if p, ok := profile.Lookup(profileName, overrides); ok {
			prof = p
		} else {
			prof = profile.Generic(explicitBinary)
		}
		binary, err = profile.FindRealBinary(explicitBinary, hostPath(e), e.Layout.Bin)
	} else {
		p, ok := profile.Lookup(profileName, overrides)
		if !ok {
			return 1, fmt.Errorf("unknown agent profile %q", profileName)
		}
		prof = p
		binary, err = profile.FindRealBinary(p.Binary, hostPath(e), e.Layout.Bin)
	}
	if err != nil {
		return 1, err
	}
	// Exec the real file, not a symlink the sandbox may not be able to read,
	// and make the agent's install tree readable. When that tree is under
	// the engineer's home (a common install location), the macOS user
	// boundary needs an ACL for it too.
	if real, err := filepath.EvalSymlinks(binary); err == nil {
		binary = real
	}
	binRoot := profile.InstallRoot(binary)
	if e.GOOS == "darwin" && strings.HasPrefix(binRoot, e.Home+string(filepath.Separator)) {
		if err := grants.ApplyACL(grants.Grant{Path: binRoot, Mode: grants.ModeRead}, e.Home, e.Engineer); err != nil {
			return 1, fmt.Errorf("make agent install readable by the sandbox user: %w", err)
		}
	}

	// Grant for cwd.
	cwd, err := grants.Canonical(e.Workdir)
	if err != nil {
		return 1, err
	}
	g, ok := honoured.Match(cwd)
	if !ok {
		mode, proceed, err := promptTrust(e.Stdin, e.Stdout, e.IsTTY, prof.Name, cwd)
		if err != nil {
			return 1, err
		}
		if !proceed {
			return 1, fmt.Errorf("cancelled")
		}
		g = grants.Grant{Path: cwd, Mode: mode, CreatedAt: time.Now().UTC(), Source: "prompt"}
		if err := addGrant(e, g); err != nil {
			return 1, err
		}
	}

	backend, err := backendFactory(cfg.Backend, e.Home)
	if err != nil {
		return 1, err
	}
	agentHome := func(name string) string { return paths.AgentHome(e.Layout, name) }
	if e.GOOS == "darwin" {
		agentHome = sysuser.AgentHome
	}
	deps := session.Deps{
		Layout: e.Layout, Log: e.Log, Config: cfg, Grants: honoured, Backend: backend,
		Providers: providerLookup, HostEnv: e.HostEnv, GOOS: e.GOOS, Engineer: e.Engineer,
		AgentHome: agentHome,
	}
	return session.Run(context.Background(), deps, session.Request{
		Profile: prof, RealBinary: binary, BinaryRoot: binRoot, Args: args, Workdir: cwd, Grant: g,
	})
}

// addGrant validates, persists, applies ACLs, and logs a grant.
func addGrant(e *env, g grants.Grant) error {
	roots := sandbox.SystemRoots(e.GOOS, nil)
	if err := grants.Refuse(g.Path, e.Home, roots); err != nil {
		return err
	}
	s, err := grants.Load(e.Layout.Grants)
	if err != nil {
		return err
	}
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}
	if err := s.Add(g); err != nil {
		return err
	}
	if err := grants.Save(e.Layout.Grants, s); err != nil {
		return err
	}
	if e.GOOS == "darwin" {
		if err := grants.ApplyACL(g, e.Home, e.Engineer); err != nil {
			return err
		}
	}
	h, err := grants.FileHash(e.Layout.Grants)
	if err != nil {
		return err
	}
	_, err = e.Log.Append(audit.TypeGrantAdd, map[string]any{
		"path": g.Path, "mode": string(g.Mode), "source": g.Source,
		"grants_hash": h, "grants": audit.GrantSnapshot(s),
	})
	return err
}

func hostPath(e *env) string {
	for _, kv := range e.HostEnv {
		if strings.HasPrefix(kv, "PATH=") {
			return kv[5:]
		}
	}
	return os.Getenv("PATH")
}
