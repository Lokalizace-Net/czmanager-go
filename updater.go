package main

import (
	"czmanager-agent/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	APIBase        = "https://lokalizace.net"
	UpdateCheckURL = APIBase + "/api/agent"
	UpdateInterval = 1 * time.Hour
)

var (
	updateMu          sync.Mutex
	updateAvailable   bool
	latestVersion     string
	latestDownloadURL string
	lastUpdateCheck   time.Time
	isUpdating        bool
)

// AgentVersionResponse from the API
type AgentVersionResponse struct {
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	DownloadURL string `json:"downloadUrl"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	Changelog   string `json:"changelog"`
}

// isAllowedDownloadURL reports whether url is an HTTPS URL whose host is
// lokalizace.net (or a subdomain of it). The self-updater downloads and
// executes this binary, so the download source must be tightly pinned.
func isAllowedDownloadURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "lokalizace.net" || strings.HasSuffix(host, ".lokalizace.net")
}

// UpdateCheckResponse for the /update-check endpoint
type UpdateCheckResponse struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl,omitempty"`
	Changelog       string `json:"changelog,omitempty"`
	LastCheck       string `json:"lastCheck"`
}

// UpdateResponse for the /update endpoint
type UpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

// Start background update checker
func startUpdateChecker() {
	go checkForUpdate()

	ticker := time.NewTicker(UpdateInterval)
	go func() {
		for range ticker.C {
			checkForUpdate()
		}
	}()
}

// checkForUpdate checks the API for a newer version
func checkForUpdate() {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}

	url := fmt.Sprintf("%s?platform=%s", UpdateCheckURL, platform)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Update check failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var versionInfo AgentVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		fmt.Printf("Failed to parse update response: %v\n", err)
		return
	}

	// Make URL absolute if relative
	downloadURL := versionInfo.DownloadURL
	if downloadURL != "" && !strings.HasPrefix(downloadURL, "http") {
		downloadURL = APIBase + downloadURL
	}

	// Only trust download URLs that point at our own domain over HTTPS. The
	// agent installs the downloaded binary and runs it, so an attacker-supplied
	// URL here would mean arbitrary code execution. Reject anything else.
	if !isAllowedDownloadURL(downloadURL) {
		fmt.Printf("Rejecting update: download URL %q is not on an allowed host\n", downloadURL)
		updateMu.Lock()
		updateAvailable = false
		latestDownloadURL = ""
		lastUpdateCheck = time.Now()
		updateMu.Unlock()
		return
	}

	updateMu.Lock()
	lastUpdateCheck = time.Now()
	latestVersion = versionInfo.Version
	latestDownloadURL = downloadURL

	if compareVersions(versionInfo.Version, Version) > 0 {
		updateAvailable = true
		fmt.Printf("Update available: %s -> %s\n", Version, versionInfo.Version)
	} else {
		updateAvailable = false
	}
	updateMu.Unlock()
}

// compareVersions compares two semantic versions
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		var n1, n2 int
		fmt.Sscanf(parts1[i], "%d", &n1)
		fmt.Sscanf(parts2[i], "%d", &n2)

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	if len(parts1) < len(parts2) {
		return -1
	}
	if len(parts1) > len(parts2) {
		return 1
	}
	return 0
}

// performUpdate downloads and installs the new version
func performUpdate() error {
	updateMu.Lock()
	if isUpdating {
		updateMu.Unlock()
		return fmt.Errorf("update already in progress")
	}
	if !updateAvailable || latestDownloadURL == "" {
		updateMu.Unlock()
		return fmt.Errorf("no update available")
	}
	isUpdating = true
	downloadURL := latestDownloadURL
	newVersion := latestVersion
	updateMu.Unlock()

	defer func() {
		updateMu.Lock()
		isUpdating = false
		updateMu.Unlock()
	}()

	// Defense in depth: re-validate the host right before downloading, in case
	// latestDownloadURL was somehow set without going through checkForUpdate.
	if !isAllowedDownloadURL(downloadURL) {
		return fmt.Errorf("refusing to download update from untrusted URL: %s", downloadURL)
	}

	fmt.Printf("Downloading update from %s...\n", downloadURL)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Download to temp file next to executable
	tmpFile := execPath + ".new"

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write update: %v", err)
	}

	// Verify downloaded file is not empty
	info, err := os.Stat(tmpFile)
	if err != nil || info.Size() == 0 {
		os.Remove(tmpFile)
		return fmt.Errorf("downloaded file is empty or invalid")
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpFile, 0755); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to set permissions: %v", err)
		}
	}

	// Replace binary
	// On Windows: running exe CAN be renamed but NOT overwritten
	// On Unix: running binary can be replaced (inode stays until process exits)
	backupPath := execPath + ".old"
	os.Remove(backupPath)

	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename current binary: %v", err)
	}

	if err := os.Rename(tmpFile, execPath); err != nil {
		// Rollback
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %v", err)
	}

	// .old stays around for rollback safety, cleaned up on next update

	fmt.Printf("Update installed successfully: %s -> %s\n", Version, newVersion)
	fmt.Println("Restarting agent...")

	// Clean up systray before restart
	removeSystray()

	go func() {
		time.Sleep(500 * time.Millisecond)
		restartAgent(execPath)
	}()

	return nil
}

// restartAgent restarts the agent process
func restartAgent(execPath string) {
	if runtime.GOOS == "windows" {
		// Windows: start new process and exit current one
		cmd := exec.Command(execPath)
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to restart: %v\n", err)
			return
		}
		os.Exit(0)
	} else {
		// Unix: replace current process with syscall.Exec
		env := os.Environ()
		err := execUnix(execPath, os.Args, env)
		if err != nil {
			// Fallback: start as child process and exit
			fmt.Printf("syscall.Exec failed: %v, using fallback\n", err)
			cmd := exec.Command(execPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = env
			if err := cmd.Start(); err != nil {
				fmt.Printf("Failed to restart: %v\n", err)
				return
			}
			os.Exit(0)
		}
	}
}

// GET /update-check
func handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	if r.URL.Query().Get("force") == "true" {
		checkForUpdate()
	}

	updateMu.Lock()
	response := UpdateCheckResponse{
		CurrentVersion:  Version,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		LastCheck:       lastUpdateCheck.Format(time.RFC3339),
	}
	if updateAvailable {
		response.DownloadURL = latestDownloadURL
	}
	updateMu.Unlock()

	writeJSON(w, http.StatusOK, response)
}

// POST /update
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	updateMu.Lock()
	available := updateAvailable
	updating := isUpdating
	version := latestVersion
	updateMu.Unlock()

	if !available {
		writeJSON(w, http.StatusOK, UpdateResponse{
			Success: false,
			Message: "No update available",
			Version: Version,
		})
		return
	}

	if updating {
		writeJSON(w, http.StatusConflict, UpdateResponse{
			Success: false,
			Message: "Update already in progress",
		})
		return
	}

	go func() {
		if err := performUpdate(); err != nil {
			fmt.Printf("Update failed: %v\n", err)
		}
	}()

	writeJSON(w, http.StatusOK, UpdateResponse{
		Success: true,
		Message: fmt.Sprintf("Updating to version %s...", version),
		Version: version,
	})
}
