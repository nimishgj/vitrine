package audit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newLog(t *testing.T) *Log {
	t.Helper()
	dir := t.TempDir()
	return Open(filepath.Join(dir, "audit.jsonl"), filepath.Join(dir, "audit.head"))
}

func TestAppendChainsAndVerifies(t *testing.T) {
	l := newLog(t)
	e1, err := l.Append(TypeInit, map[string]any{"platform": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if e1.Seq != 1 || e1.Prev != "" || e1.Hash == "" {
		t.Fatalf("bad first event: %+v", e1)
	}
	e2, _ := l.Append(TypeSessionStart, map[string]any{"session": "s1"})
	if e2.Seq != 2 || e2.Prev != e1.Hash {
		t.Fatalf("chain broken: %+v", e2)
	}
	if err := l.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
	evs, _ := l.ReadAll()
	if len(evs) != 2 || evs[1].Data["session"] != "s1" {
		t.Fatalf("readall: %+v", evs)
	}
	head, _ := os.ReadFile(l.HeadPath)
	if strings.TrimSpace(string(head)) != e2.Hash {
		t.Fatalf("head = %q want %q", head, e2.Hash)
	}
}

func TestVerifyDetectsEditedLine(t *testing.T) {
	l := newLog(t)
	l.Append(TypeInit, nil)
	l.Append(TypeSessionStart, map[string]any{"session": "s1"})
	raw, _ := os.ReadFile(l.Path)
	edited := strings.Replace(string(raw), `"s1"`, `"s2"`, 1)
	os.WriteFile(l.Path, []byte(edited), 0o600)
	var ce *ChainError
	if err := l.Verify(); !errors.As(err, &ce) || ce.Seq != 2 {
		t.Fatalf("want ChainError seq 2, got %v", err)
	}
}

func TestVerifyDetectsDeletedLine(t *testing.T) {
	l := newLog(t)
	l.Append(TypeInit, nil)
	l.Append(TypeSessionStart, nil)
	l.Append(TypeSessionEnd, nil)
	raw, _ := os.ReadFile(l.Path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	os.WriteFile(l.Path, []byte(lines[0]+"\n"+lines[2]+"\n"), 0o600)
	var ce *ChainError
	if err := l.Verify(); !errors.As(err, &ce) || ce.Seq != 3 {
		t.Fatalf("want ChainError seq 3, got %v", err)
	}
}

func TestVerifyDetectsTruncatedTail(t *testing.T) {
	l := newLog(t)
	l.Append(TypeInit, nil)
	l.Append(TypeSessionStart, nil)
	raw, _ := os.ReadFile(l.Path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	os.WriteFile(l.Path, []byte(lines[0]+"\n"), 0o600)
	var ce *ChainError
	if err := l.Verify(); !errors.As(err, &ce) || ce.Reason != "head mismatch" {
		t.Fatalf("want head mismatch, got %v", err)
	}
}

func TestEmptyLogVerifies(t *testing.T) {
	l := newLog(t)
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestFilePermissions(t *testing.T) {
	l := newLog(t)
	l.Append(TypeInit, nil)
	st, _ := os.Stat(l.Path)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", st.Mode().Perm())
	}
}
