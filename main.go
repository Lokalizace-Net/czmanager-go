package main

import (
	"crypto/rand"
	"czmanager-agent/installer"
	"czmanager-agent/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const (
	Version  = "1.4.0"
	Port     = 17892
	TokenEnv = "CZMANAGER_TOKEN"
)

var (
	authToken        string
	installerService *installer.Service
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(Version)
		return
	}

	// Check for --console flag to keep console visible (for debugging)
	showConsole := false
	for _, arg := range os.Args[1:] {
		if arg == "--console" || arg == "-c" {
			showConsole = true
			break
		}
	}

	// Hide console window on Windows unless --console flag is passed
	if !showConsole {
		hideConsoleWindow()
	}

	// Start system tray icon (Windows only, no-op on other platforms)
	initSystray()

	// Clean up old random xdelta temp dirs from previous versions
	cleanupOldXdeltaDirs()

	// Extract embedded xdelta binary (uses fixed dir, reuses if exists)
	if err := extractXdelta(); err != nil {
		fmt.Printf("Warning: Failed to extract xdelta: %v\n", err)
		fmt.Println("Patching will use external xdelta3 binary if available")
	}

	// Generate or load auth token
	authToken = os.Getenv(TokenEnv)
	if authToken == "" {
		authToken = generateToken()
		fmt.Printf("Generated auth token: %s\n", authToken)
		fmt.Printf("Set environment variable %s=%s to persist\n", TokenEnv, authToken)
	}

	// Initialize installer service with xdelta path
	installerService = installer.NewService(getXdeltaPath())

	// Setup routes
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/status", withCORS(withAuth(handleStatus)))
	http.HandleFunc("/install", withCORS(withAuth(handleInstall)))
	http.HandleFunc("/uninstall", withCORS(withAuth(handleUninstall)))
	http.HandleFunc("/progress", withCORS(withAuth(handleProgress)))
	http.HandleFunc("/logs", withCORS(withAuth(handleLogs)))
	http.HandleFunc("/cancel", withCORS(withAuth(handleCancel)))
	http.HandleFunc("/browse-folder", withCORS(withAuth(handleBrowseFolder)))
	http.HandleFunc("/browse-file", withCORS(withAuth(handleBrowseFile)))
	http.HandleFunc("/update-check", withCORS(withAuth(handleUpdateCheck)))
	http.HandleFunc("/update", withCORS(withAuth(handleUpdate)))
	http.HandleFunc("/scan-games", withCORS(withAuth(handleScanGames)))

	// Start background update checker
	startUpdateChecker()

	addr := fmt.Sprintf("127.0.0.1:%d", Port)
	fmt.Printf("CZManager Agent v%s starting on %s\n", Version, addr)
	fmt.Printf("OS: %s, Arch: %s\n", runtime.GOOS, runtime.GOARCH)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func generateToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// Middleware for CORS
func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// Middleware for authentication
func withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			auth = r.URL.Query().Get("token")
		}

		// Remove "Bearer " prefix if present
		auth = strings.TrimPrefix(auth, "Bearer ")

		if auth != authToken {
			writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
			return
		}

		handler(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// GET /ping - public endpoint
func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, models.PingResponse{
		Alive:   true,
		Version: Version,
		Token:   authToken,
	})
}

// GET /status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, models.StatusResponse{
		Running: true,
		Version: Version,
		Busy:    installerService.IsBusy(),
	})
}

// POST /install
func handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.GameSlug == "" || req.DownloadURL == "" || req.GameRoot == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Missing required fields"})
		return
	}

	if err := installerService.Install(req); err != nil {
		writeJSON(w, http.StatusConflict, models.InstallResponse{Accepted: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.InstallResponse{Accepted: true})
}

// POST /uninstall
func handleUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.UninstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.GameRoot == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "game_root is required"})
		return
	}

	if err := installerService.Uninstall(req); err != nil {
		writeJSON(w, http.StatusConflict, models.InstallResponse{Accepted: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.InstallResponse{Accepted: true})
}

// GET /progress
func handleProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, installerService.GetProgress())
}

// GET /logs
func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	since := 0
	if s := r.URL.Query().Get("since"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			since = n
		}
	}

	writeJSON(w, http.StatusOK, models.LogsResponse{Logs: installerService.GetLogs(since)})
}

// POST /cancel
func handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	installerService.Cancel()
	writeJSON(w, http.StatusOK, models.SuccessResponse{Success: true, Message: "Cancellation requested"})
}

// POST /browse-folder
func handleBrowseFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.BrowseFolderRequest
	json.NewDecoder(r.Body).Decode(&req)

	title := "Select Folder"
	if req.Title != "" {
		title = req.Title
	}

	path, canceled, err := browseForFolder(title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	if canceled {
		writeJSON(w, http.StatusOK, models.BrowseResponse{Canceled: true})
		return
	}

	writeJSON(w, http.StatusOK, models.BrowseResponse{Path: path, Canceled: false})
}

// POST /browse-file
func handleBrowseFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.BrowseFileRequest
	json.NewDecoder(r.Body).Decode(&req)

	title := "Select File"
	if req.Title != "" {
		title = req.Title
	}

	path, canceled, err := browseForFile(title, req.Filter, req.StartPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	if canceled {
		writeJSON(w, http.StatusOK, models.BrowseResponse{Canceled: true})
		return
	}

	writeJSON(w, http.StatusOK, models.BrowseResponse{Path: path, Canceled: false})
}

// POST /scan-games
func handleScanGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req models.ScanGamesRequest
	json.NewDecoder(r.Body).Decode(&req)

	var games []models.DetectedGame

	if req.GameName != "" {
		// Search for specific game
		matches := FindGameByName(req.GameName)
		for _, g := range matches {
			games = append(games, models.DetectedGame{
				Name:     g.Name,
				Path:     g.Path,
				Platform: g.Platform,
				AppID:    g.AppID,
			})
		}
	} else {
		// Return all detected games
		result := ScanForGames()
		for _, g := range result.Games {
			games = append(games, models.DetectedGame{
				Name:     g.Name,
				Path:     g.Path,
				Platform: g.Platform,
				AppID:    g.AppID,
			})
		}
	}

	writeJSON(w, http.StatusOK, models.ScanGamesResponse{Games: games})
}
