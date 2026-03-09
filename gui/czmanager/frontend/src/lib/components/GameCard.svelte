<script lang="ts">
  import type { Localization } from '../stores/games.svelte'
  import { GetImageBase64 } from '../../../wailsjs/go/main/App'
  import { onMount } from 'svelte'
  import { Heart } from 'lucide-svelte'

  let {
    game,
    focused = false,
    isFavorite = false,
    showFavoriteBtn = false,
    onclick,
    onfocus,
    onToggleFavorite
  }: {
    game: Localization
    focused?: boolean
    isFavorite?: boolean
    showFavoriteBtn?: boolean
    onclick?: () => void
    onfocus?: () => void
    onToggleFavorite?: () => void
  } = $props()

  let imageError = $state(false)
  let imageLoaded = $state(false)
  let imageSrc = $state('')

  // Load image via Go backend (avoids CORS issues)
  onMount(async () => {
    if (game.imageUrl) {
      try {
        const base64 = await GetImageBase64(game.imageUrl)
        imageSrc = base64
        imageLoaded = true
      } catch (e) {
        console.error('Failed to load image:', game.imageUrl, e)
        imageError = true
      }
    }
  })

  function getStatusLabel(status: string): string {
    switch (status) {
      case 'released': return 'Veřejná verze'
      case 'beta': return 'Beta'
      case 'translating': return 'Překládá se'
      case 'wip': return 'Rozpracováno'
      default: return 'Připravuje se'
    }
  }

  function getStatusClass(status: string): string {
    switch (status) {
      case 'released': return 'status-released'
      case 'beta': return 'status-beta'
      case 'translating': return 'status-translating'
      case 'wip': return 'status-wip'
      default: return 'status-draft'
    }
  }

  let progress = $derived(game.translatePercent || 0)
</script>

<button
  class="game-card"
  class:focused
  {onclick}
  {onfocus}
>
  <!-- Cover Image -->
  <div class="card-image">
    {#if !imageError && imageSrc}
      <img
        src={imageSrc}
        alt={game.name}
        class:loaded={imageLoaded}
      />
    {:else if !imageError && !imageLoaded}
      <!-- Loading state -->
      <div class="placeholder loading">
        <div class="spinner"></div>
      </div>
    {:else}
      <div class="placeholder">
        <span>{game.name.charAt(0)}</span>
      </div>
    {/if}

    <!-- Favorite button - top left -->
    {#if showFavoriteBtn}
      <!-- svelte-ignore node_invalid_placement_ssr -->
      <div
        class="favorite-btn"
        class:active={isFavorite}
        role="button"
        tabindex="-1"
        onclick={(e) => { e.stopPropagation(); onToggleFavorite?.() }}
        onkeydown={(e) => { if (e.key === 'Enter') { e.stopPropagation(); onToggleFavorite?.() } }}
        title={isFavorite ? 'Odebrat z oblíbených' : 'Přidat do oblíbených'}
      >
        <Heart size={18} fill={isFavorite ? 'currentColor' : 'none'} />
      </div>
    {/if}

    <!-- Status Badge - top right -->
    <div class="status-badge {getStatusClass(game.status)}">
      {getStatusLabel(game.status)}
    </div>
  </div>

  <!-- Info section -->
  <div class="card-info">
    <h3 class="card-title">{game.name}</h3>

    {#if game.teamName}
      <p class="card-team">{game.teamName}</p>
    {/if}

    <!-- Progress section -->
    <div class="progress-section">
      <div class="progress-row">
        <span class="progress-label">Přeloženo</span>
        <span class="progress-value" class:complete={progress >= 100}>{progress}%</span>
      </div>
      <div class="progress-track">
        <div class="progress-fill" class:complete={progress >= 100} style="width: {progress}%"></div>
      </div>
    </div>
  </div>
</button>

<style>
  .game-card {
    display: flex;
    flex-direction: column;
    border-radius: 12px;
    overflow: hidden;
    background: #1a1a1a;
    border: 2px solid transparent;
    transition: all 0.2s ease;
    cursor: pointer;
    text-align: left;
  }

  .game-card:hover,
  .game-card.focused,
  .game-card:focus {
    border-color: #f97316;
    transform: translateY(-4px);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.4);
    outline: none;
  }

  .card-image {
    position: relative;
    aspect-ratio: 4 / 5;
    background: linear-gradient(135deg, #2a2a2a 0%, #1a1a1a 100%);
    overflow: hidden;
  }

  .card-image img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0;
    transition: opacity 0.3s ease, transform 0.3s ease;
  }

  .card-image img.loaded {
    opacity: 1;
  }

  .game-card:hover .card-image img {
    transform: scale(1.05);
  }

  .placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
  }

  .placeholder span {
    font-size: 48px;
    font-weight: bold;
    color: rgba(255, 255, 255, 0.1);
  }

  .placeholder.loading {
    background: linear-gradient(135deg, #2a2a2a 0%, #1a1a1a 100%);
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-top-color: #f97316;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .favorite-btn {
    position: absolute;
    top: 10px;
    left: 10px;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    border: none;
    border-radius: 50%;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s;
    z-index: 2;
    opacity: 0;
  }

  .game-card:hover .favorite-btn,
  .game-card.focused .favorite-btn,
  .favorite-btn.active {
    opacity: 1;
  }

  .favorite-btn:hover {
    background: rgba(0, 0, 0, 0.8);
    color: #ef4444;
    transform: scale(1.1);
  }

  .favorite-btn.active {
    color: #ef4444;
  }

  .status-badge {
    position: absolute;
    top: 12px;
    right: 12px;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 600;
    color: white;
    letter-spacing: 0.3px;
  }

  .status-badge.status-released {
    background: linear-gradient(135deg, #10b981, #059669);
  }

  .status-badge.status-beta {
    background: linear-gradient(135deg, #8b5cf6, #7c3aed);
  }

  .status-badge.status-translating {
    background: linear-gradient(135deg, #f97316, #ea580c);
  }

  .status-badge.status-wip {
    background: linear-gradient(135deg, #eab308, #ca8a04);
  }

  .status-badge.status-draft {
    background: linear-gradient(135deg, #64748b, #475569);
  }

  .card-info {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .card-title {
    font-size: 15px;
    font-weight: 600;
    color: white;
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    line-height: 1.3;
  }

  .card-team {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .progress-section {
    margin-top: 8px;
  }

  .progress-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .progress-label {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.4);
  }

  .progress-value {
    font-size: 12px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.6);
  }

  .progress-value.complete {
    color: #10b981;
  }

  .progress-track {
    height: 4px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #10b981, #34d399);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .progress-fill.complete {
    background: linear-gradient(90deg, #10b981, #34d399);
  }
</style>
