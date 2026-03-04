<script lang="ts">
  import type { LocalizationStatus } from '../stores/games.svelte'

  export let status: LocalizationStatus

  const statusConfig: Record<LocalizationStatus, { label: string; colorClass: string }> = {
    draft: {
      label: 'Příprava',
      colorClass: 'status-draft'
    },
    translating: {
      label: 'Překládá se',
      colorClass: 'status-translating'
    },
    released: {
      label: 'Veřejná verze',
      colorClass: 'status-released'
    },
    beta: {
      label: 'Open Beta',
      colorClass: 'status-beta'
    },
    wip: {
      label: 'Rozpracováno',
      colorClass: 'status-wip'
    }
  }

  $: config = statusConfig[status] || statusConfig.wip
</script>

<span class="status-badge {config.colorClass}">
  {config.label}
</span>

<style>
  .status-badge {
    display: inline-block;
    padding: 4px 10px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-radius: 6px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  }

  .status-draft {
    background: #475569;
    color: white;
  }

  .status-translating {
    background: #ef4444;
    color: white;
  }

  .status-released {
    background: #10b981;
    color: white;
  }

  .status-beta {
    background: #f59e0b;
    color: black;
  }

  .status-wip {
    background: #64748b;
    color: white;
  }
</style>
