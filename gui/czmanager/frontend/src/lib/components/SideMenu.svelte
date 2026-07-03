<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Home, Settings, Heart, HelpCircle, Download, LogIn, LogOut, User, Crown } from 'lucide-svelte'
  import { focusStore } from '../stores/focus.svelte'
  import { authStore } from '../stores/auth.svelte'

  let {
    activeItem = 'home',
    onNavigate = () => {},
    collapsed = false,
    onLoginClick
  }: {
    activeItem?: string
    onNavigate?: (item: string) => void
    collapsed?: boolean
    onLoginClick?: () => void
  } = $props()

  let user = $derived($authStore.user)
  let subscription = $derived($authStore.subscription)
  let features = $derived($authStore.features)

  // Debug
  $effect(() => {
    console.log('SideMenu - user:', user)
    console.log('SideMenu - subscription:', subscription)
    console.log('SideMenu - features:', features)
  })

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
  let loginBtn = $state<HTMLButtonElement | undefined>(undefined)
  let logoutBtn = $state<HTMLButtonElement | undefined>(undefined)

  // Aktualizuj focus zónu když se změní stav přihlášení
  function updateFocusElements() {
    const validMenuButtons = menuButtons.filter(Boolean)
    // Přidej login nebo logout button na konec
    const authBtn = user ? logoutBtn : loginBtn
    const allButtons = authBtn ? [...validMenuButtons, authBtn] : validMenuButtons

    if (allButtons.length > 0) {
      focusStore.updateZoneElements('sidemenu', allButtons)
    }
  }

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
    setTimeout(updateFocusElements, 200)
  })

  // Reaktivně aktualizuj když se změní user
  $effect(() => {
    // Sleduj změny user
    const _ = user
    setTimeout(updateFocusElements, 50)
  })

  onDestroy(() => {
    focusStore.unregisterZone('sidemenu')
  })

  function handleClick(itemId: string) {
    onNavigate(itemId)
    // Po kliknutí přejdi na hlavní obsah
    focusStore.setActiveZone('main', false)
  }
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

  <!-- User section -->
  <div class="menu-footer">
    {#if user}
      <!-- Logged in user -->
      <div class="user-info">
        <div class="user-avatar">
          {#if user.avatar}
            <img src={user.avatar} alt={user.username} />
          {:else}
            <User size={18} />
          {/if}
        </div>
        {#if !collapsed}
          <div class="user-details">
            <span class="user-name">{user.username}</span>
            {#if subscription?.tier}
              <span class="user-tier" class:vip={subscription.tier.slug === 'vip'} class:supporter={subscription.tier.slug === 'supporter'}>
                <Crown size={12} />
                {subscription.tier.slug === 'vip' ? 'VIP' : 'Supporter'}
              </span>
            {/if}
          </div>
        {/if}
      </div>
      <button bind:this={logoutBtn} class="menu-item logout-btn" onclick={() => authStore.logout()}>
        <LogOut size={20} />
        {#if !collapsed}
          <span class="menu-label">Odhlásit se</span>
        {/if}
      </button>
    {:else}
      <!-- Not logged in -->
      <button bind:this={loginBtn} class="menu-item login-btn" onclick={onLoginClick}>
        <LogIn size={20} />
        {#if !collapsed}
          <span class="menu-label">Přihlásit se</span>
        {/if}
      </button>
    {/if}

    <!-- DEBUG - smazat později -->
    {#if !collapsed && user}
      <div style="font-size: 10px; color: #666; margin-top: 8px; padding: 8px; background: rgba(0,0,0,0.3); border-radius: 4px;">
        <div>Sub: {subscription?.tier?.slug ?? 'null'}</div>
        <div>Scanner: {features?.hasGameScanner ? 'ano' : 'ne'}</div>
      </div>
    {/if}
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

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    margin-bottom: 8px;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.03);
  }

  .collapsed .user-info {
    justify-content: center;
    padding: 12px;
  }

  .user-avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    color: rgba(255, 255, 255, 0.5);
    overflow: hidden;
    flex-shrink: 0;
  }

  .user-avatar img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .user-details {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .user-name {
    font-size: 14px;
    font-weight: 600;
    color: white;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-tier {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    width: fit-content;
  }

  .user-tier.vip {
    background: rgba(251, 191, 36, 0.2);
    color: #fbbf24;
  }

  .user-tier.supporter {
    background: rgba(244, 114, 182, 0.2);
    color: #f472b6;
  }

  .login-btn {
    color: #f97316 !important;
  }

  .logout-btn {
    color: rgba(255, 255, 255, 0.5) !important;
  }

  .logout-btn:hover {
    color: #ef4444 !important;
    background: rgba(239, 68, 68, 0.1) !important;
  }
</style>
