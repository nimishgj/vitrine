//go:build !darwin

package sysuser

import "errors"

// Exists is always false off macOS; Linux needs no dedicated user.
func Exists() bool { return false }

// Ensure is not applicable off macOS.
func Ensure(string, []string) error { return errors.New("dedicated user is only used on macOS") }
