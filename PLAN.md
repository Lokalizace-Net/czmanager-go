# Plán: CZManager GUI - Multiplatformní Aplikace

## Shrnutí
Vytvoření multiplatformní desktopové aplikace pro CZManager pomocí **Wails v2** (Go backend) + **SvelteKit** (frontend) s podporou gamepad/klávesnicové navigace pro Steam Deck.

## Technologie
- **Backend**: Wails v2 (Go) - spravuje agenta, nativní dialogy, systémové volání
- **Frontend**: SvelteKit + TypeScript + TailwindCSS
- **Navigace**: Spatial navigation (šipky + gamepad) pomocí vlastní implementace nebo `spatial-navigation-polyfill`
- **Ikony**: Lucide Svelte
- **Stav**: Svelte 5 runes ($state, $derived, $effect)

## Struktura projektu
```
gui/
├── build/                    # Wails build output
├── frontend/                 # SvelteKit aplikace
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/   # UI komponenty
│   │   │   │   ├── GameCard.svelte
│   │   │   │   ├── GameGrid.svelte
│   │   │   │   ├── Header.svelte
│   │   │   │   ├── SearchBar.svelte
│   │   │   │   ├── StatusBadge.svelte
│   │   │   │   ├── ProgressBar.svelte
│   │   │   │   ├── Modal.svelte
│   │   │   │   ├── Button.svelte
│   │   │   │   └── FocusRing.svelte
│   │   │   ├── stores/       # Svelte stores
│   │   │   │   ├── agent.ts      # Stav agenta, token
│   │   │   │   ├── games.ts      # Seznam her, lokalizací
│   │   │   │   ├── navigation.ts # Focus management
│   │   │   │   └── settings.ts   # Uživatelské nastavení
│   │   │   ├── api/          # API komunikace
│   │   │   │   ├── agent.ts      # Lokální agent API
│   │   │   │   └── lokalizace.ts # Lokalizace.NET API
│   │   │   └── utils/
│   │   │       ├── gamepad.ts    # Gamepad input handling
│   │   │       └── spatial.ts    # Spatial navigation
│   │   ├── routes/
│   │   │   ├── +layout.svelte
│   │   │   ├── +page.svelte      # Hlavní stránka (grid her)
│   │   │   ├── game/
│   │   │   │   └── [slug]/
│   │   │   │       └── +page.svelte  # Detail hry
│   │   │   └── settings/
│   │   │       └── +page.svelte  # Nastavení
│   │   ├── app.css           # Globální styly + Tailwind
│   │   └── app.html
│   ├── static/
│   ├── svelte.config.js
│   ├── tailwind.config.js
│   └── package.json
├── main.go                   # Wails hlavní soubor
├── app.go                    # Go metody volané z frontendu
├── agent.go                  # Správa agenta (spawn, komunikace)
├── wails.json
└── go.mod
```

## Fáze implementace

### Fáze 1: Základní struktura (Wails + SvelteKit)
1. Inicializace Wails projektu v `gui/`
2. Konfigurace SvelteKit pro Wails (static adapter)
3. Základní layout s headerem a navigací
4. Integrace TailwindCSS s tmavým tématem
5. Základní Go metody pro:
   - Spuštění agenta na pozadí
   - Získání tokenu z agenta
   - Kontrola běhu agenta

### Fáze 2: Agent Management
1. Go kód pro:
   - Automatické spuštění agenta při startu GUI
   - Health check agenta (/ping)
   - Restart agenta při selhání
   - Ukončení agenta při zavření GUI
2. Svelte store pro stav agenta
3. UI indikátor stavu agenta v headeru

### Fáze 3: Hlavní UI - Seznam her
1. Fetch lokalizací z Lokalizace.NET API
2. GameCard komponenta (obrázek, název, stav štítek)
3. GameGrid komponenta (responzivní mřížka)
4. Statusové štítky:
   - "Překládá se" (červený)
   - "Veřejná verze" (zelený)
   - "Open Beta" (žlutý)
   - "Rozpracováno" (šedý)
5. Tlačítko "Načíst další lokalizace"

### Fáze 4: Vyhledávání a filtrování
1. SearchBar komponenta s ikonou
2. Filtrování her podle názvu
3. Debounced search
4. Prázdný stav při žádných výsledcích

### Fáze 5: Detail hry
1. Stránka s detailem lokalizace
2. Informace o hře (popis, verze, autoři)
3. Detekce nainstalované hry (scan-games API)
4. Výběr cesty ke hře (browse-folder)
5. Tlačítka: Instalovat / Odinstalovat / Aktualizovat

### Fáze 6: Instalační proces
1. Progress bar s fázemi instalace
2. Real-time logy
3. Možnost zrušení
4. Úspěšné/chybové stavy
5. Modal dialogy (vlastní, ne alert())

### Fáze 7: Navigace šipkami + Gamepad
1. Spatial navigation systém:
   - Focus management pro grid
   - Šipky pohybují focusem
   - Enter = klik
   - Escape = zpět
2. Gamepad API integrace:
   - D-pad = šipky
   - A = Enter
   - B = Escape
   - Triggers = scroll
3. Viditelný focus ring (modrý obrys)
4. Pamatování posledního focusu při navigaci

### Fáze 8: Další funkce
1. Stránka nastavení
2. Automatické aktualizace agenta
3. Systémová notifikace po instalaci
4. Podpora pro více jazyků (i18n)
5. "Podpora týmu" odkaz

### Fáze 9: Build a distribuce
1. Windows: .exe installer (NSIS nebo WiX)
2. Linux: AppImage + .deb
3. macOS: .dmg
4. Automatický build přes GitHub Actions
5. Bundlování agenta s GUI

## API Endpointy (Lokalizace.NET)
Na základě screenshotu bude potřeba:
- `GET /api/localizations` - seznam lokalizací
- `GET /api/localizations/{slug}` - detail lokalizace
- `GET /api/agent` - verze agenta (už existuje)

## Design systém
- **Barvy**:
  - Background: #1a1a1a (tmavě šedá)
  - Surface: #2a2a2a (karty)
  - Primary: #ef4444 (červená pro akcent)
  - Text: #ffffff / #a0a0a0
- **Štítky**:
  - Překládá se: bg-red-600
  - Veřejná verze: bg-green-600
  - Open Beta: bg-yellow-600
  - Rozpracováno: bg-gray-600
- **Fonty**: System UI / Inter
- **Border radius**: rounded-lg (8px)
- **Focus ring**: ring-2 ring-blue-500

## Klávesové zkratky
- `←↑↓→` - Navigace mezi prvky
- `Enter` - Výběr/Aktivace
- `Escape` - Zpět/Zavřít modal
- `Tab` - Klasická tab navigace
- `/` nebo `Ctrl+K` - Focus na vyhledávání
- `F5` - Obnovit seznam

## Gamepad mapping
- D-pad: Navigace
- A (Xbox) / X (PS): Výběr
- B (Xbox) / O (PS): Zpět
- LB/RB: Přepínání sekcí
- LT/RT: Scroll stránky
- Start: Menu/Nastavení
