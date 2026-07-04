<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte'
  import { FileArchive, FolderOpen, Play, CheckCircle, AlertCircle, FlaskConical } from 'lucide-svelte'
  import { BrowseFile, BrowseFolder, InstallLocal, CancelInstall } from '../../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
  import { focusStore } from '../stores/focus.svelte'

  // State - Svelte 5 runes
  let zipPath = $state('')
  let gamePath = $state('')
  let installing = $state(false)
  let progress = $state(0)
  let progressStage = $state('')
  let error = $state<string | null>(null)
  let success = $state(false)
  let logs = $state<string[]>([])

  let canInstall = $derived(!!zipPath && !!gamePath && !installing)

  let pageElement = $state<HTMLElement | undefined>(undefined)

  // Zaregistruje focusable prvky stránky do hlavní 'main' zóny, aby fungovala
  // navigace šipkami i gamepadem (D-pad). Sloupec = 1, prvky jdou pod sebou.
  function updateFocusElements() {
    if (!pageElement) return
    const elements = Array.from(
      pageElement.querySelectorAll('button:not(:disabled), input:not(:disabled), [tabindex="0"]')
    ) as HTMLElement[]

    focusStore.registerZone({
      id: 'main',
      elements,
      columns: 1,
      loop: false,
      onEscape: () => focusStore.setActiveZone('sidemenu', false)
    })

    // Pokud je stránka aktivní zónou, nastav focus na první prvek
    if (elements.length > 0) {
      focusStore.focusCurrent()
    }
  }

  // Znovu naregistruj prvky vždy, když se změní obsah (přepnou se tlačítka,
  // objeví/zmizí progress, chybová/úspěšná hláška).
  $effect(() => {
    // Sleduj stav, který mění vykreslené focusable prvky
    void installing
    void success
    void error
    void zipPath
    void gamePath
    tick().then(updateFocusElements)
  })

  onMount(() => {
    setTimeout(updateFocusElements, 100)
  })

  function getStageLabel(stage: string): string {
    switch (stage) {
      case 'extracting': return 'Rozbalování souborů...'
      case 'pre_tasks': return 'Příprava instalace...'
      case 'installing': return 'Kopírování souborů...'
      case 'post_tasks': return 'Dokončování instalace...'
      case 'done': return 'Hotovo!'
      case 'error': return 'Chyba'
      default: return 'Zpracování...'
    }
  }

  function startProgressListening() {
    EventsOn('install:log', (message: string) => {
      logs = [...logs, message]
    })

    EventsOn('install:progress', (data: { stage: string; percent: number; error?: string }) => {
      progress = data.percent || 0
      progressStage = getStageLabel(data.stage)

      if (data.stage === 'done') {
        stopProgressListening()
        installing = false
        success = true
      } else if (data.stage === 'error') {
        stopProgressListening()
        installing = false
        error = data.error || 'Instalace selhala'
      }
    })
  }

  function stopProgressListening() {
    EventsOff('install:progress')
    EventsOff('install:log')
  }

  async function browseZip() {
    try {
      const path = await BrowseFile('Vyberte lokalizační balíček (ZIP)', '*.zip')
      if (path) {
        zipPath = path
        error = null
        success = false
      }
    } catch (err) {
      error = 'Nepodařilo se otevřít dialog'
    }
  }

  async function browseGameFolder() {
    try {
      const path = await BrowseFolder('Vyberte složku s hrou')
      if (path) {
        gamePath = path
        error = null
        success = false
      }
    } catch (err) {
      error = 'Nepodařilo se otevřít dialog'
    }
  }

  async function startInstall() {
    if (!zipPath) { error = 'Vyberte lokalizační balíček (ZIP)'; return }
    if (!gamePath) { error = 'Vyberte složku s hrou'; return }

    installing = true
    error = null
    success = false
    progress = 0
    progressStage = 'Zahajování instalace...'
    logs = []

    try {
      startProgressListening()
      await InstallLocal(gamePath, zipPath)
    } catch (err) {
      stopProgressListening()
      error = err instanceof Error ? err.message : 'Chyba při instalaci'
      installing = false
    }
  }

  async function cancelInstall() {
    try {
      await CancelInstall()
    } catch (err) {
      console.error('Cancel error:', err)
    }
    stopProgressListening()
    installing = false
  }

  onDestroy(() => {
    stopProgressListening()
    // Ukliď po sobě, ať v 'main' zóně nezůstanou odpojené prvky
    focusStore.updateZoneElements('main', [])
  })
</script>

<div class="page-content" bind:this={pageElement}>
  <div class="header">
    <FlaskConical size={28} />
    <div>
      <h1 class="page-title">Manuální instalace</h1>
      <p class="page-subtitle">Otestujte svůj lokalizační balíček z lokálního ZIP archivu, bez nahrávání na server.</p>
    </div>
  </div>

  <div class="install-card">
    <!-- Výběr ZIP balíčku -->
    <div class="input-section">
      <label for="zip-path-input">Lokalizační balíček (ZIP)</label>
      <div class="input-row">
        <div class="input-wrapper">
          <input
            id="zip-path-input"
            type="text"
            bind:value={zipPath}
            placeholder="Vyberte ZIP archiv s INSTALL_INSTRUCTIONS.json..."
            disabled={installing}
          />
          {#if zipPath}
            <CheckCircle size={16} class="input-icon" />
          {/if}
        </div>
        <button class="btn-secondary" onclick={browseZip} disabled={installing}>
          <FileArchive size={16} />
          <span>Procházet</span>
        </button>
      </div>
    </div>

    <!-- Výběr složky s hrou -->
    <div class="input-section">
      <label for="game-path-input">Cesta ke hře</label>
      <div class="input-row">
        <div class="input-wrapper">
          <input
            id="game-path-input"
            type="text"
            bind:value={gamePath}
            placeholder="Vyberte složku s hrou..."
            disabled={installing}
          />
          {#if gamePath}
            <CheckCircle size={16} class="input-icon" />
          {/if}
        </div>
        <button class="btn-secondary" onclick={browseGameFolder} disabled={installing}>
          <FolderOpen size={16} />
          <span>Procházet</span>
        </button>
      </div>
    </div>

    <!-- Progress + logy: zůstávají viditelné i po dokončení/chybě, aby
         tvůrce viděl co se stalo (nakopírované soubory, tasky, warningy). -->
    {#if installing || logs.length > 0}
      <div class="install-progress">
        <div class="install-header">
          <span>{progressStage || 'Instalace...'}</span>
          <span class="install-percent">{progress}%</span>
        </div>
        <div class="install-track">
          <div class="install-fill" class:done={success} class:failed={!!error} style="width: {progress}%"></div>
        </div>
        {#if logs.length > 0}
          <div class="install-logs">
            {#each logs as log, i (i)}
              <div class="log-line">{log}</div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Messages -->
    {#if error}
      <div class="message error">
        <AlertCircle size={18} />
        <span>{error}</span>
      </div>
    {/if}

    {#if success && !installing}
      <div class="message success">
        <CheckCircle size={18} />
        <span>Instalace balíčku byla úspěšně dokončena!</span>
      </div>
    {/if}

    <!-- Actions -->
    {#if installing}
      <button class="btn-cancel" onclick={cancelInstall}>
        Zrušit
      </button>
    {:else}
      <button class="btn-primary" onclick={startInstall} disabled={!canInstall}>
        <Play size={18} />
        Nainstalovat balíček
      </button>
    {/if}
  </div>
</div>

<style>
  .page-content {
    padding: 32px 48px;
  }

  .header {
    display: flex;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 32px;
    color: #f97316;
  }

  .page-title {
    font-size: 28px;
    font-weight: 700;
    color: white;
    margin: 0 0 6px 0;
  }

  .page-subtitle {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.5);
    margin: 0;
    line-height: 1.5;
    max-width: 640px;
  }

  .install-card {
    background: #1a1a1a;
    border-radius: 16px;
    padding: 28px;
    border: 1px solid rgba(255, 255, 255, 0.05);
    display: flex;
    flex-direction: column;
    gap: 24px;
    width: 100%;
  }

  .input-section label {
    display: block;
    font-size: 11px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.4);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }

  .input-row {
    display: flex;
    gap: 8px;
  }

  .input-wrapper {
    flex: 1;
    position: relative;
  }

  .input-wrapper input {
    width: 100%;
    height: 44px;
    padding: 0 16px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    font-size: 14px;
    color: white;
    outline: none;
    transition: all 0.2s;
  }

  .input-wrapper input::placeholder { color: rgba(255, 255, 255, 0.2); }
  .input-wrapper input:focus { border-color: #f97316; box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.3); }
  .input-wrapper input:disabled { opacity: 0.5; }

  .input-wrapper :global(.input-icon) {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    color: #34d399;
  }

  .btn-secondary {
    height: 44px;
    padding: 0 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover,
  .btn-secondary:focus { background: rgba(255, 255, 255, 0.1); color: white; outline: none; box-shadow: 0 0 0 2px #f97316; }
  .btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

  .install-progress { display: flex; flex-direction: column; gap: 12px; }

  .install-header {
    display: flex;
    justify-content: space-between;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.7);
  }

  .install-percent { font-weight: 700; color: #f97316; }

  .install-track {
    height: 8px;
    background: rgba(0, 0, 0, 0.3);
    border-radius: 4px;
    overflow: hidden;
  }

  .install-fill {
    height: 100%;
    background: linear-gradient(90deg, #f97316, #fb923c);
    border-radius: 4px;
    transition: width 0.3s, background 0.3s;
  }

  .install-fill.done {
    background: linear-gradient(90deg, #16a34a, #22c55e);
  }

  .install-fill.failed {
    background: linear-gradient(90deg, #dc2626, #ef4444);
  }

  .install-logs {
    background: rgba(0, 0, 0, 0.3);
    border-radius: 8px;
    padding: 12px;
    max-height: 200px;
    overflow-y: auto;
    font-family: monospace;
    font-size: 12px;
  }

  .log-line { color: rgba(255, 255, 255, 0.4); padding: 2px 0; }

  .message {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-radius: 8px;
    font-size: 14px;
  }

  .message.error {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #f87171;
  }

  .message.success {
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.2);
    color: #4ade80;
  }

  .btn-primary {
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    background: #f97316;
    border: none;
    border-radius: 12px;
    font-size: 16px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary:hover:not(:disabled),
  .btn-primary:focus { background: #ea580c; outline: none; box-shadow: 0 0 0 2px #fff; }
  .btn-primary:disabled { opacity: 0.4; cursor: not-allowed; }

  .btn-cancel {
    height: 56px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    font-size: 16px;
    color: rgba(255, 255, 255, 0.6);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-cancel:hover,
  .btn-cancel:focus { background: rgba(255, 255, 255, 0.1); color: white; outline: none; box-shadow: 0 0 0 2px #f97316; }
</style>
