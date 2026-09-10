package fake

import (
	"context"
	"errors"
	"testing"

	"github.com/nimishgj/vitrine/internal/provider"
)

func TestMintAndCleanup(t *testing.T) {
	var cleaned []string
	p := &Provider{Cleaned: &cleaned}
	c, err := p.Mint(context.Background(), provider.Target{Name: "t", Kind: "fake"}, provider.SessionInfo{ID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	if string(c.Files["fake/token"]) != "fake-token-s1" || c.Identity != "fake:s1" || c.Env["FAKE_TOKEN_FILE"] != "$CRED_DIR/fake/token" {
		t.Fatalf("%+v", c)
	}
	if err := p.Cleanup(context.Background(), c); err != nil || len(cleaned) != 1 || cleaned[0] != "s1" {
		t.Fatal("cleanup")
	}
	p2 := &Provider{MintErr: errors.New("boom")}
	if _, err := p2.Mint(context.Background(), provider.Target{}, provider.SessionInfo{}); err == nil {
		t.Fatal("expected error")
	}
}
