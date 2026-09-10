package sandbox

import (
	"sort"
	"strings"

	"github.com/nimishgj/vitrine/internal/shim"
)

// EnvInput is the material BuildEnv assembles.
type EnvInput struct {
	HostEnv  []string // os.Environ()
	ShimDir  string
	Home     string
	Scratch  string
	Extra    map[string]string // profile extras and credential env; override host
	GitName  string
	GitEmail string
}

var passthrough = map[string]bool{
	"PATH": true, "TERM": true, "LANG": true, "COLORTERM": true,
}

// BuildEnv constructs the sandbox environment from an explicit allowlist.
// Nothing else from the host environment crosses the boundary.
func BuildEnv(in EnvInput) map[string]string {
	env := map[string]string{}
	for _, kv := range in.HostEnv {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if passthrough[k] || strings.HasPrefix(k, "LC_") {
			env[k] = v
		}
	}
	if p, ok := env["PATH"]; ok && in.ShimDir != "" {
		env["PATH"] = shim.StripFromPath(p, in.ShimDir)
	}
	env["HOME"] = in.Home
	env["TMPDIR"] = in.Scratch
	env["VITRINE"] = "1"
	if in.GitName != "" {
		env["GIT_AUTHOR_NAME"] = in.GitName
		env["GIT_COMMITTER_NAME"] = in.GitName
	}
	if in.GitEmail != "" {
		env["GIT_AUTHOR_EMAIL"] = in.GitEmail
		env["GIT_COMMITTER_EMAIL"] = in.GitEmail
	}
	for k, v := range in.Extra {
		env[k] = v
	}
	return env
}

// GitIdentity reads the engineer's git user.name and user.email via runGit.
func GitIdentity(runGit func(args ...string) (string, error)) (name, email string) {
	if out, err := runGit("config", "--get", "user.name"); err == nil {
		name = strings.TrimSpace(out)
	}
	if out, err := runGit("config", "--get", "user.email"); err == nil {
		email = strings.TrimSpace(out)
	}
	return
}

// EnvSlice renders a map as KEY=VALUE pairs, sorted for determinism.
func EnvSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}
