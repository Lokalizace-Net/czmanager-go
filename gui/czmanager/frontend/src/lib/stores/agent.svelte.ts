// Agent store - the agent now runs in-process inside the GUI (no HTTP, no
// separate binary), so the connection is effectively always available. This
// store is kept with the same shape so existing components keep working.
import { writable, derived, get } from 'svelte/store'
import { GetAgentStatus } from '../../../wailsjs/go/main/App'

export type AgentStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

interface AgentState {
  status: AgentStatus
  version: string | null
  busy: boolean
  error: string | null
}

const initialState: AgentState = {
  status: 'disconnected',
  version: null,
  busy: false,
  error: null
}

function createAgentStore() {
  const { subscribe, set, update } = writable<AgentState>(initialState)

  // connect verifies the in-process installer is ready. Kept async + returning
  // boolean so existing callers (App.svelte, GameDetail.svelte) are unchanged.
  async function connect(): Promise<boolean> {
    update(s => ({ ...s, status: 'connecting', error: null }))
    try {
      const status = await GetAgentStatus()
      if (!status.running) {
        throw new Error('Installer není připraven')
      }
      update(s => ({
        ...s,
        status: 'connected',
        version: status.version,
        busy: status.busy,
        error: null
      }))
      return true
    } catch (err) {
      update(s => ({
        ...s,
        status: 'error',
        error: err instanceof Error ? err.message : 'Instalátor není dostupný'
      }))
      return false
    }
  }

  // refreshStatus updates the busy flag from the backend.
  async function refreshStatus() {
    try {
      const status = await GetAgentStatus()
      update(s => ({ ...s, busy: status.busy, version: status.version }))
      return status
    } catch {
      return null
    }
  }

  return {
    subscribe,
    set,
    connect,
    refreshStatus
  }
}

export const agentStore = createAgentStore()

// Derived stores for convenience
export const isConnected = derived(agentStore, $agent => $agent.status === 'connected')
