package models

import (
	"time"
)

// InstallStage represents the current stage of installation
type InstallStage string

const (
	StageIdle        InstallStage = "idle"
	StageDownloading InstallStage = "downloading"
	StageExtracting  InstallStage = "extracting"
	StagePreTasks    InstallStage = "pre_tasks"
	StageInstalling  InstallStage = "installing"
	StagePostTasks   InstallStage = "post_tasks"
	StageDone        InstallStage = "done"
	StageError       InstallStage = "error"
)

// TaskType represents the type of pre/post task
type TaskType string

const (
	TaskRunFile    TaskType = "run_file"
	TaskDeleteFile TaskType = "delete_file"
	TaskMoveFile   TaskType = "move_file"
	TaskCopyFile   TaskType = "copy_file"
	TaskCreateDir  TaskType = "create_dir"
	TaskDeleteDir  TaskType = "delete_dir"
)

// InstallRequest is sent by the web app to start installation
type InstallRequest struct {
	GameSlug    string `json:"game_slug"`
	GameID      int    `json:"game_id,omitempty"`      // ID hry z API
	Version     string `json:"version"`
	DownloadURL string `json:"download_url,omitempty"` // Volitelné - agent si zjistí sám
	GameRoot    string `json:"game_root"`
}

// UninstallRequest is sent to uninstall a localization
type UninstallRequest struct {
	GameRoot string `json:"game_root"`
}

// ProgressResponse is returned by /progress endpoint
type ProgressResponse struct {
	Stage    InstallStage `json:"stage"`
	Percent  int          `json:"percent"`
	Message  string       `json:"message"`
	Error    string       `json:"error,omitempty"`
	GameSlug string       `json:"game_slug,omitempty"`
	Version  string       `json:"version,omitempty"`
}

// LogEntry represents a single log message
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// LogsResponse is returned by /logs endpoint
type LogsResponse struct {
	Logs []LogEntry `json:"logs"`
}

// StatusResponse is returned by /status endpoint
type StatusResponse struct {
	Running bool   `json:"running"`
	Version string `json:"version"`
	Busy    bool   `json:"busy"`
}

// PingResponse is returned by /ping endpoint
type PingResponse struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
	Token   string `json:"token"`
}

// BrowseFolderRequest for folder selection dialog
type BrowseFolderRequest struct {
	Title       string `json:"title,omitempty"`
	StartPath   string `json:"start_path,omitempty"`
}

// BrowseFileRequest for file selection dialog
type BrowseFileRequest struct {
	Title       string `json:"title,omitempty"`
	Filter      string `json:"filter,omitempty"`
	StartPath   string `json:"start_path,omitempty"`
}

// BrowseResponse is returned by browse dialogs
type BrowseResponse struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

// ErrorResponse for error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse for simple success responses
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// InstallResponse for install/uninstall endpoints (frontend expects "accepted")
type InstallResponse struct {
	Accepted bool   `json:"accepted"`
	Error    string `json:"error,omitempty"`
}

// InstallInstructions is the structure of INSTALL_INSTRUCTIONS.json
type InstallInstructions struct {
	Generals   *Generals     `json:"generals,omitempty"`
	PreTasks   []InstallTask `json:"pre_tasks,omitempty"`
	PostTasks  []InstallTask `json:"post_tasks,omitempty"`
	ModdFiles  []ModdFile    `json:"modd_files,omitempty"`
}

// Generals contains general installation settings
type Generals struct {
	ShowReadmeAfterInstall bool   `json:"show_readme_after_install,omitempty"`
	GameRootData           string `json:"game_root_data,omitempty"`
}

// InstallTask represents a pre or post installation task
type InstallTask struct {
	Priority int    `json:"priority"`
	Command  string `json:"command"`
	Source   string `json:"source,omitempty"`
	Target   string `json:"target,omitempty"`
	Extra    string `json:"extra,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// ModdFile represents a file to be installed
type ModdFile struct {
	Priority    int    `json:"priority"`
	Name        string `json:"name"`
	OriginalMD5 string `json:"originalMD5,omitempty"`
	ModedMD5    string `json:"modedMD5,omitempty"`
	InstallType string `json:"installType"`
	Optional    bool   `json:"optional,omitempty"`
}

// UninstallInfo stored in .czmanager folder for uninstallation
type UninstallInfo struct {
	GameSlug    string    `json:"game_slug"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

// ScanGamesRequest for /scan-games endpoint
type ScanGamesRequest struct {
	GameName string `json:"game_name,omitempty"` // Optional: search for specific game
}

// ScanGamesResponse returned by /scan-games endpoint
type ScanGamesResponse struct {
	Games []DetectedGame `json:"games"`
}

// DetectedGame represents a detected game installation
type DetectedGame struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Platform string `json:"platform"` // steam, epic, gog, origin, other
	AppID    string `json:"appId,omitempty"`
}
