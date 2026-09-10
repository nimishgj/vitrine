//go:build !linux

package probe

import "errors"

func tryTIOCSTI() error { return errors.New("not applicable") }
