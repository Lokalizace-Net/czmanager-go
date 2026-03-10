@echo off
cd /d "%~dp0"

echo ========================================
echo   CZ Manager GUI - Create Release
echo ========================================
echo.

set /p VERSION="Zadej verzi (napr. 1.5.3): "

if "%VERSION%"=="" (
    echo Verze nebyla zadana!
    pause
    exit /b 1
)

echo.
echo Vytvarim tag v%VERSION%...

git add .
git commit -m "Release v%VERSION%"
git tag v%VERSION%
git push origin main
git push origin v%VERSION%

echo.
echo ========================================
echo   Release v%VERSION% vytvoren!
echo ========================================
echo.
echo GitHub Actions ted builduje:
echo   - Windows AMD64
echo   - Linux AMD64 + ARM64
echo   - macOS AMD64 + ARM64
echo   - Flatpak bundle
echo.
echo Sleduj prubeh:
echo   https://github.com/Lokalizace-Net/czmanager-gui/actions
echo.
echo Az dobehne, release najdes na:
echo   https://github.com/Lokalizace-Net/czmanager-gui/releases/tag/v%VERSION%
echo.
pause
