<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import GameCard from './GameCard.svelte'
  import { gamesStore, filteredLocalizations, type Localization } from '../stores/games.svelte'
  import { navigationStore } from '../stores/navigation.svelte'
  import { Loader2 } from 'lucide-svelte'

  export let onGameSelect: ((game: Localization) => void) | undefined = undefined

  let gridElement: HTMLElement
  let cardElements: HTMLElement[] = []
  let scrollContainer: HTMLElement | null = null

  function updateGridColumns() {
    if (!gridElement) return
    const width = gridElement.clientWidth
    const cardMinWidth = 180
    const gap = 16
    const columns = Math.floor((width + gap) / (cardMinWidth + gap))
    navigationStore.setGridColumns(Math.max(2, Math.min(8, columns)))
  }

  function handleScroll() {
    if (!scrollContainer || $gamesStore.loading || !$gamesStore.hasMore) return

    const { scrollTop, scrollHeight, clientHeight } = scrollContainer
    if (scrollHeight - scrollTop - clientHeight < 400) {
      gamesStore.fetchLocalizations()
    }
  }

  onMount(() => {
    updateGridColumns()
    window.addEventListener('resize', updateGridColumns)

    // Find scroll container (main element)
    scrollContainer = document.querySelector('main')
    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', handleScroll)
    }
  })

  onDestroy(() => {
    window.removeEventListener('resize', updateGridColumns)
    if (scrollContainer) {
      scrollContainer.removeEventListener('scroll', handleScroll)
    }
  })

  $: if (cardElements.length > 0) {
    navigationStore.registerFocusables(cardElements)
  }

  function handleCardClick(game: Localization, index: number) {
    navigationStore.setFocusedIndex(index)
    onGameSelect?.(game)
  }

  $: gameCount = $filteredLocalizations.length
  $: totalCount = $gamesStore.total
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
      bind:element={cardElements[index]}
      {game}
      focused={$navigationStore.focusedIndex === index}
      onclick={() => handleCardClick(game, index)}
    />
  {/each}
</div>

{#if $gamesStore.loading}
  <div class="flex justify-center py-8">
    <Loader2 size={32} class="animate-spin text-primary" />
  </div>
{/if}

{#if $gamesStore.hasMore && !$gamesStore.loading && $filteredLocalizations.length > 0}
  <div class="flex justify-center py-6">
    <button
      class="focusable px-6 py-3 bg-surface hover:bg-surface-hover border border-border rounded-lg font-medium
             transition-colors text-sm"
      on:click={() => gamesStore.fetchLocalizations()}
    >
      Načíst další lokalizace
    </button>
  </div>
{/if}

{#if $filteredLocalizations.length === 0 && !$gamesStore.loading}
  <div class="flex flex-col items-center justify-center py-16 text-text-muted">
    <p class="text-lg">Žádné lokalizace nenalezeny</p>
    {#if $gamesStore.searchQuery}
      <p class="text-sm mt-2">Zkuste upravit vyhledávání</p>
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
</style>
