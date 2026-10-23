<script lang="ts">
  // The display name input. Saves the cleaned name on blur or Enter; Escape
  // puts back the saved name. Follows `value` whenever it isn't being edited,
  // so a rolled-back save shows the saved name again.
  import { createEventDispatcher } from 'svelte'
  import { cleanName, NAME_MAX } from '../../stores/profile'

  export let value = ''
  export let id = 'display-name'

  const dispatch = createEventDispatcher<{ save: string }>()
  let draft = value
  let editing = false

  $: if (!editing) draft = value

  function commit() {
    editing = false
    const name = cleanName(draft)
    if (name !== value) dispatch('save', name)
  }

  function onKeydown(e: KeyboardEvent) {
    const input = e.currentTarget as HTMLInputElement
    if (e.key === 'Enter') {
      e.preventDefault()
      input.blur()
    } else if (e.key === 'Escape') {
      draft = value
      editing = false
      input.blur()
    }
  }
</script>

<input
  {id}
  class="name-field"
  type="text"
  maxlength={NAME_MAX}
  autocomplete="name"
  placeholder="Your name"
  bind:value={draft}
  on:focus={() => (editing = true)}
  on:blur={() => editing && commit()}
  on:keydown={onKeydown}
/>

<style>
  .name-field {
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface);
    color: var(--color-ink);
    font-size: var(--text-body-size);
    transition: border-color var(--motion-quick) var(--ease-standard);
  }

  .name-field:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px var(--color-primary-wash);
  }
</style>
