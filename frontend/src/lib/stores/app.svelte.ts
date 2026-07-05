// App store - informace o aplikaci (verze + auto-update z GitHubu)
import { writable, get } from 'svelte/store'
import { GetVersion, CheckUpdate, OpenReleasePage, PerformUpdate, Log } from '../../../wailsjs/go/main/App'
import { EventsOn } from '../../../wailsjs/runtime/runtime'

// log zapíše zprávu do Debug Logu (přes Go backend). Používá se pro logování
// uživatelských akcí z frontendu (navigace, kliknutí, ...).
export function debugLog(message: string) {
  Log(message).catch(() => {})
}

export interface UpdateInfo {
  available: boolean
  currentVersion: string
  latestVersion: string
  releaseUrl: string
  releaseNotes: string
}

export interface UpdateProgress {
  stage: string    // downloading | installing | restarting | error
  percent: number
  message: string
}

interface AppState {
  version: string
  update: UpdateInfo | null       // null = ještě nezkontrolováno
  dismissed: boolean              // uživatel zavřel notifikaci
  updating: boolean               // probíhá self-update
  progress: UpdateProgress | null // průběh self-update
  updateError: string | null      // chyba self-update
}

function createAppStore() {
  const { subscribe, update } = writable<AppState>({
    version: '',
    update: null,
    dismissed: false,
    updating: false,
    progress: null,
    updateError: null,
  })

  // Poslouchej průběh self-update z Go backendu
  EventsOn('update:progress', (p: UpdateProgress) => {
    update(s => ({
      ...s,
      progress: p,
      updateError: p.stage === 'error' ? p.message : s.updateError,
      updating: p.stage !== 'error',
    }))
  })

  // Načte verzi z Go backendu (vloženou při buildu z git tagu)
  async function init() {
    try {
      const version = await GetVersion()
      update(s => ({ ...s, version }))
    } catch (e) {
      console.error('Nepodařilo se načíst verzi aplikace:', e)
    }
  }

  // Zkontroluje aktualizaci na GitHubu (public repo, bez tokenu)
  async function checkUpdate() {
    try {
      const info = await CheckUpdate() as UpdateInfo
      update(s => ({ ...s, update: info }))
    } catch (e) {
      // Tichá chyba - update check nesmí rušit běh aplikace
      console.error('Kontrola aktualizace selhala:', e)
    }
  }

  // Otevře stránku s vydáním v prohlížeči
  function openRelease() {
    const state = get({ subscribe })
    OpenReleasePage(state.update?.releaseUrl || '')
  }

  // Otevře libovolné URL v systémovém prohlížeči (přes Go backend)
  function openUrl(url: string) {
    OpenReleasePage(url)
  }

  // Spustí automatickou aktualizaci (stáhne + nahradí + restartuje).
  // Při selhání nastaví updateError; volající pak může nabídnout ruční stažení.
  async function performUpdate() {
    update(s => ({ ...s, updating: true, updateError: null, progress: { stage: 'downloading', percent: 0, message: 'Zahajuji...' } }))
    try {
      await PerformUpdate()
      // Pokud vše proběhne, aplikace se restartuje - sem se většinou nedostaneme
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      update(s => ({ ...s, updating: false, updateError: msg }))
    }
  }

  // Skryje notifikaci o aktualizaci
  function dismissUpdate() {
    update(s => ({ ...s, dismissed: true }))
  }

  function getVersion(): string {
    return get({ subscribe }).version
  }

  return {
    subscribe,
    init,
    checkUpdate,
    openRelease,
    openUrl,
    performUpdate,
    dismissUpdate,
    getVersion,
  }
}

export const appStore = createAppStore()
