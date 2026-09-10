package sysuser

import (
	"strings"
	"testing"
)

func TestSudoersContent(t *testing.T) {
	s := SudoersContent("/opt/homebrew/bin/vitrine", []string{"nimisha", "bob"})
	for _, want := range []string{
		"nimisha ALL=(vitrine) NOPASSWD: /opt/homebrew/bin/vitrine launch *",
		"bob ALL=(vitrine) NOPASSWD: /opt/homebrew/bin/vitrine launch *",
		"Defaults!/opt/homebrew/bin/vitrine !requiretty",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if !strings.HasSuffix(s, "\n") {
		t.Error("sudoers files must end with a newline")
	}
}

func TestAgentHome(t *testing.T) {
	if AgentHome("claude") != "/var/lib/vitrine/agents/claude/home" {
		t.Fatal(AgentHome("claude"))
	}
}
