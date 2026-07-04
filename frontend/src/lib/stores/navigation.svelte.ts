// Navigation store - manages spatial navigation and gamepad input
import { writable, get } from 'svelte/store'

export type View = 'home' | 'game' | 'settings'

interface NavigationState {
  currentView: View
  selectedGameSlug: string | null
  focusedIndex: number
  gridColumns: number
  modalOpen: boolean
}

function createNavigationStore() {
  const { subscribe, set, update } = writable<NavigationState>({
    currentView: 'home',
    selectedGameSlug: null,
    focusedIndex: 0,
    gridColumns: 6,
    modalOpen: false
  })

  let focusableElements: HTMLElement[] = []

  function setView(view: View, gameSlug?: string) {
    update(s => ({
      ...s,
      currentView: view,
      selectedGameSlug: gameSlug || null,
      focusedIndex: 0
    }))
  }

  function goBack() {
    const state = get({ subscribe })
    if (state.modalOpen) {
      update(s => ({ ...s, modalOpen: false }))
      return
    }

    switch (state.currentView) {
      case 'game':
      case 'settings':
        setView('home')
        break
    }
  }

  function setGridColumns(columns: number) {
    update(s => ({ ...s, gridColumns: columns }))
  }

  function setModalOpen(open: boolean) {
    update(s => ({ ...s, modalOpen: open }))
  }

  function registerFocusables(elements: HTMLElement[]) {
    focusableElements = elements
  }

  function moveFocus(direction: 'up' | 'down' | 'left' | 'right') {
    if (focusableElements.length === 0) return

    const state = get({ subscribe })
    const cols = state.gridColumns
    const current = state.focusedIndex
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

    // Clamp to valid range
    if (next >= 0 && next < focusableElements.length) {
      update(s => ({ ...s, focusedIndex: next }))
      focusableElements[next]?.focus()
    }
  }

  function selectCurrent() {
    const state = get({ subscribe })
    focusableElements[state.focusedIndex]?.click()
  }

  function setFocusedIndex(index: number) {
    update(s => ({
      ...s,
      focusedIndex: Math.max(0, Math.min(index, focusableElements.length - 1))
    }))
    const state = get({ subscribe })
    focusableElements[state.focusedIndex]?.focus()
  }

  // Keyboard handler
  function handleKeydown(event: KeyboardEvent) {
    // Don't handle if typing in input
    if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
      if (event.key === 'Escape') {
        (event.target as HTMLElement).blur()
      }
      return
    }

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
        moveFocus('left')
        break
      case 'ArrowRight':
        event.preventDefault()
        moveFocus('right')
        break
      case 'Enter':
      case ' ':
        event.preventDefault()
        selectCurrent()
        break
      case 'Escape':
      case 'Backspace':
        event.preventDefault()
        goBack()
        break
      case '/':
        event.preventDefault()
        document.querySelector<HTMLInputElement>('[data-search-input]')?.focus()
        break
    }
  }

  return {
    subscribe,
    setView,
    goBack,
    setGridColumns,
    setModalOpen,
    registerFocusables,
    moveFocus,
    selectCurrent,
    setFocusedIndex,
    handleKeydown
  }
}

export const navigationStore = createNavigationStore()
