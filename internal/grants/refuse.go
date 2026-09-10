package grants

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Refuse returns an error if a grant on canonical path p must be refused.
// Rules: root, system roots and anything under them, the home directory
// itself, any direct child of home whose name starts with ".", and anything
// under ~/.vitrine.
func Refuse(p, home string, systemRoots []string) error {
	sep := string(filepath.Separator)
	if p == sep {
		return fmt.Errorf("refusing to grant the filesystem root")
	}
	for _, r := range systemRoots {
		if isUnder(p, r) {
			return fmt.Errorf("refusing to grant %s: inside system root %s", p, r)
		}
	}
	if p == home {
		return fmt.Errorf("refusing to grant the home directory itself")
	}
	vit := filepath.Join(home, ".vitrine")
	if isUnder(p, vit) {
		return fmt.Errorf("refusing to grant %s: inside %s", p, vit)
	}
	if strings.HasPrefix(p, home+sep) {
		rel := strings.TrimPrefix(p, home+sep)
		first := strings.SplitN(rel, sep, 2)[0]
		if strings.HasPrefix(first, ".") {
			return fmt.Errorf("refusing to grant %s: dotfiles and dot-directories directly under home often hold credentials", p)
		}
	}
	return nil
}
