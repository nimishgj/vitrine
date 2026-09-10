//go:build darwin

package grants

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nimishgj/vitrine/internal/sysuser"
)

const (
	readPerms  = "read,readattr,readextattr,readsecurity,list,search,execute"
	writePerms = readPerms + ",write,append,delete,add_file,add_subdirectory,delete_child,writeattr,writeextattr"
	inherit    = ",file_inherit,directory_inherit"
	searchOnly = "search"
)

func chmod(args ...string) error {
	out, err := exec.Command("chmod", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("chmod %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return nil
}

func ace(user, perms string) string { return "user:" + user + " allow " + perms }

// ancestors lists directories from home down to the parent of p (inclusive of home).
func ancestors(p, home string) []string {
	var out []string
	cur := filepath.Dir(p)
	for {
		out = append(out, cur)
		if cur == home || cur == "/" || len(cur) < len(home) {
			break
		}
		cur = filepath.Dir(cur)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ApplyACL grants the vitrine user access to g.Path and gives the engineer
// an inherited full-access ACE so vitrine-created files stay theirs to edit.
func ApplyACL(g Grant, home, engineer string) error {
	if strings.HasPrefix(g.Path, home+"/") {
		for _, anc := range ancestors(g.Path, home) {
			if err := chmod("+a", ace(sysuser.Name, searchOnly), anc); err != nil {
				return err
			}
		}
	}
	perms := readPerms
	if g.Mode == ModeWrite {
		perms = writePerms
	}
	if err := chmod("-R", "+a", ace(sysuser.Name, perms+inherit), g.Path); err != nil {
		return err
	}
	if g.Mode == ModeWrite && engineer != "" {
		if err := chmod("-R", "+a", ace(engineer, writePerms+inherit), g.Path); err != nil {
			return err
		}
	}
	return nil
}

// RemoveACL strips Vitrine's ACEs from g.Path and from ancestors no longer
// needed by any remaining grant. chmod -a removes an ACE only when the
// principal and permission set match exactly, so removing both variants is
// safe and only the entries Vitrine added disappear.
func RemoveACL(g Grant, home string, remaining []Grant) error {
	for _, perms := range []string{writePerms + inherit, readPerms + inherit} {
		_ = chmod("-R", "-a", ace(sysuser.Name, perms), g.Path)
	}
	if engineer := currentUser(); engineer != "" {
		_ = chmod("-R", "-a", ace(engineer, writePerms+inherit), g.Path)
	}
	still := map[string]bool{}
	for _, r := range remaining {
		for _, anc := range ancestors(r.Path, home) {
			still[anc] = true
		}
	}
	if strings.HasPrefix(g.Path, home+"/") {
		for _, anc := range ancestors(g.Path, home) {
			if !still[anc] {
				_ = chmod("-a", ace(sysuser.Name, searchOnly), anc)
			}
		}
	}
	return nil
}

func currentUser() string {
	out, err := exec.Command("id", "-un").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
