# CZManager GUI

Multiplatformní desktopová aplikace pro instalaci českých lokalizací her. Postavená na **Wails v2** (Go backend) + **Svelte 5** (frontend) s podporou gamepad a klávesnicové navigace pro Steam Deck.

Komunikuje s webovou aplikací [Lokalizace.NET](https://lokalizace.net) a instalaci lokalizačních balíčků provádí přímo v procesu aplikace (žádný samostatný agent na pozadí).

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)
![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-412991)
![TailwindCSS](https://img.shields.io/badge/Tailwind_CSS-4-06B6D4?logo=tailwindcss&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20Steam%20Deck-lightgrey)
![License](https://img.shields.io/badge/License-GPL_3.0-blue)

## Funkce

- **Prohlížení lokalizací** — přehledný grid s kartami her, vyhledávání a filtrování (nejnovější nahoře)
- **Oblíbené** — možnost přidat hry do oblíbených pro rychlý přístup
- **Instalace / odinstalace** — stažení a aplikace lokalizačních patchů jedním kliknutím
- **Manuální instalace** — instalace lokalizačního balíčku z lokálního ZIP archivu (pro tvůrce k testování balíčků před nahráním)
- **Detekce her** — automatické rozpoznání nainstalovaných her (Steam, Epic Games, GOG, Origin, Ubisoft Connect)
- **Progress & logy** — real-time progress bar a log výstup během instalace, s perzistentním logem i po dokončení
- **Gamepad navigace** — podpora D-pad a tlačítek pro Steam Deck
- **Klávesnicová navigace** — šipky, Enter, Escape, `/` pro vyhledávání
- **Autentizace** — přihlášení přes Lokalizace.NET účet

## Prerekvizity

- [Go](https://go.dev/) 1.23+
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
# Instalace frontend závislostí
cd frontend && npm install && cd ..

# Spuštění v dev režimu s hot-reload
wails dev
```

## Build

```bash
# Build pro aktuální platformu
wails build

# Výstup: build/bin/cz-agent-gui[.exe]
```

Pro dávkový build lze použít pomocné skripty v kořeni repozitáře:

```bash
# Windows
build.bat

# Linux / macOS
./build.sh
```

## Architektura

Instalace i skenování běží **v procesu** aplikace přes Wails bindings — frontend
volá Go metody přímo, žádný samostatný HTTP agent už neexistuje. Progress a logy
se do frontendu streamují přes Wails eventy (`install:progress`, `install:log`).

```
┌─────────────────────────────────────────┐
│              CZManager GUI               │
│  ┌───────────┐      ┌─────────────────┐  │
│  │ Svelte 5  │◄────►│   Wails (Go)    │  │
│  │ Frontend  │ Wails│   Backend       │  │
│  └───────────┘ bind.└────────┬────────┘  │
│                              │           │
│                     ┌────────▼────────┐  │
│                     │ internal/       │  │
│                     │ • installer     │  │
│                     │ • scanner       │  │
│                     │ • xdelta3       │  │
│                     └─────────────────┘  │
└───────────────────────────┬──────────────┘
                            │ HTTPS
                   ┌────────▼────────┐
                   │  Lokalizace.NET │
                   │  (web API)      │
                   └─────────────────┘
```

### Struktura projektu

```
czmanager-gui/
├── main.go                  # Vstupní bod Wails aplikace
├── app.go                   # Go backend (Wails bindings volané z frontendu)
├── wails.json               # Wails konfigurace
│
├── internal/
│   ├── installer/           # Instalační logika (ZIP, xdelta3 patching, tasky)
│   ├── models/              # Datové modely (InstallRequest, instrukce, ...)
│   ├── scanner/             # Detekce her z herních platforem
│   └── xdelta/              # Embedded xdelta3 binárky pro patching
│
└── frontend/                # Svelte 5 aplikace
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
| Backend | Go 1.23 |
| Frontend | Svelte 5 (runes) + TypeScript |
| Styling | Tailwind CSS 4 |
| Ikony | Lucide Svelte |
| Patching | xdelta3 (embedded) |

## Ovládání

### Klávesnice

| Klávesa | Akce |
|---------|------|
| `←↑↓→` | Navigace mezi prvky |
| `Enter` | Výběr / aktivace |
| `Escape` | Zpět / zavřít modal |
| `/` | Vyhledávání |

### Gamepad (Steam Deck)

| Tlačítko | Akce |
|----------|------|
| D-pad | Navigace |
| A | Výběr |
| B | Zpět |

## Licence

Tento projekt je open-source pod licencí **[GNU General Public License v3.0](LICENSE)**.

Copyright © 2026 Lokalizace.NET

Ve zkratce to znamená, že smíte software volně používat, studovat, upravovat
i šířit (včetně komerčního použití). Pokud ale šíříte upravenou verzi, musíte
ji uvolnit rovněž pod GPL-3.0 a zpřístupnit její zdrojový kód. Plné znění je
v souboru [LICENSE](LICENSE), informace o použitých komponentách třetích stran
(např. xdelta3) v souboru [NOTICE](NOTICE).

## Autor

**michalss** — [Lokalizace.NET](https://lokalizace.net)
