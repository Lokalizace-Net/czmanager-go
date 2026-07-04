// App store - informace o aplikaci (verze + auto-update z GitHubu)
import { writable, get } from 'svelte/store'
import { GetVersion, CheckUpdate, OpenReleasePage } from '../../../wailsjs/go/main/App'

export interface UpdateInfo {
  available: boolean
  currentVersion: string
  latestVersion: string
  releaseUrl: string
  releaseNotes: string
}

interface AppState {
  version: string
  update: UpdateInfo | null   // null = ještě nezkontrolováno
  dismissed: boolean          // uživatel zavřel notifikaci
}

function createAppStore() {
  const { subscribe, update } = writable<AppState>({
    version: '',
    update: null,
    dismissed: false,
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
    dismissUpdate,
    getVersion,
  }
}

export const appStore = createAppStore()
