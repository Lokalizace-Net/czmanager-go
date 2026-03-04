//go:build windows

package main

import "fmt"

// execUnix is not used on Windows, always returns error to trigger fallback
func execUnix(path string, args []string, env []string) error {
	return fmt.Errorf("syscall.Exec not supported on Windows")
}
