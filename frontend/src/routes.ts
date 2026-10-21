// Route table and sidebar registry. A destination is listed in the sidebar only
// once it exists here, so the nav never shows unbuilt pages. Profile is routed
// but reached from the profile button at the foot of the sidebar.

import type { SvelteComponent } from 'svelte'
import type { IconName } from './ui/icons'
import DashboardView from './views/dashboard/DashboardView.svelte'
import ProfileView from './views/profile/ProfileView.svelte'
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
  { path: '/', label: 'Dashboard', icon: 'dashboard', component: DashboardView },
  { path: '/chat', label: 'Chat', icon: 'chat', component: SearchView },
  { path: '/settings', label: 'Settings', icon: 'settings', component: SettingsView },
]

export const routes: Record<string, typeof SvelteComponent> = {
  ...Object.fromEntries(destinations.map((d) => [d.path, d.component])),
  '/profile': ProfileView,
  '*': Redirect,
}
