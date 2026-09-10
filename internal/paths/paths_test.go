package paths

import (
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	l := Resolve("/home/u")
	want := map[string]string{
		"Root":      "/home/u/.vitrine",
		"Config":    "/home/u/.vitrine/config.toml",
		"Grants":    "/home/u/.vitrine/grants.json",
		"Audit":     "/home/u/.vitrine/audit.jsonl",
		"AuditHead": "/home/u/.vitrine/audit.head",
		"Bin":       "/home/u/.vitrine/bin",
		"Agents":    "/home/u/.vitrine/agents",
	}
	got := map[string]string{
		"Root": l.Root, "Config": l.Config, "Grants": l.Grants, "Audit": l.Audit,
		"AuditHead": l.AuditHead, "Bin": l.Bin, "Agents": l.Agents,
	}
	for k, v := range want {
		if got[k] != filepath.FromSlash(v) {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestAgentHome(t *testing.T) {
	l := Resolve("/home/u")
	if got := AgentHome(l, "claude"); got != filepath.FromSlash("/home/u/.vitrine/agents/claude/home") {
		t.Errorf("AgentHome = %q", got)
	}
}
