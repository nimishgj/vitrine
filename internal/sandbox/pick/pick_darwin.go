//go:build darwin

package pick

import (
	"github.com/nimishgj/vitrine/internal/sandbox"
	"github.com/nimishgj/vitrine/internal/sandbox/darwin"
)

func native(home string) sandbox.Backend { return darwin.New(home) }
