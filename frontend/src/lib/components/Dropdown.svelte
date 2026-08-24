<script lang="ts">
  import { onDestroy, tick } from 'svelte'
  import { ChevronDown, Check, Crown } from 'lucide-svelte'
  import { focusStore } from '../stores/focus.svelte'

  export interface DropdownOption {
    value: number | string
    label: string        // hlavní text (např. verze)
    detail?: string      // doplňkový text (velikost, datum)
    disabled?: boolean   // nelze vybrat (např. VIP bez nároku)
    badge?: 'vip'        // volitelný štítek
  }

  let {
    options = [],
    value = $bindable(),
    disabled = false,
    placeholder = 'Vyberte...',
    /** ID focus zóny, do které se má vrátit fokus po zavření */
    returnZone = 'modal',
  }: {
    options?: DropdownOption[]
    value?: number | string | null
    disabled?: boolean
    placeholder?: string
    returnZone?: string
  } = $props()

  let open = $state(false)
  let triggerEl = $state<HTMLButtonElement | undefined>(undefined)
  let listEl = $state<HTMLElement | undefined>(undefined)

  let selected = $derived(options.find(o => o.value === value) ?? null)

  // Registruje položky seznamu jako vlastní focus zónu, aby šly ovládat
  // D-padem/šipkami. Escape/B zavře a vrátí fokus na trigger.
  async function registerListZone() {
    await tick()
    if (!listEl) return
    const items = Array.from(listEl.querySelectorAll('button:not(:disabled)')) as HTMLElement[]

    focusStore.registerZone({
      id: 'dropdown',
      elements: items,
      columns: 1,
      loop: true,
      onEscape: () => close()
    })
    focusStore.setActiveZone('dropdown', false)

    // Předvyber aktuálně zvolenou položku
    const idx = items.findIndex(el => el.dataset.value === String(value))
    focusStore.setFocusedIndex(idx >= 0 ? idx : 0)
  }

  function toggle() {
    if (disabled) return
    open ? close() : openList()
  }

  function openList() {
    open = true
    registerListZone()
  }

  function close() {
    open = false
    focusStore.unregisterZone('dropdown')
    focusStore.setActiveZone(returnZone, false)
    // Vrať fokus na trigger, ať navigace pokračuje odsud
    setTimeout(() => triggerEl?.focus(), 20)
  }

  function pick(option: DropdownOption) {
    if (option.disabled) return
    value = option.value
    close()
  }

  onDestroy(() => {
    if (open) focusStore.unregisterZone('dropdown')
  })
</script>

<div class="dropdown">
  <button
    bind:this={triggerEl}
    class="dropdown-trigger"
    class:open
    {disabled}
    onclick={toggle}
    aria-haspopup="listbox"
    aria-expanded={open}
  >
    <span class="trigger-text">
      {#if selected}
        {#if selected.badge === 'vip'}<Crown size={14} />{/if}
        <span class="trigger-label">{selected.label}</span>
        {#if selected.detail}<span class="trigger-detail">{selected.detail}</span>{/if}
      {:else}
        <span class="trigger-placeholder">{placeholder}</span>
      {/if}
    </span>
    <ChevronDown size={18} class="chevron" />
  </button>

  {#if open}
    <!-- Kliknutí mimo zavře seznam -->
    <button class="dropdown-overlay" onclick={close} aria-label="Zavřít výběr"></button>

    <div class="dropdown-list" bind:this={listEl} role="listbox" tabindex="-1">
      {#each options as option (option.value)}
        <button
          class="dropdown-item"
          class:selected={option.value === value}
          class:locked={option.disabled}
          data-value={option.value}
          disabled={option.disabled}
          onclick={() => pick(option)}
          role="option"
          aria-selected={option.value === value}
        >
          <span class="item-main">
            {#if option.badge === 'vip'}<Crown size={13} />{/if}
            <span class="item-label">{option.label}</span>
          </span>
          {#if option.detail}
            <span class="item-detail">{option.detail}</span>
          {/if}
          {#if option.value === value}
            <Check size={16} class="item-check" />
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .dropdown {
    position: relative;
    width: 100%;
  }

  .dropdown-trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    width: 100%;
    height: 44px;
    padding: 0 14px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    font-size: 14px;
    color: white;
    cursor: pointer;
    transition: all 0.2s;
    text-align: left;
  }

  .dropdown-trigger:hover:not(:disabled),
  .dropdown-trigger:focus {
    border-color: #f97316;
    outline: none;
    box-shadow: 0 0 0 2px rgba(249, 115, 22, 0.3);
  }

  .dropdown-trigger:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .dropdown-trigger.open :global(.chevron) {
    transform: rotate(180deg);
  }

  .dropdown-trigger :global(.chevron) {
    flex-shrink: 0;
    color: rgba(255, 255, 255, 0.4);
    transition: transform 0.2s;
  }

  .trigger-text {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    overflow: hidden;
    color: #fbbf24;
  }

  .trigger-label {
    color: white;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .trigger-detail {
    color: rgba(255, 255, 255, 0.4);
    font-size: 13px;
    white-space: nowrap;
  }

  .trigger-placeholder {
    color: rgba(255, 255, 255, 0.3);
  }

  /* Neviditelná vrstva pro zavření kliknutím mimo */
  .dropdown-overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: transparent;
    border: none;
    cursor: default;
  }

  .dropdown-list {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 101;
    max-height: 280px;
    overflow-y: auto;
    padding: 6px;
    background: #1f1f1f;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 10px;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.6);
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 12px;
    background: transparent;
    border: 2px solid transparent;
    border-radius: 8px;
    font-size: 14px;
    color: rgba(255, 255, 255, 0.85);
    cursor: pointer;
    text-align: left;
    transition: all 0.15s;
  }

  .dropdown-item:hover:not(:disabled),
  .dropdown-item:focus {
    background: rgba(255, 255, 255, 0.07);
    outline: none;
    border-color: #f97316;
    color: white;
  }

  .dropdown-item.selected {
    background: rgba(249, 115, 22, 0.12);
    color: white;
  }

  .dropdown-item.locked {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .item-main {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    color: #fbbf24;
  }

  .item-label {
    color: inherit;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dropdown-item .item-label {
    color: white;
  }

  .dropdown-item.locked .item-label {
    color: rgba(255, 255, 255, 0.6);
  }

  .item-detail {
    margin-left: auto;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.4);
    white-space: nowrap;
  }

  .dropdown-item :global(.item-check) {
    flex-shrink: 0;
    color: #f97316;
  }
</style>
