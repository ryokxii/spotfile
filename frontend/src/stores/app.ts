// App-wide store instances wired to the Wails bindings. Components import from
// here; tests construct the stores directly with their own runtimes.

import { derived } from 'svelte/store'
import { ClearIndex, EngineStatus, GetSettings, IndexFolder, Search, SelectFolder, SetAppearance } from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'
import { createEngineStore } from './engine'
import { createMotionStore } from './motion'
import { createSearchStore } from './search'
import { createSettingsStore, type MotionPreference, type ThemePreference } from './settings'
import { createThemeStore } from './theme'

export const settings = createSettingsStore({
  // Go validates these values before saving, so the casts only narrow the type.
  get: async () => {
    const saved = await GetSettings()
    return { theme: saved.theme as ThemePreference, motion: saved.motion as MotionPreference }
  },
  setAppearance: SetAppearance,
})

export const engine = createEngineStore({
  on: EventsOn,
  status: EngineStatus,
  selectFolder: SelectFolder,
  indexFolder: IndexFolder,
  clearIndex: ClearIndex,
})

export const search = createSearchStore({ search: Search })

const root = document.documentElement

export const theme = createThemeStore(
  window.matchMedia('(prefers-color-scheme: light)'),
  root,
  derived(settings, (s) => s.theme),
)

export const motion = createMotionStore(
  root,
  derived(settings, (s) => s.motion),
)
