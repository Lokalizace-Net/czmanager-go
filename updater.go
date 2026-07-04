package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// assetName vrací jméno GitHub release assetu pro aktuální platformu.
// Vrací prázdný řetězec pro platformy, které self-replace nepodporují (macOS).
func assetName() string {
	switch goruntime.GOOS {
	case "windows":
		return "cz-agent-gui-windows-amd64.exe"
	case "linux":
		if goruntime.GOARCH == "arm64" {
			return "cz-agent-gui-linux-arm64"
		}
		return "cz-agent-gui-linux-amd64"
	default:
		// macOS (.app v zipu) - self-replace neřešíme, fallback na release stránku
		return ""
	}
}

// PerformUpdate stáhne nejnovější binárku pro aktuální platformu, nahradí
// běžící spustitelný soubor a restartuje aplikaci. Průběh se streamuje přes
// event "update:progress". Vrací chybu, pokud update selže (volající pak
// může nabídnout ruční stažení).
func (a *App) PerformUpdate() error {
	asset := assetName()
	if asset == "" {
		return fmt.Errorf("automatická aktualizace není na této platformě podporována")
	}

	a.emitUpdate("downloading", 0, "Zjišťuji nejnovější verzi...")

	// Zjisti download URL assetu z nejnovějšího release
	downloadURL, err := a.latestAssetURL(asset)
	if err != nil {
		a.emitUpdate("error", 0, err.Error())
		return err
	}

	// Cesta k běžícímu spustitelnému souboru
	exePath, err := os.Executable()
	if err != nil {
		a.emitUpdate("error", 0, "nelze zjistit cestu k aplikaci")
		return fmt.Errorf("nelze zjistit cestu k aplikaci: %v", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	// Stáhni novou binárku vedle stávající (jako .new)
	newPath := exePath + ".new"
	a.emitUpdate("downloading", 10, "Stahuji novou verzi...")
	if err := a.downloadTo(downloadURL, newPath); err != nil {
		os.Remove(newPath)
		a.emitUpdate("error", 0, fmt.Sprintf("stahování selhalo: %v", err))
		return err
	}

	// Nastav práva pro spuštění (Linux/macOS)
	if goruntime.GOOS != "windows" {
		os.Chmod(newPath, 0755)
	}

	a.emitUpdate("installing", 90, "Instaluji aktualizaci...")

	// Přejmenuj starou binárku na .old (běžící soubor nejde přepsat, ale jde
	// přejmenovat na Windows i Unixu), pak přesuň novou na jeho místo.
	oldPath := exePath + ".old"
	os.Remove(oldPath) // úklid po předchozím updatu
	if err := os.Rename(exePath, oldPath); err != nil {
		os.Remove(newPath)
		a.emitUpdate("error", 0, "nelze nahradit aplikaci (spusťte jako správce?)")
		return fmt.Errorf("nelze přejmenovat starou binárku: %v", err)
	}
	if err := os.Rename(newPath, exePath); err != nil {
		// Pokus o návrat zpět
		os.Rename(oldPath, exePath)
		os.Remove(newPath)
		a.emitUpdate("error", 0, "nelze nainstalovat novou verzi")
		return fmt.Errorf("nelze přesunout novou binárku: %v", err)
	}

	a.emitUpdate("restarting", 100, "Restartuji aplikaci...")

	// Restartuj a ukonči tento proces
	if err := a.restart(exePath); err != nil {
		a.emitUpdate("error", 0, "aktualizace hotova, restartujte aplikaci ručně")
		return err
	}
	return nil
}

// latestAssetURL najde download URL daného assetu v nejnovějším release.
func (a *App) latestAssetURL(asset string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("nepodařilo se spojit s GitHubem: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API vrátilo status %d", resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("nepodařilo se zpracovat odpověď GitHubu: %v", err)
	}

	for _, as := range release.Assets {
		if as.Name == asset {
			return as.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("v nejnovějším vydání nebyl nalezen soubor %s", asset)
}

// downloadTo stáhne URL do souboru s průběhem (10-90 %).
func (a *App) downloadTo(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server vrátil status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if total > 0 {
				pct := 10 + int(downloaded*80/total)
				a.emitUpdate("downloading", pct, fmt.Sprintf("Stahuji... %d%%", downloaded*100/total))
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return nil
}

func (a *App) emitUpdate(stage string, percent int, message string) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]any{
			"stage":   stage,
			"percent": percent,
			"message": message,
		})
	}
}
