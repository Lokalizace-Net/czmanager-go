<script lang="ts">
  import { navigationStore } from '../stores/navigation.svelte'

  export let open = false
  export let onClose: (() => void) | undefined = undefined

  function handleClose() {
    onClose?.()
    navigationStore.setModalOpen(false)
  }

  function handleBackdropClick(event: MouseEvent) {
    if (event.target === event.currentTarget) {
      handleClose()
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      handleClose()
    }
  }

  $: if (open) {
    navigationStore.setModalOpen(true)
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
</script>

<svelte:window on:keydown={open ? handleKeydown : undefined} />

{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    on:click={handleBackdropClick}
  >
    <div class="modal-content" role="dialog" aria-modal="true">
      <slot />
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 50;
    display: flex;
    align-items: stretch;
    justify-content: center;
    padding: 40px 10%;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(2px);
  }

  .modal-content {
    width: 100%;
    height: 100%;
    overflow-y: auto;
    border-radius: 16px;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    animation: fade-in 0.2s ease-out, zoom-in 0.2s ease-out;
  }

  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes zoom-in {
    from { transform: scale(0.95); }
    to { transform: scale(1); }
  }
</style>
