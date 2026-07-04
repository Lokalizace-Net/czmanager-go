<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import SideMenu from './lib/components/SideMenu.svelte'
  import ManualInstall from './lib/components/ManualInstall.svelte'
  import WelcomeBanner from './lib/components/WelcomeBanner.svelte'
  import GameGrid from './lib/components/GameGrid.svelte'
  import GameDetail from './lib/components/GameDetail.svelte'
  import Modal from './lib/components/Modal.svelte'
  import LoginModal from './lib/components/LoginModal.svelte'
  import LogPanel from './lib/components/LogPanel.svelte'
  import GameCard from './lib/components/GameCard.svelte'
  import { gamesStore, type Localization } from './lib/stores/games.svelte'
  import { focusStore } from './lib/stores/focus.svelte'
  import { authStore } from './lib/stores/auth.svelte'
  import { favoritesStore, favoriteLocalizations } from './lib/stores/favorites.svelte'
  import { appStore } from './lib/stores/app.svelte'
  import UpdateNotice from './lib/components/UpdateNotice.svelte'
  import { startGamepadPolling, stopGamepadPolling } from './lib/utils/gamepad'
  import { FetchGameDetail } from '../wailsjs/go/main/App'
  import { Loader2, Search, Terminal, Heart, ExternalLink, Github, RefreshCw, CheckCircle, Sparkles } from 'lucide-svelte'

  let selectedGame = $state<Localization | null>(null)
  let showGameDetail = $state(false)
  let showLoginModal = $state(false)
  let showLogPanel = $state(false)
  let initializing = $state(true)
  let activeMenuItem = $state('home')
  let searchQuery = $state('')
  let searchInput = $state<HTMLInputElement | undefined>(undefined)
  let favGridElement = $state<HTMLElement | undefined>(undefined)
  let checkingUpdate = $state(false)

  const API_WEB = 'https://lokalizace.net'
  const GITHUB_URL = 'https://github.com/Lokalizace-Net/czmanager-go'

  // Ruční kontrola aktualizace z Nastavení
  async function checkForUpdate() {
    checkingUpdate = true
    await appStore.checkUpdate()
    checkingUpdate = false
  }

  function openExternal(url: string) {
    appStore.openUrl(url)
  }

  // Registruj focus zónu pro favorites grid když se změní obsah
  $effect(() => {
    const favs = $favoriteLocalizations
    if (activeMenuItem === 'favorites' && favGridElement && favs.length > 0) {
      setTimeout(() => {
        if (!favGridElement) return
        const cards = Array.from(favGridElement.querySelectorAll('.game-card')) as HTMLButtonElement[]
        const width = favGridElement.clientWidth
        const cols = Math.max(2, Math.min(8, Math.floor((width + 20) / 240)))
        focusStore.registerZone({
          id: 'main',
          elements: cards,
          columns: cols,
          loop: false,
          onEscape: () => focusStore.setActiveZone('sidemenu', false)
        })
      }, 50)
    }
  })

  function handleGamepadFavorite() {
    if (!$authStore.user) return
    const game = getFocusedGame()
    if (game) {
      favoritesStore.toggleFavorite(game.id)
    }
  }

  onMount(async () => {
    // Načti verzi aplikace
    await appStore.init()

    // Zkontroluj aktualizaci na GitHubu (na pozadí, tichá chyba)
    appStore.checkUpdate()

    // Start gamepad polling
    startGamepadPolling()

    // Gamepad Y = toggle favorite
    window.addEventListener('gamepad:favorite', handleGamepadFavorite)

    // Load games
    await gamesStore.fetchLocalizations(true)

    // Initialize auth (check stored tokens)
    await authStore.init()

    // Načti oblíbené z API pokud je uživatel přihlášen
    await favoritesStore.fetchFromApi()

    initializing = false
  })

  onDestroy(() => {
    stopGamepadPolling()
    window.removeEventListener('gamepad:favorite', handleGamepadFavorite)
  })

  function handleGameSelect(game: Localization) {
    selectedGame = game
    showGameDetail = true
    focusStore.setActiveZone('modal')
  }

  const API_BASE = 'https://lokalizace.net'

  function mapStatus(status: string): Localization['status'] {
    const m: Record<string, Localization['status']> = {
      'draft': 'draft', 'translating': 'translating', 'alpha': 'wip', 'open_beta': 'beta', 'public': 'released'
    }
    return m[status?.toLowerCase()] || 'wip'
  }

  async function handleFavoriteGameSelect(favGame: Localization) {
    // Nejprve zkus najít hru v hlavním games storu (má plná data)
    const fromStore = $gamesStore.localizations.find(g => g.id === favGame.id)
    if (fromStore) {
      handleGameSelect(fromStore)
      return
    }

    // Jinak načti plný detail z API
    try {
      const detail = await FetchGameDetail(favGame.id)
      const item = detail as any
      const status = mapStatus(item.status)
      const availability = item.availability || 'web_only'

      const fullGame: Localization = {
        id: item.id,
        slug: item.slug,
        name: item.name,
        description: item.story || '',
        imageUrl: item.thumbnail ? `${API_BASE}${item.thumbnail}` : favGame.imageUrl,
        heroImageUrl: item.heroImage ? `${API_BASE}${item.heroImage}` : undefined,
        status,
        version: item.version || '1.0.0',
        downloadUrl: item.downloadUrl || undefined,
        teamName: item.teamName,
        teamSlug: item.teamSlug,
        translatePercent: item.translatePercent || 0,
        correctionPercent: item.correctionPercent || 0,
        testingPercent: item.testingPercent || 0,
        rating: item.rating,
        totalRatings: item.totalRatings,
        availability,
        supportsAppInstall: availability === 'app_only' || availability === 'both'
      }

      handleGameSelect(fullGame)
    } catch (e) {
      console.error('Failed to fetch game detail:', e)
      // Fallback — otevři s neúplnými daty
      handleGameSelect(favGame)
    }
  }

  function handleCloseDetail() {
    showGameDetail = false
    selectedGame = null
    focusStore.setActiveZone('main')
  }

  function handleMenuNavigate(item: string) {
    activeMenuItem = item
    focusStore.setActiveZone('main')
  }

  function handleSearch() {
    gamesStore.setSearchQuery(searchQuery)
  }

  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSearch()
      searchInput?.blur()
    } else if (e.key === 'Escape') {
      searchInput?.blur()
      focusStore.setActiveZone('main')
    }
  }

  // Hlavní keyboard handler pro navigaci
  function handleGlobalKeydown(e: KeyboardEvent) {
    const inSearch = e.target === searchInput
    const inInput = e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement

    // Zkratka / pro vyhledávání
    if (e.key === '/' && !inInput) {
      e.preventDefault()
      searchInput?.focus()
      return
    }

    // SEARCH INPUT navigace
    if (inSearch) {
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        searchInput?.blur()
        focusStore.setActiveZone('sidemenu', false)
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        searchInput?.blur()
        focusStore.setActiveZone('main', false)
      } else if (e.key === 'Escape') {
        e.preventDefault()
        searchInput?.blur()
        focusStore.setActiveZone('main', false)
      }
      return
    }

    // Ostatní inputy (např. "Cesta ke hře" v modalu): deleguj na focusStore,
    // který u šipek nahoru/dolů + Escape vyskočí z inputu ven, ale šipky
    // vlevo/vpravo nechá nativní (pohyb kurzoru v textu).
    if (inInput) {
      focusStore.handleKeydown(e)
      return
    }

    const currentZone = $focusStore.activeZone
    const state = $focusStore
    const zone = state.zones.get('main')

    // Support button je na indexu 0 - speciální navigace
    const hasSupportBtn = zone?.elements[0]?.classList.contains('support-btn')
    const onSupportBtn = hasSupportBtn && state.focusedIndex === 0
    const cols = zone?.columns || 4

    // Ze support buttonu: nahoru -> search, doleva -> menu, dolů/doprava -> první karta
    if (onSupportBtn && currentZone === 'main') {
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        searchInput?.focus()
        return
      }
      if (e.key === 'ArrowLeft' || e.key === 'Escape') {
        e.preventDefault()
        focusStore.setActiveZone('sidemenu', false)
        return
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
        e.preventDefault()
        focusStore.setFocusedIndex(1) // První karta
        return
      }
    }

    // Šipka nahoru z prvního řádku karet -> support button (pokud existuje) nebo search
    if (e.key === 'ArrowUp' && currentZone === 'main' && zone) {
      // První řádek karet (indexy 1 až cols pokud je support btn, jinak 0 až cols-1)
      const firstCardIndex = hasSupportBtn ? 1 : 0
      const lastFirstRowIndex = hasSupportBtn ? cols : cols - 1

      if (state.focusedIndex >= firstCardIndex && state.focusedIndex <= lastFirstRowIndex) {
        e.preventDefault()
        if (hasSupportBtn) {
          focusStore.setFocusedIndex(0) // Jdi na support button
        } else {
          searchInput?.focus()
        }
        return
      }
    }

    // Escape nebo šipka doleva z main (i prázdného) -> sidemenu
    if ((e.key === 'Escape' || e.key === 'ArrowLeft') && currentZone === 'main') {
      // Escape vždy jde na menu, ArrowLeft jen pokud jsme na levém kraji gridu nebo nemáme elementy
      if (!zone || zone.elements.length === 0) {
        e.preventDefault()
        focusStore.setActiveZone('sidemenu', false)
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        focusStore.setActiveZone('sidemenu', false)
        return
      }
      // ArrowLeft -> menu jen pokud jsme na levém okraji (první sloupec)
      if (e.key === 'ArrowLeft' && state.focusedIndex % cols === 0) {
        e.preventDefault()
        focusStore.setActiveZone('sidemenu', false)
        return
      }
    }

    // F = toggle oblíbené na fokusnuté kartě
    if ((e.key === 'f' || e.key === 'F') && currentZone === 'main' && $authStore.user) {
      e.preventDefault()
      const focusedGame = getFocusedGame()
      if (focusedGame) {
        favoritesStore.toggleFavorite(focusedGame.id)
      }
      return
    }

    // Vše ostatní deleguj na focusStore
    focusStore.handleKeydown(e)
  }

  // Získej hru na aktuálně fokusnutém indexu
  function getFocusedGame(): Localization | null {
    const idx = $focusStore.focusedIndex
    if (activeMenuItem === 'favorites') {
      const favs = $favoriteLocalizations
      return favs[idx] || null
    }
    // Na home stránce - offset kvůli support buttonu
    const zone = $focusStore.zones.get('main')
    const hasSupportBtn = zone?.elements[0]?.classList.contains('support-btn')
    const gameIdx = hasSupportBtn ? idx - 1 : idx
    const games = $gamesStore.localizations
    return games[gameIdx] || null
  }
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<div class="app-container">
  <SideMenu
    activeItem={activeMenuItem}
    onNavigate={handleMenuNavigate}
    onLoginClick={() => showLoginModal = true}
  />

  <div class="main-area">
    <!-- Top bar se searchem -->
    <header class="top-bar">
      <div class="search-box">
        <Search size={18} />
        <input
          bind:this={searchInput}
          type="text"
          placeholder="Hledat lokalizace... (stiskni /)"
          bind:value={searchQuery}
          onkeydown={handleSearchKeydown}
          oninput={handleSearch}
        />
      </div>

    </header>

    {#if initializing}
      <div class="loading-container">
        <div class="loading-content">
          <Loader2 size={48} class="spinning" />
          <p class="loading-text">Načítání...</p>
        </div>
      </div>
    {:else}
      <main class="main-content">
        {#if activeMenuItem === 'home'}
          <WelcomeBanner />
          <GameGrid onGameSelect={handleGameSelect} />

        {:else if activeMenuItem === 'favorites'}
          <div class="page-content">
            <div class="favorites-header">
              <h1 class="page-title">Oblíbené</h1>
              {#if $authStore.user && $authStore.features}
                <span class="favorites-count">
                  {$favoritesStore.ids.length} / {$authStore.features.maxFavorites}
                </span>
              {/if}
            </div>

            {#if $favoritesStore.limitError}
              <div class="favorites-limit-msg">
                <Heart size={16} />
                <span>{$favoritesStore.limitError}</span>
                <button class="limit-dismiss" onclick={() => favoritesStore.clearLimitError()}>×</button>
              </div>
            {/if}

            {#if !$authStore.user}
              <div class="favorites-empty">
                <Heart size={48} />
                <p class="empty-title">Pro používání oblíbených se přihlaste</p>
                <button class="btn-login" onclick={() => showLoginModal = true}>Přihlásit se</button>
              </div>
            {:else if $favoriteLocalizations.length === 0}
              <div class="favorites-empty">
                <Heart size={48} />
                <p class="empty-title">Zatím nemáte žádné oblíbené lokalizace</p>
                <p class="empty-hint">Klikněte na srdíčko u libovolné hry pro přidání do oblíbených.</p>
              </div>
            {:else}
              <div class="favorites-grid" bind:this={favGridElement}>
                {#each $favoriteLocalizations as game, index (game.id)}
                  <GameCard
                    {game}
                    focused={$focusStore.activeZone === 'main' && $focusStore.focusedIndex === index}
                    isFavorite={true}
                    showFavoriteBtn={true}
                    onclick={() => {
                      focusStore.setFocusedIndex(index)
                      handleFavoriteGameSelect(game)
                    }}
                    onfocus={() => {
                      focusStore.setActiveZone('main', false)
                      focusStore.setFocusedIndex(index)
                    }}
                    onToggleFavorite={() => favoritesStore.toggleFavorite(game.id)}
                  />
                {/each}
              </div>
            {/if}
          </div>

        {:else if activeMenuItem === 'downloads'}
          <div class="page-content">
            <h1 class="page-title">Stažené</h1>
            <p class="page-empty">Historie stahování bude brzy k dispozici.</p>
          </div>

        {:else if activeMenuItem === 'manual-install'}
          <ManualInstall />

        {:else if activeMenuItem === 'settings'}
          <div class="page-content">
            <h1 class="page-title">Nastavení</h1>

            <div class="settings-grid">
              <div class="settings-card about-card">
                <h3>O aplikaci</h3>

                <div class="about-head">
                  <span class="about-name">CZManager</span>
                  <span class="about-badge">{$appStore.version || '...'}</span>
                </div>
                <p class="about-desc">
                  Instalátor českých lokalizací her. Prohlížení, instalace a správa
                  lokalizací z <a href={API_WEB} onclick={(e) => { e.preventDefault(); openExternal(API_WEB) }}>Lokalizace.NET</a>.
                </p>

                <div class="about-rows">
                  <div class="about-row"><span class="about-key">Verze</span><span class="about-val">{$appStore.version || '...'}</span></div>
                  <div class="about-row"><span class="about-key">Autor</span><span class="about-val">Lokalizace.NET</span></div>
                  <div class="about-row"><span class="about-key">Licence</span><span class="about-val">GPL-3.0</span></div>
                </div>

                <!-- Stav aktualizace -->
                {#if $appStore.update?.available}
                  <div class="about-update available">
                    <Sparkles size={16} />
                    <span>K dispozici nová verze {$appStore.update.latestVersion}</span>
                    <button class="about-update-btn" onclick={() => appStore.openRelease()}>Stáhnout</button>
                  </div>
                {:else if $appStore.update && !$appStore.update.available}
                  <div class="about-update ok">
                    <CheckCircle size={16} />
                    <span>Máte nejnovější verzi</span>
                  </div>
                {/if}

                <div class="about-actions">
                  <button class="about-btn" onclick={checkForUpdate} disabled={checkingUpdate}>
                    <RefreshCw size={15} class={checkingUpdate ? 'spinning' : ''} />
                    {checkingUpdate ? 'Kontroluji...' : 'Zkontrolovat aktualizace'}
                  </button>
                  <button class="about-btn" onclick={() => openExternal(GITHUB_URL)}>
                    <Github size={15} />
                    GitHub
                  </button>
                  <button class="about-btn" onclick={() => openExternal(API_WEB)}>
                    <ExternalLink size={15} />
                    Web
                  </button>
                </div>
              </div>

              <div class="settings-card">
                <h3>Ovládání klávesnicí</h3>
                <div class="shortcut-list">
                  <div class="shortcut"><kbd>↑↓←→</kbd><span>Navigace</span></div>
                  <div class="shortcut"><kbd>Enter</kbd><span>Výběr</span></div>
                  <div class="shortcut"><kbd>Esc</kbd><span>Zpět</span></div>
                  <div class="shortcut"><kbd>/</kbd><span>Hledat</span></div>
                </div>
              </div>

              <div class="settings-card">
                <h3>Ovládání gamepadem</h3>
                <div class="shortcut-list">
                  <div class="shortcut"><kbd>D-pad</kbd><span>Navigace</span></div>
                  <div class="shortcut"><kbd>A</kbd><span>Výběr</span></div>
                  <div class="shortcut"><kbd>B</kbd><span>Zpět</span></div>
                </div>
              </div>
            </div>
          </div>

        {:else if activeMenuItem === 'help'}
          <div class="page-content">
            <h1 class="page-title">Nápověda</h1>
            <div class="help-content">
              <h3>Jak používat CZManager</h3>
              <ol>
                <li>Vyberte lokalizaci ze seznamu</li>
                <li>Nastavte cestu ke hře</li>
                <li>Klikněte na "Nainstalovat"</li>
              </ol>
              <p>Pokud máte problémy, navštivte <a href="https://lokalizace.net" target="_blank">lokalizace.net</a></p>
            </div>
          </div>
        {/if}
      </main>
    {/if}
  </div>

  <!-- Game Detail Modal -->
  <Modal
    open={showGameDetail}
    onClose={handleCloseDetail}
  >
    {#if selectedGame}
      <GameDetail game={selectedGame} onClose={handleCloseDetail} />
    {/if}
  </Modal>

  <!-- Login Modal -->
  <LoginModal
    open={showLoginModal}
    onClose={() => showLoginModal = false}
  />

  <!-- Notifikace o aktualizaci (z GitHubu) -->
  <UpdateNotice />

  <!-- Log Panel Toggle -->
  <button class="log-toggle" onclick={() => showLogPanel = !showLogPanel} title="Debug Log">
    <Terminal size={16} />
  </button>

  <!-- Log Panel -->
  {#if showLogPanel}
    <LogPanel onClose={() => showLogPanel = false} />
  {/if}
</div>

<style>
  .app-container {
    display: flex;
    height: 100%;
    background: #121212;
  }

  .main-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .top-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 32px;
    background: #0d0d0d;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .search-box {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 400px;
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.05);
    border: 2px solid transparent;
    border-radius: 12px;
    color: rgba(255, 255, 255, 0.4);
    transition: all 0.2s;
  }

  .search-box:focus-within {
    border-color: #f97316;
    background: rgba(255, 255, 255, 0.08);
    box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.2);
  }

  .search-box input {
    flex: 1;
    background: none;
    border: none;
    outline: none;
    font-size: 15px;
    color: white;
  }

  .search-box input::placeholder {
    color: rgba(255, 255, 255, 0.4);
  }

  .loading-container {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .loading-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
  }

  .loading-text {
    color: rgba(255, 255, 255, 0.5);
    margin: 0;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
    color: #f97316;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .main-content {
    flex: 1;
    overflow-y: auto;
  }

  .page-content {
    padding: 32px 48px;
  }

  .page-title {
    font-size: 28px;
    font-weight: 700;
    color: white;
    margin: 0 0 24px 0;
  }

  .page-empty {
    color: rgba(255, 255, 255, 0.4);
    font-size: 16px;
  }

  .favorites-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
  }

  .favorites-header .page-title {
    margin: 0;
  }

  .favorites-count {
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.4);
    background: rgba(255, 255, 255, 0.05);
    padding: 4px 12px;
    border-radius: 20px;
  }

  .favorites-limit-msg {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.2);
    border-radius: 10px;
    color: #fbbf24;
    font-size: 14px;
    margin-bottom: 24px;
  }

  .limit-dismiss {
    margin-left: auto;
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.4);
    font-size: 18px;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }

  .limit-dismiss:hover {
    color: white;
  }

  .favorites-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 0;
    color: rgba(255, 255, 255, 0.2);
  }

  .favorites-empty .empty-title {
    font-size: 18px;
    color: rgba(255, 255, 255, 0.5);
    margin: 20px 0 8px;
  }

  .favorites-empty .empty-hint {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.3);
    margin: 0;
  }

  .btn-login {
    margin-top: 20px;
    height: 44px;
    padding: 0 32px;
    background: #f97316;
    border: none;
    border-radius: 10px;
    font-size: 15px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-login:hover {
    background: #ea580c;
  }

  .favorites-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 20px;
  }

  .settings-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
  }

  .settings-card {
    background: #1a1a1a;
    border-radius: 16px;
    padding: 24px;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .settings-card h3 {
    font-size: 16px;
    font-weight: 600;
    color: white;
    margin: 0 0 16px 0;
  }

  .settings-card p {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0 0 8px 0;
  }

  /* O aplikaci */
  .about-head {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }

  .about-name {
    font-size: 18px;
    font-weight: 700;
    color: #f97316;
  }

  .about-badge {
    font-size: 12px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.7);
    background: rgba(255, 255, 255, 0.08);
    padding: 3px 10px;
    border-radius: 20px;
  }

  .about-desc {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
    line-height: 1.6;
    margin: 0 0 16px 0;
  }

  .about-desc a {
    color: #f97316;
    text-decoration: none;
  }

  .about-desc a:hover {
    text-decoration: underline;
  }

  .about-rows {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }

  .about-row {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
  }

  .about-key {
    color: rgba(255, 255, 255, 0.4);
  }

  .about-val {
    color: rgba(255, 255, 255, 0.8);
    font-weight: 500;
  }

  .about-update {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-radius: 10px;
    font-size: 13px;
    margin-bottom: 16px;
  }

  .about-update.available {
    background: rgba(249, 115, 22, 0.1);
    border: 1px solid rgba(249, 115, 22, 0.25);
    color: #fbbf24;
  }

  .about-update.ok {
    background: rgba(34, 197, 94, 0.08);
    border: 1px solid rgba(34, 197, 94, 0.2);
    color: #4ade80;
  }

  .about-update-btn {
    margin-left: auto;
    height: 28px;
    padding: 0 12px;
    background: #f97316;
    border: none;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
  }

  .about-update-btn:hover,
  .about-update-btn:focus {
    background: #ea580c;
    outline: none;
    box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.4);
  }

  .about-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .about-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 34px;
    padding: 0 12px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    font-size: 13px;
    color: rgba(255, 255, 255, 0.75);
    cursor: pointer;
    transition: all 0.2s;
  }

  .about-btn:hover:not(:disabled),
  .about-btn:focus {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    outline: none;
    box-shadow: 0 0 0 2px #f97316;
  }

  .about-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .shortcut-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .shortcut {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 14px;
  }

  .shortcut kbd {
    background: rgba(255, 255, 255, 0.1);
    padding: 4px 10px;
    border-radius: 6px;
    font-family: inherit;
    font-size: 13px;
    color: rgba(255, 255, 255, 0.8);
  }

  .shortcut span {
    color: rgba(255, 255, 255, 0.5);
  }

  .help-content {
    background: #1a1a1a;
    border-radius: 16px;
    padding: 24px;
    border: 1px solid rgba(255, 255, 255, 0.05);
  }

  .help-content h3 {
    font-size: 18px;
    font-weight: 600;
    color: white;
    margin: 0 0 16px 0;
  }

  .help-content ol {
    margin: 0 0 16px 0;
    padding-left: 20px;
    color: rgba(255, 255, 255, 0.6);
    line-height: 1.8;
  }

  .help-content p {
    color: rgba(255, 255, 255, 0.5);
    margin: 0;
  }

  .help-content a {
    color: #f97316;
    text-decoration: none;
  }

  .help-content a:hover {
    text-decoration: underline;
  }

  .log-toggle {
    position: fixed;
    bottom: 16px;
    right: 16px;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 50%;
    color: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    transition: all 0.2s;
    z-index: 999;
  }

  .log-toggle:hover {
    background: rgba(255, 255, 255, 0.15);
    color: white;
  }
</style>
