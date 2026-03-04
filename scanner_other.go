//go:build !windows

package main

// getAvailableDrives returns available mount points on non-Windows systems
func getAvailableDrives() []string {
	// On Linux/macOS, we don't use drive letters
	// The scanner functions already handle non-Windows paths
	return []string{"/"}
}
