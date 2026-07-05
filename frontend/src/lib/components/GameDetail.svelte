<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { FolderOpen, Download, Trash2, RefreshCw, CheckCircle, AlertCircle, X, ExternalLink, Clock, AlertTriangle, Heart } from 'lucide-svelte'
  import type { Localization } from '../stores/games.svelte'
  import { focusStore } from '../stores/focus.svelte'
  import { authStore } from '../stores/auth.svelte'
  import { favoritesStore } from '../stores/favorites.svelte'
  import { BrowseFolder, ScanGames, FetchGameDetail, DownloadLocalization, Install, Uninstall, CancelInstall, IsInstalled } from '../../../wailsjs/go/main/App'
  import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

  const API_BASE = 'https://lokalizace.net'

  // Props - Svelte 5
  let { game, onClose }: { game: Localization; onClose?: () => void } = $props()

  // State - Svelte 5 runes
  let gamePath = $state(game.gamePath || '')
  let installing = $state(false)
  let uninstalling = $state(false)
  let downloading = $state(false)
  let progress = $state(0)
  let progressStage = $state('')
  let error = $state<string | null>(null)
  let success = $state(false)
  let logs = $state<string[]>([])
  let detectedPath = $state<string | null>(null)
  let scanning = $state(false)
  let isInstalled = $state(false)  // je lokalizace nainstalovaná v gamePath?

  // Zkontroluj instalaci pokaždé, když se změní cesta ke hře
  $effect(() => {
    const path = gamePath
    if (!path) { isInstalled = false; return }
    IsInstalled(path).then(v => { isInstalled = v }).catch(() => { isInstalled = false })
  })

  // Derived state - Svelte 5
  let safeDescription = $derived(sanitizeHtml(game.description || ''))
  let progressPercent = $derived(game.translatePercent || 0)
  let supportsAppInstall = $derived(game.supportsAppInstall === true)
  let isReady = $derived(game.status === 'released' || game.status === 'beta')
  let statusMessage = $derived(getInstallStatusMessage(game.status, progressPercent, supportsAppInstall))

  // VIP/Supporter funkce - automatické hledání her
  let canAutoScan = $derived($authStore.features?.hasGameScanner || false)

  // Oblíbené
  let isFav = $derived($favoritesStore.ids.includes(game.id))
  let isLoggedIn = $derived(!!$authStore.user)

  function sanitizeHtml(html: string): string {
    if (!html) return ''
    return html
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '')
      .replace(/on\w+="[^"]*"/gi, '')
      .replace(/on\w+='[^']*'/gi, '')
  }

  function getInstallStatusMessage(status: string, percent: number, appInstall: boolean): { text: string; type: 'success' | 'warning' | 'info' } {
    if (status === 'released' && appInstall) {
      return { text: 'Lokalizace je připravena k instalaci', type: 'success' }
    }
    if (status === 'released' && !appInstall) {
      return { text: 'Lokalizace je k dispozici ke stažení na webu', type: 'success' }
    }
    if (status === 'beta' && appInstall) {
      return { text: 'Beta verze - může obsahovat chyby', type: 'warning' }
    }
    if (status === 'beta' && !appInstall) {
      return { text: 'Beta verze je k dispozici ke stažení na webu', type: 'warning' }
    }
    if (status === 'translating') {
      return { text: `Překlad probíhá (${percent}% hotovo)`, type: 'info' }
    }
    if (status === 'wip') {
      return { text: 'Lokalizace je v rané fázi vývoje', type: 'info' }
    }
    return { text: 'Lokalizace zatím není k dispozici', type: 'info' }
  }

  function getStatusLabel(status: string): string {
    switch (status) {
      case 'released': return 'Vydáno'
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
      default: return 'status-default'
    }
  }

  async function scanForGame() {
    scanning = true
    try {
      const games = await ScanGames(game.name)
      if (games && games.length > 0) {
        detectedPath = games[0].path
        gamePath = games[0].path
      }
    } catch (err) {
      console.error('Scan failed:', err)
    }
    scanning = false
  }

  async function browseFolder() {
    try {
      const path = await BrowseFolder('Vyberte složku s hrou')
      if (path) {
        gamePath = path
        detectedPath = null
      }
    } catch (err) {
      error = 'Nepodařilo se otevřít dialog'
    }
  }

  // Listen to install/uninstall progress + log events emitted by the Go
  // backend (replaces the old HTTP polling of /progress and /logs).
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
        uninstalling = false
        success = true
        // Po instalaci/odinstalaci přepočítej stav (ukáže/skryje "Odebrat")
        if (gamePath) {
          IsInstalled(gamePath).then(v => { isInstalled = v }).catch(() => {})
        }
      } else if (data.stage === 'error') {
        stopProgressListening()
        installing = false
        uninstalling = false
        error = data.error || 'Operace selhala'
      }
    })
  }

  function stopProgressListening() {
    EventsOff('install:progress')
    EventsOff('install:log')
  }

  async function startInstall() {
    if (!gamePath) { error = 'Vyberte složku s hrou'; return }

    installing = true
    error = null
    success = false
    progress = 0
    progressStage = 'Načítání informací o lokalizaci...'
    logs = []

    try {
      // Získáme detail hry z Go backendu (obejití CORS)
      const detail = await FetchGameDetail(game.id)

      if (!detail.files || detail.files.length === 0) {
        throw new Error('Pro tuto lokalizaci zatím nejsou nahrány žádné soubory ke stažení')
      }

      // Vybereme nejnovější soubor (poslední v poli)
      const files = detail.files as Array<{ id: number; version: string; fileName: string }>
      const latestFile = files[files.length - 1]
      const downloadUrl = `${API_BASE}/api/download/${latestFile.id}`

      logs = [...logs, `Stahování: ${latestFile.fileName} (verze ${latestFile.version})`]
      progressStage = 'Zahajování instalace...'

      // Start listening before the operation so we don't miss early events.
      startProgressListening()

      await Install(
        game.slug,
        latestFile.version || game.version || '1.0.0',
        downloadUrl,
        gamePath
      )

    } catch (err) {
      stopProgressListening()
      error = err instanceof Error ? err.message : 'Chyba při instalaci'
      installing = false
    }
  }

  function getStageLabel(stage: string): string {
    switch (stage) {
      case 'downloading': return 'Stahování balíčku...'
      case 'extracting': return 'Rozbalování souborů...'
      case 'pre_tasks': return 'Příprava instalace...'
      case 'installing': return 'Kopírování souborů...'
      case 'post_tasks': return 'Dokončování instalace...'
      case 'done': return 'Hotovo!'
      case 'error': return 'Chyba'
      default: return 'Zpracování...'
    }
  }

  async function startUninstall() {
    if (!gamePath) { error = 'Chybí cesta ke hře'; return }

    uninstalling = true
    error = null
    success = false
    progress = 0
    logs = ['Odstraňování lokalizace...']

    try {
      startProgressListening()
      await Uninstall(gamePath)
    } catch (err) {
      stopProgressListening()
      error = err instanceof Error ? err.message : 'Chyba při odinstalaci'
      uninstalling = false
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
    uninstalling = false
  }

  function openOnWeb() {
    // Otevři v systémovém prohlížeči (ne uvnitř webview)
    BrowserOpenURL(`${API_BASE}/localizations/${game.slug}`)
  }

  async function downloadLocalization() {
    downloading = true
    error = null
    success = false
    progress = 0
    logs = ['Připojování k serveru...']

    // Posloucháme progress eventy
    EventsOn('download:progress', (data: { status: string; percent: number; file: string }) => {
      progress = data.percent
      if (data.status === 'downloading' && data.percent === 0) {
        logs = [...logs, `Stahování: ${data.file}`]
      }
      if (data.status === 'complete') {
        logs = [...logs, `Hotovo! Soubor ${data.file} byl stažen do složky Stažené soubory.`]
        downloading = false
        success = true
        EventsOff('download:progress')
      }
    })

    try {
      const savedPath = await DownloadLocalization(game.id)
      console.log('Downloaded to:', savedPath)
    } catch (err) {
      error = err instanceof Error ? err.message : 'Stahování selhalo'
      downloading = false
      EventsOff('download:progress')
    }
  }

  let modalElement: HTMLDivElement

  function updateFocusables() {
    if (!modalElement) return
    const focusableElements = Array.from(
      modalElement.querySelectorAll('button:not(:disabled), input:not(:disabled), [tabindex="0"]')
    ) as HTMLElement[]

    focusStore.updateZoneElements('modal', focusableElements)

    // Focus první element pokud je modal aktivní
    if (focusableElements.length > 0) {
      focusStore.focusCurrent()
    }
  }

  onMount(() => {
    // Registruj modal jako focus zónu
    focusStore.registerZone({
      id: 'modal',
      elements: [],
      columns: 1,
      loop: false,
      onEscape: () => {
        onClose?.()
      }
    })

    // Automatické hledání her pouze pro VIP/Supporter uživatele
    if (canAutoScan) {
      scanForGame()
    }
    // Počkáme na renderování a pak nastavíme focusables
    setTimeout(updateFocusables, 100)
  })

  onDestroy(() => {
    stopProgressListening()
    focusStore.unregisterZone('modal')
  })

  // Aktualizuj focusables když se změní stav
  $effect(() => {
    if (installing || uninstalling || downloading || success || error) {
      setTimeout(updateFocusables, 50)
    }
  })
</script>

<div class="modal-card" bind:this={modalElement}>
  <!-- Header -->
  <div class="modal-header">
    <button class="close-btn" onclick={onClose}>
      <X size={18} />
    </button>

    <div class="header-content">
      <img src={game.imageUrl} alt={game.name} class="cover-image" />

      <div class="header-info">
        <div class="meta-row">
          <span class="status {getStatusClass(game.status)}">{getStatusLabel(game.status)}</span>
          {#if game.teamName}
            <span class="separator">•</span>
            <span class="team">{game.teamName}</span>
          {/if}
        </div>

        <h2 class="game-title">{game.name}</h2>

        <div class="progress-section">
          <div class="progress-label">
            <span>Přeloženo</span>
            <span class="progress-value" class:complete={progressPercent >= 100}>{progressPercent}%</span>
          </div>
          <div class="progress-track">
            <div class="progress-fill" class:complete={progressPercent >= 100} style="width: {progressPercent}%"></div>
          </div>
        </div>

        <div class="header-actions">
          {#if isLoggedIn}
            <button
              class="favorite-link"
              class:active={isFav}
              onclick={() => favoritesStore.toggleFavorite(game.id)}
            >
              <Heart size={14} fill={isFav ? 'currentColor' : 'none'} />
              {isFav ? 'V oblíbených' : 'Přidat do oblíbených'}
            </button>
          {/if}
          <button class="web-link" onclick={openOnWeb}>
            <ExternalLink size={12} />
            Zobrazit na webu
          </button>
        </div>
      </div>
    </div>
  </div>

  <div class="divider"></div>

  <!-- Content -->
  <div class="modal-content">
    {#if safeDescription}
      <div class="description">
        {@html safeDescription}
      </div>
    {/if}

    <!-- Status message -->
    <div class="status-message {statusMessage.type}">
      {#if statusMessage.type === 'success'}
        <CheckCircle size={18} />
      {:else if statusMessage.type === 'warning'}
        <AlertTriangle size={18} />
      {:else}
        <Clock size={18} />
      {/if}
      <span>{statusMessage.text}</span>
    </div>

    <!-- Stav 1: Podporuje přímou instalaci přes aplikaci -->
    {#if supportsAppInstall && isReady}
      <div class="input-section">
        <label for="game-path-input">Cesta ke hře</label>
        <div class="input-row">
          <div class="input-wrapper">
            <input
              id="game-path-input"
              type="text"
              bind:value={gamePath}
              placeholder="Vyberte složku s hrou..."
              disabled={installing || uninstalling}
            />
            {#if detectedPath}
              <CheckCircle size={16} class="input-icon" />
            {/if}
          </div>
          <button class="btn-secondary" onclick={browseFolder} disabled={installing || uninstalling}>
            <FolderOpen size={16} />
            <span>Procházet</span>
          </button>
          {#if canAutoScan}
            <button class="btn-icon" onclick={scanForGame} disabled={installing || uninstalling || scanning} title="Automaticky vyhledat">
              <RefreshCw size={16} class={scanning ? 'spinning' : ''} />
            </button>
          {/if}
        </div>
        {#if detectedPath && canAutoScan}
          <p class="detected-msg">Hra byla automaticky nalezena</p>
        {/if}
      </div>

      <!-- Progress + logy: zůstávají viditelné i po dokončení/chybě, aby
           uživatel viděl co se stalo (nakopírované soubory, warningy). -->
      {#if installing || uninstalling || logs.length > 0}
        <div class="install-progress">
          <div class="install-header">
            <span>{progressStage || (installing ? 'Instalace...' : 'Odinstalace...')}</span>
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

      {#if success && !installing && !uninstalling}
        <div class="message success">
          <CheckCircle size={18} />
          <span>Hotovo!</span>
        </div>
      {/if}

      <!-- Action buttons -->
      {#if !installing && !uninstalling && !downloading}
        <div class="actions">
          {#if isInstalled}
            <button class="btn-danger" onclick={startUninstall} disabled={!gamePath}>
              <Trash2 size={18} />
              Odebrat
            </button>
          {/if}
          <button class="btn-primary" onclick={startInstall} disabled={!gamePath}>
            <Download size={18} />
            {isInstalled ? 'Přeinstalovat' : 'Nainstalovat'}
          </button>
          <button class="btn-secondary" onclick={downloadLocalization} disabled={downloading}>
            <Download size={18} />
            Jen stáhnout
          </button>
        </div>
      {:else if installing || uninstalling}
        <button class="btn-cancel" onclick={cancelInstall}>
          Zrušit
        </button>
      {:else if downloading}
        <button class="btn-cancel" onclick={() => { downloading = false; EventsOff('download:progress') }}>
          Zrušit stahování
        </button>
      {/if}

    <!-- Stav 2: Je hotová ale nepodporuje přímou instalaci - jen stažení z webu -->
    {:else if isReady && !supportsAppInstall}
      <div class="web-download-section">
        {#if !downloading && !success}
          <p class="web-download-info">
            Tato lokalizace nepodporuje přímou instalaci přes aplikaci.<br>
            Stáhněte si ji a nainstalujte ručně podle návodu na webu.
          </p>
          <div class="download-buttons">
            <button class="btn-primary" onclick={downloadLocalization}>
              <Download size={18} />
              Stáhnout lokalizaci
            </button>
            <button class="btn-secondary-large" onclick={openOnWeb}>
              <ExternalLink size={18} />
              Zobrazit na webu
            </button>
          </div>
        {:else if downloading}
          <div class="install-progress">
            <div class="install-header">
              <span>Stahování...</span>
              <span class="install-percent">{progress}%</span>
            </div>
            <div class="install-track">
              <div class="install-fill" style="width: {progress}%"></div>
            </div>
            <div class="install-logs">
              {#each logs as log}
                <div class="log-line">{log}</div>
              {/each}
            </div>
          </div>
          <button class="btn-cancel" onclick={() => { downloading = false }}>
            Zrušit
          </button>
        {:else if success}
          <div class="message success">
            <CheckCircle size={18} />
            <span>Soubor byl úspěšně stažen!</span>
          </div>
          <div class="download-buttons">
            <button class="btn-secondary-large" onclick={() => { success = false }}>
              <Download size={18} />
              Stáhnout znovu
            </button>
            <button class="btn-secondary-large" onclick={openOnWeb}>
              <ExternalLink size={18} />
              Návod k instalaci
            </button>
          </div>
        {/if}
      </div>

    <!-- Stav 3: Není hotová -->
    {:else}
      <div class="not-available">
        <p>Tato lokalizace zatím není připravena ke stažení.</p>
        <p>Sledujte postup překladu na našem webu.</p>
        <button class="btn-secondary-large" onclick={openOnWeb}>
          <ExternalLink size={18} />
          Zobrazit na webu
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .modal-card {
    background: #1a1a1a;
    border-radius: 16px;
    width: 100%;
    max-height: 100%;
    /* Celý card scrolluje - na malém okně se nic neuřízne (header i obsah) */
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    position: relative;
    padding: 32px 40px;
    flex-shrink: 0;
  }

  .close-btn {
    position: absolute;
    top: 16px;
    right: 16px;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.5);
    border: none;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s;
  }

  .close-btn:hover,
  .close-btn:focus {
    background: rgba(0, 0, 0, 0.7);
    color: white;
    outline: none;
    box-shadow: 0 0 0 2px #f97316;
  }

  .header-content {
    display: flex;
    gap: 20px;
  }

  .cover-image {
    width: 180px;
    height: 240px;
    object-fit: cover;
    border-radius: 12px;
    flex-shrink: 0;
  }

  .header-info {
    flex: 1;
    min-width: 0;
    padding-top: 4px;
  }

  .meta-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
  }

  .status {
    font-size: 12px;
    font-weight: 500;
  }

  .status-released { color: #34d399; }
  .status-beta { color: #60a5fa; }
  .status-translating { color: #c084fc; }
  .status-wip { color: #fbbf24; }
  .status-default { color: #94a3b8; }

  .separator {
    color: rgba(255, 255, 255, 0.3);
    font-size: 12px;
  }

  .team {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
  }

  .game-title {
    font-size: 28px;
    font-weight: 700;
    color: white;
    margin: 0 0 20px 0;
    line-height: 1.2;
  }

  .progress-section {
    margin-bottom: 12px;
  }

  .progress-label {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
    margin-bottom: 6px;
  }

  .progress-value {
    font-weight: 600;
    color: rgba(255, 255, 255, 0.7);
  }

  .progress-value.complete {
    color: #34d399;
  }

  .progress-track {
    height: 6px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: #f97316;
    border-radius: 3px;
    transition: width 0.3s;
  }

  .progress-fill.complete {
    background: #22c55e;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .favorite-link {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.4);
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    transition: color 0.2s;
  }

  .favorite-link:hover {
    color: #ef4444;
  }

  .favorite-link.active {
    color: #ef4444;
  }

  .web-link {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.4);
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    transition: color 0.2s;
  }

  .web-link:hover {
    color: rgba(255, 255, 255, 0.7);
  }

  .divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.05);
    margin: 0 40px;
  }

  .modal-content {
    padding: 28px 40px 40px;
    display: flex;
    flex-direction: column;
    gap: 24px;
    flex: 1;
  }

  .description {
    font-size: 14px;
    color: rgba(255, 255, 255, 0.6);
    line-height: 1.6;
    padding-right: 8px;
  }

  .description :global(p) { margin: 0 0 12px 0; }
  .description :global(p:last-child) { margin-bottom: 0; }
  .description :global(a) { color: #f97316; text-decoration: none; }
  .description :global(a:hover) { text-decoration: underline; }
  .description :global(strong), .description :global(b) { color: rgba(255, 255, 255, 0.8); font-weight: 600; }
  .description :global(ul), .description :global(ol) { margin: 8px 0; padding-left: 20px; }
  .description :global(li) { margin: 4px 0; }

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

  .btn-icon {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-icon:hover,
  .btn-icon:focus { background: rgba(255, 255, 255, 0.1); color: white; outline: none; box-shadow: 0 0 0 2px #f97316; }
  .btn-icon:disabled { opacity: 0.5; cursor: not-allowed; }

  :global(.spinning) { animation: spin 1s linear infinite; }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

  .detected-msg { margin: 8px 0 0; font-size: 12px; color: #34d399; }

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
    max-height: 100px;
    overflow-y: auto;
    font-family: monospace;
    font-size: 12px;
  }

  .log-line { color: rgba(255, 255, 255, 0.4); padding: 2px 0; }

  .message {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
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

  .actions { display: flex; gap: 12px; padding-top: 4px; }

  .btn-primary {
    flex: 1;
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

  .btn-primary:hover,
  .btn-primary:focus { background: #ea580c; outline: none; box-shadow: 0 0 0 2px #fff; }
  .btn-primary:disabled { opacity: 0.4; cursor: not-allowed; }

  .btn-danger {
    flex: 1;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 12px;
    font-size: 16px;
    font-weight: 500;
    color: #f87171;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-danger:hover,
  .btn-danger:focus { background: rgba(239, 68, 68, 0.2); outline: none; box-shadow: 0 0 0 2px #f97316; }
  .btn-danger:disabled { opacity: 0.4; cursor: not-allowed; }

  .btn-cancel {
    width: 100%;
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

  .status-message {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 18px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
  }

  .status-message.success {
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.2);
    color: #4ade80;
  }

  .status-message.warning {
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.2);
    color: #fbbf24;
  }

  .status-message.info {
    background: rgba(99, 102, 241, 0.1);
    border: 1px solid rgba(99, 102, 241, 0.2);
    color: #a5b4fc;
  }

  .web-download-section { padding: 8px 0; }

  .web-download-info {
    color: rgba(255, 255, 255, 0.6);
    font-size: 14px;
    line-height: 1.6;
    margin: 0 0 20px 0;
    text-align: center;
  }

  .download-buttons { display: flex; gap: 12px; }
  .download-buttons .btn-primary { flex: 1; }
  .download-buttons .btn-secondary-large { flex: 1; }

  .not-available { text-align: center; padding: 24px 0; }
  .not-available p { color: rgba(255, 255, 255, 0.5); font-size: 14px; margin: 0 0 8px 0; }
  .not-available p:last-of-type { margin-bottom: 20px; }

  .btn-secondary-large {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    height: 48px;
    padding: 0 24px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    color: rgba(255, 255, 255, 0.8);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary-large:hover,
  .btn-secondary-large:focus {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.25);
    color: white;
    outline: none;
    box-shadow: 0 0 0 2px #f97316;
  }
</style>
