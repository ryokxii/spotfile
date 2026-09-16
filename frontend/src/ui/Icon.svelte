<script lang="ts">
  import { ICONS, type IconName } from './icons'

  export let name: IconName
  export let size = 18
</script>

<svg
  class="icon"
  width={size}
  height={size}
  viewBox="0 0 20 20"
  fill="none"
  stroke="currentColor"
  stroke-width="1.6"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#each ICONS[name] as d}
    <path {d} />
  {/each}
</svg>

<style>
  .icon {
    flex-shrink: 0;
    display: block;
  }
</style>
