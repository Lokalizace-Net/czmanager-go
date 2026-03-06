package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	AgentDownloadBaseURL = "https://lokalizace.net/downloads/agent"
	AgentVersion         = "latest"
)

// App struct
type App struct {
	ctx          context.Context
	agentProcess *os.Process
	agentPath    string
}

// DetectedGame represents a detected game installation
type DetectedGame struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Platform string `json:"platform"`
	AppID    string `json:"appId,omitempty"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.findAgentPath()
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	a.StopAgent()
}

// findAgentPath locates the agent executable
func (a *App) findAgentPath() {
	// First check the standard install location
	installPath := a.getAgentInstallPath()
	if _, err := os.Stat(installPath); err == nil {
		a.agentPath = installPath
		fmt.Printf("Found agent at install location: %s\n", installPath)
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execDir := filepath.Dir(execPath)

	// Agent binary name based on OS
	agentName := "czmanager-agent"
	agentNameWithArch := "czmanager-agent"
	if goruntime.GOOS == "windows" {
		agentName = "czmanager-agent.exe"
		agentNameWithArch = "czmanager-agent-windows-amd64.exe"
	} else if goruntime.GOOS == "linux" {
		agentNameWithArch = "czmanager-agent-linux-amd64"
	} else if goruntime.GOOS == "darwin" {
		agentNameWithArch = "czmanager-agent-macos-amd64"
	}

	// Get working directory for development mode
	workDir, _ := os.Getwd()

	// Look for agent in various locations (development paths)
	possiblePaths := []string{
		filepath.Join(execDir, agentName),
		filepath.Join(execDir, agentNameWithArch),
		filepath.Join(workDir, "build", agentNameWithArch),
		filepath.Join(workDir, "..", "build", agentNameWithArch),
		filepath.Join(workDir, "..", "..", "build", agentNameWithArch),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			a.agentPath = path
			fmt.Printf("Found agent at: %s\n", path)
			return
		}
	}

	fmt.Println("Agent not found - will need to download")
}

// StartAgent starts the agent process
func (a *App) StartAgent() error {
	// First check if agent is already running
	if a.isAgentRunning() {
		fmt.Println("Agent is already running")
		return nil
	}

	// Try to find agent path again if not set
	if a.agentPath == "" {
		a.findAgentPath()
	}

	if a.agentPath == "" {
		fmt.Println("Agent executable not found")
		return fmt.Errorf("agent not found - please ensure czmanager-agent is in the build directory")
	}

	fmt.Printf("Starting agent from: %s\n", a.agentPath)

	cmd := exec.Command(a.agentPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start agent: %v\n", err)
		return fmt.Errorf("failed to start agent: %v", err)
	}

	a.agentProcess = cmd.Process
	fmt.Printf("Agent process started with PID: %d\n", cmd.Process.Pid)

	// Wait for agent to be ready
	for i := 0; i < 50; i++ {
		if a.isAgentRunning() {
			fmt.Println("Agent is now running and responding")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("agent did not start in time")
}

// StopAgent stops the agent process
func (a *App) StopAgent() {
	if a.agentProcess != nil {
		a.agentProcess.Kill()
		a.agentProcess = nil
	}
}

// GetAgentPath returns the expected agent path
func (a *App) GetAgentPath() string {
	return a.getAgentInstallPath()
}

// getAgentInstallPath returns the path where agent should be installed
func (a *App) getAgentInstallPath() string {
	// Get user's local app data directory
	var baseDir string
	if goruntime.GOOS == "windows" {
		baseDir = os.Getenv("LOCALAPPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	} else if goruntime.GOOS == "darwin" {
		baseDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	} else {
		baseDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}

	agentDir := filepath.Join(baseDir, "CZManager")

	// Agent binary name
	agentName := "czmanager-agent"
	if goruntime.GOOS == "windows" {
		agentName = "czmanager-agent.exe"
	}

	return filepath.Join(agentDir, agentName)
}

// getAgentDownloadURL returns the download URL for current platform
func (a *App) getAgentDownloadURL() string {
	var platform string
	switch goruntime.GOOS {
	case "windows":
		platform = "windows-amd64.exe"
	case "darwin":
		platform = "macos-amd64"
	case "linux":
		if goruntime.GOARCH == "arm64" {
			platform = "linux-arm64"
		} else {
			platform = "linux-amd64"
		}
	default:
		platform = "linux-amd64"
	}

	return fmt.Sprintf("%s/czmanager-agent-%s", AgentDownloadBaseURL, platform)
}

// IsAgentInstalled checks if agent is installed
func (a *App) IsAgentInstalled() bool {
	agentPath := a.getAgentInstallPath()
	_, err := os.Stat(agentPath)
	return err == nil
}

// DownloadAgent downloads the agent from the web
func (a *App) DownloadAgent() error {
	agentPath := a.getAgentInstallPath()
	agentDir := filepath.Dir(agentPath)

	// Create directory if not exists
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Download URL
	downloadURL := a.getAgentDownloadURL()
	fmt.Printf("Downloading agent from: %s\n", downloadURL)

	// Emit progress event
	wailsruntime.EventsEmit(a.ctx, "agent:download:progress", map[string]interface{}{
		"status":  "downloading",
		"percent": 0,
	})

	// Create HTTP request
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Get content length for progress
	contentLength := resp.ContentLength

	// Create temp file
	tmpFile, err := os.CreateTemp(agentDir, "agent-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()

	// Download with progress
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("failed to write: %v", writeErr)
			}
			downloaded += int64(n)

			if contentLength > 0 {
				percent := int(float64(downloaded) / float64(contentLength) * 100)
				wailsruntime.EventsEmit(a.ctx, "agent:download:progress", map[string]interface{}{
					"status":  "downloading",
					"percent": percent,
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("download error: %v", err)
		}
	}
	tmpFile.Close()

	// Remove old agent if exists
	os.Remove(agentPath)

	// Rename temp file to final path
	if err := os.Rename(tmpPath, agentPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to install agent: %v", err)
	}

	// Make executable on Unix
	if goruntime.GOOS != "windows" {
		os.Chmod(agentPath, 0755)
	}

	// Update agent path
	a.agentPath = agentPath

	wailsruntime.EventsEmit(a.ctx, "agent:download:progress", map[string]interface{}{
		"status":  "complete",
		"percent": 100,
	})

	fmt.Printf("Agent installed to: %s\n", agentPath)
	return nil
}

// DownloadAndStartAgent downloads agent if needed and starts it
func (a *App) DownloadAndStartAgent() error {
	// Check if already running
	if a.isAgentRunning() {
		return nil
	}

	// Check if installed
	if !a.IsAgentInstalled() {
		if err := a.DownloadAgent(); err != nil {
			return err
		}
	}

	// Update path and start
	a.agentPath = a.getAgentInstallPath()
	return a.StartAgent()
}

// isAgentRunning checks if the agent is responding
func (a *App) isAgentRunning() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://127.0.0.1:17892/ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetAgentStatus returns the agent status
func (a *App) GetAgentStatus() (map[string]interface{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:17892/ping")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// BrowseFolder opens a folder selection dialog
func (a *App) BrowseFolder(title string) (string, error) {
	if title == "" {
		title = "Vyberte složku"
	}

	path, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
	})

	if err != nil {
		return "", err
	}

	return path, nil
}

// BrowseFile opens a file selection dialog
func (a *App) BrowseFile(title string, filters string) (string, error) {
	if title == "" {
		title = "Vyberte soubor"
	}

	var dialogFilters []wailsruntime.FileFilter
	if filters != "" {
		dialogFilters = []wailsruntime.FileFilter{
			{
				DisplayName: "Soubory",
				Pattern:     filters,
			},
		}
	}

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   title,
		Filters: dialogFilters,
	})

	if err != nil {
		return "", err
	}

	return path, nil
}

// ScanGames scans for installed games matching the name
func (a *App) ScanGames(gameName string) ([]DetectedGame, error) {
	// For now, delegate to agent
	client := &http.Client{Timeout: 30 * time.Second}

	// Get token first
	pingResp, err := client.Get("http://127.0.0.1:17892/ping")
	if err != nil {
		return nil, err
	}
	defer pingResp.Body.Close()

	var pingData map[string]interface{}
	json.NewDecoder(pingResp.Body).Decode(&pingData)
	token, _ := pingData["token"].(string)

	// Create request
	reqBody := fmt.Sprintf(`{"game_name": "%s"}`, gameName)
	req, _ := http.NewRequest("POST", "http://127.0.0.1:17892/scan-games", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Games []DetectedGame `json:"games"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Games, nil
}

// LoginResult represents the login response
type LoginResult struct {
	AccessToken      string                 `json:"accessToken"`
	RefreshToken     string                 `json:"refreshToken"`
	ExpiresAt        string                 `json:"expiresAt"`
	RefreshExpiresAt string                 `json:"refreshExpiresAt"`
	User             map[string]interface{} `json:"user"`
}

// Login authenticates user with lokalizace.net API
func (a *App) Login(username string, password string) (*LoginResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	resp, err := client.Post("https://lokalizace.net/api/auth/login", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("neplatné přihlašovací údaje")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("účet je zablokován")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chyba serveru: %d", resp.StatusCode)
	}

	var result LoginResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	return &result, nil
}

// RefreshToken refreshes the access token
func (a *App) RefreshToken(refreshToken string) (*LoginResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"refreshToken": "%s"}`, refreshToken)
	resp, err := client.Post("https://lokalizace.net/api/auth/refresh", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("neplatný refresh token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	var result LoginResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	return &result, nil
}

// FetchSubscription fetches subscription info for authenticated user
func (a *App) FetchSubscription(accessToken string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, _ := http.NewRequest("GET", "https://lokalizace.net/api/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chyba serveru: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	return result, nil
}

// FetchGames fetches games from lokalizace.net API
func (a *App) FetchGames(page int, limit int, search string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf("https://lokalizace.net/api/games?page=%d&limit=%d", page, limit)
	if search != "" {
		url += "&search=" + search
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch games: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return result, nil
}

// FetchGameDetail fetches game detail including files from lokalizace.net API
func (a *App) FetchGameDetail(gameId int) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf("https://lokalizace.net/api/games/%d", gameId)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game detail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return result, nil
}

// DownloadLocalization downloads localization file - lets user choose where to save
func (a *App) DownloadLocalization(gameId int) (string, error) {
	// Nejprve získáme detail hry
	detail, err := a.FetchGameDetail(gameId)
	if err != nil {
		return "", err
	}

	files, ok := detail["files"].([]interface{})
	if !ok || len(files) == 0 {
		return "", fmt.Errorf("žádné soubory ke stažení")
	}

	// Vezmeme poslední (nejnovější) soubor
	lastFile := files[len(files)-1].(map[string]interface{})
	fileId := int(lastFile["id"].(float64))
	fileName := lastFile["fileName"].(string)

	// Výchozí složka Downloads
	var defaultDir string
	if goruntime.GOOS == "windows" {
		defaultDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	} else {
		defaultDir = filepath.Join(os.Getenv("HOME"), "Downloads")
	}

	// Dialog pro výběr kam uložit
	destPath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Uložit lokalizaci jako",
		DefaultDirectory: defaultDir,
		DefaultFilename: fileName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "ZIP soubory", Pattern: "*.zip"},
			{DisplayName: "Všechny soubory", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("chyba dialogu: %v", err)
	}
	if destPath == "" {
		return "", fmt.Errorf("stahování zrušeno")
	}

	// Download URL
	downloadURL := fmt.Sprintf("https://lokalizace.net/api/download/%d", fileId)

	// Emit progress
	wailsruntime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
		"status":  "downloading",
		"percent": 0,
		"file":    fileName,
	})

	// Stáhneme soubor
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("nepodařilo se stáhnout: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server vrátil status %d", resp.StatusCode)
	}

	// Vytvoříme soubor
	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("nepodařilo se vytvořit soubor: %v", err)
	}
	defer out.Close()

	// Stahujeme s progressem
	totalSize := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return "", writeErr
			}
			downloaded += int64(n)

			if totalSize > 0 {
				percent := int(downloaded * 100 / totalSize)
				wailsruntime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
					"status":  "downloading",
					"percent": percent,
					"file":    fileName,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	wailsruntime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
		"status":  "complete",
		"percent": 100,
		"file":    fileName,
	})

	return destPath, nil
}
