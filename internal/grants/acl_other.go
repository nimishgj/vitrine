//go:build !darwin

package grants

// ApplyACL is a no-op off macOS: the Linux backend uses mount namespaces.
func ApplyACL(Grant, string, string) error { return nil }

// RemoveACL is a no-op off macOS.
func RemoveACL(Grant, string, []Grant) error { return nil }
