<script lang="ts">
  // App shell: sidebar navigation + routed view. Owns app-wide lifecycles
  // (engine events, OS theme) so they outlive any single view.
  import { onDestroy, onMount } from 'svelte'
  import Router from 'svelte-spa-router'
  import { WindowFullscreen, WindowIsFullscreen, WindowUnfullscreen } from '../wailsjs/runtime/runtime.js'
  import { destinations, routes } from './routes'
  import { engine, search, theme } from './stores/app'
  import Sidebar from './ui/Sidebar.svelte'

  // DESIGN.md §7: full sidebar at ≥1180px, icon rail below. While a preview is
  // open the rail is used up to 1440px so results and preview both have room.
  const RAIL_BELOW = 1180
  const RAIL_WITH_PREVIEW_BELOW = 1440

  let width = window.innerWidth
  $: compact = width < RAIL_BELOW || ($search.preview !== null && width < RAIL_WITH_PREVIEW_BELOW)

  const stopTheme = theme.subscribe(() => {})

  onMount(() => {
    void engine.start()
  })

  onDestroy(() => {
    engine.stop()
    stopTheme()
  })

  // F11 toggles full screen on every platform (macOS also has ⌃⌘F via the Window menu).
  async function onKeydown(e: KeyboardEvent) {
    if (e.key !== 'F11') return
    e.preventDefault()
    if (await WindowIsFullscreen()) WindowUnfullscreen()
    else WindowFullscreen()
  }
</script>

<svelte:window bind:innerWidth={width} on:keydown={onKeydown} />

<div class="shell" class:compact>
  <Sidebar {destinations} {compact} />
  <main class="view">
    <Router {routes} />
  </main>
</div>

<style>
  .shell {
    height: 100%;
    display: grid;
    grid-template-columns: var(--sidebar) minmax(0, 1fr);
    background: var(--room-decor), var(--color-canvas);
  }

  @media (min-width: 1440px) {
    .shell {
      grid-template-columns: var(--sidebar-wide) minmax(0, 1fr);
    }
  }

  .shell.compact {
    grid-template-columns: var(--sidebar-rail) minmax(0, 1fr);
  }

  .view {
    min-width: 0;
    min-height: 0;
    height: 100%;
    overflow: hidden;
  }
</style>
