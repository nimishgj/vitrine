// Package provider defines how Vitrine mints read-only credentials for a target.
package provider

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/nimishgj/vitrine/internal/config"
)

// Target is a configured remote system.
type Target = config.Target

// SessionInfo identifies the session a credential is minted for.
type SessionInfo struct {
	ID       string
	Agent    string
	Workdir  string
	Engineer string // OS username of the engineer
}

// Artifact is a one-time setup file for the platform team (manifest, SQL, ...).
type Artifact struct {
	Name    string
	Content []byte
}

// Credential is what enters the sandbox. Files are relative paths placed
// under the session credential directory, mode 0400. Env values may contain
// the literal $CRED_DIR, which the core replaces with the absolute directory.
type Credential struct {
	Files     map[string][]byte
	Env       map[string]string
	Identity  string
	ExpiresAt time.Time
	Handle    any
}

// Provider knows one kind of target.
type Provider interface {
	Kind() string
	Setup(ctx context.Context, t Target) ([]Artifact, error)
	Mint(ctx context.Context, t Target, s SessionInfo) (Credential, error)
	Cleanup(ctx context.Context, c Credential) error
}

var (
	mu       sync.RWMutex
	registry = map[string]Provider{}
)

// Register adds p under its kind, replacing any previous registration.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Kind()] = p
}

// Lookup returns the provider for kind.
func Lookup(kind string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[kind]
	return p, ok
}

// Kinds lists registered kinds, sorted.
func Kinds() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// reset clears the registry (tests only).
func reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = map[string]Provider{}
}
