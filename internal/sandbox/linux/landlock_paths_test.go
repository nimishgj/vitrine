package linux

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestClassifyPaths(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)
	file := filepath.Join(dir, "resolv.conf")
	os.WriteFile(file, []byte("x"), 0o644)
	missing := filepath.Join(dir, "nope")

	dirs, files := classifyPaths([]string{sub, file, missing}, os.Stat)
	if !reflect.DeepEqual(dirs, []string{sub}) {
		t.Fatalf("dirs %v", dirs)
	}
	if !reflect.DeepEqual(files, []string{file}) {
		t.Fatalf("files %v", files)
	}
}
