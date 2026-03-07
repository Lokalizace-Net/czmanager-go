@echo off
cd /d "%~dp0"

echo ========================================
echo   CZ Agent GUI - Windows Build
echo ========================================
echo.
echo   (Pro Linux buildy pouzij build.sh na Linuxu)
echo.

echo Building Windows AMD64...
go run github.com/wailsapp/wails/v2/cmd/wails@latest build -platform windows/amd64
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
