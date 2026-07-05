package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"czmanager/internal/installer"
	"czmanager/internal/models"
	"czmanager/internal/scanner"
	"czmanager/internal/xdelta"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ApiBaseURL = "https://lokalizace.net"

// GitHub repo pro kontrolu aktualizací (public, bez tokenu)
const (
	githubOwner = "Lokalizace-Net"
	githubRepo  = "czmanager-go"
)

// App struct
type App struct {
	ctx       context.Context
	logFile   *os.File
	logs      []string
	installer *installer.Service
	version   string
}

// GetVersion vrací verzi aplikace (vloženou při buildu z git tagu).
func (a *App) GetVersion() string {
	if a.version == "" {
		return "dev"
	}
	return a.version
}

// UpdateInfo popisuje výsledek kontroly aktualizace.
type UpdateInfo struct {
	Available      bool   `json:"available"`      // je k dispozici novější verze?
	CurrentVersion string `json:"currentVersion"` // aktuální verze aplikace
	LatestVersion  string `json:"latestVersion"`  // nejnovější verze na GitHubu
	ReleaseURL     string `json:"releaseUrl"`     // odkaz na release stránku
	ReleaseNotes   string `json:"releaseNotes"`   // popis vydání
}

// CheckUpdate zkontroluje nejnovější GitHub Release a porovná s aktuální verzí.
// Repo je public, takže se nepoužívá žádný token.
func (a *App) CheckUpdate() (*UpdateInfo, error) {
	current := a.GetVersion()

	info := &UpdateInfo{
		Available:      false,
		CurrentVersion: current,
	}

	// Dev buildy neaktualizujeme
	if current == "dev" || strings.HasPrefix(current, "dev-") {
		return info, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	client := &http.Client{Timeout: 15 * time.Second}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nepodařilo se spojit s GitHubem: %v", err)
	}
	defer resp.Body.Close()

	// Žádný release zatím nebyl vydán
	if resp.StatusCode == http.StatusNotFound {
		return info, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API vrátilo status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("nepodařilo se zpracovat odpověď GitHubu: %v", err)
	}

	info.LatestVersion = release.TagName
	info.ReleaseURL = release.HTMLURL
	info.ReleaseNotes = release.Body
	info.Available = isNewerVersion(current, release.TagName)

	if info.Available {
		a.log("Kontrola aktualizace: k dispozici nová verze %s (aktuální %s)", release.TagName, current)
	} else {
		a.log("Kontrola aktualizace: máte nejnovější verzi (%s)", current)
	}

	return info, nil
}

// isNewerVersion vrací true, pokud je latest novější než current.
// Verze mají tvar vMAJOR.MINOR.PATCH (např. v1.6.1). Porovnává se numericky
// po složkách; předpona "v" je volitelná.
func isNewerVersion(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	for i := 0; i < 3; i++ {
		if lat[i] > cur[i] {
			return true
		}
		if lat[i] < cur[i] {
			return false
		}
	}
	return false
}

// parseVersion rozloží "v1.6.1" na [1, 6, 1]. Chybějící/nečíselné složky = 0.
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n := 0
		fmt.Sscanf(parts[i], "%d", &n)
		out[i] = n
	}
	return out
}

// OpenReleasePage otevře stránku s vydáním v systémovém prohlížeči.
func (a *App) OpenReleasePage(url string) {
	if url == "" {
		url = fmt.Sprintf("https://github.com/%s/%s/releases/latest", githubOwner, githubRepo)
	}
	wailsruntime.BrowserOpenURL(a.ctx, url)
}

// getLogPath returns path to log file
func (a *App) getLogPath() string {
	var baseDir string
	if goruntime.GOOS == "windows" {
		baseDir = os.Getenv("LOCALAPPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	} else {
		baseDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	return filepath.Join(baseDir, "CZManager", "gui.log")
}

// initLogging initializes log file
func (a *App) initLogging() {
	logPath := a.getLogPath()
	os.MkdirAll(filepath.Dir(logPath), 0755)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}
	a.logFile = f
	a.log("=== CZ Agent GUI started ===")
}

// log writes to log file and stores in memory
func (a *App) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] %s", timestamp, msg)

	fmt.Println(line)

	a.logs = append(a.logs, line)
	if len(a.logs) > 500 {
		a.logs = a.logs[len(a.logs)-500:]
	}

	if a.logFile != nil {
		a.logFile.WriteString(line + "\n")
	}

	// Emit to frontend
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "log", line)
	}
}

// GetLogs returns recent logs
func (a *App) GetLogs() []string {
	return a.logs
}

// Log zapíše zprávu z frontendu do Debug Logu (stejný kanál jako backend logy).
// Frontend tak může logovat uživatelské akce (navigace, kliknutí, ...).
func (a *App) Log(message string) {
	a.log("%s", message)
}

// GetLogPath returns the log file path
func (a *App) GetLogPath() string {
	return a.getLogPath()
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
	a.initLogging()

	// Úklid po případném self-update: smaž starou binárku (.old)
	if exe, err := os.Executable(); err == nil {
		if exe, err := filepath.EvalSymlinks(exe); err == nil {
			os.Remove(exe + ".old")
			os.Remove(exe + ".new")
		}
	}

	// Prepare the embedded xdelta3 binary and the installer service. The
	// installer runs in-process now - there is no separate agent binary.
	xdelta.CleanupOldDirs()
	if err := xdelta.Extract(); err != nil {
		a.log("Warning: failed to extract xdelta: %v", err)
	}
	a.installer = installer.NewService(xdelta.Path())
	a.log("Installer service ready (in-process)")
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	if a.installer != nil {
		a.installer.Cancel()
	}
	if a.logFile != nil {
		a.logFile.Close()
	}
}

// --- Installer bindings (in-process, replaces the former HTTP agent) ---

// IsBusy reports whether an install/uninstall is currently running.
func (a *App) IsBusy() bool {
	return a.installer != nil && a.installer.IsBusy()
}

// IsInstalled reports whether a localization is installed in the given folder
// (detected via .ORIG/.IMPORT backup files left by the installer).
func (a *App) IsInstalled(gameRoot string) bool {
	return a.installer != nil && a.installer.IsInstalled(gameRoot)
}

// Install starts a localization install and streams progress/logs to the
// frontend via the "install:progress" and "install:log" events.
func (a *App) Install(gameSlug, version, downloadURL, gameRoot string) error {
	if a.installer == nil {
		return fmt.Errorf("installer not ready")
	}
	req := models.InstallRequest{
		GameSlug:    gameSlug,
		Version:     version,
		DownloadURL: downloadURL,
		GameRoot:    gameRoot,
	}
	if err := a.installer.Install(req); err != nil {
		return err
	}
	go a.streamProgress()
	return nil
}

// InstallLocal installs a localization from a local ZIP archive instead of
// downloading it. Used by the "Manual install" tab so creators can test their
// packages before uploading them. Progress/logs stream via the same events.
func (a *App) InstallLocal(gameRoot, zipPath string) error {
	if a.installer == nil {
		return fmt.Errorf("installer not ready")
	}
	if zipPath == "" {
		return fmt.Errorf("nebyl vybrán žádný archiv")
	}
	if gameRoot == "" {
		return fmt.Errorf("nebyla vybrána složka s hrou")
	}
	req := models.InstallRequest{
		GameSlug: "manual",
		Version:  "manual",
		GameRoot: gameRoot,
		LocalZip: zipPath,
	}
	if err := a.installer.Install(req); err != nil {
		return err
	}
	go a.streamProgress()
	return nil
}

// Uninstall removes a previously installed localization from gameRoot.
func (a *App) Uninstall(gameRoot string) error {
	if a.installer == nil {
		return fmt.Errorf("installer not ready")
	}
	req := models.UninstallRequest{GameRoot: gameRoot}
	if err := a.installer.Uninstall(req); err != nil {
		return err
	}
	go a.streamProgress()
	return nil
}

// CancelInstall cancels the current install/uninstall operation.
func (a *App) CancelInstall() {
	if a.installer != nil {
		a.installer.Cancel()
	}
}

// streamProgress polls the installer service and emits progress + new log
// lines to the frontend until the operation reaches a terminal stage. This
// replaces the old HTTP polling the frontend did against /progress and /logs.
func (a *App) streamProgress() {
	sentLogs := 0
	for {
		// Flush any new log lines first so the UI stays in sync.
		newLogs := a.installer.GetLogs(sentLogs)
		for _, entry := range newLogs {
			wailsruntime.EventsEmit(a.ctx, "install:log", entry.Message)
		}
		sentLogs += len(newLogs)

		progress := a.installer.GetProgress()
		wailsruntime.EventsEmit(a.ctx, "install:progress", progress)

		if progress.Stage == models.StageDone || progress.Stage == models.StageError {
			return
		}
		if !a.installer.IsBusy() {
			// Safety net: operation ended without a terminal stage.
			wailsruntime.EventsEmit(a.ctx, "install:progress", a.installer.GetProgress())
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// ScanGames scans for installed games matching the name (in-process).
func (a *App) ScanGames(gameName string) ([]DetectedGame, error) {
	if gameName != "" {
		a.log("Hledám nainstalovanou hru: %s", gameName)
	} else {
		a.log("Skenuji nainstalované hry...")
	}

	var found []scanner.InstalledGame
	if gameName != "" {
		found = scanner.FindGameByName(gameName)
	} else {
		found = scanner.ScanForGames().Games
	}

	games := make([]DetectedGame, 0, len(found))
	for _, g := range found {
		games = append(games, DetectedGame{
			Name:     g.Name,
			Path:     g.Path,
			Platform: g.Platform,
			AppID:    g.AppID,
		})
	}
	a.log("Sken dokončen: nalezeno %d her", len(games))
	return games, nil
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
	a.log("Přihlašování uživatele: %s", username)
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, password)
	resp, err := client.Post(ApiBaseURL+"/api/auth/login", "application/json", strings.NewReader(reqBody))
	if err != nil {
		a.log("Přihlášení selhalo: chyba připojení k serveru")
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	if resp.StatusCode == 401 {
		a.log("Přihlášení selhalo: neplatné přihlašovací údaje")
		return nil, fmt.Errorf("neplatné přihlašovací údaje")
	}
	if resp.StatusCode == 403 {
		a.log("Přihlášení selhalo: účet je zablokován")
		return nil, fmt.Errorf("účet je zablokován")
	}
	if resp.StatusCode != http.StatusOK {
		a.log("Přihlášení selhalo: server vrátil status %d", resp.StatusCode)
		return nil, fmt.Errorf("chyba serveru: %d", resp.StatusCode)
	}

	var result LoginResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	a.log("Uživatel %s úspěšně přihlášen", username)
	return &result, nil
}

// RefreshToken refreshes the access token
func (a *App) RefreshToken(refreshToken string) (*LoginResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"refreshToken": "%s"}`, refreshToken)
	resp, err := client.Post(ApiBaseURL+"/api/auth/refresh", "application/json", strings.NewReader(reqBody))
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

	req, _ := http.NewRequest("GET", ApiBaseURL+"/api/subscription", nil)
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

	apiUrl := fmt.Sprintf(ApiBaseURL+"/api/games?page=%d&limit=%d", page, limit)
	if search != "" {
		// URL encode search string (handles spaces and special characters)
		encoded := strings.ReplaceAll(search, " ", "+")
		apiUrl += "&search=" + encoded
	}

	if search != "" {
		a.log("Načítám lokalizace (stránka %d, hledání: %q)", page, search)
	} else {
		a.log("Načítám lokalizace (stránka %d)", page)
	}

	resp, err := client.Get(apiUrl)
	if err != nil {
		a.log("Načtení lokalizací selhalo: %v", err)
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

	if games, ok := result["games"].([]interface{}); ok {
		a.log("Načteno %d lokalizací", len(games))
	}

	return result, nil
}

// FetchGameDetail fetches game detail including files from lokalizace.net API
func (a *App) FetchGameDetail(gameId int) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf(ApiBaseURL+"/api/games/%d", gameId)

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
	downloadURL := fmt.Sprintf(ApiBaseURL+"/api/download/%d", fileId)

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

// FetchFavorites fetches user's favorite games from lokalizace.net API
func (a *App) FetchFavorites(accessToken string) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, _ := http.NewRequest("GET", ApiBaseURL+"/api/favorites", nil)
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

// AddFavorite adds a game to favorites via POST /api/favorites
func (a *App) AddFavorite(accessToken string, gameId int) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"gameId": %d}`, gameId)
	req, _ := http.NewRequest("POST", ApiBaseURL+"/api/favorites", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("%s", extractErrorMessage(body, "Dosáhli jste limitu oblíbených her"))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chyba serveru: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	return result, nil
}

// extractErrorMessage vytáhne čitelnou chybovou hlášku z odpovědi API.
// Podporuje JSON {"error":"..."} / {"message":"..."}. Pokud odpověď není
// JSON (např. HTML error stránka), vrátí zadaný fallback místo surového těla.
func extractErrorMessage(body []byte, fallback string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		for _, key := range []string{"error", "message", "detail"} {
			if v, ok := parsed[key].(string); ok && v != "" {
				return v
			}
		}
	}
	return fallback
}

// RemoveFavorite removes a game from favorites via DELETE /api/favorites
func (a *App) RemoveFavorite(accessToken string, gameId int) (map[string]interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := fmt.Sprintf(`{"gameId": %d}`, gameId)
	req, _ := http.NewRequest("DELETE", ApiBaseURL+"/api/favorites", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chyba připojení k serveru")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chyba čtení odpovědi")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chyba serveru: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("chyba parsování odpovědi")
	}

	return result, nil
}

// GetImageBase64 fetches image from URL and returns as base64 data URL
func (a *App) GetImageBase64(imageUrl string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(imageUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image fetch failed: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Convert to base64 data URL
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}
