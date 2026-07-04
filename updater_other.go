//go:build !windows

package main

import (
	"os"
	"syscall"
)

// restart nahradí běžící proces novou binárkou pomocí exec (Unix). Tím se
// zachová stejné PID a rovnou naběhne nová verze.
func (a *App) restart(exePath string) error {
	args := []string{exePath}
	return syscall.Exec(exePath, args, os.Environ())
}
