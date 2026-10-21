<script lang="ts">
  // Modal confirmation on the native <dialog>; window.confirm freezes the Wails
  // WebView. Controlled by `open`: Cancel and Esc dispatch `cancel`, and the
  // parent closes the dialog. Focus starts on Cancel so Enter never confirms.
  import { createEventDispatcher, tick } from 'svelte'
  import Button from './Button.svelte'

  export let open = false
  export let title: string
  export let confirmLabel: string
  export let cancelLabel = 'Cancel'

  const dispatch = createEventDispatcher<{ confirm: void; cancel: void }>()
  const titleId = `confirm-title-${Math.random().toString(36).slice(2)}`
  let dialog: HTMLDialogElement

  $: if (dialog) void sync(open)

  async function sync(shouldOpen: boolean) {
    if (shouldOpen && !dialog.open) {
      dialog.showModal()
      await tick()
      dialog.querySelector('button')?.focus() // Cancel is the first button
    } else if (!shouldOpen && dialog.open) {
      dialog.close()
    }
  }

  function onEscape(e: Event) {
    e.preventDefault()
    dispatch('cancel')
  }
</script>

<dialog bind:this={dialog} class="confirm" aria-labelledby={titleId} on:cancel={onEscape}>
  <h2 id={titleId} class="title">{title}</h2>
  <div class="body"><slot /></div>
  <div class="actions">
    <Button on:click={() => dispatch('cancel')}>{cancelLabel}</Button>
    <Button variant="destructive" on:click={() => dispatch('confirm')}>{confirmLabel}</Button>
  </div>
</dialog>

<style>
  .confirm {
    width: min(440px, calc(100vw - 2 * var(--space-lg)));
    margin: auto;
    padding: var(--space-lg);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface-raised);
    color: var(--color-ink);
    box-shadow: var(--shadow-raised);
  }

  .confirm::backdrop {
    background-color: color-mix(in srgb, var(--color-canvas) 70%, transparent);
  }

  .title {
    overflow-wrap: anywhere; /* long file names in titles break instead of overflowing */
    font-size: var(--text-title-size);
    line-height: var(--text-title-line);
    font-weight: var(--text-title-weight);
  }

  .body {
    margin-top: var(--space-xs);
    color: var(--color-ink-secondary);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-xs);
    margin-top: var(--space-xl);
  }
</style>
