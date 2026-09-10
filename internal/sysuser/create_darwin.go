//go:build darwin

package sysuser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func dscl(args ...string) error {
	out, err := exec.Command("dscl", append([]string{"."}, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dscl %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return nil
}

// Exists reports whether the vitrine user is present.
func Exists() bool {
	return exec.Command("id", "-u", Name).Run() == nil
}

// nextSystemUID finds a free UID below 500 (hidden system range).
func nextSystemUID() (int, error) {
	out, err := exec.Command("dscl", ".", "-list", "/Users", "UniqueID").Output()
	if err != nil {
		return 0, err
	}
	used := map[int]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 {
			if n, err := strconv.Atoi(f[1]); err == nil {
				used[n] = true
			}
		}
	}
	for uid := 499; uid > 200; uid-- {
		if !used[uid] {
			return uid, nil
		}
	}
	return 0, fmt.Errorf("no free system uid")
}

// Ensure creates the vitrine user and its directories, writes the sudoers
// rule, and configures git for the vitrine user. Must run as root.
func Ensure(vitrineBin string, engineers []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must run as root: sudo vitrine init")
	}
	if !Exists() {
		uid, err := nextSystemUID()
		if err != nil {
			return err
		}
		path := "/Users/" + Name
		if err := dscl("-create", "/Groups/"+Name); err != nil {
			return err
		}
		if err := dscl("-create", "/Groups/"+Name, "PrimaryGroupID", strconv.Itoa(uid)); err != nil {
			return err
		}
		steps := [][]string{
			{"-create", path},
			{"-create", path, "UserShell", "/bin/zsh"},
			{"-create", path, "RealName", "Vitrine Agent Sandbox"},
			{"-create", path, "UniqueID", strconv.Itoa(uid)},
			{"-create", path, "PrimaryGroupID", strconv.Itoa(uid)},
			{"-create", path, "NFSHomeDirectory", Home},
			{"-create", path, "IsHidden", "1"},
		}
		for _, s := range steps {
			if err := dscl(s...); err != nil {
				return err
			}
		}
	}
	uidOut, err := exec.Command("id", "-u", Name).Output()
	if err != nil {
		return err
	}
	uid, _ := strconv.Atoi(strings.TrimSpace(string(uidOut)))
	gidOut, _ := exec.Command("id", "-g", Name).Output()
	gid, _ := strconv.Atoi(strings.TrimSpace(string(gidOut)))

	for _, d := range []string{Home, filepath.Join(Home, "agents")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
		if err := os.Chown(d, uid, gid); err != nil {
			return err
		}
	}
	// /var/lib/vitrine itself must be traversable by engineers so they can
	// create session directories under sessions/.
	if err := os.Chmod(Home, 0o711); err != nil {
		return err
	}
	if err := os.MkdirAll(SessionsDir(), 0o1777); err != nil {
		return err
	}
	if err := os.Chmod(SessionsDir(), 0o1777); err != nil {
		return err
	}
	gitcfg := filepath.Join(Home, ".gitconfig")
	if err := os.WriteFile(gitcfg, []byte("[safe]\n\tdirectory = *\n"), 0o644); err != nil {
		return err
	}
	if err := os.Chown(gitcfg, uid, gid); err != nil {
		return err
	}
	content := SudoersContent(vitrineBin, engineers)
	tmp := SudoersPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o440); err != nil {
		return err
	}
	if out, err := exec.Command("visudo", "-cf", tmp).CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("sudoers validation failed: %v: %s", err, out)
	}
	return os.Rename(tmp, SudoersPath)
}
