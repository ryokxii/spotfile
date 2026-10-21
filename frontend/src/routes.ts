// Route table and sidebar registry. A destination is listed in the sidebar only
// once it exists here, so the nav never shows unbuilt pages. Later pieces add a
// '/chats' entry.

import type { SvelteComponent } from 'svelte'
import type { IconName } from './ui/icons'
import Redirect from './views/Redirect.svelte'
import SearchView from './views/search/SearchView.svelte'
import SettingsView from './views/settings/SettingsView.svelte'

export interface Destination {
  path: string
  label: string
  icon: IconName
  component: typeof SvelteComponent
}

export const destinations: Destination[] = [
  { path: '/', label: 'Search', icon: 'search', component: SearchView },
  { path: '/settings', label: 'Settings', icon: 'settings', component: SettingsView },
]

export const routes: Record<string, typeof SvelteComponent> = {
  ...Object.fromEntries(destinations.map((d) => [d.path, d.component])),
  '*': Redirect,
}
