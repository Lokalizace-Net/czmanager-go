package xdelta

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed resources/xdelta3.exe resources/xdelta3_linux resources/xdelta3_mac
var xdeltaFiles embed.FS

var extractedXdeltaPath string

// Extract extracts the appropriate xdelta binary for the current OS.
// Uses a fixed directory to avoid leaving temp files behind on os.Exit.
func Extract() error {
	var filename string
	switch runtime.GOOS {
	case "windows":
		filename = "xdelta3.exe"
	case "linux":
		filename = "xdelta3_linux"
	case "darwin":
		filename = "xdelta3_mac"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Use fixed directory instead of random temp dir
	tempDir := filepath.Join(os.TempDir(), "czmanager-agent")
	os.MkdirAll(tempDir, 0755)

	extractedXdeltaPath = filepath.Join(tempDir, filename)

	// Skip extraction if already exists with correct size
	data, err := xdeltaFiles.ReadFile("resources/" + filename)
	if err != nil {
		return fmt.Errorf("failed to read embedded xdelta: %v", err)
	}

	if info, err := os.Stat(extractedXdeltaPath); err == nil && info.Size() == int64(len(data)) {
		return nil
	}

	if err := os.WriteFile(extractedXdeltaPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write xdelta: %v", err)
	}

	return nil
}

// Path returns path to xdelta binary (extracted or local).
func Path() string {
	// First check if we have extracted version
	if extractedXdeltaPath != "" {
		if _, err := os.Stat(extractedXdeltaPath); err == nil {
			return extractedXdeltaPath
		}
	}

	// Fallback to looking next to executable
	var filename string
	switch runtime.GOOS {
	case "windows":
		filename = "xdelta3.exe"
	case "linux":
		filename = "xdelta3_linux"
	case "darwin":
		filename = "xdelta3_mac"
	default:
		filename = "xdelta3"
	}

	execPath, err := os.Executable()
	if err == nil {
		localPath := filepath.Join(filepath.Dir(execPath), filename)
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}

	// Last resort - hope it's in PATH
	return filename
}

// CleanupOldDirs removes leftover random temp directories from previous versions.
func CleanupOldDirs() {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "czmanager-xdelta-*"))
	for _, dir := range matches {
		os.RemoveAll(dir)
	}
}
