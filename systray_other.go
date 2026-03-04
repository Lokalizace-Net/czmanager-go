//go:build !windows

package main

// initSystray is a no-op on non-Windows platforms
// Linux/macOS run as systemd service or LaunchAgent
func initSystray() {}

// removeSystray is a no-op on non-Windows platforms
func removeSystray() {}
