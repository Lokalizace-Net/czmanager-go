<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import SideMenu from './lib/components/SideMenu.svelte'
  import AgentBanner from './lib/components/AgentBanner.svelte'
  import WelcomeBanner from './lib/components/WelcomeBanner.svelte'
  import GameGrid from './lib/components/GameGrid.svelte'
  import GameDetail from './lib/components/GameDetail.svelte'
  import Modal from './lib/components/Modal.svelte'
  import { agentStore } from './lib/stores/agent.svelte'
  import { gamesStore, type Localization } from './lib/stores/games.svelte'
  import { focusStore } from './lib/stores/focus.svelte'
  import { startGamepadPolling, stopGamepadPolling } from './lib/utils/gamepad'
  import { StartAgent } from '../wailsjs/go/main/App'
  import { Loader2, Search } from 'lucide-svelte'

  let selectedGame = $state<Localization | null>(null)
  let showGameDetail = $state(false)
  let initializing = $state(true)
  let activeMenuItem = $state('home')
  let searchQuery = $state('')
  let searchInput = $state<HTMLInputElement | undefined>(undefined)

  onMount(async () => {
    // Setup keyboard handler
    window.addEventListener('keydown', focusStore.handleKeydown)

    // Start gamepad polling
    startGamepadPolling()

    // Start agent via Wails
    try {
      await StartAgent()
    } catch (err) {
      console.log('Could not start agent via Wails, may already be running')
    }

    // Connect to agent
    let connected = false
    for (let i = 0; i < 10; i++) {
      connected = await agentStore.connect()
      if (connected) break
      await new Promise(r => setTimeout(r, 1000))
    }

    // Load games
    await gamesStore.fetchLocalizations(true)

    initializing = false
  })

  onDestroy(() => {
    agentStore.stopHealthCheck()
    window.removeEventListener('keydown', focusStore.handleKeydown)
    stopGamepadPolling()
  })

  function handleGameSelect(game: Localization) {
    selectedGame = game
    showGameDetail = true
    focusStore.setActiveZone('modal')
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

  // Zkratka / pro vyhledávání, Tab pro přepínání zón
  function handleGlobalKeydown(e: KeyboardEvent) {
    if (e.key === '/' && !(e.target instanceof HTMLInputElement)) {
      e.preventDefault()
      searchInput?.focus()
    }
    // Tab přepíná mezi zónami: main -> sidemenu -> search -> main
    if (e.key === 'Tab' && !(e.target instanceof HTMLInputElement)) {
      e.preventDefault()
      const currentZone = $focusStore.activeZone
      if (currentZone === 'main') {
        focusStore.setActiveZone('sidemenu')
      } else if (currentZone === 'sidemenu') {
        searchInput?.focus()
      }
    }
    // Šipka nahoru ze search inputu přepne na sidemenu
    if (e.key === 'ArrowUp' && e.target === searchInput) {
      e.preventDefault()
      searchInput?.blur()
      focusStore.setActiveZone('sidemenu')
    }
    // Šipka dolů ze search inputu přepne na main grid
    if (e.key === 'ArrowDown' && e.target === searchInput) {
      e.preventDefault()
      searchInput?.blur()
      focusStore.setActiveZone('main')
    }
    // Šipka nahoru z horního řádku main gridu -> search
    if (e.key === 'ArrowUp' && !(e.target instanceof HTMLInputElement)) {
      const currentZone = $focusStore.activeZone
      const state = $focusStore
      const zone = state.zones.get('main')
      if (currentZone === 'main' && zone) {
        const cols = zone.columns || 4
        const currentIndex = state.focusedIndex
        // Pokud jsme na horním řádku
        if (currentIndex < cols) {
          e.preventDefault()
          e.stopPropagation()
          searchInput?.focus()
        }
      }
    }
  }
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<div class="app-container">
  <SideMenu
    activeItem={activeMenuItem}
    onNavigate={handleMenuNavigate}
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

      <div class="top-bar-right">
        <div class="agent-indicator" class:connected={$agentStore.status === 'connected'}>
          <span class="indicator-dot"></span>
          <span class="indicator-text">
            {$agentStore.status === 'connected' ? 'Agent připojen' : 'Agent odpojen'}
          </span>
        </div>
      </div>
    </header>

    <AgentBanner />

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
            <h1 class="page-title">Oblíbené</h1>
            <p class="page-empty">Zatím nemáte žádné oblíbené lokalizace.</p>
          </div>

        {:else if activeMenuItem === 'downloads'}
          <div class="page-content">
            <h1 class="page-title">Stažené</h1>
            <p class="page-empty">Historie stahování bude brzy k dispozici.</p>
          </div>

        {:else if activeMenuItem === 'settings'}
          <div class="page-content">
            <h1 class="page-title">Nastavení</h1>

            <div class="settings-grid">
              <div class="settings-card">
                <h3>O aplikaci</h3>
                <p>CZManager v3</p>
                <p>Agent: {$agentStore.version || 'nepřipojen'}</p>
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

  .top-bar-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .agent-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border-radius: 20px;
    background: rgba(239, 68, 68, 0.1);
  }

  .agent-indicator.connected {
    background: rgba(34, 197, 94, 0.1);
  }

  .indicator-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ef4444;
  }

  .agent-indicator.connected .indicator-dot {
    background: #22c55e;
  }

  .indicator-text {
    font-size: 13px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.7);
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
</style>
