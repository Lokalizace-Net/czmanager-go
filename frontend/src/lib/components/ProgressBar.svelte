<script lang="ts">
  export let value: number
  export let max = 100
  export let label = ''
  export let showPercentage = true
  export let variant: 'default' | 'success' | 'error' = 'default'

  $: percentage = Math.min(100, Math.max(0, (value / max) * 100))
</script>

<div class="progress-wrapper">
  {#if label || showPercentage}
    <div class="progress-header">
      {#if label}
        <span class="progress-label">{label}</span>
      {/if}
      {#if showPercentage}
        <span class="progress-percentage">{Math.round(percentage)}%</span>
      {/if}
    </div>
  {/if}

  <div class="progress-track">
    <div
      class="progress-fill {variant}"
      style="width: {percentage}%"
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
    ></div>
  </div>
</div>

<style>
  .progress-wrapper {
    width: 100%;
  }

  .progress-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
    font-size: 14px;
  }

  .progress-label {
    color: rgba(255, 255, 255, 0.5);
  }

  .progress-percentage {
    color: white;
    font-weight: 500;
  }

  .progress-track {
    width: 100%;
    height: 8px;
    background: #2a2a2a;
    border-radius: 9999px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    border-radius: 9999px;
    transition: all 0.3s ease-out;
  }

  .progress-fill.default {
    background: #f97316;
  }

  .progress-fill.success {
    background: #22c55e;
  }

  .progress-fill.error {
    background: #ef4444;
  }
</style>
