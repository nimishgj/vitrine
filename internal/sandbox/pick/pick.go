// Package pick chooses the sandbox backend for this OS and config.
package pick

import (
	"fmt"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

// Default returns the backend named in config for this OS.
func Default(cfgBackend, engineerHome string) (sandbox.Backend, error) {
	switch cfgBackend {
	case "", "native":
		return native(engineerHome), nil
	default:
		return nil, fmt.Errorf("backend %q is not available in this version", cfgBackend)
	}
}
