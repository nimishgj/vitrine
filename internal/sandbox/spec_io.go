package sandbox

import (
	"encoding/json"
	"io"
)

// WriteSpec serialises a Spec as JSON.
func WriteSpec(w io.Writer, s Spec) error { return json.NewEncoder(w).Encode(s) }

// ReadSpec parses a Spec written by WriteSpec.
func ReadSpec(r io.Reader) (Spec, error) {
	var s Spec
	err := json.NewDecoder(r).Decode(&s)
	return s, err
}
