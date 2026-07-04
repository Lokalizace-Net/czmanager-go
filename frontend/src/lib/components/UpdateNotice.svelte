<script lang="ts">
  import { Download, X, Sparkles } from 'lucide-svelte'
  import { appStore } from '../stores/app.svelte'

  // Zobraz jen když je aktualizace k dispozici a uživatel ji nezavřel
  let show = $derived($appStore.update?.available === true && !$appStore.dismissed)
  let latest = $derived($appStore.update?.latestVersion ?? '')
</script>

{#if show}
  <div class="update-notice" role="alert">
    <div class="update-icon">
      <Sparkles size={18} />
    </div>
    <div class="update-text">
      <span class="update-title">K dispozici je nová verze {latest}</span>
      <span class="update-sub">Klikni pro stažení z GitHubu</span>
    </div>
    <button class="update-btn" onclick={() => appStore.openRelease()}>
      <Download size={16} />
      Stáhnout
    </button>
    <button class="update-close" onclick={() => appStore.dismissUpdate()} title="Zavřít">
      <X size={16} />
    </button>
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
    max-width: 480px;
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

  .update-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .update-title {
    font-size: 14px;
    font-weight: 600;
    color: white;
    white-space: nowrap;
  }

  .update-sub {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
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
</style>
