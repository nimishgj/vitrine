package provider

import (
	"context"
	"testing"
)

type stub struct{ kind string }

func (s stub) Kind() string                                                   { return s.kind }
func (s stub) Setup(context.Context, Target) ([]Artifact, error)             { return nil, nil }
func (s stub) Mint(context.Context, Target, SessionInfo) (Credential, error) { return Credential{}, nil }
func (s stub) Cleanup(context.Context, Credential) error                     { return nil }

func TestRegistry(t *testing.T) {
	reset()
	Register(stub{"a"})
	Register(stub{"b"})
	if p, ok := Lookup("a"); !ok || p.Kind() != "a" {
		t.Fatal("lookup a")
	}
	if _, ok := Lookup("zzz"); ok {
		t.Fatal("unknown kind")
	}
	if k := Kinds(); len(k) != 2 || k[0] != "a" || k[1] != "b" {
		t.Fatalf("kinds %v", k)
	}
}
