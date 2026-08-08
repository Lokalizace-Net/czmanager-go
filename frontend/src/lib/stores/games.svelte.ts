// Games store - manages localization data from Lokalizace.NET
import { writable, derived, get } from 'svelte/store'
import { FetchGames } from '../../../wailsjs/go/main/App'
import { debugLog } from './app.svelte'

export type LocalizationStatus = 'translating' | 'released' | 'beta' | 'wip' | 'draft'

export interface Localization {
  id: number
  slug: string
  name: string
  description: string
  imageUrl: string
  heroImageUrl?: string
  status: LocalizationStatus
  version: string
  downloadUrl?: string
  installedVersion?: string
  gamePath?: string
  teamName?: string
  teamSlug?: string
  translatePercent?: number
  correctionPercent?: number
  testingPercent?: number
  rating?: number
  totalRatings?: number
  // Dostupnost: 'web_only' | 'app_only' | 'both' - z API pole "availability"
  availability?: 'web_only' | 'app_only' | 'both'
  supportsAppInstall?: boolean  // zda podporuje instalaci přes CZManager
}

interface GamesState {
  localizations: Localization[]
  loading: boolean
  error: string | null
  searchQuery: string
  page: number
  hasMore: boolean
  total: number
}

const API_BASE = 'https://lokalizace.net'

// Počet lokalizací na stránku. POZOR: API vrací HTTP 500 při limitu >= 90
// (server neustojí větší dávku), proto držíme bezpečnou hodnotu s rezervou.
// Zbytek se donačítá stránkováním (scroll / tlačítko "Načíst další").
const PAGE_SIZE = 50

function createGamesStore() {
  const { subscribe, set, update } = writable<GamesState>({
    localizations: [],
    loading: false,
    error: null,
    searchQuery: '',
    page: 1,
    hasMore: true,
    total: 0
  })

  async function fetchLocalizations(reset = false) {
    const state = get({ subscribe })
    if (state.loading) return

    update(s => ({ ...s, loading: true, error: null }))

    if (reset) {
      update(s => ({ ...s, page: 1, localizations: [], hasMore: true }))
    }

    try {
      const currentState = get({ subscribe })

      // Use Wails backend to fetch games (avoids CORS)
      const data = await FetchGames(currentState.page, PAGE_SIZE, currentState.searchQuery)

      const newLocalizations: Localization[] = (data.games as any[])?.map((item: any) => {
        const status = mapStatus(item.status)
        const translatePercent = item.translatePercent || 0

        // Dostupnost z API: 'web_only', 'app_only', 'both' - default je 'web_only'
        const availability = item.availability || 'web_only'
        // Podporuje přímou instalaci přes CZManager pouze pokud je 'app_only' nebo 'both'
        const supportsAppInstall = availability === 'app_only' || availability === 'both'

        return {
          id: item.id,
          slug: item.slug,
          name: item.name,
          description: item.story || '',
          imageUrl: item.thumbnail ? `${API_BASE}${item.thumbnail}` : `${API_BASE}/uploads/games/${item.id}/thumbnail.webp`,
          heroImageUrl: item.heroImage ? `${API_BASE}${item.heroImage}` : null,
          status,
          version: item.version || '1.0.0',
          downloadUrl: item.downloadUrl || null,
          teamName: item.teamName,
          teamSlug: item.teamSlug,
          translatePercent,
          correctionPercent: item.correctionPercent || 0,
          testingPercent: item.testingPercent || 0,
          rating: item.rating,
          totalRatings: item.totalRatings,
          availability,
          supportsAppInstall
        }
      }) || []

      update(s => {
        const merged = reset ? newLocalizations : [...s.localizations, ...newLocalizations]
        // Naposledy přidané nahoře - vyšší id = novější záznam (auto-increment)
        const sorted = merged.slice().sort((a, b) => b.id - a.id)
        return {
          ...s,
          localizations: sorted,
          hasMore: newLocalizations.length === PAGE_SIZE,
          page: s.page + 1,
          total: (data.total as number) || newLocalizations.length,
          loading: false
        }
      })
    } catch (err) {
      // Chybu ukaž uživateli - dřív se místo ní tiše načetla mock data
      // (Gothic, Legacy of Kain...), což vypadalo, že "nejsou projekty".
      const msg = err instanceof Error ? err.message : String(err)
      console.error('Failed to fetch localizations:', err)
      debugLog(`Načtení lokalizací selhalo: ${msg}`)
      update(s => ({
        ...s,
        error: `Nepodařilo se načíst lokalizace: ${msg}`,
        hasMore: false,
        loading: false
      }))
    }
  }

  function mapStatus(apiStatus: string): LocalizationStatus {
    const statusMap: Record<string, LocalizationStatus> = {
      'draft': 'draft',
      'translating': 'translating',
      'alpha': 'wip',
      'open_beta': 'beta',
      'public': 'released'
    }
    return statusMap[apiStatus?.toLowerCase()] || 'wip'
  }

  function setSearchQuery(query: string) {
    update(s => ({ ...s, searchQuery: query }))
    // Automaticky spustíme nové vyhledávání
    fetchLocalizations(true)
  }

  function getLocalizationBySlug(slug: string): Localization | undefined {
    const state = get({ subscribe })
    return state.localizations.find(loc => loc.slug === slug)
  }

  return {
    subscribe,
    fetchLocalizations,
    setSearchQuery,
    getLocalizationBySlug
  }
}

export const gamesStore = createGamesStore()

// Derived store for filtered localizations
export const filteredLocalizations = derived(gamesStore, $games => {
  if (!$games.searchQuery) {
    return $games.localizations
  }
  const query = $games.searchQuery.toLowerCase()
  return $games.localizations.filter(loc =>
    loc.name.toLowerCase().includes(query) ||
    loc.description?.toLowerCase().includes(query)
  )
})
