// Auth store pro správu přihlášení a tokenů
import { writable, get } from 'svelte/store'

// Import Wails bindings - budou dostupné po wails generate
// @ts-ignore - bindings se generují dynamicky
import { Login, RefreshToken, FetchSubscription } from '../../../wailsjs/go/main/App'

export interface User {
  id: number
  username: string
  role: 'user' | 'moderator' | 'admin'
  avatar?: string | null
  email?: string
}

export interface SubscriptionFeatures {
  maxProjects: number
  maxFavorites: number
  hasAds: boolean
  hasGameScanner: boolean
  hasCustomSignature: boolean
  hasBadge: boolean
  hasDiscordRole: boolean
  hasWebBot: boolean
  hasDiscordInstall: boolean
  hasRemoteSupport: boolean
  canSuggestProject: boolean
}

export interface Subscription {
  id: number
  tierSlug: 'supporter' | 'vip'
  status: string
  expiresAt: string
}

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  expiresAt: string | null
  refreshExpiresAt: string | null
  features: SubscriptionFeatures | null
  subscription: Subscription | null
  isLoading: boolean
  error: string | null
}

const STORAGE_KEY = 'czmanager_auth'

function createAuthStore() {
  // Načti uložený stav z localStorage
  const savedState = loadFromStorage()

  const { subscribe, set, update } = writable<AuthState>({
    user: savedState?.user || null,
    accessToken: savedState?.accessToken || null,
    refreshToken: savedState?.refreshToken || null,
    expiresAt: savedState?.expiresAt || null,
    refreshExpiresAt: savedState?.refreshExpiresAt || null,
    features: null,
    subscription: null,
    isLoading: false,
    error: null
  })

  function loadFromStorage(): Partial<AuthState> | null {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        return JSON.parse(stored)
      }
    } catch (e) {
      console.error('Failed to load auth from storage:', e)
    }
    return null
  }

  function saveToStorage(state: AuthState) {
    try {
      const toSave = {
        user: state.user,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        expiresAt: state.expiresAt,
        refreshExpiresAt: state.refreshExpiresAt
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave))
    } catch (e) {
      console.error('Failed to save auth to storage:', e)
    }
  }

  function clearStorage() {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch (e) {
      console.error('Failed to clear auth storage:', e)
    }
  }

  // Login - volá Go backend přes Wails
  async function login(username: string, password: string): Promise<boolean> {
    update(s => ({ ...s, isLoading: true, error: null }))

    try {
      // Volání Go funkce přes Wails
      const data = await Login(username, password)

      const newState: AuthState = {
        user: data.user,
        accessToken: data.accessToken,
        refreshToken: data.refreshToken,
        expiresAt: data.expiresAt,
        refreshExpiresAt: data.refreshExpiresAt,
        features: null,
        subscription: null,
        isLoading: false,
        error: null
      }

      set(newState)
      saveToStorage(newState)

      // Načti subscription info
      await fetchSubscription()

      return true
    } catch (e: any) {
      console.error('Login error:', e)
      // Go vrací error message přímo
      const errorMsg = typeof e === 'string' ? e : (e?.message || 'Chyba připojení k serveru')
      update(s => ({ ...s, isLoading: false, error: errorMsg }))
      return false
    }
  }

  // Logout - pouze lokální vymazání (JWT nepotřebuje server-side logout)
  function logout() {
    set({
      user: null,
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      refreshExpiresAt: null,
      features: null,
      subscription: null,
      isLoading: false,
      error: null
    })
    clearStorage()
  }

  // Refresh token - volá Go backend přes Wails
  async function refreshAccessToken(): Promise<boolean> {
    const state = get({ subscribe })

    if (!state.refreshToken) {
      return false
    }

    try {
      const data = await RefreshToken(state.refreshToken)

      update(s => {
        const newState = {
          ...s,
          user: data.user,
          accessToken: data.accessToken,
          refreshToken: data.refreshToken,
          expiresAt: data.expiresAt,
          refreshExpiresAt: data.refreshExpiresAt
        }
        saveToStorage(newState)
        return newState
      })

      return true
    } catch (e) {
      console.error('Token refresh error:', e)
      // Refresh token neplatný - odhlásit
      logout()
      return false
    }
  }

  // Zkontroluj a případně obnov token
  async function ensureValidToken(): Promise<string | null> {
    const state = get({ subscribe })

    if (!state.accessToken || !state.expiresAt) {
      return null
    }

    // Zkontroluj jestli token brzy nevyprší (5 minut předem)
    const expiresAt = new Date(state.expiresAt)
    const now = new Date()
    const fiveMinutes = 5 * 60 * 1000

    if (expiresAt.getTime() - now.getTime() < fiveMinutes) {
      const refreshed = await refreshAccessToken()
      if (!refreshed) {
        return null
      }
      return get({ subscribe }).accessToken
    }

    return state.accessToken
  }

  // Načti subscription info - volá Go backend přes Wails
  async function fetchSubscription() {
    const state = get({ subscribe })

    if (!state.accessToken) {
      return
    }

    try {
      const data = await FetchSubscription(state.accessToken)
      update(s => ({
        ...s,
        features: data.features as SubscriptionFeatures,
        subscription: data.subscription as Subscription
      }))
    } catch (e) {
      console.error('Failed to fetch subscription:', e)
    }
  }

  // Inicializace - zkontroluj uložený token
  async function init() {
    const state = get({ subscribe })

    if (state.accessToken) {
      // Zkus obnovit token a načíst subscription
      const valid = await ensureValidToken()
      if (valid) {
        await fetchSubscription()
      }
    }
  }

  return {
    subscribe,
    login,
    logout,
    refreshAccessToken,
    ensureValidToken,
    fetchSubscription,
    init,
    clearError: () => update(s => ({ ...s, error: null }))
  }
}

export const authStore = createAuthStore()

// Derived stores pro snadný přístup
export const isLoggedIn = {
  subscribe: (fn: (value: boolean) => void) => {
    return authStore.subscribe(state => fn(!!state.user))
  }
}

export const isVip = {
  subscribe: (fn: (value: boolean) => void) => {
    return authStore.subscribe(state => fn(state.subscription?.tierSlug === 'vip'))
  }
}

export const isSupporter = {
  subscribe: (fn: (value: boolean) => void) => {
    return authStore.subscribe(state => fn(
      state.subscription?.tierSlug === 'supporter' ||
      state.subscription?.tierSlug === 'vip'
    ))
  }
}

export const hasGameScanner = {
  subscribe: (fn: (value: boolean) => void) => {
    return authStore.subscribe(state => fn(state.features?.hasGameScanner || false))
  }
}
