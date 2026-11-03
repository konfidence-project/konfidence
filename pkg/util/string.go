// Package util holds small, dependency-free helpers shared across the codebase.
package util

// OptionalString returns a pointer to s, or nil when s is empty. It maps an
// optional string onto a *string that is omitted when unset.
func OptionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
