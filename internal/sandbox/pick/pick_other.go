//go:build !linux

package pick

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nimishgj/vitrine/internal/sandbox"
)

type unavailable struct{}

func (unavailable) Name() string { return "unsupported" }
func (unavailable) Available() error {
	return fmt.Errorf("no native sandbox backend for %s", runtime.GOOS)
}
func (unavailable) Run(context.Context, sandbox.Spec, []string) (int, error) {
	return 1, fmt.Errorf("no native sandbox backend for %s", runtime.GOOS)
}

func native(string) sandbox.Backend { return unavailable{} }
