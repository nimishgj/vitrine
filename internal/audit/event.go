// Package audit implements the hash-chained, append-only audit log.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Event types.
const (
	TypeSessionStart = "session.start"
	TypeSessionEnd   = "session.end"
	TypeGrantAdd     = "grant.add"
	TypeGrantRevoke  = "grant.revoke"
	TypeGrantAccept  = "grant.accept"
	TypeConfigWrite  = "config.write"
	TypeTamper       = "tamper"
	TypeDoctor       = "doctor"
	TypeInit         = "init"
)

// Event is one line of the audit log.
type Event struct {
	Seq  int            `json:"seq"`
	TS   time.Time      `json:"ts"`
	Prev string         `json:"prev"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
	Hash string         `json:"hash,omitempty"`
}

// HashEvent computes sha256 over the canonical JSON of e without its Hash field.
// encoding/json sorts map keys, so the encoding is deterministic.
func HashEvent(e Event) string {
	e.Hash = ""
	b, err := json.Marshal(e)
	if err != nil {
		// Event contains only JSON-safe types; this cannot happen with our data.
		panic(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
