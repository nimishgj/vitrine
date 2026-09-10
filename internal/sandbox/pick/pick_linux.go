//go:build linux

package pick

import (
	"github.com/nimishgj/vitrine/internal/sandbox"
	lnx "github.com/nimishgj/vitrine/internal/sandbox/linux"
)

func native(home string) sandbox.Backend { return lnx.New(home) }
