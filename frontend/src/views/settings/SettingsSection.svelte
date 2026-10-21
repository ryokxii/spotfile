<script lang="ts">
  // A titled group of setting rows.
  export let title: string
  const headingId = `settings-${title.toLowerCase().replace(/\s+/g, '-')}`
</script>

<section class="section" aria-labelledby={headingId}>
  <h2 id={headingId} class="eyebrow">{title}</h2>
  <div class="rows"><slot /></div>
</section>

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .eyebrow {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .rows {
    display: flex;
    flex-direction: column;
  }
</style>
