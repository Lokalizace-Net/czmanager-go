@echo off
setlocal enabledelayedexpansion
cd /d "%~dp0"

echo ========================================
echo   CZManager GUI - Release
echo ========================================
echo.
echo Tento skript vytvori git tag a pushne ho na GitHub.
echo Tim se spusti workflow "Build ^& Release", ktery:
echo   - zbuilduje Windows / Linux / macOS binarky
echo   - vytvori .flatpak bundle
echo   - vytvori GitHub Release s vsemi soubory
echo.

REM Zjisti posledni tag pro info
for /f "delims=" %%t in ('git describe --tags --abbrev^=0 2^>nul') do set LASTTAG=%%t
if defined LASTTAG (
  echo Posledni verze: !LASTTAG!
) else (
  echo Zatim zadny tag.
)
echo.

set /p VERSION="Zadej novou verzi (napr. v1.6.1): "

if "!VERSION!"=="" (
  echo.
  echo CHYBA: Verze nezadana. Koncim.
  pause
  exit /b 1
)

REM Musi zacinat 'v' (workflow reaguje na tagy v*)
echo !VERSION!| findstr /r "^v[0-9]" >nul
if errorlevel 1 (
  echo.
  echo CHYBA: Verze musi zacinat 'v' a cislem, napr. v1.6.1
  pause
  exit /b 1
)

REM Overeni ze tag jeste neexistuje
git rev-parse "!VERSION!" >nul 2>&1
if not errorlevel 1 (
  echo.
  echo CHYBA: Tag !VERSION! uz existuje.
  pause
  exit /b 1
)

echo.
echo Vytvorim a pushnu tag: !VERSION!
set /p CONFIRM="Pokracovat? (a/n): "
if /i not "!CONFIRM!"=="a" (
  echo Zruseno.
  pause
  exit /b 0
)

echo.
echo Vytvarim tag !VERSION!...
git tag -a "!VERSION!" -m "Release !VERSION!"
if errorlevel 1 goto :error

echo Pushuji tag na GitHub...
git push origin "!VERSION!"
if errorlevel 1 goto :error

echo.
echo ========================================
echo   Hotovo! Tag !VERSION! pushnut.
echo ========================================
echo.
echo Workflow bezi zde:
echo   https://github.com/Lokalizace-Net/czmanager-go/actions
echo.
echo Release se objevi zde (az workflow dobehne):
echo   https://github.com/Lokalizace-Net/czmanager-go/releases
echo.
pause
goto :eof

:error
echo.
echo CHYBA pri vytvareni/pushovani tagu!
pause
exit /b 1
