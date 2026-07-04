@echo off
cd /d "%~dp0"

echo ========================================
echo   CZ Agent GUI - Windows Build
echo ========================================
echo.
echo   (Pro Linux buildy pouzij build.sh na Linuxu)
echo.

REM Zjisti verzi z git tagu (jinak "dev")
set VERSION=dev
for /f "delims=" %%v in ('git describe --tags --always 2^>nul') do set VERSION=%%v
echo Verze: %VERSION%

echo Building Windows AMD64...
go run github.com/wailsapp/wails/v2/cmd/wails@latest build -platform windows/amd64 -ldflags "-X main.Version=%VERSION%"
if %ERRORLEVEL% NEQ 0 goto :error

echo.
echo ========================================
echo   Build complete!
echo ========================================
echo.
echo Output: build\bin\cz-agent-gui.exe
echo.
pause
goto :eof

:error
echo.
echo BUILD FAILED!
pause
