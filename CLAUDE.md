# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CZManager Agent is a cross-platform Go application that serves as a local HTTP agent for installing game localizations. It runs as a background service on port 17892, communicating with the Lokalizace.NET web application to handle game file patching and localization installations.

## Build Commands

```bash
# Development build (current platform)
go build -o build/czmanager-agent .

# Run for development
go run .

# Multi-platform build (Windows)
build.bat

# Multi-platform build (Linux/macOS)
make all

# Individual platform builds
make windows-amd64
make linux-amd64
make linux-arm64

# Check code
go vet ./...
go fmt ./...
```

The build uses `-ldflags "-s -w"` for smaller binaries. Windows builds include `-H windowsgui` to hide the console window.

## Architecture

### HTTP API Server (main.go)
- Listens on `127.0.0.1:17892`
- Token-based authentication via `Authorization` header or `token` query param
- CORS enabled for web app communication

**Endpoints:**
- `GET /ping` - Public health check, returns token
- `GET /status` - Agent status and busy state
- `POST /install` - Start localization installation
- `POST /uninstall` - Remove localization (restores .ORIG files)
- `GET /progress` - Installation progress
- `GET /logs` - Installation logs (supports `?since=N`)
- `POST /cancel` - Cancel current operation
- `POST /browse-folder` / `POST /browse-file` - Native file dialogs
- `POST /scan-games` - Detect installed games
- `GET /update-check` / `POST /update` - Self-update

### Installer Service (installer/installer.go)
Processes `INSTALL_INSTRUCTIONS.json` from downloaded ZIP archives:
- **Pre-tasks**: Commands run before file installation
- **ModdFiles**: Files to patch/replace/insert (sorted by priority)
- **Post-tasks**: Commands run after installation

**Install types:**
- `patch` - Apply xdelta3 binary diff (backs up original as `.ORIG`)
- `insert`/`patch_insert` - Copy new file (creates `.IMPORT` marker for new files)
- `replace` - Replace existing file (backs up as `.ORIG`)

**Path placeholders:** `{GAME_ROOT}`, `{APPDATA}`, `{LOCALAPPDATA}`, `{MY_DOCUMENTS}`, etc.

### Game Scanner (scanner.go + scanner_*.go)
Detects installed games from multiple platforms:
- Steam (parses `libraryfolders.vdf` and `appmanifest_*.acf`)
- Epic Games (parses `.item` manifests)
- GOG, Origin/EA, Ubisoft Connect

Platform-specific implementations use build tags (`scanner_windows.go`, `scanner_other.go`).

### Self-Updater (updater.go)
- Checks `https://lokalizace.net/api/agent` hourly
- Downloads new binary, renames current to `.old`, replaces and restarts

### Platform-Specific Files
- `dialog_*.go` - Native folder/file picker dialogs
- `hide_*.go` - Console window hiding (Windows only)
- `systray_*.go` - System tray icon (Windows only)
- `exec_*.go` - Process execution (Unix uses syscall.Exec for restart)
- `xdelta_embed.go` - Embedded xdelta3 binaries for all platforms

## Key Dependencies

- `github.com/sqweek/dialog` - Native file dialogs
- Embedded `xdelta3` binaries in `resources/` for binary patching

## Important Patterns

1. **Backup System**: Modified files get `.ORIG` suffix, new files get `.IMPORT` marker - uninstall reverses this
2. **Cross-platform paths**: ZIP files may contain Windows backslashes; code normalizes to forward slashes
3. **Case-insensitive file matching**: Linux builds include case-insensitive file lookup for compatibility with Windows-created ZIPs
