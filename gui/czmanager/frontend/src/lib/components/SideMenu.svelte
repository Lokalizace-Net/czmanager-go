<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Home, Settings, Heart, HelpCircle, Download } from 'lucide-svelte'
  import { focusStore } from '../stores/focus.svelte'
  import { agentStore } from '../stores/agent.svelte'

  let {
    activeItem = 'home',
    onNavigate = () => {},
    collapsed = false
  }: {
    activeItem?: string
    onNavigate?: (item: string) => void
    collapsed?: boolean
  } = $props()

  interface MenuItem {
    id: string
    label: string
    icon: any
  }

  const menuItems: MenuItem[] = [
    { id: 'home', label: 'Domů', icon: Home },
    { id: 'favorites', label: 'Oblíbené', icon: Heart },
    { id: 'downloads', label: 'Stažené', icon: Download },
    { id: 'settings', label: 'Nastavení', icon: Settings },
    { id: 'help', label: 'Nápověda', icon: HelpCircle },
  ]

  let menuButtons: HTMLButtonElement[] = []

  onMount(() => {
    // Registruj menu jako focus zónu
    focusStore.registerZone({
      id: 'sidemenu',
      elements: [],
      columns: 1,
      loop: true,
      onEscape: () => {
        focusStore.setActiveZone('main', false)
      }
    })

    // Počkej na renderování a pak zaregistruj elementy
    setTimeout(() => {
      const validButtons = menuButtons.filter(Boolean)
      if (validButtons.length > 0) {
        focusStore.updateZoneElements('sidemenu', validButtons)
      }
    }, 200)
  })

  onDestroy(() => {
    focusStore.unregisterZone('sidemenu')
  })

  function handleClick(itemId: string) {
    onNavigate(itemId)
    // Po kliknutí přejdi na hlavní obsah
    focusStore.setActiveZone('main', false)
  }

  let agentStatus = $derived($agentStore.status)
</script>

<aside class="side-menu" class:collapsed>
  <!-- Logo -->
  <div class="menu-header">
    {#if !collapsed}
      <span class="logo-text">CZManager</span>
    {:else}
      <span class="logo-text-small">CZ</span>
    {/if}
  </div>

  <!-- Menu items -->
  <nav class="menu-nav">
    {#each menuItems as item, index (item.id)}
      <button
        bind:this={menuButtons[index]}
        class="menu-item"
        class:active={activeItem === item.id}
        onclick={() => handleClick(item.id)}
      >
        <item.icon size={22} />
        {#if !collapsed}
          <span class="menu-label">{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Agent status -->
  <div class="menu-footer">
    <div class="agent-status" class:connected={agentStatus === 'connected'}>
      <div class="status-dot"></div>
      {#if !collapsed}
        <span class="status-text">
          {agentStatus === 'connected' ? 'Agent připojen' : 'Agent odpojen'}
        </span>
      {/if}
    </div>
  </div>
</aside>

<style>
  .side-menu {
    display: flex;
    flex-direction: column;
    width: 220px;
    height: 100%;
    background: #0d0d0d;
    border-right: 1px solid rgba(255, 255, 255, 0.05);
    transition: width 0.2s ease;
  }

  .side-menu.collapsed {
    width: 70px;
  }

  .menu-header {
    padding: 24px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .logo-text {
    font-size: 20px;
    font-weight: 700;
    color: #f97316;
  }

  .logo-text-small {
    font-size: 20px;
    font-weight: 700;
    color: #f97316;
    display: block;
    text-align: center;
  }

  .menu-nav {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 16px 12px;
    gap: 4px;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px;
    border: 2px solid transparent;
    border-radius: 12px;
    background: transparent;
    color: rgba(255, 255, 255, 0.6);
    font-size: 15px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
    text-align: left;
  }

  .collapsed .menu-item {
    justify-content: center;
    padding: 14px;
  }

  .menu-item:hover {
    background: rgba(255, 255, 255, 0.05);
    color: white;
  }

  .menu-item:focus {
    outline: none;
    border-color: #f97316;
    background: rgba(249, 115, 22, 0.1);
    color: white;
  }

  .menu-item.active {
    background: rgba(249, 115, 22, 0.15);
    color: #f97316;
  }

  .menu-item.active:focus {
    background: rgba(249, 115, 22, 0.2);
  }

  .menu-label {
    white-space: nowrap;
  }

  .menu-footer {
    padding: 16px 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }

  .agent-status {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    border-radius: 10px;
    background: rgba(239, 68, 68, 0.1);
  }

  .agent-status.connected {
    background: rgba(34, 197, 94, 0.1);
  }

  .collapsed .agent-status {
    justify-content: center;
    padding: 12px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ef4444;
    flex-shrink: 0;
  }

  .agent-status.connected .status-dot {
    background: #22c55e;
  }

  .status-text {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.6);
    white-space: nowrap;
  }
</style>
