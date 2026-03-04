<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { WifiOff, Play, RefreshCw, AlertTriangle, Download, CheckCircle } from 'lucide-svelte'
  import { agentStore } from '../stores/agent.svelte'
  import { DownloadAndStartAgent, IsAgentInstalled } from '../../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

  let starting = false
  let downloading = false
  let downloadProgress = 0
  let downloadStatus = ''
  let agentInstalled = false
  let error: string | null = null

  $: status = $agentStore.status
  $: isDisconnected = status === 'disconnected' || status === 'error'
  $: isConnecting = status === 'connecting'

  onMount(async () => {
    // Check if agent is installed
    try {
      agentInstalled = await IsAgentInstalled()
    } catch {
      agentInstalled = false
    }

    // Listen for download progress
    EventsOn('agent:download:progress', (data: { status: string; percent: number }) => {
      downloadStatus = data.status
      downloadProgress = data.percent
      if (data.status === 'complete') {
        downloading = false
        agentInstalled = true
      }
    })
  })

  onDestroy(() => {
    EventsOff('agent:download:progress')
  })

  async function handleDownloadAndStart() {
    starting = true
    downloading = true
    downloadProgress = 0
    error = null

    try {
      await DownloadAndStartAgent()

      // Wait and try to connect
      await new Promise(resolve => setTimeout(resolve, 1000))

      for (let i = 0; i < 5; i++) {
        const connected = await agentStore.connect()
        if (connected) break
        await new Promise(resolve => setTimeout(resolve, 500))
      }
    } catch (err) {
      console.error('Failed to download/start agent:', err)
      error = err instanceof Error ? err.message : 'Nepodařilo se stáhnout agenta'
      downloading = false
    }

    starting = false
  }

  async function handleRetryConnect() {
    starting = true
    error = null
    await agentStore.connect()
    starting = false
  }
</script>

{#if isDisconnected || isConnecting || downloading}
  <div class="agent-banner" class:error={status === 'error' || error}>
    <div class="banner-content">
      <div class="banner-icon">
        {#if downloading}
          <Download size={20} />
        {:else if status === 'error' || error}
          <AlertTriangle size={20} />
        {:else}
          <WifiOff size={20} />
        {/if}
      </div>

      <div class="banner-text">
        {#if downloading}
          <span class="title">Stahování CZManager agenta...</span>
          <div class="progress-container">
            <div class="progress-bar">
              <div class="progress-fill" style="width: {downloadProgress}%"></div>
            </div>
            <span class="progress-text">{downloadProgress}%</span>
          </div>
        {:else if isConnecting || starting}
          <span class="title">Připojování k agentovi...</span>
        {:else if error}
          <span class="title">Chyba</span>
          <span class="subtitle">{error}</span>
        {:else if status === 'error'}
          <span class="title">Spojení s agentem selhalo</span>
          <span class="subtitle">Agent pravděpodobně neběží</span>
        {:else if !agentInstalled}
          <span class="title">Agent není nainstalován</span>
          <span class="subtitle">Pro instalaci lokalizací je potřeba stáhnout CZManager agenta</span>
        {:else}
          <span class="title">Agent není spuštěn</span>
          <span class="subtitle">Pro instalaci lokalizací je potřeba spustit CZManager agenta</span>
        {/if}
      </div>

      <div class="banner-actions">
        {#if downloading}
          <!-- Progress is shown in text area -->
        {:else if isConnecting || starting}
          <div class="connecting-indicator">
            <RefreshCw size={18} class="spinning" />
          </div>
        {:else}
          <button class="btn-start" on:click={handleDownloadAndStart}>
            {#if !agentInstalled}
              <Download size={16} />
              Stáhnout a spustit
            {:else}
              <Play size={16} />
              Spustit agenta
            {/if}
          </button>
          {#if agentInstalled && (status === 'error' || error)}
            <button class="btn-retry" on:click={handleRetryConnect}>
              <RefreshCw size={16} />
              Zkusit znovu
            </button>
          {/if}
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .agent-banner {
    background: linear-gradient(135deg, rgba(245, 158, 11, 0.15) 0%, rgba(245, 158, 11, 0.08) 100%);
    border-bottom: 1px solid rgba(245, 158, 11, 0.3);
  }

  .agent-banner.error {
    background: linear-gradient(135deg, rgba(239, 68, 68, 0.15) 0%, rgba(239, 68, 68, 0.08) 100%);
    border-bottom: 1px solid rgba(239, 68, 68, 0.3);
  }

  .banner-content {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 48px;
  }

  .banner-icon {
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(245, 158, 11, 0.2);
    border-radius: 10px;
    color: #fbbf24;
  }

  .error .banner-icon {
    background: rgba(239, 68, 68, 0.2);
    color: #f87171;
  }

  .banner-text {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .title {
    font-size: 14px;
    font-weight: 600;
    color: white;
  }

  .subtitle {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.5);
  }

  .progress-container {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 4px;
  }

  .progress-bar {
    flex: 1;
    height: 6px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    overflow: hidden;
    max-width: 300px;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #f97316, #fb923c);
    border-radius: 3px;
    transition: width 0.2s ease;
  }

  .progress-text {
    font-size: 13px;
    font-weight: 600;
    color: #f97316;
    min-width: 40px;
  }

  .banner-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .connecting-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    color: #fbbf24;
  }

  .btn-start {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 40px;
    padding: 0 20px;
    background: linear-gradient(135deg, #22c55e, #16a34a);
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
  }

  .btn-start:hover {
    background: linear-gradient(135deg, #16a34a, #15803d);
    transform: translateY(-1px);
  }

  .btn-retry {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 40px;
    padding: 0 16px;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.8);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-retry:hover {
    background: rgba(255, 255, 255, 0.15);
    color: white;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
