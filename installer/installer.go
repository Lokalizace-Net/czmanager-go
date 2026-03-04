package installer

import (
	"archive/zip"
	"czmanager-agent/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service handles installation and uninstallation
type Service struct {
	mu         sync.Mutex
	isBusy     bool
	cancelChan chan struct{}
	progress   models.ProgressResponse
	logs       []models.LogEntry
	logsMu     sync.RWMutex
	currentReq *models.InstallRequest
	xdeltaPath string
}

// NewService creates a new installer service
func NewService(xdeltaPath string) *Service {
	return &Service{
		progress: models.ProgressResponse{
			Stage:   models.StageIdle,
			Percent: 0,
			Message: "",
		},
		logs:       make([]models.LogEntry, 0),
		xdeltaPath: xdeltaPath,
	}
}

// IsBusy returns whether an operation is in progress
func (s *Service) IsBusy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isBusy
}

// GetProgress returns the current progress
func (s *Service) GetProgress() models.ProgressResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// GetLogs returns all logs since given index
func (s *Service) GetLogs(since int) []models.LogEntry {
	s.logsMu.RLock()
	defer s.logsMu.RUnlock()
	if since >= len(s.logs) {
		return []models.LogEntry{}
	}
	return s.logs[since:]
}

// ClearLogs clears all logs
func (s *Service) ClearLogs() {
	s.logsMu.Lock()
	defer s.logsMu.Unlock()
	s.logs = make([]models.LogEntry, 0)
}

// Cancel cancels the current operation
func (s *Service) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelChan != nil {
		close(s.cancelChan)
		s.cancelChan = nil
	}
}

func (s *Service) log(level, message string) {
	// Print to console
	fmt.Printf("[%s] %s\n", level, message)

	s.logsMu.Lock()
	defer s.logsMu.Unlock()
	s.logs = append(s.logs, models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
}

func (s *Service) logInfo(message string) {
	s.log("INFO", message)
}

func (s *Service) logError(message string) {
	s.log("ERROR", message)
}

func (s *Service) logWarning(message string) {
	s.log("WARNING", message)
}

func (s *Service) setProgress(stage models.InstallStage, percent int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = models.ProgressResponse{
		Stage:   stage,
		Percent: percent,
		Message: message,
	}
	if s.currentReq != nil {
		s.progress.GameSlug = s.currentReq.GameSlug
		s.progress.Version = s.currentReq.Version
	}
}

func (s *Service) setError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress.Stage = models.StageError
	s.progress.Error = err
}

func (s *Service) isCancelled() bool {
	select {
	case <-s.cancelChan:
		return true
	default:
		return false
	}
}

// Install starts the installation process
func (s *Service) Install(req models.InstallRequest) error {
	s.mu.Lock()
	if s.isBusy {
		s.mu.Unlock()
		return fmt.Errorf("another operation is in progress")
	}
	s.isBusy = true
	s.cancelChan = make(chan struct{})
	s.currentReq = &req
	s.mu.Unlock()

	s.ClearLogs()

	go s.doInstall(req)
	return nil
}

func (s *Service) doInstall(req models.InstallRequest) {
	defer func() {
		s.mu.Lock()
		s.isBusy = false
		s.currentReq = nil
		s.mu.Unlock()
	}()

	s.logInfo(fmt.Sprintf("Zahajuji instalaci %s v%s", req.GameSlug, req.Version))
	s.logInfo(fmt.Sprintf("Cílová složka: %s", req.GameRoot))

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "czmanager-*")
	if err != nil {
		s.setError(fmt.Sprintf("Nepodařilo se vytvořit dočasnou složku: %v", err))
		s.logError(err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "localization.zip")
	extractPath := filepath.Join(tempDir, "extracted")

	// Download
	s.setProgress(models.StageDownloading, 0, "Stahování lokalizace...")
	s.logInfo(fmt.Sprintf("Stahování z: %s", req.DownloadURL))

	if err := s.downloadFile(req.DownloadURL, zipPath); err != nil {
		s.setError(fmt.Sprintf("Stahování selhalo: %v", err))
		s.logError(err.Error())
		return
	}

	if s.isCancelled() {
		s.setError("Operace byla zrušena")
		return
	}

	// Extract
	s.setProgress(models.StageExtracting, 30, "Rozbalování archivu...")
	s.logInfo("Rozbaluji archiv...")

	if err := s.extractZip(zipPath, extractPath); err != nil {
		s.setError(fmt.Sprintf("Rozbalení selhalo: %v", err))
		s.logError(err.Error())
		return
	}

	if s.isCancelled() {
		s.setError("Operace byla zrušena")
		return
	}

	// Load instructions
	instructionsPath := filepath.Join(extractPath, "INSTALL_INSTRUCTIONS.json")
	instructions, err := s.loadInstructions(instructionsPath)
	if err != nil {
		s.setError(fmt.Sprintf("Nepodařilo se načíst instrukce: %v", err))
		s.logError(err.Error())
		return
	}

	s.logInfo(fmt.Sprintf("Načteno %d pre-tasks, %d modd_files, %d post-tasks",
		len(instructions.PreTasks), len(instructions.ModdFiles), len(instructions.PostTasks)))

	// Sort tasks and files by priority
	sort.Slice(instructions.PreTasks, func(i, j int) bool {
		return instructions.PreTasks[i].Priority < instructions.PreTasks[j].Priority
	})
	sort.Slice(instructions.ModdFiles, func(i, j int) bool {
		return instructions.ModdFiles[i].Priority < instructions.ModdFiles[j].Priority
	})
	sort.Slice(instructions.PostTasks, func(i, j int) bool {
		return instructions.PostTasks[i].Priority < instructions.PostTasks[j].Priority
	})

	// Pre-tasks
	if len(instructions.PreTasks) > 0 {
		s.setProgress(models.StagePreTasks, 35, "Provádím přípravné úkoly...")
		s.logInfo(fmt.Sprintf("Provádím %d přípravných úkolů", len(instructions.PreTasks)))

		if err := s.executeTasks(instructions.PreTasks, extractPath, req.GameRoot); err != nil {
			s.setError(fmt.Sprintf("Přípravný úkol selhal: %v", err))
			s.logError(err.Error())
			return
		}
	}

	if s.isCancelled() {
		s.setError("Operace byla zrušena")
		return
	}

	// Install modd_files
	s.setProgress(models.StageInstalling, 45, "Instaluji soubory...")

	for i, moddFile := range instructions.ModdFiles {
		if s.isCancelled() {
			s.setError("Operace byla zrušena")
			return
		}

		progress := 45 + (i * 35 / max(len(instructions.ModdFiles), 1))
		s.setProgress(models.StageInstalling, progress, fmt.Sprintf("Instaluji %d/%d: %s", i+1, len(instructions.ModdFiles), moddFile.Name))

		if err := s.installFile(moddFile, extractPath, req.GameRoot); err != nil {
			if moddFile.Optional {
				s.logWarning(fmt.Sprintf("Volitelný soubor přeskočen: %s - %v", moddFile.Name, err))
				continue
			}
			s.setError(fmt.Sprintf("Instalace souboru selhala: %s - %v", moddFile.Name, err))
			s.logError(err.Error())
			return
		}
	}

	// Post-tasks
	if len(instructions.PostTasks) > 0 {
		s.setProgress(models.StagePostTasks, 85, "Provádím dokončovací úkoly...")
		s.logInfo(fmt.Sprintf("Provádím %d dokončovacích úkolů", len(instructions.PostTasks)))

		if err := s.executeTasks(instructions.PostTasks, extractPath, req.GameRoot); err != nil {
			s.setError(fmt.Sprintf("Dokončovací úkol selhal: %v", err))
			s.logError(err.Error())
			return
		}
	}

	// Cleanup - remove INSTALL_INSTRUCTIONS.json from game root if exists
	gameInstructions := filepath.Join(req.GameRoot, "INSTALL_INSTRUCTIONS.json")
	os.Remove(gameInstructions)

	s.setProgress(models.StageDone, 100, "Instalace dokončena!")
	s.logInfo("Instalace úspěšně dokončena!")
}

func (s *Service) installFile(moddFile models.ModdFile, extractPath, gameRoot string) error {
	// Normalize path separators for cross-platform compatibility
	normalizedName := strings.ReplaceAll(moddFile.Name, "\\", "/")
	sourcePath := filepath.Join(extractPath, normalizedName)
	targetPath := filepath.Join(gameRoot, normalizedName)
	origPath := targetPath + ".ORIG"
	importPath := targetPath + ".IMPORT"

	// On Linux, filesystem is case-sensitive but ZIP/instructions may have different case
	// Try to find actual file with case-insensitive match
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			if actualPath := s.findFileCaseInsensitive(extractPath, normalizedName); actualPath != "" {
				s.logInfo(fmt.Sprintf("Case mismatch opraven: %s -> %s", sourcePath, actualPath))
				sourcePath = actualPath
			}
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("cannot create directory: %v", err)
	}

	installType := strings.ToLower(moddFile.InstallType)

	switch installType {
	case "patch":
		// Apply xdelta3 patch
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if moddFile.Optional {
				return fmt.Errorf("file not found for patching")
			}
			return fmt.Errorf("original file not found: %s", targetPath)
		}

		// Find patch file - try different extensions
		patchPath := sourcePath
		patchExtensions := []string{".xdelta", ".delta", ".patch", ".vcdiff", ""}
		found := false
		for _, ext := range patchExtensions {
			testPath := sourcePath + ext
			if _, err := os.Stat(testPath); err == nil {
				patchPath = testPath
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("patch file not found: %s (tried .xdelta, .patch, .vcdiff)", moddFile.Name)
		}

		// Backup original if not already backed up
		if _, err := os.Stat(origPath); os.IsNotExist(err) {
			s.logInfo(fmt.Sprintf("Zálohuji: %s", moddFile.Name))
			if err := copyFile(targetPath, origPath); err != nil {
				return fmt.Errorf("backup failed: %v", err)
			}
		}

		s.logInfo(fmt.Sprintf("Aplikuji patch: %s (z %s)", moddFile.Name, filepath.Base(patchPath)))
		if err := s.applyPatch(targetPath, patchPath, targetPath); err != nil {
			return fmt.Errorf("patch failed: %v", err)
		}

	case "insert", "patch_insert":
		// Copy new file, backup if exists
		if _, err := os.Stat(targetPath); err == nil {
			// File exists - backup it
			if _, err := os.Stat(origPath); os.IsNotExist(err) {
				s.logInfo(fmt.Sprintf("Zálohuji existující: %s", moddFile.Name))
				if err := copyFile(targetPath, origPath); err != nil {
					s.logWarning(fmt.Sprintf("Záloha selhala: %v", err))
				}
			}
		} else {
			// File doesn't exist - mark as imported
			if _, err := os.Create(importPath); err == nil {
				s.logInfo(fmt.Sprintf("Nový soubor označen: %s", moddFile.Name))
			}
		}

		s.logInfo(fmt.Sprintf("Kopíruji: %s", moddFile.Name))
		if err := copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("copy failed: %v", err)
		}

	case "replace":
		// Simple replacement, backup original
		if _, err := os.Stat(targetPath); err == nil {
			if _, err := os.Stat(origPath); os.IsNotExist(err) {
				s.logInfo(fmt.Sprintf("Zálohuji: %s", moddFile.Name))
				if err := copyFile(targetPath, origPath); err != nil {
					s.logWarning(fmt.Sprintf("Záloha selhala: %v", err))
				}
			}
		}

		s.logInfo(fmt.Sprintf("Nahrazuji: %s", moddFile.Name))
		if err := copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("replace failed: %v", err)
		}

	default:
		s.logWarning(fmt.Sprintf("Neznámý typ instalace '%s' pro %s, použiji replace", installType, moddFile.Name))
		if err := copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("copy failed: %v", err)
		}
	}

	return nil
}

func (s *Service) downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server vrátil status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			if totalSize > 0 {
				percent := int(downloaded * 30 / totalSize)
				s.setProgress(models.StageDownloading, percent, fmt.Sprintf("Staženo %d%%", percent*100/30))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) extractZip(zipPath, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		// Normalize path separators - ZIP files may contain Windows-style backslashes
		// which don't work correctly on Linux/Mac
		normalizedName := strings.ReplaceAll(f.Name, "\\", "/")
		fpath := filepath.Join(destPath, normalizedName)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destPath)+string(os.PathSeparator)) {
			return fmt.Errorf("neplatná cesta v archivu: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) loadInstructions(path string) (*models.InstallInstructions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM if present (0xEF 0xBB 0xBF)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	var instructions models.InstallInstructions
	if err := json.Unmarshal(data, &instructions); err != nil {
		return nil, err
	}

	return &instructions, nil
}

func (s *Service) executeTasks(tasks []models.InstallTask, extractPath, gameRoot string) error {
	for i, task := range tasks {
		comment := task.Comment
		if comment == "" {
			comment = task.Command
		}
		s.logInfo(fmt.Sprintf("Úkol %d/%d: %s", i+1, len(tasks), comment))

		cmd := strings.ToLower(task.Command)

		switch cmd {
		case "run_file":
			if err := s.executeRunFile(task, extractPath, gameRoot); err != nil {
				return err
			}

		case "delete_file":
			path := s.resolvePath(task.Source, extractPath, gameRoot)
			if path == "" {
				path = s.resolvePath(task.Target, extractPath, gameRoot)
			}
			s.logInfo(fmt.Sprintf("Mažu soubor: %s", path))
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				s.logWarning(fmt.Sprintf("Nelze smazat: %v", err))
			}

		case "move_file":
			src := s.resolvePath(task.Source, extractPath, gameRoot)
			dst := s.resolvePath(task.Target, extractPath, gameRoot)
			s.logInfo(fmt.Sprintf("Přesouvám: %s -> %s", src, dst))
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := os.Rename(src, dst); err != nil {
				// Try copy+delete if rename fails (cross-device)
				if err := copyFile(src, dst); err != nil {
					return err
				}
				os.Remove(src)
			}

		case "copy_file":
			src := s.resolvePath(task.Source, extractPath, gameRoot)
			dst := s.resolvePath(task.Target, extractPath, gameRoot)
			s.logInfo(fmt.Sprintf("Kopíruji: %s -> %s", src, dst))
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := copyFile(src, dst); err != nil {
				return err
			}

		case "new_file":
			path := s.resolvePath(task.Target, extractPath, gameRoot)
			if path == "" {
				path = s.resolvePath(task.Source, extractPath, gameRoot)
			}
			s.logInfo(fmt.Sprintf("Vytvářím soubor: %s", path))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			f, err := os.Create(path)
			if err != nil {
				return err
			}
			f.Close()

		case "delete_folder", "delete_dir":
			path := s.resolvePath(task.Source, extractPath, gameRoot)
			if path == "" {
				path = s.resolvePath(task.Target, extractPath, gameRoot)
			}
			s.logInfo(fmt.Sprintf("Mažu složku: %s", path))
			if err := os.RemoveAll(path); err != nil {
				s.logWarning(fmt.Sprintf("Nelze smazat složku: %v", err))
			}

		case "new_folder", "new_dir", "create_dir":
			path := s.resolvePath(task.Target, extractPath, gameRoot)
			if path == "" {
				path = s.resolvePath(task.Source, extractPath, gameRoot)
			}
			s.logInfo(fmt.Sprintf("Vytvářím složku: %s", path))
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}

		case "copy_folder", "move_folder":
			s.logWarning(fmt.Sprintf("Příkaz %s není implementován", cmd))

		case "decompress_file_zip", "decompress_zip_file", "decopress_file_zip":
			src := s.resolvePath(task.Source, extractPath, gameRoot)
			dst := s.resolvePath(task.Target, extractPath, gameRoot)
			if dst == "" {
				dst = filepath.Dir(src)
			}
			s.logInfo(fmt.Sprintf("Rozbaluji: %s -> %s", src, dst))
			if err := s.extractZip(src, dst); err != nil {
				return err
			}

		case "wait_process_finish", "wait_proccess_finnish":
			// Wait for a process - usually handled by run_file with extra
			s.logInfo("Čekám na dokončení procesu...")

		case "wait_task":
			// Wait for specified number of seconds from task.Extra
			seconds := 1
			if task.Extra != "" {
				if _, err := fmt.Sscanf(task.Extra, "%d", &seconds); err != nil {
					seconds = 1
				}
			}
			s.logInfo(fmt.Sprintf("Čekám %d sekund na další příkaz...", seconds))
			time.Sleep(time.Duration(seconds) * time.Second)

		default:
			s.logWarning(fmt.Sprintf("Neznámý příkaz: %s", cmd))
		}
	}

	return nil
}

func (s *Service) executeRunFile(task models.InstallTask, extractPath, gameRoot string) error {
	// Resolve executable path
	path := s.resolvePath(task.Source, extractPath, gameRoot)
	if path == "" {
		path = s.resolvePath(task.Target, extractPath, gameRoot)
	}

	// Working directory is the directory of the script (like C# version)
	workDir := filepath.Dir(path)

	// For arguments use replacePlaceholders (not resolvePath which adds temp folder)
	// Always set args from task.Target (like C# version)
	args := s.replacePlaceholders(task.Target, gameRoot)

	// Convert Unix single quotes to Windows double quotes (CZMaker uses Unix style)
	args = strings.ReplaceAll(args, "'", "\"")

	// Check if should run in background
	extraLower := strings.ToLower(task.Extra)
	runInBackground := strings.Contains(extraLower, "background") || strings.Contains(extraLower, "nowait")

	s.logInfo(fmt.Sprintf("[SCRIPT] Spouštím: %s", path))
	if args != "" {
		s.logInfo(fmt.Sprintf("[SCRIPT] Argumenty: %s", args))
	}
	s.logInfo(fmt.Sprintf("[SCRIPT] Pracovní adresář: %s", workDir))

	var cmd *exec.Cmd
	if args != "" {
		// Parse arguments properly, respecting quoted strings
		argList := parseArguments(args)
		s.logInfo(fmt.Sprintf("[SCRIPT] Parsed args: %v", argList))
		cmd = exec.Command(path, argList...)
	} else {
		cmd = exec.Command(path)
	}
	cmd.Dir = workDir

	if runInBackground {
		if err := cmd.Start(); err != nil {
			s.logWarning(fmt.Sprintf("[SCRIPT] Nelze spustit: %v", err))
		}
		s.logInfo("[SCRIPT] Spuštěno na pozadí")
	} else {
		output, err := cmd.CombinedOutput()
		if len(output) > 0 {
			s.logInfo(fmt.Sprintf("[SCRIPT] Výstup: %s", string(output)))
		}
		if err != nil {
			s.logWarning(fmt.Sprintf("[SCRIPT] Chyba: %v", err))
		}
		s.logInfo("[SCRIPT] Dokončeno")
	}

	return nil
}

func (s *Service) resolvePath(path, extractPath, gameRoot string) string {
	if path == "" {
		return ""
	}

	// Normalize path separators for cross-platform compatibility
	path = strings.ReplaceAll(path, "\\", "/")

	// Replace placeholders
	replacements := map[string]string{
		"{GAME_ROOT}":                   gameRoot,
		"{GAME_DIR}":                    gameRoot,
		"{TEMP}":                        os.TempDir(),
		"{TEMP_DIR}":                    os.TempDir(),
		"{MY_APPLICATION_DATA_ROAMING}": filepath.Join(os.Getenv("APPDATA")),
		"{APPDATA}":                     os.Getenv("APPDATA"),
		"{MY_APPLICATION_DATA_LOCAL}":   os.Getenv("LOCALAPPDATA"),
		"{LOCALAPPDATA}":                os.Getenv("LOCALAPPDATA"),
		"{MY_APPLICATION_DATA_LOW}":     filepath.Join(os.Getenv("LOCALAPPDATA") + "Low"),
		"{MY_USER_PROFILE}":             os.Getenv("USERPROFILE"),
		"{USER_HOME_DIR}":               os.Getenv("USERPROFILE"),
		"{MY_DOCUMENTS}":                filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
		"{DOCUMENTS}":                   filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
	}

	for placeholder, value := range replacements {
		path = strings.ReplaceAll(path, placeholder, value)
	}

	// If path is relative and doesn't contain resolved placeholder, assume relative to game root
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, gameRoot) {
		// Check if file exists in extract path first
		testPath := filepath.Join(extractPath, path)
		if _, err := os.Stat(testPath); err == nil {
			return testPath
		}
		// Otherwise use game root
		path = filepath.Join(gameRoot, path)
	}

	return filepath.Clean(path)
}

// replacePlaceholders just replaces placeholders in text without path logic
// Used for arguments where we don't want to add temp/game root paths
func (s *Service) replacePlaceholders(text, gameRoot string) string {
	if text == "" {
		return ""
	}

	// Normalize path separators for cross-platform compatibility
	text = strings.ReplaceAll(text, "\\", "/")

	replacements := map[string]string{
		"{GAME_ROOT}":                   gameRoot,
		"{GAME_DIR}":                    gameRoot,
		"{TEMP}":                        os.TempDir(),
		"{TEMP_DIR}":                    os.TempDir(),
		"{MY_APPLICATION_DATA_ROAMING}": os.Getenv("APPDATA"),
		"{APPDATA}":                     os.Getenv("APPDATA"),
		"{MY_APPLICATION_DATA_LOCAL}":   os.Getenv("LOCALAPPDATA"),
		"{LOCALAPPDATA}":                os.Getenv("LOCALAPPDATA"),
		"{MY_APPLICATION_DATA_LOW}":     os.Getenv("LOCALAPPDATA") + "Low",
		"{MY_USER_PROFILE}":             os.Getenv("USERPROFILE"),
		"{USER_HOME_DIR}":               os.Getenv("USERPROFILE"),
		"{MY_DOCUMENTS}":                filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
		"{DOCUMENTS}":                   filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
	}

	for placeholder, value := range replacements {
		text = strings.ReplaceAll(text, placeholder, value)
	}

	return text
}

func (s *Service) applyPatch(originalPath, patchPath, outputPath string) error {
	// Use xdelta path from service (set at init from embedded or external)
	xdeltaBin := s.xdeltaPath
	if xdeltaBin == "" {
		// Fallback: determine xdelta3 binary name based on OS
		xdeltaBin = "xdelta3"
		switch runtime.GOOS {
		case "windows":
			xdeltaBin = "xdelta3.exe"
		case "linux":
			xdeltaBin = "xdelta3_linux"
		case "darwin":
			xdeltaBin = "xdelta3_mac"
		}

		// Look for xdelta3 in same directory as executable
		execPath, err := os.Executable()
		if err == nil {
			localXdelta := filepath.Join(filepath.Dir(execPath), xdeltaBin)
			if _, err := os.Stat(localXdelta); err == nil {
				xdeltaBin = localXdelta
			}
		}
	}

	// Create temp output file
	tempOutput := outputPath + ".tmp"

	cmd := exec.Command(xdeltaBin, "-d", "-s", originalPath, patchPath, tempOutput)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xdelta3 selhal: %v, výstup: %s", err, string(output))
	}

	// Replace original with patched file
	if err := os.Remove(outputPath); err != nil && !os.IsNotExist(err) {
		os.Remove(tempOutput)
		return err
	}

	if err := os.Rename(tempOutput, outputPath); err != nil {
		return err
	}

	return nil
}

// Uninstall starts the uninstallation process
func (s *Service) Uninstall(req models.UninstallRequest) error {
	s.mu.Lock()
	if s.isBusy {
		s.mu.Unlock()
		return fmt.Errorf("another operation is in progress")
	}
	s.isBusy = true
	s.cancelChan = make(chan struct{})
	s.mu.Unlock()

	s.ClearLogs()

	go s.doUninstall(req)
	return nil
}

func (s *Service) doUninstall(req models.UninstallRequest) {
	defer func() {
		s.mu.Lock()
		s.isBusy = false
		s.mu.Unlock()
	}()

	s.logInfo("Zahajuji odinstalaci...")
	s.setProgress(models.StageInstalling, 10, "Hledám nainstalované soubory...")

	// Find all .ORIG and .IMPORT files
	var origFiles, importFiles []string

	err := filepath.Walk(req.GameRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ORIG") {
			origFiles = append(origFiles, path)
		} else if strings.HasSuffix(path, ".IMPORT") {
			importFiles = append(importFiles, path)
		}
		return nil
	})

	if err != nil {
		s.setError(fmt.Sprintf("Chyba při prohledávání: %v", err))
		s.logError(err.Error())
		return
	}

	totalFiles := len(origFiles) + len(importFiles)
	if totalFiles == 0 {
		s.setError("Nenalezeny žádné soubory k odinstalaci")
		s.logWarning("Žádné .ORIG nebo .IMPORT soubory nenalezeny")
		return
	}

	s.logInfo(fmt.Sprintf("Nalezeno %d .ORIG a %d .IMPORT souborů", len(origFiles), len(importFiles)))

	processed := 0

	// Restore .ORIG files
	for _, origPath := range origFiles {
		if s.isCancelled() {
			s.setError("Operace byla zrušena")
			return
		}

		originalPath := strings.TrimSuffix(origPath, ".ORIG")
		s.logInfo(fmt.Sprintf("Obnovuji: %s", filepath.Base(originalPath)))

		// Remove modified file
		os.Remove(originalPath)

		// Restore original
		if err := os.Rename(origPath, originalPath); err != nil {
			s.logWarning(fmt.Sprintf("Nelze obnovit: %v", err))
		}

		processed++
		progress := 10 + (processed * 80 / totalFiles)
		s.setProgress(models.StageInstalling, progress, fmt.Sprintf("Obnoveno %d/%d", processed, totalFiles))
	}

	// Delete .IMPORT files and their corresponding files
	for _, importPath := range importFiles {
		if s.isCancelled() {
			s.setError("Operace byla zrušena")
			return
		}

		installedPath := strings.TrimSuffix(importPath, ".IMPORT")
		s.logInfo(fmt.Sprintf("Mažu importovaný: %s", filepath.Base(installedPath)))

		// Remove installed file
		os.Remove(installedPath)

		// Remove import marker
		os.Remove(importPath)

		processed++
		progress := 10 + (processed * 80 / totalFiles)
		s.setProgress(models.StageInstalling, progress, fmt.Sprintf("Zpracováno %d/%d", processed, totalFiles))
	}

	s.setProgress(models.StageDone, 100, "Odinstalace dokončena!")
	s.logInfo("Odinstalace úspěšně dokončena!")
}

// findFileCaseInsensitive finds a file matching the given relative path case-insensitively
// Returns the actual path if found, empty string otherwise
func (s *Service) findFileCaseInsensitive(basePath, relativePath string) string {
	// Split path into components
	parts := strings.Split(relativePath, "/")
	currentPath := basePath

	for _, part := range parts {
		if part == "" {
			continue
		}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return ""
		}

		found := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				currentPath = filepath.Join(currentPath, entry.Name())
				found = true
				break
			}
		}

		if !found {
			return ""
		}
	}

	// Verify it's a file (not directory) at the end
	info, err := os.Stat(currentPath)
	if err != nil || info.IsDir() {
		return ""
	}

	return currentPath
}

// Helper function to copy files
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// parseArguments splits an argument string respecting quoted strings
// Input: `"D:\path with spaces" arg2`
// Output: ["D:\path with spaces", "arg2"]
func parseArguments(args string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, r := range args {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			// Start of quoted string
			inQuotes = true
			quoteChar = r
		case r == quoteChar && inQuotes:
			// End of quoted string
			inQuotes = false
			quoteChar = 0
		case r == ' ' && !inQuotes:
			// Space outside quotes - end of argument
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Don't forget the last argument
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
