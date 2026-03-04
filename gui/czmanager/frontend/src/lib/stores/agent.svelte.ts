// Agent store - manages connection to the local agent
import { writable, derived, get } from 'svelte/store'

const AGENT_URL = 'http://127.0.0.1:17892'

export type AgentStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

interface AgentState {
  status: AgentStatus
  token: string | null
  version: string | null
  busy: boolean
  error: string | null
}

const initialState: AgentState = {
  status: 'disconnected',
  token: null,
  version: null,
  busy: false,
  error: null
}

function createAgentStore() {
  const { subscribe, set, update } = writable<AgentState>(initialState)

  let pingInterval: ReturnType<typeof setInterval> | null = null

  async function connect(): Promise<boolean> {
    update(s => ({ ...s, status: 'connecting', error: null }))

    try {
      const response = await fetch(`${AGENT_URL}/ping`, {
        method: 'GET',
        signal: AbortSignal.timeout(5000)
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      const data = await response.json()
      update(s => ({
        ...s,
        token: data.token,
        version: data.version,
        status: 'connected'
      }))

      // Start health check
      startHealthCheck()

      return true
    } catch (err) {
      update(s => ({
        ...s,
        status: 'error',
        error: err instanceof Error ? err.message : 'Nepodařilo se připojit k agentovi'
      }))
      return false
    }
  }

  function startHealthCheck() {
    if (pingInterval) {
      clearInterval(pingInterval)
    }

    pingInterval = setInterval(async () => {
      try {
        const response = await fetch(`${AGENT_URL}/ping`, {
          method: 'GET',
          signal: AbortSignal.timeout(3000)
        })

        if (!response.ok) {
          throw new Error('Agent not responding')
        }

        update(s => {
          if (s.status !== 'connected') {
            return { ...s, status: 'connected', error: null }
          }
          return s
        })
      } catch {
        update(s => ({
          ...s,
          status: 'error',
          error: 'Spojení s agentem bylo přerušeno'
        }))
      }
    }, 10000)
  }

  function stopHealthCheck() {
    if (pingInterval) {
      clearInterval(pingInterval)
      pingInterval = null
    }
  }

  async function fetchWithAuth<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const state = get({ subscribe })
    if (!state.token) {
      throw new Error('Not connected to agent')
    }

    const response = await fetch(`${AGENT_URL}${endpoint}`, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${state.token}`,
        'Content-Type': 'application/json'
      }
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: `HTTP ${response.status}` }))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    return response.json()
  }

  async function getStatus() {
    const data = await fetchWithAuth<{ running: boolean; version: string; busy: boolean }>('/status')
    update(s => ({
      ...s,
      busy: data.busy,
      version: data.version
    }))
    return data
  }

  return {
    subscribe,
    connect,
    stopHealthCheck,
    fetchWithAuth,
    getStatus
  }
}

export const agentStore = createAgentStore()

// Derived stores for convenience
export const isConnected = derived(agentStore, $agent => $agent.status === 'connected')
export const agentToken = derived(agentStore, $agent => $agent.token)
