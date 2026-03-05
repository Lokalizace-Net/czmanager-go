<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { get } from 'svelte/store'
  import GameCard from './GameCard.svelte'
  import { gamesStore, filteredLocalizations, type Localization } from '../stores/games.svelte'
  import { focusStore } from '../stores/focus.svelte'
  import { Loader2 } from 'lucide-svelte'

  let { onGameSelect }: { onGameSelect?: (game: Localization) => void } = $props()

  let gridElement: HTMLElement
  let scrollContainer: HTMLElement | null = null
  let gridColumns = $state(4)

  function updateGridColumns() {
    if (!gridElement) return
    const width = gridElement.clientWidth
    const cardMinWidth = 220
    const gap = 20
    gridColumns = Math.floor((width + gap) / (cardMinWidth + gap))
    gridColumns = Math.max(2, Math.min(8, gridColumns))

    // Aktualizuj zónu s novým počtem sloupců a elementy
    updateGridElements()
  }

  function updateGridElements() {
    if (!gridElement) return
    const buttons = Array.from(gridElement.querySelectorAll('.game-card')) as HTMLButtonElement[]
    const state = get(focusStore)
    const zone = state.zones.get('main')

    focusStore.registerZone({
      id: 'main',
      elements: buttons,
      columns: gridColumns,
      loop: false,
      onEscape: zone?.onEscape
    })
  }

  function handleScroll() {
    const gs = get(gamesStore)
    if (!scrollContainer || gs.loading || !gs.hasMore) return

    const { scrollTop, scrollHeight, clientHeight } = scrollContainer
    if (scrollHeight - scrollTop - clientHeight < 400) {
      gamesStore.fetchLocalizations()
    }
  }

  onMount(() => {
    // Registruj hlavní zónu pro grid
    focusStore.registerZone({
      id: 'main',
      elements: [],
      columns: gridColumns,
      loop: false,
      onEscape: () => {
        focusStore.setActiveZone('sidemenu', false)
      }
    })

    updateGridColumns()
    window.addEventListener('resize', updateGridColumns)

    // Find scroll container (main element)
    scrollContainer = document.querySelector('main')
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll)
    }

    // Aktualizuj elementy po renderování
    setTimeout(updateGridElements, 100)
  })

  onDestroy(() => {
    window.removeEventListener('resize', updateGridColumns)
    if (scrollContainer) {
      scrollContainer.removeEventListener('scroll', handleScroll)
    }
    focusStore.unregisterZone('main')
  })

  // Aktualizuj elementy když se změní seznam her
  $effect(() => {
    const games = $filteredLocalizations
    if (games.length > 0) {
      // Počkej na renderování
      setTimeout(updateGridElements, 50)
    }
  })

  function handleCardClick(game: Localization, index: number) {
    focusStore.setFocusedIndex(index)
    onGameSelect?.(game)
  }

  function handleCardFocus(index: number) {
    focusStore.setActiveZone('main', false)
    focusStore.setFocusedIndex(index)
  }

  let gameCount = $derived($filteredLocalizations.length)
  let totalCount = $derived($gamesStore.total)
  let focusedIndex = $derived($focusStore.focusedIndex)
  let isMainActive = $derived($focusStore.activeZone === 'main')
</script>

<!-- Header with count -->
<div class="grid-header">
  <h2>
    Lokalizace
    <span>({gameCount}{totalCount > gameCount ? ` z ${totalCount}` : ''})</span>
  </h2>
</div>

<div
  bind:this={gridElement}
  class="game-grid"
>
  {#each $filteredLocalizations as game, index (game.id)}
    <GameCard
      {game}
      focused={isMainActive && focusedIndex === index}
      onclick={() => handleCardClick(game, index)}
      onfocus={() => handleCardFocus(index)}
    />
  {/each}
</div>

{#if $gamesStore.loading}
  <div class="loading-indicator">
    <Loader2 size={32} class="spinning" />
  </div>
{/if}

{#if $gamesStore.hasMore && !$gamesStore.loading && $filteredLocalizations.length > 0}
  <div class="load-more-container">
    <button
      class="load-more-btn"
      onclick={() => gamesStore.fetchLocalizations()}
    >
      Načíst další lokalizace
    </button>
  </div>
{/if}

{#if $filteredLocalizations.length === 0 && !$gamesStore.loading}
  <div class="empty-state">
    <p class="empty-title">Žádné lokalizace nenalezeny</p>
    {#if $gamesStore.searchQuery}
      <p class="empty-hint">Zkuste upravit vyhledávání</p>
    {/if}
  </div>
{/if}

<style>
  .grid-header {
    padding: 24px 48px 16px 48px;
  }

  .grid-header h2 {
    font-size: 18px;
    font-weight: 600;
    color: white;
  }

  .grid-header span {
    font-weight: 400;
    color: rgba(255, 255, 255, 0.5);
    margin-left: 8px;
  }

  .game-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 20px;
    padding: 0 48px 48px 48px;
  }

  .loading-indicator {
    display: flex;
    justify-content: center;
    padding: 32px 0;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
    color: #f97316;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .load-more-container {
    display: flex;
    justify-content: center;
    padding: 24px 0;
  }

  .load-more-btn {
    padding: 12px 24px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s;
  }

  .load-more-btn:hover,
  .load-more-btn:focus {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    outline: none;
    box-shadow: 0 0 0 2px #f97316;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 64px 0;
  }

  .empty-title {
    font-size: 18px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0;
  }

  .empty-hint {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.3);
    margin: 8px 0 0 0;
  }
</style>
