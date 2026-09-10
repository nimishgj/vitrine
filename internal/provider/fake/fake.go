// Package fake is a test-only provider. It is registered by tests, never by
// the release binary.
package fake

import (
	"context"
	"time"

	"github.com/nimishgj/vitrine/internal/provider"
)

// Provider mints a predictable token file.
type Provider struct {
	MintErr error
	Cleaned *[]string
}

func (p *Provider) Kind() string { return "fake" }

func (p *Provider) Setup(context.Context, provider.Target) ([]provider.Artifact, error) {
	return []provider.Artifact{{Name: "README.txt", Content: []byte("fake provider needs no setup\n")}}, nil
}

func (p *Provider) Mint(_ context.Context, _ provider.Target, s provider.SessionInfo) (provider.Credential, error) {
	if p.MintErr != nil {
		return provider.Credential{}, p.MintErr
	}
	return provider.Credential{
		Files:     map[string][]byte{"fake/token": []byte("fake-token-" + s.ID)},
		Env:       map[string]string{"FAKE_TOKEN_FILE": "$CRED_DIR/fake/token"},
		Identity:  "fake:" + s.ID,
		ExpiresAt: time.Now().Add(time.Hour),
		Handle:    s.ID,
	}, nil
}

func (p *Provider) Cleanup(_ context.Context, c provider.Credential) error {
	if p.Cleaned != nil {
		*p.Cleaned = append(*p.Cleaned, c.Handle.(string))
	}
	return nil
}
