package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// InstalledGame represents a detected game installation
type InstalledGame struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Platform string `json:"platform"` // steam, epic, gog, origin, uplay, other
	AppID    string `json:"appId,omitempty"`
}

// ScanResult holds all detected games
type ScanResult struct {
	Games []InstalledGame `json:"games"`
}

// ScanForGames scans the system for installed games
func ScanForGames() ScanResult {
	var games []InstalledGame

	// Scan Steam
	steamGames := scanSteam()
	games = append(games, steamGames...)

	// Scan Epic Games
	epicGames := scanEpic()
	games = append(games, epicGames...)

	// Scan GOG
	gogGames := scanGOG()
	games = append(games, gogGames...)

	// Scan Origin/EA
	originGames := scanOrigin()
	games = append(games, originGames...)

	// Scan Ubisoft Connect
	ubisoftGames := scanUbisoft()
	games = append(games, ubisoftGames...)

	return ScanResult{Games: games}
}

// normalizeForSearch removes special characters and spaces for fuzzy matching
func normalizeForSearch(s string) string {
	s = strings.ToLower(s)
	// Remove common words
	s = strings.ReplaceAll(s, "the ", "")
	// Remove special characters that might differ between web name and folder name
	replacements := []string{" ", "-", "_", ":", "'", "'", ".", ",", "!", "?", "(", ")", "[", "]", "&"}
	for _, r := range replacements {
		s = strings.ReplaceAll(s, r, "")
	}
	return s
}

// FindGameByName searches for a specific game by name (fuzzy match)
func FindGameByName(searchName string) []InstalledGame {
	allGames := ScanForGames()
	var matches []InstalledGame

	searchNormalized := normalizeForSearch(searchName)

	for _, game := range allGames.Games {
		gameNormalized := normalizeForSearch(game.Name)

		// Check if search term is contained in game name or vice versa
		if strings.Contains(gameNormalized, searchNormalized) || strings.Contains(searchNormalized, gameNormalized) {
			matches = append(matches, game)
		}
	}

	return matches
}

// ============================================================================
// STEAM SCANNER
// ============================================================================

func scanSteam() []InstalledGame {
	var games []InstalledGame

	// Find Steam installation paths
	steamPaths := getSteamLibraryPaths()

	for _, libraryPath := range steamPaths {
		commonPath := filepath.Join(libraryPath, "steamapps", "common")
		if _, err := os.Stat(commonPath); os.IsNotExist(err) {
			continue
		}

		// Read game folders
		entries, err := os.ReadDir(commonPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				gamePath := filepath.Join(commonPath, entry.Name())
				games = append(games, InstalledGame{
					Name:     entry.Name(),
					Path:     gamePath,
					Platform: "steam",
				})
			}
		}
	}

	// Try to get app IDs from manifest files
	games = enrichSteamGamesWithAppIDs(games, getSteamLibraryPaths())

	return games
}

func getSteamLibraryPaths() []string {
	var paths []string

	if runtime.GOOS == "windows" {
		// Get all available drives and check common Steam paths
		drives := getAvailableDrives()
		steamFolderNames := []string{"Steam", "SteamLibrary", "Games\\Steam", "Games\\SteamLibrary"}

		for _, drive := range drives {
			// Check Program Files paths on each drive
			programFilesX86 := filepath.Join(drive, "Program Files (x86)", "Steam")
			programFiles := filepath.Join(drive, "Program Files", "Steam")

			if _, err := os.Stat(programFilesX86); err == nil {
				paths = append(paths, programFilesX86)
			}
			if _, err := os.Stat(programFiles); err == nil {
				paths = append(paths, programFiles)
			}

			// Check root-level Steam folders
			for _, folderName := range steamFolderNames {
				steamPath := filepath.Join(drive, folderName)
				if _, err := os.Stat(steamPath); err == nil {
					paths = append(paths, steamPath)
				}
			}
		}

		// Parse libraryfolders.vdf for additional library paths
		for _, steamPath := range paths {
			additionalPaths := parseSteamLibraryFolders(steamPath)
			paths = append(paths, additionalPaths...)
		}

	} else {
		// Linux/macOS paths
		home, _ := os.UserHomeDir()
		linuxPaths := []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			"/home/deck/.steam/steam", // Steam Deck
		}

		for _, p := range linuxPaths {
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}

		// Parse libraryfolders.vdf
		for _, steamPath := range paths {
			additionalPaths := parseSteamLibraryFolders(steamPath)
			paths = append(paths, additionalPaths...)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique
}

func parseSteamLibraryFolders(steamPath string) []string {
	var paths []string

	vdfPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
	file, err := os.Open(vdfPath)
	if err != nil {
		return paths
	}
	defer file.Close()

	// Simple VDF parser - look for "path" keys
	scanner := bufio.NewScanner(file)
	pathRegex := regexp.MustCompile(`"path"\s+"([^"]+)"`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := pathRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			path := strings.ReplaceAll(matches[1], "\\\\", "\\")
			if _, err := os.Stat(path); err == nil {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

func enrichSteamGamesWithAppIDs(games []InstalledGame, steamPaths []string) []InstalledGame {
	// Build a map of install dirs to app IDs from manifest files
	installDirToAppID := make(map[string]string)

	for _, steamPath := range steamPaths {
		manifestsPath := filepath.Join(steamPath, "steamapps")
		entries, err := os.ReadDir(manifestsPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "appmanifest_") && strings.HasSuffix(entry.Name(), ".acf") {
				manifestPath := filepath.Join(manifestsPath, entry.Name())
				appID, installDir := parseAppManifest(manifestPath)
				if appID != "" && installDir != "" {
					installDirToAppID[strings.ToLower(installDir)] = appID
				}
			}
		}
	}

	// Enrich games with app IDs
	for i := range games {
		gameDirName := filepath.Base(games[i].Path)
		if appID, ok := installDirToAppID[strings.ToLower(gameDirName)]; ok {
			games[i].AppID = appID
		}
	}

	return games
}

func parseAppManifest(path string) (appID, installDir string) {
	file, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	appIDRegex := regexp.MustCompile(`"appid"\s+"(\d+)"`)
	installDirRegex := regexp.MustCompile(`"installdir"\s+"([^"]+)"`)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := appIDRegex.FindStringSubmatch(line); len(matches) > 1 {
			appID = matches[1]
		}
		if matches := installDirRegex.FindStringSubmatch(line); len(matches) > 1 {
			installDir = matches[1]
		}
	}

	return appID, installDir
}

// ============================================================================
// EPIC GAMES SCANNER
// ============================================================================

func scanEpic() []InstalledGame {
	var games []InstalledGame

	if runtime.GOOS != "windows" {
		return games // Epic primarily Windows
	}

	// Also check ProgramData for installed games (manifests)
	programData := os.Getenv("PROGRAMDATA")
	if programData != "" {
		epicManifests := filepath.Join(programData, "Epic", "EpicGamesLauncher", "Data", "Manifests")
		entries, err := os.ReadDir(epicManifests)
		if err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".item") {
					manifestPath := filepath.Join(epicManifests, entry.Name())
					game := parseEpicManifest(manifestPath)
					if game.Name != "" {
						games = append(games, game)
					}
				}
			}
		}
	}

	// Scan all drives for Epic Games folders
	drives := getAvailableDrives()
	epicFolderNames := []string{"Epic Games", "Games\\Epic Games", "Games\\Epic"}

	for _, drive := range drives {
		// Check Program Files
		programFilesPath := filepath.Join(drive, "Program Files", "Epic Games")
		if _, err := os.Stat(programFilesPath); err == nil {
			scanEpicFolder(programFilesPath, &games)
		}

		// Check root-level folders
		for _, folderName := range epicFolderNames {
			epicPath := filepath.Join(drive, folderName)
			if _, err := os.Stat(epicPath); err == nil {
				scanEpicFolder(epicPath, &games)
			}
		}
	}

	return games
}

func scanEpicFolder(basePath string, games *[]InstalledGame) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "Launcher" && entry.Name() != "DirectXRedist" {
			gamePath := filepath.Join(basePath, entry.Name())
			// Check if not already added
			found := false
			for _, g := range *games {
				if g.Path == gamePath {
					found = true
					break
				}
			}
			if !found {
				*games = append(*games, InstalledGame{
					Name:     entry.Name(),
					Path:     gamePath,
					Platform: "epic",
				})
			}
		}
	}
}

func parseEpicManifest(path string) InstalledGame {
	var game InstalledGame
	game.Platform = "epic"

	data, err := os.ReadFile(path)
	if err != nil {
		return game
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return game
	}

	if name, ok := manifest["DisplayName"].(string); ok {
		game.Name = name
	}
	if installLoc, ok := manifest["InstallLocation"].(string); ok {
		game.Path = installLoc
	}
	if appName, ok := manifest["AppName"].(string); ok {
		game.AppID = appName
	}

	return game
}

// ============================================================================
// GOG SCANNER
// ============================================================================

func scanGOG() []InstalledGame {
	var games []InstalledGame

	if runtime.GOOS != "windows" {
		// GOG on Linux uses different paths
		home, _ := os.UserHomeDir()
		gogPath := filepath.Join(home, "GOG Games")
		if _, err := os.Stat(gogPath); err == nil {
			entries, _ := os.ReadDir(gogPath)
			for _, entry := range entries {
				if entry.IsDir() {
					games = append(games, InstalledGame{
						Name:     entry.Name(),
						Path:     filepath.Join(gogPath, entry.Name()),
						Platform: "gog",
					})
				}
			}
		}
		return games
	}

	// Scan all drives for GOG folders
	drives := getAvailableDrives()
	gogFolderNames := []string{"GOG Games", "Games\\GOG", "Games\\GOG Galaxy"}

	for _, drive := range drives {
		// Check Program Files (x86) for GOG Galaxy
		gogGalaxyPath := filepath.Join(drive, "Program Files (x86)", "GOG Galaxy", "Games")
		if _, err := os.Stat(gogGalaxyPath); err == nil {
			scanGOGFolder(gogGalaxyPath, &games)
		}

		// Check root-level folders
		for _, folderName := range gogFolderNames {
			gogPath := filepath.Join(drive, folderName)
			if _, err := os.Stat(gogPath); err == nil {
				scanGOGFolder(gogPath, &games)
			}
		}
	}

	return games
}

func scanGOGFolder(basePath string, games *[]InstalledGame) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			gamePath := filepath.Join(basePath, entry.Name())
			// Check if not already added
			found := false
			for _, g := range *games {
				if g.Path == gamePath {
					found = true
					break
				}
			}
			if !found {
				*games = append(*games, InstalledGame{
					Name:     entry.Name(),
					Path:     gamePath,
					Platform: "gog",
				})
			}
		}
	}
}

// ============================================================================
// ORIGIN/EA SCANNER
// ============================================================================

func scanOrigin() []InstalledGame {
	var games []InstalledGame

	if runtime.GOOS != "windows" {
		return games
	}

	// Scan all drives for Origin/EA folders
	drives := getAvailableDrives()
	originFolderNames := []string{"Origin Games", "EA Games", "Games\\Origin", "Games\\EA"}

	for _, drive := range drives {
		// Check Program Files paths
		programFilesX86 := filepath.Join(drive, "Program Files (x86)", "Origin Games")
		programFiles := filepath.Join(drive, "Program Files", "EA Games")

		if _, err := os.Stat(programFilesX86); err == nil {
			scanOriginFolder(programFilesX86, &games)
		}
		if _, err := os.Stat(programFiles); err == nil {
			scanOriginFolder(programFiles, &games)
		}

		// Check root-level folders
		for _, folderName := range originFolderNames {
			originPath := filepath.Join(drive, folderName)
			if _, err := os.Stat(originPath); err == nil {
				scanOriginFolder(originPath, &games)
			}
		}
	}

	return games
}

func scanOriginFolder(basePath string, games *[]InstalledGame) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			gamePath := filepath.Join(basePath, entry.Name())
			// Check if not already added
			found := false
			for _, g := range *games {
				if g.Path == gamePath {
					found = true
					break
				}
			}
			if !found {
				*games = append(*games, InstalledGame{
					Name:     entry.Name(),
					Path:     gamePath,
					Platform: "origin",
				})
			}
		}
	}
}

// ============================================================================
// UBISOFT CONNECT SCANNER
// ============================================================================

func scanUbisoft() []InstalledGame {
	var games []InstalledGame

	if runtime.GOOS != "windows" {
		return games
	}

	// Scan all drives for Ubisoft folders
	drives := getAvailableDrives()
	ubisoftFolderNames := []string{
		"Ubisoft Game Launcher\\games",
		"Ubisoft\\Ubisoft Game Launcher\\games",
		"Games\\Ubisoft",
		"Ubisoft Games",
	}

	for _, drive := range drives {
		// Check Program Files paths
		programFilesX86 := filepath.Join(drive, "Program Files (x86)", "Ubisoft", "Ubisoft Game Launcher", "games")
		programFiles := filepath.Join(drive, "Program Files", "Ubisoft", "Ubisoft Game Launcher", "games")

		if _, err := os.Stat(programFilesX86); err == nil {
			scanUbisoftFolder(programFilesX86, &games)
		}
		if _, err := os.Stat(programFiles); err == nil {
			scanUbisoftFolder(programFiles, &games)
		}

		// Check root-level folders
		for _, folderName := range ubisoftFolderNames {
			ubisoftPath := filepath.Join(drive, folderName)
			if _, err := os.Stat(ubisoftPath); err == nil {
				scanUbisoftFolder(ubisoftPath, &games)
			}
		}
	}

	return games
}

func scanUbisoftFolder(basePath string, games *[]InstalledGame) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			gamePath := filepath.Join(basePath, entry.Name())
			// Check if not already added
			found := false
			for _, g := range *games {
				if g.Path == gamePath {
					found = true
					break
				}
			}
			if !found {
				*games = append(*games, InstalledGame{
					Name:     entry.Name(),
					Path:     gamePath,
					Platform: "ubisoft",
				})
			}
		}
	}
}
