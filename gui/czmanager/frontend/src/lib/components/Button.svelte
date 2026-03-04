<script lang="ts">
  import { Loader2 } from 'lucide-svelte'

  export let variant: 'primary' | 'secondary' | 'danger' | 'ghost' = 'primary'
  export let size: 'sm' | 'md' | 'lg' = 'md'
  export let disabled = false
  export let loading = false
  export let onclick: (() => void) | undefined = undefined
</script>

<button
  class="btn btn-{variant} btn-{size}"
  disabled={disabled || loading}
  on:click={onclick}
>
  {#if loading}
    <Loader2 size={16} class="spinning" />
  {/if}
  <slot />
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    border-radius: 8px;
    font-weight: 500;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: #f97316;
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: #ea580c;
  }

  .btn-secondary {
    background: #1e1e1e;
    color: white;
    border: 1px solid #333;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #2a2a2a;
  }

  .btn-danger {
    background: #dc2626;
    color: white;
  }

  .btn-danger:hover:not(:disabled) {
    background: #b91c1c;
  }

  .btn-ghost {
    background: transparent;
    color: white;
  }

  .btn-ghost:hover:not(:disabled) {
    background: #2a2a2a;
  }

  .btn-sm {
    padding: 6px 12px;
    font-size: 14px;
  }

  .btn-md {
    padding: 8px 16px;
    font-size: 14px;
  }

  .btn-lg {
    padding: 12px 24px;
    font-size: 16px;
  }

  :global(.spinning) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
