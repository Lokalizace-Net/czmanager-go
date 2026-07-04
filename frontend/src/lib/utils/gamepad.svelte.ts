// Gamepad support for Steam Deck and controllers
import { writable, get } from 'svelte/store'
import { navigationStore } from '../stores/navigation.svelte'

interface GamepadState {
  connected: boolean
  lastInput: number
}

const DEADZONE = 0.5
const REPEAT_DELAY = 500
const REPEAT_RATE = 100

function createGamepadHandler() {
  const { subscribe, set, update } = writable<GamepadState>({
    connected: false,
    lastInput: 0
  })

  let animationFrame: number | null = null
  let lastDirection: string | null = null
  let directionHeldSince: number = 0

  function start() {
    if (animationFrame !== null) return

    window.addEventListener('gamepadconnected', onConnect)
    window.addEventListener('gamepaddisconnected', onDisconnect)

    const gamepads = navigator.getGamepads()
    for (const gp of gamepads) {
      if (gp) {
        update(s => ({ ...s, connected: true }))
        break
      }
    }

    pollGamepad()
  }

  function stop() {
    if (animationFrame !== null) {
      cancelAnimationFrame(animationFrame)
      animationFrame = null
    }

    window.removeEventListener('gamepadconnected', onConnect)
    window.removeEventListener('gamepaddisconnected', onDisconnect)
  }

  function onConnect(event: GamepadEvent) {
    console.log('Gamepad connected:', event.gamepad.id)
    update(s => ({ ...s, connected: true }))
  }

  function onDisconnect(event: GamepadEvent) {
    console.log('Gamepad disconnected:', event.gamepad.id)
    const gamepads = navigator.getGamepads()
    const anyConnected = gamepads.some(gp => gp !== null)
    update(s => ({ ...s, connected: anyConnected }))
  }

  function pollGamepad() {
    const gamepads = navigator.getGamepads()
    const now = Date.now()
    const state = get({ subscribe })

    for (const gamepad of gamepads) {
      if (!gamepad) continue

      const leftX = gamepad.axes[0] || 0
      const leftY = gamepad.axes[1] || 0

      const dpadUp = gamepad.buttons[12]?.pressed
      const dpadDown = gamepad.buttons[13]?.pressed
      const dpadLeft = gamepad.buttons[14]?.pressed
      const dpadRight = gamepad.buttons[15]?.pressed

      let direction: 'up' | 'down' | 'left' | 'right' | null = null

      if (dpadUp || leftY < -DEADZONE) direction = 'up'
      else if (dpadDown || leftY > DEADZONE) direction = 'down'
      else if (dpadLeft || leftX < -DEADZONE) direction = 'left'
      else if (dpadRight || leftX > DEADZONE) direction = 'right'

      if (direction) {
        if (direction !== lastDirection) {
          navigationStore.moveFocus(direction)
          lastDirection = direction
          directionHeldSince = now
        } else if (now - directionHeldSince > REPEAT_DELAY) {
          if (now - state.lastInput > REPEAT_RATE) {
            navigationStore.moveFocus(direction)
            update(s => ({ ...s, lastInput: now }))
          }
        }
      } else {
        lastDirection = null
      }

      const buttonA = gamepad.buttons[0]?.pressed
      if (buttonA && now - state.lastInput > 200) {
        navigationStore.selectCurrent()
        update(s => ({ ...s, lastInput: now }))
      }

      const buttonB = gamepad.buttons[1]?.pressed
      if (buttonB && now - state.lastInput > 200) {
        navigationStore.goBack()
        update(s => ({ ...s, lastInput: now }))
      }

      const buttonStart = gamepad.buttons[9]?.pressed
      if (buttonStart && now - state.lastInput > 200) {
        navigationStore.setView('settings')
        update(s => ({ ...s, lastInput: now }))
      }

      const triggerLeft = gamepad.buttons[6]?.value || 0
      const triggerRight = gamepad.buttons[7]?.value || 0

      if (triggerLeft > 0.5 && now - state.lastInput > 50) {
        window.scrollBy({ top: -100, behavior: 'smooth' })
        update(s => ({ ...s, lastInput: now }))
      }
      if (triggerRight > 0.5 && now - state.lastInput > 50) {
        window.scrollBy({ top: 100, behavior: 'smooth' })
        update(s => ({ ...s, lastInput: now }))
      }
    }

    animationFrame = requestAnimationFrame(pollGamepad)
  }

  return {
    subscribe,
    start,
    stop
  }
}

export const gamepadHandler = createGamepadHandler()
