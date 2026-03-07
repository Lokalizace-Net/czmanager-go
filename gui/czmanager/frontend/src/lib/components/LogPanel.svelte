<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { X, Trash2, FolderOpen } from 'lucide-svelte'
  import { GetLogs, GetLogPath } from '../../../wailsjs/go/main/App'
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

  let { onClose }: { onClose?: () => void } = $props()

  let logs = $state<string[]>([])
  let logPath = $state('')
  let logContainer: HTMLDivElement

  async function loadLogs() {
    logs = await GetLogs()
    logPath = await GetLogPath()
    scrollToBottom()
  }

  function scrollToBottom() {
    setTimeout(() => {
      if (logContainer) {
        logContainer.scrollTop = logContainer.scrollHeight
      }
    }, 50)
  }

  function clearLogs() {
    logs = []
  }

  function openLogFolder() {
    // Open folder in explorer
    if (logPath) {
      const folder = logPath.substring(0, logPath.lastIndexOf('\\'))
      window.open('file:///' + folder)
    }
  }

  onMount(() => {
    loadLogs()

    EventsOn('log', (line: string) => {
      logs = [...logs, line]
      scrollToBottom()
    })
  })

  onDestroy(() => {
    EventsOff('log')
  })
</script>

<div class="log-panel">
  <div class="log-header">
    <h3>Debug Log</h3>
    <div class="log-actions">
      <button class="icon-btn" onclick={clearLogs} title="Vymazat logy">
        <Trash2 size={16} />
      </button>
      <button class="icon-btn" onclick={onClose} title="Zavřít">
        <X size={16} />
      </button>
    </div>
  </div>

  <div class="log-content" bind:this={logContainer}>
    {#each logs as log}
      <div class="log-line" class:error={log.includes('ERROR')} class:agent={log.includes('[Agent')}>
        {log}
      </div>
    {/each}
    {#if logs.length === 0}
      <div class="log-empty">Žádné logy</div>
    {/if}
  </div>

  <div class="log-footer">
    <span class="log-path" title={logPath}>{logPath}</span>
  </div>
</div>

<style>
  .log-panel {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 300px;
    background: #0d0d0d;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    z-index: 1000;
  }

  .log-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 16px;
    background: rgba(255, 255, 255, 0.03);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .log-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.7);
  }

  .log-actions {
    display: flex;
    gap: 8px;
  }

  .icon-btn {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    border-radius: 4px;
    color: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .log-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px 16px;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 12px;
  }

  .log-line {
    padding: 2px 0;
    color: rgba(255, 255, 255, 0.6);
    white-space: pre-wrap;
    word-break: break-all;
  }

  .log-line.error {
    color: #f87171;
  }

  .log-line.agent {
    color: #60a5fa;
  }

  .log-empty {
    color: rgba(255, 255, 255, 0.3);
    font-style: italic;
  }

  .log-footer {
    padding: 6px 16px;
    background: rgba(255, 255, 255, 0.02);
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }

  .log-path {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.3);
    font-family: monospace;
  }
</style>
