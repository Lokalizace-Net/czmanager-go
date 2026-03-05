// Focus management store pro TV/gamepad navigaci
import { writable, get } from 'svelte/store'

export interface FocusZone {
  id: string
  elements: HTMLElement[]
  columns?: number // Pro grid layout
  parent?: string // ID rodičovské zóny
  onEscape?: () => void // Co udělat při Escape
  loop?: boolean // Zacyklit navigaci
}

interface FocusState {
  activeZone: string
  focusedIndex: number
  zones: Map<string, FocusZone>
  history: string[] // Pro návrat zpět
}

function createFocusStore() {
  const { subscribe, set, update } = writable<FocusState>({
    activeZone: 'main',
    focusedIndex: 0,
    zones: new Map(),
    history: []
  })

  // Registruje novou focus zónu
  function registerZone(zone: FocusZone) {
    update(s => {
      const zones = new Map(s.zones)
      zones.set(zone.id, zone)
      return { ...s, zones }
    })
  }

  // Odregistruje zónu
  function unregisterZone(id: string) {
    update(s => {
      const zones = new Map(s.zones)
      zones.delete(id)
      return { ...s, zones }
    })
  }

  // Aktualizuje elementy v zóně
  function updateZoneElements(zoneId: string, elements: HTMLElement[]) {
    update(s => {
      const zones = new Map(s.zones)
      const zone = zones.get(zoneId)
      if (zone) {
        zones.set(zoneId, { ...zone, elements })
      }
      return { ...s, zones }
    })
  }

  // Přepne na jinou zónu
  function setActiveZone(zoneId: string, saveHistory = true) {
    update(s => {
      // Pouze uložíme historii pokud přecházíme DO modalu
      const history = saveHistory && zoneId === 'modal' && s.activeZone !== 'modal'
        ? [...s.history, s.activeZone]
        : s.history
      return {
        ...s,
        activeZone: zoneId,
        focusedIndex: 0,
        history
      }
    })
    focusCurrent()
  }

  // Vrátí se na předchozí zónu (používá se pro Escape/B)
  function goBack() {
    const state = get({ subscribe })
    const zone = state.zones.get(state.activeZone)

    // Pokud jsme v modalu, zavři ho
    if (state.activeZone === 'modal' && zone?.onEscape) {
      zone.onEscape()
      return true
    }

    // Z menu nebo gridu se vrať na hlavní grid
    if (state.activeZone === 'sidemenu') {
      setActiveZone('main', false)
      return true
    }

    // Nebo se vrať v historii (pro modal)
    if (state.history.length > 0) {
      const prevZone = state.history[state.history.length - 1]
      update(s => ({
        ...s,
        activeZone: prevZone,
        focusedIndex: 0,
        history: s.history.slice(0, -1)
      }))
      focusCurrent()
      return true
    }

    return false
  }

  // Nastaví focus index
  function setFocusedIndex(index: number) {
    update(s => {
      const zone = s.zones.get(s.activeZone)
      if (!zone) return s
      const maxIndex = zone.elements.length - 1
      const newIndex = Math.max(0, Math.min(index, maxIndex))
      return { ...s, focusedIndex: newIndex }
    })
    focusCurrent()
  }

  // Pohyb focusu v aktuální zóně
  function moveFocus(direction: 'up' | 'down' | 'left' | 'right'): boolean {
    const state = get({ subscribe })
    const zone = state.zones.get(state.activeZone)

    if (!zone || zone.elements.length === 0) {
      return false
    }

    const cols = zone.columns || 1
    const current = state.focusedIndex
    const total = zone.elements.length
    let next = current

    switch (direction) {
      case 'up':
        next = current - cols
        break
      case 'down':
        next = current + cols
        break
      case 'left':
        next = current - 1
        break
      case 'right':
        next = current + 1
        break
    }

    // Zacyklení nebo omezení
    if (zone.loop) {
      if (next < 0) next = total + next
      if (next >= total) next = next - total
    } else {
      if (next < 0 || next >= total) return false
    }

    setFocusedIndex(next)
    return true
  }

  // Aktivuje aktuální element (klik)
  function selectCurrent() {
    const state = get({ subscribe })
    const zone = state.zones.get(state.activeZone)
    if (!zone) return

    const element = zone.elements[state.focusedIndex]
    if (element) {
      element.click()
    }
  }

  // Nastaví focus na aktuální element
  function focusCurrent() {
    setTimeout(() => {
      const state = get({ subscribe })
      const zone = state.zones.get(state.activeZone)
      if (!zone || zone.elements.length === 0) {
        return
      }

      const index = Math.min(state.focusedIndex, zone.elements.length - 1)
      const element = zone.elements[index]
      if (element) {
        element.focus()
        element.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
      }
    }, 20)
  }

  // Globální keyboard handler
  function handleKeydown(event: KeyboardEvent) {
    // Ignoruj pokud je focus v inputu
    const inInput = event.target instanceof HTMLInputElement ||
                    event.target instanceof HTMLTextAreaElement
    if (inInput) return

    const state = get({ subscribe })
    const currentZone = state.activeZone
    const zone = state.zones.get(currentZone)

    switch (event.key) {
      case 'ArrowUp':
        event.preventDefault()
        moveFocus('up')
        break
      case 'ArrowDown':
        event.preventDefault()
        moveFocus('down')
        break
      case 'ArrowLeft':
        event.preventDefault()
        // Z levého okraje main gridu -> sidemenu
        if (currentZone === 'main' && zone) {
          const cols = zone.columns || 1
          if (state.focusedIndex % cols === 0) {
            setActiveZone('sidemenu', false)
            return
          }
        }
        moveFocus('left')
        break
      case 'ArrowRight':
        event.preventDefault()
        // Ze sidemenu -> main
        if (currentZone === 'sidemenu') {
          setActiveZone('main', false)
          return
        }
        moveFocus('right')
        break
      case 'Enter':
      case ' ':
        // Nechej nativní click na button
        if (event.target instanceof HTMLButtonElement) return
        event.preventDefault()
        selectCurrent()
        break
      case 'Escape':
        event.preventDefault()
        goBack()
        break
    }
  }

  // Gamepad handler (volá se z gamepad utility)
  function handleGamepadInput(button: string) {
    switch (button) {
      case 'dpad_up':
        moveFocus('up')
        break
      case 'dpad_down':
        moveFocus('down')
        break
      case 'dpad_left':
        moveFocus('left')
        break
      case 'dpad_right':
        moveFocus('right')
        break
      case 'a': // Confirm
        selectCurrent()
        break
      case 'b': // Back
        goBack()
        break
    }
  }

  return {
    subscribe,
    registerZone,
    unregisterZone,
    updateZoneElements,
    setActiveZone,
    goBack,
    setFocusedIndex,
    moveFocus,
    selectCurrent,
    focusCurrent,
    handleKeydown,
    handleGamepadInput
  }
}

export const focusStore = createFocusStore()

// Svelte action pro automatickou registraci focusable elementů
export function focusable(node: HTMLElement, zoneId: string) {
  // Přidej do zóny
  const state = get(focusStore)
  const zone = state.zones.get(zoneId)
  if (zone) {
    focusStore.updateZoneElements(zoneId, [...zone.elements, node])
  }

  return {
    destroy() {
      const state = get(focusStore)
      const zone = state.zones.get(zoneId)
      if (zone) {
        focusStore.updateZoneElements(
          zoneId,
          zone.elements.filter(el => el !== node)
        )
      }
    }
  }
}
