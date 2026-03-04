// Gamepad utility pro Steam Deck / Xbox controller navigaci
import { focusStore } from '../stores/focus.svelte'

interface GamepadState {
  connected: boolean
  lastInput: number
  inputDelay: number
}

const state: GamepadState = {
  connected: false,
  lastInput: 0,
  inputDelay: 150 // ms mezi vstupy
}

let animationFrameId: number | null = null

// Mapování tlačítek pro standardní gamepad (Xbox layout)
const BUTTON_MAP = {
  0: 'a',        // A - Confirm
  1: 'b',        // B - Back
  2: 'x',        // X
  3: 'y',        // Y
  12: 'dpad_up',
  13: 'dpad_down',
  14: 'dpad_left',
  15: 'dpad_right',
  9: 'start',
  8: 'select'
}

function processGamepad() {
  const gamepads = navigator.getGamepads()
  const now = Date.now()

  for (const gamepad of gamepads) {
    if (!gamepad) continue

    // Kontrola tlačítek
    for (let i = 0; i < gamepad.buttons.length; i++) {
      const button = gamepad.buttons[i]
      if (button.pressed && now - state.lastInput > state.inputDelay) {
        const buttonName = BUTTON_MAP[i as keyof typeof BUTTON_MAP]
        if (buttonName) {
          state.lastInput = now
          focusStore.handleGamepadInput(buttonName)
        }
      }
    }

    // Kontrola analogových páček (levá páčka pro navigaci)
    const leftStickX = gamepad.axes[0]
    const leftStickY = gamepad.axes[1]
    const deadzone = 0.5

    if (now - state.lastInput > state.inputDelay) {
      if (leftStickY < -deadzone) {
        state.lastInput = now
        focusStore.handleGamepadInput('dpad_up')
      } else if (leftStickY > deadzone) {
        state.lastInput = now
        focusStore.handleGamepadInput('dpad_down')
      } else if (leftStickX < -deadzone) {
        state.lastInput = now
        focusStore.handleGamepadInput('dpad_left')
      } else if (leftStickX > deadzone) {
        state.lastInput = now
        focusStore.handleGamepadInput('dpad_right')
      }
    }
  }

  animationFrameId = requestAnimationFrame(processGamepad)
}

export function startGamepadPolling() {
  if (animationFrameId !== null) return

  window.addEventListener('gamepadconnected', (e) => {
    console.log('Gamepad připojen:', e.gamepad.id)
    state.connected = true
  })

  window.addEventListener('gamepaddisconnected', () => {
    console.log('Gamepad odpojen')
    state.connected = false
  })

  // Zahájení polling smyčky
  animationFrameId = requestAnimationFrame(processGamepad)
}

export function stopGamepadPolling() {
  if (animationFrameId !== null) {
    cancelAnimationFrame(animationFrameId)
    animationFrameId = null
  }
}
