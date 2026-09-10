// Package shim installs the agent-named symlinks that point at vitrine.
package shim

import (
	"os"
	"path/filepath"
	"strings"
)

// Install creates binDir/<name> -> vitrineBinary for each name. Idempotent.
func Install(binDir, vitrineBinary string, names []string) error {
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return err
	}
	for _, n := range names {
		link := filepath.Join(binDir, n)
		cur, err := os.Readlink(link)
		if err == nil && cur == vitrineBinary {
			continue
		}
		if fileExists(link) {
			if err := os.Remove(link); err != nil {
				return err
			}
		}
		if err := os.Symlink(vitrineBinary, link); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// StripFromPath removes every entry equal to binDir from a PATH string.
func StripFromPath(pathEnv, binDir string) string {
	want := filepath.Clean(binDir)
	var keep []string
	for _, d := range filepath.SplitList(pathEnv) {
		if d == "" || filepath.Clean(d) == want {
			continue
		}
		keep = append(keep, d)
	}
	return strings.Join(keep, string(os.PathListSeparator))
}
