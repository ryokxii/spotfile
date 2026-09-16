// Route table and sidebar registry. A destination is listed in the sidebar only
// once it exists here, so the nav never shows unbuilt pages. Later pieces add
// '/settings' and '/chats' entries.

import type { SvelteComponent } from 'svelte'
import type { IconName } from './ui/icons'
import Redirect from './views/Redirect.svelte'
import SearchView from './views/search/SearchView.svelte'

export interface Destination {
  path: string
  label: string
  icon: IconName
  component: typeof SvelteComponent
}

export const destinations: Destination[] = [{ path: '/', label: 'Search', icon: 'search', component: SearchView }]

export const routes: Record<string, typeof SvelteComponent> = {
  ...Object.fromEntries(destinations.map((d) => [d.path, d.component])),
  '*': Redirect,
}
