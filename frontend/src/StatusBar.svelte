<script lang="ts">
  export let indexing: boolean = false
  export let reindexing: boolean = false
  export let watcherActive: boolean = false
  export let indexedCount: number = 0
  export let totalFiles: number = 0
  export let currentFile: string = ''

  $: percent = totalFiles > 0 ? Math.round((indexedCount / totalFiles) * 100) : 0
  $: shortName = currentFile ? currentFile.split('/').pop() ?? currentFile : ''
</script>

{#if indexing || reindexing || watcherActive}
  <div class="status-bar" class:active={indexing || reindexing}>
    <div class="bar-left">
      {#if indexing}
        <span class="pulse" />
        <span class="label">Indexing</span>
        {#if currentFile}
          <span class="filename" title={currentFile}>{shortName}</span>
        {/if}
        <span class="count">{indexedCount} chunks</span>
        {#if totalFiles > 0}
          <span class="sep">·</span>
          <span class="files">{percent}% of {totalFiles} file{totalFiles !== 1 ? 's' : ''}</span>
        {/if}
      {:else if reindexing}
        <span class="pulse reindex" />
        <span class="label">Re-indexing</span>
        {#if currentFile}
          <span class="filename" title={currentFile}>{shortName}</span>
        {/if}
      {:else if watcherActive}
        <span class="dot watching" />
        <span class="label quiet">Watching for changes</span>
      {/if}
    </div>

    {#if indexing && totalFiles > 0}
      <div class="track">
        <div class="fill" style="width:{percent}%" />
      </div>
    {/if}
  </div>
{/if}

<style>
  .status-bar {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 28px;
    background: rgba(10, 20, 35, 0.92);
    border-top: 1px solid rgba(100, 181, 246, 0.15);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0 1rem;
    font-size: 0.78rem;
    color: #78909c;
    z-index: 500;
  }

  .bar-left {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    flex: 1;
    min-width: 0;
  }

  .label {
    font-weight: 600;
    color: #90caf9;
    white-space: nowrap;
  }

  .label.quiet {
    color: #546e7a;
    font-weight: 400;
  }

  .filename {
    color: #b0bec5;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 260px;
  }

  .count, .files {
    white-space: nowrap;
    color: #607d8b;
  }

  .sep {
    color: #37474f;
  }

  /* animated pulse dot */
  .pulse {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #42a5f5;
    flex-shrink: 0;
    animation: pulse 1.4s ease-in-out infinite;
  }

  .pulse.reindex {
    background: #ffa726;
  }

  .dot.watching {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #4caf50;
    flex-shrink: 0;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; transform: scale(1); }
    50%       { opacity: 0.35; transform: scale(0.7); }
  }

  /* mini progress track on the right */
  .track {
    width: 120px;
    height: 3px;
    background: rgba(100, 181, 246, 0.15);
    border-radius: 2px;
    overflow: hidden;
    flex-shrink: 0;
  }

  .fill {
    height: 100%;
    background: linear-gradient(90deg, #1e88e5, #42a5f5);
    border-radius: 2px;
    transition: width 0.3s ease;
  }
</style>
