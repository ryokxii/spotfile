<script lang="ts">
  // Every stale file, in a modal. Controlled by `open`; Esc and Close dispatch `close`.
  import { createEventDispatcher, tick } from 'svelte'
  import type { StaleFile } from '../../stores/dashboard'
  import Button from '../../ui/Button.svelte'
  import StaleFileList from './StaleFileList.svelte'

  export let open = false
  export let files: StaleFile[]
  export let busy = false
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const dispatch = createEventDispatcher<{ close: void }>()
  let dialog: HTMLDialogElement

  $: if (dialog) void sync(open)

  async function sync(shouldOpen: boolean) {
    if (shouldOpen && !dialog.open) {
      dialog.showModal()
      await tick()
    } else if (!shouldOpen && dialog.open) {
      dialog.close()
    }
  }

  function onEscape(e: Event) {
    e.preventDefault()
    dispatch('close')
  }
</script>

<dialog bind:this={dialog} class="stale-dialog" aria-labelledby="stale-dialog-title" on:cancel={onEscape}>
  <header class="head">
    <h2 id="stale-dialog-title">Free up space</h2>
    <Button on:click={() => dispatch('close')}>Close</Button>
  </header>
  <div class="body">
    <StaleFileList {files} {busy} {errors} {now} on:remove on:trash />
  </div>
</dialog>

<style>
  .stale-dialog {
    width: min(720px, calc(100vw - 2 * var(--space-xl)));
    max-height: calc(100vh - 2 * var(--space-xxl));
    padding: 0;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface-raised);
    color: var(--color-ink);
    box-shadow: var(--shadow-raised);
  }

  .stale-dialog::backdrop {
    background: rgb(0 0 0 / 0.45);
  }

  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--color-hairline);
  }

  h2 {
    font-size: var(--text-title-size);
    line-height: var(--text-title-line);
    font-weight: var(--text-title-weight);
  }

  .body {
    padding: var(--space-sm) var(--space-lg) var(--space-lg);
    overflow-y: auto;
  }
</style>
