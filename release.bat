@echo off
cd /d "%~dp0"

echo ========================================
echo   CZ Agent GUI - Create Release
echo ========================================
echo.

set /p VERSION="Zadej verzi (napr. 1.0.0): "

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
git push
git push origin v%VERSION%

echo.
echo ========================================
echo   Release v%VERSION% vytvoren!
echo ========================================
echo.
echo GitHub Actions ted builduje vsechny platformy.
echo Az dobehne, najdes soubory v:
echo   https://github.com/user/repo/releases/tag/v%VERSION%
echo.
pause
