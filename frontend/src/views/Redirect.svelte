<script lang="ts">
  // Unknown routes land on Search.
  import { onMount } from 'svelte'
  import { replace } from 'svelte-spa-router'

  onMount(() => {
    void replace('/')
  })
</script>
