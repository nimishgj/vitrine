package audit

import (
	"os"
	"time"

	"github.com/nimishgj/vitrine/internal/grants"
)

// Diff describes how the current grants file differs from the recognised set.
type Diff struct {
	Added   []grants.Grant `json:"added"`
	Removed []grants.Grant `json:"removed"`
	Changed []grants.Grant `json:"changed"` // current entry whose mode differs
}

// GrantSnapshot renders a store as JSON-safe maps for embedding in events.
func GrantSnapshot(s *grants.Store) []map[string]any {
	out := make([]map[string]any, 0, len(s.Grants))
	for _, g := range s.Grants {
		out = append(out, map[string]any{
			"path": g.Path, "mode": string(g.Mode),
			"created_at": g.CreatedAt.UTC().Format(time.RFC3339Nano), "source": g.Source,
		})
	}
	return out
}

func isGrantEvent(t string) bool {
	return t == TypeGrantAdd || t == TypeGrantRevoke || t == TypeGrantAccept
}

// ExpectedHashes returns the last recorded grants and config file hashes.
func ExpectedHashes(evs []Event) (grantsHash, configHash string) {
	for _, e := range evs {
		if isGrantEvent(e.Type) {
			if h, ok := e.Data["grants_hash"].(string); ok {
				grantsHash = h
			}
		}
		if e.Type == TypeConfigWrite {
			if h, ok := e.Data["config_hash"].(string); ok {
				configHash = h
			}
		}
	}
	return
}

// RecognisedGrants reconstructs the grant set from the latest grant event snapshot.
func RecognisedGrants(evs []Event) []grants.Grant {
	var snap []any
	for _, e := range evs {
		if isGrantEvent(e.Type) {
			if s, ok := e.Data["grants"].([]any); ok {
				snap = s
			}
		}
	}
	out := make([]grants.Grant, 0, len(snap))
	for _, item := range snap {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		g := grants.Grant{Source: str(m["source"])}
		g.Path = str(m["path"])
		g.Mode = grants.Mode(str(m["mode"]))
		if ts, err := time.Parse(time.RFC3339Nano, str(m["created_at"])); err == nil {
			g.CreatedAt = ts
		}
		out = append(out, g)
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// GrantsDiff compares recognised against current by path.
func GrantsDiff(recognised, current []grants.Grant) Diff {
	rec := map[string]grants.Grant{}
	for _, g := range recognised {
		rec[g.Path] = g
	}
	cur := map[string]grants.Grant{}
	for _, g := range current {
		cur[g.Path] = g
	}
	var d Diff
	for p, g := range cur {
		r, ok := rec[p]
		switch {
		case !ok:
			d.Added = append(d.Added, g)
		case r.Mode != g.Mode:
			d.Changed = append(d.Changed, g)
		}
	}
	for p, g := range rec {
		if _, ok := cur[p]; !ok {
			d.Removed = append(d.Removed, g)
		}
	}
	return d
}

// CheckGrants verifies the grants file against the chain. On mismatch it logs
// a tamper event and returns only grants that are both recognised and present.
// A changed entry is honoured at the narrower of the two modes.
func CheckGrants(l *Log, grantsPath string, current *grants.Store) (*grants.Store, bool, error) {
	evs, err := l.ReadAll()
	if err != nil {
		return nil, false, err
	}
	expected, _ := ExpectedHashes(evs)
	actual, err := grants.FileHash(grantsPath)
	if err != nil {
		return nil, false, err
	}
	if expected == actual {
		return current, false, nil
	}
	recognised := RecognisedGrants(evs)
	diff := GrantsDiff(recognised, current.Grants)
	mtime := ""
	if st, err := os.Stat(grantsPath); err == nil {
		mtime = st.ModTime().UTC().Format(time.RFC3339)
	}
	_, err = l.Append(TypeTamper, map[string]any{
		"file": "grants", "expected_hash": expected, "actual_hash": actual,
		"mtime": mtime, "diff": diff,
	})
	if err != nil {
		return nil, true, err
	}
	rec := map[string]grants.Grant{}
	for _, g := range recognised {
		rec[g.Path] = g
	}
	honoured := &grants.Store{Version: current.Version}
	for _, g := range current.Grants {
		r, ok := rec[g.Path]
		if !ok {
			continue
		}
		mode := g.Mode
		if r.Mode == grants.ModeRead || g.Mode == grants.ModeRead {
			mode = grants.ModeRead
		}
		honoured.Grants = append(honoured.Grants, grants.Grant{Path: g.Path, Mode: mode, CreatedAt: r.CreatedAt, Source: r.Source})
	}
	return honoured, true, nil
}

// CheckConfig verifies the config file hash and logs a tamper event on mismatch.
func CheckConfig(l *Log, configPath string) (bool, error) {
	evs, err := l.ReadAll()
	if err != nil {
		return false, err
	}
	_, expected := ExpectedHashes(evs)
	actual, err := grants.FileHash(configPath)
	if err != nil {
		return false, err
	}
	if expected == actual {
		return false, nil
	}
	mtime := ""
	if st, err := os.Stat(configPath); err == nil {
		mtime = st.ModTime().UTC().Format(time.RFC3339)
	}
	_, err = l.Append(TypeTamper, map[string]any{
		"file": "config", "expected_hash": expected, "actual_hash": actual, "mtime": mtime,
	})
	return true, err
}
