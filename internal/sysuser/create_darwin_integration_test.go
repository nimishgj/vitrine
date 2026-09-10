//go:build darwin && integration

package sysuser

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestEnsureIdempotent(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("run as root")
	}
	bin := "/usr/local/bin/vitrine-test"
	os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	defer os.Remove(bin)
	for i := 0; i < 2; i++ {
		if err := Ensure(bin, []string{"root"}); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	if !Exists() {
		t.Fatal("user missing")
	}
	s, _ := os.ReadFile(SudoersPath)
	if !strings.Contains(string(s), "root ALL=(vitrine) NOPASSWD: "+bin+" launch *") {
		t.Fatalf("sudoers:\n%s", s)
	}
	st, err := os.Stat(SessionsDir())
	if err != nil || st.Mode().Perm() != 0o777 || st.Mode()&os.ModeSticky == 0 {
		t.Fatalf("sessions mode %v %v", st.Mode(), err)
	}
	if err := exec.Command("sudo", "-n", "-u", Name, bin, "launch", "--check").Run(); err != nil {
		t.Fatalf("sudo rule not effective: %v", err)
	}
}
