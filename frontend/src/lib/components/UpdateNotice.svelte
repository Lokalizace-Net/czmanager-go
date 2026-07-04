<script lang="ts">
  import { Download, X, Sparkles, Loader2, AlertCircle } from 'lucide-svelte'
  import { appStore } from '../stores/app.svelte'

  // Zobraz jen když je aktualizace k dispozici a uživatel ji nezavřel
  let show = $derived($appStore.update?.available === true && !$appStore.dismissed)
  let latest = $derived($appStore.update?.latestVersion ?? '')
  let updating = $derived($appStore.updating)
  let progress = $derived($appStore.progress)
  let error = $derived($appStore.updateError)
</script>

{#if show}
  <div class="update-notice" role="alert">
    {#if updating}
      <!-- Průběh automatické aktualizace -->
      <div class="update-icon">
        <Loader2 size={18} class="spinning" />
      </div>
      <div class="update-text">
        <span class="update-title">{progress?.message ?? 'Aktualizuji...'}</span>
        <div class="update-track">
          <div class="update-fill" style="width: {progress?.percent ?? 0}%"></div>
        </div>
      </div>
    {:else if error}
      <!-- Chyba - nabídni ruční stažení -->
      <div class="update-icon err">
        <AlertCircle size={18} />
      </div>
      <div class="update-text">
        <span class="update-title">Automatická aktualizace selhala</span>
        <span class="update-sub">{error}</span>
      </div>
      <button class="update-btn" onclick={() => appStore.openRelease()}>
        <Download size={16} />
        Stáhnout ručně
      </button>
      <button class="update-close" onclick={() => appStore.dismissUpdate()} title="Zavřít">
        <X size={16} />
      </button>
    {:else}
      <!-- Nabídka aktualizace -->
      <div class="update-icon">
        <Sparkles size={18} />
      </div>
      <div class="update-text">
        <span class="update-title">K dispozici je nová verze {latest}</span>
        <span class="update-sub">Aktualizovat automaticky a restartovat</span>
      </div>
      <button class="update-btn" onclick={() => appStore.performUpdate()}>
        <Download size={16} />
        Aktualizovat
      </button>
      <button class="update-close" onclick={() => appStore.dismissUpdate()} title="Zavřít">
        <X size={16} />
      </button>
    {/if}
  </div>
{/if}

<style>
  .update-notice {
    position: fixed;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 12px 16px;
    background: #1a1a1a;
    border: 1px solid rgba(249, 115, 22, 0.4);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
    z-index: 1100;
    width: 480px;
    max-width: calc(100vw - 32px);
    animation: slide-up 0.3s ease;
  }

  @keyframes slide-up {
    from { opacity: 0; transform: translate(-50%, 20px); }
    to { opacity: 1; transform: translate(-50%, 0); }
  }

  .update-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    flex-shrink: 0;
    border-radius: 10px;
    background: rgba(249, 115, 22, 0.15);
    color: #f97316;
  }

  .update-icon.err {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }

  .update-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    flex: 1;
  }

  .update-title {
    font-size: 14px;
    font-weight: 600;
    color: white;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .update-sub {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .update-track {
    height: 6px;
    background: rgba(0, 0, 0, 0.4);
    border-radius: 3px;
    overflow: hidden;
  }

  .update-fill {
    height: 100%;
    background: linear-gradient(90deg, #f97316, #fb923c);
    border-radius: 3px;
    transition: width 0.3s;
  }

  .update-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 38px;
    padding: 0 16px;
    background: #f97316;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .update-btn:hover,
  .update-btn:focus {
    background: #ea580c;
    outline: none;
    box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.4);
  }

  .update-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    flex-shrink: 0;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.4);
    cursor: pointer;
    transition: all 0.2s;
  }

  .update-close:hover,
  .update-close:focus {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    outline: none;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
