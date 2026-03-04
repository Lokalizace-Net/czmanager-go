@echo off
setlocal

echo ========================================
echo CZManager Agent - Multi-platform Build
echo ========================================
echo.

cd /d "%~dp0"

:: Clean build folder
if not exist "build" mkdir build

:: Windows AMD64
echo [1/4] Building Windows AMD64...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-s -w -H windowsgui" -o build/czmanager-agent-windows-amd64.exe .
if %errorlevel% neq 0 goto :error

:: Linux AMD64
echo [2/4] Building Linux AMD64...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-s -w" -o build/czmanager-agent-linux-amd64 .
if %errorlevel% neq 0 goto :error

:: Linux ARM64 (Steam Deck compatibility)
echo [3/4] Building Linux ARM64...
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags "-s -w" -o build/czmanager-agent-linux-arm64 .
if %errorlevel% neq 0 goto :error

:: macOS AMD64
echo [4/4] Building macOS AMD64...
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-s -w" -o build/czmanager-agent-macos-amd64 .
if %errorlevel% neq 0 goto :error

echo.
echo ========================================
echo Build complete! Files in build folder:
echo ========================================
dir /b build\czmanager-agent-*
echo.
echo xdelta3 binaries are EMBEDDED in executables.
echo Each file is standalone - no external dependencies!
echo ========================================

goto :end

:error
echo.
echo BUILD FAILED!
exit /b 1

:end
endlocal
