<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Header from './lib/components/Header.svelte'
  import AgentBanner from './lib/components/AgentBanner.svelte'
  import WelcomeBanner from './lib/components/WelcomeBanner.svelte'
  import GameGrid from './lib/components/GameGrid.svelte'
  import GameDetail from './lib/components/GameDetail.svelte'
  import Modal from './lib/components/Modal.svelte'
  import { agentStore } from './lib/stores/agent.svelte'
  import { gamesStore, type Localization } from './lib/stores/games.svelte'
  import { navigationStore } from './lib/stores/navigation.svelte'
  import { gamepadHandler } from './lib/utils/gamepad.svelte'
  import { StartAgent } from '../wailsjs/go/main/App'
  import { Loader2 } from 'lucide-svelte'

  let selectedGame: Localization | null = null
  let showGameDetail = false
  let initializing = true

  onMount(async () => {
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

    // Start gamepad handler
    gamepadHandler.start()

    // Setup keyboard handler
    window.addEventListener('keydown', navigationStore.handleKeydown)

    initializing = false
  })

  onDestroy(() => {
    agentStore.stopHealthCheck()
    gamepadHandler.stop()
    window.removeEventListener('keydown', navigationStore.handleKeydown)
  })

  function handleGameSelect(game: Localization) {
    selectedGame = game
    showGameDetail = true
    navigationStore.setView('game', game.slug)
  }

  function handleCloseDetail() {
    showGameDetail = false
    selectedGame = null
    navigationStore.setView('home')
  }

  function handleSearch(query: string) {
    gamesStore.setSearchQuery(query)
  }
</script>

<div class="app-container">
  <Header onSearch={handleSearch} />
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
      {#if $navigationStore.currentView === 'home'}
        <WelcomeBanner />
        <GameGrid onGameSelect={handleGameSelect} />
      {:else if $navigationStore.currentView === 'settings'}
        <div class="settings-page">
          <h2 class="settings-title">Nastavení</h2>
          <div class="settings-sections">
            <div class="settings-card">
              <h3 class="card-title">O aplikaci</h3>
              <p class="card-text">
                CZManager v3 - Manažer českých lokalizací
              </p>
              <p class="card-text">
                Agent: {$agentStore.version || 'nepřipojen'}
              </p>
            </div>

            <div class="settings-card">
              <h3 class="card-title">Klávesové zkratky</h3>
              <div class="shortcuts-grid">
                <span>Šipky</span><span>Navigace</span>
                <span>Enter / Mezerník</span><span>Výběr</span>
                <span>Escape / Backspace</span><span>Zpět</span>
                <span>/</span><span>Vyhledávání</span>
              </div>
            </div>

            <div class="settings-card">
              <h3 class="card-title">Gamepad</h3>
              <p class="gamepad-status" class:connected={$gamepadHandler.connected}>
                {$gamepadHandler.connected ? 'Gamepad připojen' : 'Žádný gamepad nepřipojen'}
              </p>
              <div class="shortcuts-grid">
                <span>D-pad / Levá páčka</span><span>Navigace</span>
                <span>A / X</span><span>Výběr</span>
                <span>B / O</span><span>Zpět</span>
                <span>LT / RT</span><span>Scrollování</span>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </main>
  {/if}

  <!-- Game Detail Modal -->
  <Modal
    open={showGameDetail}
    title={selectedGame?.name}
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
    flex-direction: column;
    height: 100%;
    background: #121212;
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

  .settings-page {
    padding: 32px 48px;
  }

  .settings-title {
    font-size: 24px;
    font-weight: 700;
    color: white;
    margin: 0 0 24px 0;
  }

  .settings-sections {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .settings-card {
    background: #1e1e1e;
    border-radius: 12px;
    padding: 16px;
    border: 1px solid #333;
  }

  .card-title {
    font-size: 16px;
    font-weight: 600;
    color: white;
    margin: 0 0 8px 0;
  }

  .card-text {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0 0 4px 0;
  }

  .shortcuts-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
  }

  .gamepad-status {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0 0 8px 0;
  }

  .gamepad-status.connected {
    color: #22c55e;
  }
</style>
