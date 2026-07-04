// App store - informace o aplikaci (verze, později auto-update stav)
import { writable, get } from 'svelte/store'
import { GetVersion } from '../../../wailsjs/go/main/App'

interface AppState {
  version: string
}

function createAppStore() {
  const { subscribe, update } = writable<AppState>({
    version: '',
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

  function getVersion(): string {
    return get({ subscribe }).version
  }

  return {
    subscribe,
    init,
    getVersion,
  }
}

export const appStore = createAppStore()
