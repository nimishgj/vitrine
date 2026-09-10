package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Log is an append-only JSONL file with a head file holding the last hash.
type Log struct {
	Path     string
	HeadPath string
}

// ChainError reports the first broken link in the chain.
type ChainError struct {
	Seq    int
	Reason string
}

func (c *ChainError) Error() string {
	return fmt.Sprintf("audit chain broken at seq %d: %s", c.Seq, c.Reason)
}

// Open returns a Log for the given paths. Files are created on first Append.
func Open(path, headPath string) *Log {
	return &Log{Path: path, HeadPath: headPath}
}

// ReadAll parses every event in order. A missing file is an empty log.
func (l *Log) ReadAll() ([]Event, error) {
	f, err := os.Open(l.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return out, fmt.Errorf("audit line %d: %w", len(out)+1, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// Last returns the final event, or ok=false for an empty log.
func (l *Log) Last() (Event, bool, error) {
	evs, err := l.ReadAll()
	if err != nil || len(evs) == 0 {
		return Event{}, false, err
	}
	return evs[len(evs)-1], true, nil
}

// Append writes one event chained to the previous one and updates the head file.
func (l *Log) Append(typ string, data map[string]any) (Event, error) {
	if data == nil {
		data = map[string]any{}
	}
	last, ok, err := l.Last()
	if err != nil {
		return Event{}, err
	}
	e := Event{Seq: 1, TS: time.Now().UTC().Truncate(time.Millisecond), Type: typ, Data: data}
	if ok {
		e.Seq = last.Seq + 1
		e.Prev = last.Hash
	}
	e.Hash = HashEvent(e)
	b, err := json.Marshal(e)
	if err != nil {
		return Event{}, err
	}
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Event{}, err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return Event{}, err
	}
	if err := f.Close(); err != nil {
		return Event{}, err
	}
	if err := os.WriteFile(l.HeadPath, []byte(e.Hash+"\n"), 0o600); err != nil {
		return Event{}, err
	}
	return e, nil
}

// Verify recomputes every hash and prev link and checks the head file.
func (l *Log) Verify() error {
	evs, err := l.ReadAll()
	if err != nil {
		return err
	}
	prev := ""
	for i, e := range evs {
		if e.Seq != i+1 {
			return &ChainError{Seq: e.Seq, Reason: fmt.Sprintf("expected seq %d", i+1)}
		}
		if e.Prev != prev {
			return &ChainError{Seq: e.Seq, Reason: "prev does not match previous hash"}
		}
		if HashEvent(e) != e.Hash {
			return &ChainError{Seq: e.Seq, Reason: "hash does not match content"}
		}
		prev = e.Hash
	}
	headRaw, err := os.ReadFile(l.HeadPath)
	if errors.Is(err, os.ErrNotExist) {
		if len(evs) == 0 {
			return nil
		}
		return &ChainError{Seq: len(evs), Reason: "head mismatch"}
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(headRaw)) != prev {
		return &ChainError{Seq: len(evs), Reason: "head mismatch"}
	}
	return nil
}
