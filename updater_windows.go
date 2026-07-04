//go:build windows

package main

import (
	"os"
	"os/exec"
)

// restart spustí novou binárku jako samostatný proces a ukončí ten současný.
// Na Windows nejde nahradit běžící proces in-place, takže spustíme nový a
// tenhle necháme skončit.
func (a *App) restart(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	// Dej novému procesu chvíli na start, pak ukonči tenhle
	go func() {
		os.Exit(0)
	}()
	return nil
}
