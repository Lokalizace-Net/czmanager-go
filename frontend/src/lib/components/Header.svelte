<script lang="ts">
  import { Home, Heart, HelpCircle, Search, Gamepad2 } from 'lucide-svelte'
  import { navigationStore } from '../stores/navigation.svelte'
  import { gamepadHandler } from '../utils/gamepad.svelte'

  export let onSearch: ((query: string) => void) | undefined = undefined

  let searchQuery = ''
  let searchFocused = false

  $: gamepadConnected = $gamepadHandler.connected

  function handleSearchInput(event: Event) {
    const value = (event.target as HTMLInputElement).value
    searchQuery = value
    onSearch?.(value)
  }

  function handleNavClick(view: 'home' | 'settings') {
    navigationStore.setView(view)
  }
</script>

<header class="header">
  <!-- Left side - Logo and navigation -->
  <div class="header-left">
    <div class="logo">
      <span class="logo-name">CZManager</span>
      <span class="logo-version">v3</span>
    </div>

    <nav class="nav">
      <button
        class="nav-item"
        class:active={$navigationStore.currentView === 'home'}
        on:click={() => handleNavClick('home')}
      >
        <Home size={18} />
        <span>Domů</span>
      </button>

      <a
        href="https://lokalizace.net/support"
        target="_blank"
        rel="noopener noreferrer"
        class="nav-item support"
      >
        <Heart size={18} />
        <span>Podpora Týmu</span>
      </a>

      <button
        class="nav-item"
        class:active={$navigationStore.currentView === 'settings'}
        on:click={() => handleNavClick('settings')}
      >
        <HelpCircle size={18} />
        <span>Pomoc</span>
      </button>
    </nav>
  </div>

  <!-- Right side - Search and status -->
  <div class="header-right">
    <!-- Search -->
    <div class="search-wrapper">
      <Search size={18} class="search-icon" />
      <input
        type="text"
        placeholder="Hledat hru..."
        data-search-input
        bind:value={searchQuery}
        on:input={handleSearchInput}
        on:focus={() => searchFocused = true}
        on:blur={() => searchFocused = false}
        class="search-input"
        class:focused={searchFocused}
      />
    </div>

    <!-- Gamepad indicator -->
    {#if gamepadConnected}
      <div class="gamepad-indicator" title="Gamepad připojen">
        <Gamepad2 size={18} />
      </div>
    {/if}
  </div>
</header>

<style>
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 48px;
    background: #1e1e1e;
    border-bottom: 1px solid #333;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 32px;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .logo-name {
    font-size: 20px;
    font-weight: 700;
    color: #f97316;
  }

  .logo-version {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
  }

  .nav {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 14px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.7);
    background: transparent;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
    text-decoration: none;
  }

  .nav-item:hover {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .nav-item.active {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .nav-item.support {
    color: #f97316;
  }

  .nav-item.support:hover {
    background: rgba(249, 115, 22, 0.1);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .search-wrapper {
    position: relative;
  }

  .search-wrapper :global(.search-icon) {
    position: absolute;
    left: 14px;
    top: 50%;
    transform: translateY(-50%);
    color: rgba(255, 255, 255, 0.4);
    pointer-events: none;
  }

  .search-input {
    width: 280px;
    height: 42px;
    padding: 0 16px 0 44px;
    background: #121212;
    border: 1px solid #333;
    border-radius: 10px;
    font-size: 14px;
    color: white;
    outline: none;
    transition: all 0.2s;
  }

  .search-input::placeholder {
    color: rgba(255, 255, 255, 0.3);
  }

  .search-input:hover {
    border-color: #444;
  }

  .search-input.focused,
  .search-input:focus {
    border-color: #f97316;
    background: #1a1a1a;
  }

  .gamepad-indicator {
    display: flex;
    align-items: center;
    color: #22c55e;
  }

</style>
