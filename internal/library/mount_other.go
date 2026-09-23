//go:build !darwin && !linux

package library

// IsNetworkMount reports false where mount types cannot be inspected, which
// selects the shorter local refresh interval.
func IsNetworkMount(string) bool { return false }
