// Favorites store - synchronizace oblíbených lokalizací s API
import { writable, derived, get } from 'svelte/store'
import { type Localization } from './games.svelte'
import { authStore } from './auth.svelte'
import { debugLog } from './app.svelte'
// @ts-ignore - bindings se generují dynamicky
import { FetchFavorites, AddFavorite, RemoveFavorite } from '../../../wailsjs/go/main/App'

const STORAGE_KEY = 'czmanager_favorites'

interface FavoritesState {
  ids: number[]
  games: FavoriteGame[]
  loading: boolean
  limitError: string | null
}

export interface FavoriteGame {
  id: number
  name: string
  slug: string
  thumbnail: string | null
  status: string
  translatePercent: number
  teamName: string
  teamSlug: string
}

function createFavoritesStore() {
  const cachedIds = loadCacheFromStorage()

  const { subscribe, set, update } = writable<FavoritesState>({
    ids: cachedIds,
    games: [],
    loading: false,
    limitError: null
  })

  function loadCacheFromStorage(): number[] {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) return JSON.parse(stored)
    } catch (e) {
      console.error('Failed to load favorites cache:', e)
    }
    return []
  }

  function saveCacheToStorage(ids: number[]) {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(ids))
    } catch (e) {
      console.error('Failed to save favorites cache:', e)
    }
  }

  // Načtení oblíbených z API
  async function fetchFromApi() {
    const auth = get(authStore)
    console.log('[FAV] fetchFromApi called, hasToken:', !!auth.accessToken)
    if (!auth.accessToken) return

    update(s => ({ ...s, loading: true }))

    try {
      const data = await FetchFavorites(auth.accessToken)
      console.log('[FAV] API response:', JSON.stringify(data))
      const favorites = (data.favorites as FavoriteGame[]) || []
      const ids = favorites.map(f => f.id)
      console.log('[FAV] Parsed favorites:', favorites.length, 'ids:', ids)

      update(s => ({
        ...s,
        ids,
        games: favorites,
        loading: false,
        limitError: null
      }))

      saveCacheToStorage(ids)
    } catch (e) {
      console.error('[FAV] Failed to fetch favorites:', e)
      update(s => ({ ...s, loading: false }))
    }
  }

  // Toggle oblíbené přes API
  async function toggleFavorite(gameId: number) {
    const auth = get(authStore)
    if (!auth.user || !auth.accessToken) return

    // Optimistický update
    const state = get({ subscribe })
    const wasFavorite = state.ids.includes(gameId)

    if (wasFavorite) {
      debugLog(`Odebráno z oblíbených (hra #${gameId})`)
      const newIds = state.ids.filter(id => id !== gameId)
      const newGames = state.games.filter(g => g.id !== gameId)
      update(s => ({ ...s, ids: newIds, games: newGames, limitError: null }))
      saveCacheToStorage(newIds)
    } else {
      debugLog(`Přidáno do oblíbených (hra #${gameId})`)
      const newIds = [...state.ids, gameId]
      update(s => ({ ...s, ids: newIds, limitError: null }))
      saveCacheToStorage(newIds)
    }

    try {
      if (wasFavorite) {
        await RemoveFavorite(auth.accessToken, gameId)
      } else {
        await AddFavorite(auth.accessToken, gameId)
        // Refetchni pro kompletní data hry
        await fetchFromApi()
      }
    } catch (e: any) {
      // Rollback optimistického updatu
      const errorMsg = typeof e === 'string' ? e : (e?.message || 'Chyba při změně oblíbených')

      if (wasFavorite) {
        update(s => ({ ...s, ids: [...s.ids, gameId], limitError: null }))
      } else {
        update(s => ({
          ...s,
          ids: s.ids.filter(id => id !== gameId),
          limitError: errorMsg
        }))
      }

      saveCacheToStorage(get({ subscribe }).ids)
    }
  }

  function clearLimitError() {
    update(s => ({ ...s, limitError: null }))
  }

  function clear() {
    set({ ids: [], games: [], loading: false, limitError: null })
    localStorage.removeItem(STORAGE_KEY)
  }

  return {
    subscribe,
    fetchFromApi,
    toggleFavorite,
    clearLimitError,
    clear
  }
}

export const favoritesStore = createFavoritesStore()

// Derived store - oblíbené lokalizace přímo z favorites API
export const favoriteLocalizations = derived(
  favoritesStore,
  ($favorites) => {
    return $favorites.games.map(g => ({
      id: g.id,
      name: g.name,
      slug: g.slug,
      description: '',
      imageUrl: g.thumbnail ? `https://lokalizace.net${g.thumbnail}` : '',
      status: mapApiStatus(g.status),
      version: '',
      teamName: g.teamName,
      translatePercent: g.translatePercent || 0
    } as Localization))
  }
)

function mapApiStatus(status: string): 'translating' | 'released' | 'beta' | 'wip' | 'draft' {
  const map: Record<string, 'translating' | 'released' | 'beta' | 'wip' | 'draft'> = {
    'draft': 'draft',
    'translating': 'translating',
    'alpha': 'wip',
    'open_beta': 'beta',
    'public': 'released'
  }
  return map[status?.toLowerCase()] || 'wip'
}
