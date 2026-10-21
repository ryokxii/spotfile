<script lang="ts">
  // Disk: the index size, its files and chunks, files by type, and the last
  // file the index updated.
  import { formatBytes } from '../../lib/format'
  import { filename } from '../../lib/path'
  import { ago } from '../../lib/time'
  import type { DashboardData } from '../../stores/dashboard'

  export let data: DashboardData
  export let now = new Date()

  // A fixed order keeps each type on its colour as the counts change.
  const TYPE_ORDER = ['PDF', 'Markdown', 'Text', 'Code', 'Data']
  const rank = (type: string) => (TYPE_ORDER.includes(type) ? TYPE_ORDER.indexOf(type) : TYPE_ORDER.length)
  const colorOf = (type: string) => `var(--color-identity-${rank(type) % 6})`
  const plural = (n: number, word: string) => `${n.toLocaleString('en-US')} ${word}${n === 1 ? '' : 's'}`

  $: size = formatBytes(data.indexBytes)
  $: value = size.slice(0, size.lastIndexOf(' '))
  $: unit = size.slice(size.lastIndexOf(' ') + 1)
  $: segments = [...data.types].sort((a, b) => rank(a.type) - rank(b.type))
</script>

<section class="tile disk" aria-labelledby="disk-title">
  <h2 id="disk-title" class="eyebrow">Disk</h2>
  <p class="big"><span class="value">{value}</span><span class="unit">{unit} index</span></p>
  <p class="muted">{plural(data.files, 'file')} · {plural(data.chunks, 'chunk')}</p>

  {#if segments.length > 0}
    <div class="bar" role="img" aria-label={segments.map((s) => `${s.type} ${s.files}`).join(', ')}>
      {#each segments as s (s.type)}
        <span class="segment" style="flex-grow: {s.files}; background-color: {colorOf(s.type)}" />
      {/each}
    </div>
    <ul class="legend">
      {#each segments as s (s.type)}
        <li><i style="background-color: {colorOf(s.type)}" aria-hidden="true" />{s.type} <span class="count">{s.files.toLocaleString('en-US')}</span></li>
      {/each}
    </ul>
  {/if}

  <div class="well">
    <span class="muted">Last update</span>
    {#if data.lastUpdate}
      <span title={data.lastUpdate.path}>
        <strong>{filename(data.lastUpdate.path)}</strong>
        <span class="muted">· {ago(new Date(data.lastUpdate.at), now)}</span>
      </span>
    {:else}
      <span class="muted">Nothing updated yet</span>
    {/if}
  </div>
</section>

<style>
  .big {
    display: flex;
    align-items: baseline;
    gap: var(--space-xs);
    margin-top: var(--space-sm);
  }

  .value {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
    font-variant-numeric: tabular-nums;
  }

  .unit {
    color: var(--color-ink-secondary);
    font-size: var(--text-heading-size);
  }

  .bar {
    display: flex;
    gap: 2px;
    height: 10px;
    margin: var(--space-md) 0 var(--space-sm);
    border-radius: 5px;
    overflow: hidden;
  }

  .segment {
    min-width: 4px;
  }

  .legend {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs) var(--space-md);
    font-size: var(--text-small-size);
    color: var(--color-ink-secondary);
  }

  .legend i {
    display: inline-block;
    width: 8px;
    height: 8px;
    margin-right: var(--space-xxs);
    border-radius: 2px;
  }

  .count {
    font-variant-numeric: tabular-nums;
  }

  .well {
    display: flex;
    flex-direction: column;
    gap: var(--space-xxs);
    margin-top: var(--space-lg);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    background-color: var(--color-surface-sunken);
  }

  strong {
    font-weight: var(--text-body-strong-weight);
  }
</style>
