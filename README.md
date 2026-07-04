# CZManager GUI

Multiplatformní desktopová aplikace pro instalaci českých lokalizací her. Postavená na **Wails v2** (Go backend) + **Svelte 5** (frontend) s podporou gamepad a klávesnicové navigace pro Steam Deck.

Komunikuje s webovou aplikací [Lokalizace.NET](https://lokalizace.net) a lokálním HTTP agentem pro správu herních souborů.

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go&logoColor=white)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-412991)
![TailwindCSS](https://img.shields.io/badge/Tailwind_CSS-4-06B6D4?logo=tailwindcss&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20Steam%20Deck-lightgrey)

## Funkce

- **Prohlížení lokalizací** — přehledný grid s kartami her, vyhledávání a filtrování
- **Oblíbené** — možnost přidat hry do oblíbených pro rychlý přístup
- **Instalace / odinstalace** — stažení a aplikace lokalizačních patchů jedním kliknutím
- **Detekce her** — automatické rozpoznání nainstalovaných her (Steam, Epic Games, GOG, Origin, Ubisoft Connect)
- **Progress & logy** — real-time progress bar a log výstup během instalace
- **Gamepad navigace** — plná podpora D-pad, tlačítek a triggerů pro Steam Deck
- **Klávesnicová navigace** — šipky, Enter, Escape, Ctrl+K pro vyhledávání
- **Správa agenta** — automatické stažení, spuštění a health-check lokálního agenta
- **Autentizace** — přihlášení přes Lokalizace.NET účet

## Prerekvizity

- [Go](https://go.dev/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Systémové závislosti (Linux)

```bash
# Ubuntu/Debian
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

# Fedora
sudo dnf install gtk3-devel webkit2gtk3-devel
```

## Spuštění (vývoj)

```bash
cd gui/czmanager

# Instalace frontend závislostí
cd frontend && npm install && cd ..

# Spuštění v dev režimu s hot-reload
wails dev
```

## Build

```bash
cd gui/czmanager

# Build pro aktuální platformu
wails build

# Výstup: build/bin/cz-agent-gui[.exe]
```

### Agent (HTTP služba na pozadí)

```bash
# Build agenta
go build -ldflags "-s -w" -o build/czmanager-agent .

# Multiplatformní build (Windows)
build.bat

# Multiplatformní build (Linux/macOS)
make all
```

| Target | Příkaz |
|--------|--------|
| Windows AMD64 | `make windows-amd64` |
| Linux AMD64 | `make linux-amd64` |
| Linux ARM64 (Steam Deck) | `make linux-arm64` |

## Architektura

```
┌─────────────────────────────────────┐
│           CZManager GUI             │
│  ┌───────────┐  ┌────────────────┐  │
│  │  Svelte 5 │◄►│  Wails (Go)    │  │
│  │  Frontend  │  │  Backend       │  │
│  └───────────┘  └───────┬────────┘  │
│                         │           │
└─────────────────────────┼───────────┘
                          │ HTTP :17892
┌─────────────────────────▼───────────┐
│         CZManager Agent             │
│  • Instalace / odinstalace patchů   │
│  • Skenování her                    │
│  • Nativní dialogy                  │
│  • Self-update                      │
└─────────────────────────────────────┘
```

### Struktura projektu

```
czmanager-gui/
├── main.go                  # HTTP agent server (:17892)
├── installer/               # Instalační logika (xdelta3 patching)
├── scanner.go               # Detekce her z platforem
├── updater.go               # Self-update agenta
├── Makefile                 # Multiplatformní build
│
└── gui/czmanager/           # Wails GUI aplikace
    ├── app.go               # Go backend (Wails bindings)
    ├── wails.json           # Wails konfigurace
    └── frontend/            # Svelte 5 aplikace
        └── src/
            ├── App.svelte           # Hlavní komponenta
            ├── lib/components/      # UI komponenty
            ├── lib/stores/          # Svelte 5 runes stores
            └── lib/utils/           # Gamepad, navigace
```

### Technologie

| Vrstva | Technologie |
|--------|------------|
| Desktop framework | Wails v2 |
| Backend | Go 1.21 |
| Frontend | Svelte 5 (runes) + TypeScript |
| Styling | Tailwind CSS 4 |
| Ikony | Lucide Svelte |
| Patching | xdelta3 (embedded) |

## Agent API

Agent běží na `127.0.0.1:17892` s token autentizací.

| Metoda | Endpoint | Popis |
|--------|----------|-------|
| `GET` | `/ping` | Health check (veřejný) |
| `GET` | `/status` | Stav agenta |
| `POST` | `/install` | Spustit instalaci |
| `POST` | `/uninstall` | Odinstalovat lokalizaci |
| `GET` | `/progress` | Průběh instalace |
| `GET` | `/logs` | Logy instalace |
| `POST` | `/cancel` | Zrušit operaci |
| `POST` | `/scan-games` | Detekce nainstalovaných her |

## Ovládání

### Klávesnice

| Klávesa | Akce |
|---------|------|
| `←↑↓→` | Navigace mezi prvky |
| `Enter` | Výběr / aktivace |
| `Escape` | Zpět / zavřít modal |
| `Ctrl+K` | Vyhledávání |
| `F5` | Obnovit seznam |

### Gamepad (Steam Deck)

| Tlačítko | Akce |
|----------|------|
| D-pad | Navigace |
| A / X | Výběr |
| B / O | Zpět |
| LB / RB | Přepínání sekcí |
| LT / RT | Scroll |

## Licence

Proprietární software. Všechna práva vyhrazena.

## Autor

**michalss**
