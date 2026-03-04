// Games store - manages localization data from Lokalizace.NET
import { writable, derived, get } from 'svelte/store'
import { FetchGames } from '../../../wailsjs/go/main/App'

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
      const data = await FetchGames(currentState.page, 100, currentState.searchQuery)

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

      update(s => ({
        ...s,
        localizations: reset ? newLocalizations : [...s.localizations, ...newLocalizations],
        hasMore: newLocalizations.length === 100,
        page: s.page + 1,
        total: (data.total as number) || newLocalizations.length,
        loading: false
      }))
    } catch (err) {
      console.error('Failed to fetch localizations:', err)
      update(s => ({
        ...s,
        error: err instanceof Error ? err.message : 'Nepodařilo se načíst lokalizace',
        loading: false
      }))

      // Fallback: load mock data for development
      const currentState = get({ subscribe })
      if (currentState.localizations.length === 0) {
        update(s => ({
          ...s,
          localizations: getMockData(),
          hasMore: false,
          error: null
        }))
      }
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

// Mock data for development - fallback when API is not available
function getMockData(): Localization[] {
  const BASE = 'https://lokalizace.net'
  return [
    {
      id: 1,
      slug: 'gothic-ii-gold-edition',
      name: 'Gothic II: Gold Edition',
      description: 'RPG klasika s českým dabingem od Piranha Bytes',
      imageUrl: `${BASE}/uploads/games/gothic-ii-gold-edition/thumbnail.webp`,
      status: 'released',
      version: '3.0.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 100,
      correctionPercent: 100,
      testingPercent: 100
    },
    {
      id: 2,
      slug: 'legacy-of-kain-defiance',
      name: 'Legacy of Kain: Defiance',
      description: 'Akční adventura z temného světa Legacy of Kain',
      imageUrl: `${BASE}/uploads/games/legacy-of-kain-defiance/thumbnail.webp`,
      status: 'translating',
      version: '2.1.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 75,
      correctionPercent: 50,
      testingPercent: 20
    },
    {
      id: 3,
      slug: 'julia-among-the-stars',
      name: 'J.U.L.I.A.: Among the Stars',
      description: 'Sci-fi adventura od českého studia CBE software',
      imageUrl: `${BASE}/uploads/games/julia-among-the-stars/thumbnail.webp`,
      status: 'released',
      version: '1.5.0',
      teamName: 'CBE Software',
      translatePercent: 100,
      correctionPercent: 100,
      testingPercent: 100
    },
    {
      id: 4,
      slug: 'metal-gear-solid-master-collection-vol-1',
      name: 'Metal Gear Solid: Master Collection',
      description: 'Legendární stealth série od Hideo Kojimy',
      imageUrl: `${BASE}/uploads/games/metal-gear-solid-master-collection-vol-1/thumbnail.webp`,
      status: 'beta',
      version: '0.9.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 90,
      correctionPercent: 70,
      testingPercent: 40
    },
    {
      id: 5,
      slug: 'gothic',
      name: 'Gothic',
      description: 'První díl kultovní RPG série',
      imageUrl: `${BASE}/uploads/games/gothic/thumbnail.webp`,
      status: 'released',
      version: '2.0.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 100,
      correctionPercent: 100,
      testingPercent: 100
    },
    {
      id: 6,
      slug: 'soul-reaver',
      name: 'Legacy of Kain: Soul Reaver',
      description: 'Akční adventura s Razielem',
      imageUrl: `${BASE}/uploads/games/soul-reaver/thumbnail.webp`,
      status: 'translating',
      version: '1.0.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 60,
      correctionPercent: 30,
      testingPercent: 10
    },
    {
      id: 7,
      slug: 'soul-reaver-2',
      name: 'Legacy of Kain: Soul Reaver 2',
      description: 'Pokračování příběhu Raziela',
      imageUrl: `${BASE}/uploads/games/soul-reaver-2/thumbnail.webp`,
      status: 'wip',
      version: '0.5.0',
      teamName: 'Lokalizace.NET',
      translatePercent: 20,
      correctionPercent: 5,
      testingPercent: 0
    }
  ]
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
