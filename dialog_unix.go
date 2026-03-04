//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"strings"
)

// browseForFolder opens folder picker using zenity (GTK) or kdialog (KDE)
func browseForFolder(title string) (string, bool, error) {
	// Try zenity first (GNOME/GTK)
	if path, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command(path, "--file-selection", "--directory", "--title="+title)
		output, err := cmd.Output()
		if err != nil {
			// Exit code 1 means user cancelled
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return "", true, nil
			}
			return "", false, err
		}
		return strings.TrimSpace(string(output)), false, nil
	}

	// Try kdialog (KDE)
	if path, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.Command(path, "--getexistingdirectory", ".", "--title", title)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return "", true, nil
			}
			return "", false, err
		}
		return strings.TrimSpace(string(output)), false, nil
	}

	return "", false, errors.New("no dialog tool available (install zenity or kdialog)")
}

// browseForFile opens file picker using zenity or kdialog
func browseForFile(title, filter, startPath string) (string, bool, error) {
	// Try zenity first
	if path, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=" + title}
		if filter != "" {
			// Parse filter and add to zenity
			parts := strings.Split(filter, "|")
			if len(parts) >= 2 {
				args = append(args, "--file-filter="+parts[0]+"|"+parts[1])
			}
		}
		cmd := exec.Command(path, args...)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return "", true, nil
			}
			return "", false, err
		}
		return strings.TrimSpace(string(output)), false, nil
	}

	// Try kdialog
	if path, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--getopenfilename", "."}
		if filter != "" {
			parts := strings.Split(filter, "|")
			if len(parts) >= 2 {
				args = append(args, parts[1])
			}
		}
		args = append(args, "--title", title)
		cmd := exec.Command(path, args...)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				return "", true, nil
			}
			return "", false, err
		}
		return strings.TrimSpace(string(output)), false, nil
	}

	return "", false, errors.New("no dialog tool available (install zenity or kdialog)")
}
