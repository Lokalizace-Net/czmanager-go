//go:build !windows

package main

import "syscall"

// execUnix replaces the current process with a new one (Unix only)
func execUnix(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}
