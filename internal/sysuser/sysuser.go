// Package sysuser describes the dedicated OS user Vitrine uses on macOS.
package sysuser

import (
	"path/filepath"
	"strings"
)

const (
	// Name is the hidden system user the agent runs as.
	Name = "vitrine"
	// Home is that user's home directory.
	Home = "/var/lib/vitrine"
	// SudoersPath is the rule file installed by `vitrine init`.
	SudoersPath = "/etc/sudoers.d/vitrine"
)

// AgentHome returns the persistent state home for an agent profile.
func AgentHome(profileName string) string {
	return filepath.Join(Home, "agents", profileName, "home")
}

// SessionsDir holds per-session credential and scratch directories that
// must be readable by the vitrine user.
func SessionsDir() string { return filepath.Join(Home, "sessions") }

// SudoersContent renders the rule allowing each engineer to run
// `vitrine launch` as the vitrine user without a password.
func SudoersContent(vitrineBin string, engineers []string) string {
	var b strings.Builder
	b.WriteString("# Managed by vitrine init. Allows engineers to launch sandboxed agents as the vitrine user.\n")
	b.WriteString("Defaults!" + vitrineBin + " !requiretty\n")
	for _, e := range engineers {
		b.WriteString(e + " ALL=(" + Name + ") NOPASSWD: " + vitrineBin + " launch *\n")
	}
	return b.String()
}
