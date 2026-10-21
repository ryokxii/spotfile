// App-wide store instances wired to the Wails bindings. Components import from
// here; tests construct the stores directly with their own runtimes.

import { derived } from 'svelte/store'
import {
  ClearHistory,
  ClearIndex,
  EngineStatus,
  ForgetOpened,
  GetDashboard,
  GetSettings,
  IndexFolder,
  MoveToTrash,
  RecordOpened,
  AddFiles,
  RemoveFile,
  RemoveFolder,
  RemoveFromIndex,
  Search,
  SelectFiles,
  SelectFolder,
  SetAppearance,
  SetPrivacy,
  SetProfile,
} from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'
import { createDashboardStore, type DashboardData } from './dashboard'
import { createEngineStore } from './engine'
import { createMotionStore } from './motion'
import { createProfileStore, type StaleDays } from './profile'
import { createSearchStore } from './search'
import { createSettingsStore, type MotionPreference, type ThemePreference } from './settings'
import { createThemeStore } from './theme'

const logFailure = (what: string) => (err: unknown) => console.error(what, err)

export const settings = createSettingsStore({
  // Go validates these values before saving, so the casts only narrow the type.
  get: async () => {
    const saved = await GetSettings()
    return { theme: saved.theme as ThemePreference, motion: saved.motion as MotionPreference }
  },
  setAppearance: SetAppearance,
})

export const profile = createProfileStore({
  get: async () => {
    const saved = await GetSettings()
    return { name: saved.name, history: saved.history, staleDays: saved.staleDays as StaleDays }
  },
  setProfile: SetProfile,
  setPrivacy: SetPrivacy,
  clearHistory: ClearHistory,
})

export const engine = createEngineStore({
  on: EventsOn,
  status: EngineStatus,
  selectFolder: SelectFolder,
  indexFolder: IndexFolder,
  clearIndex: ClearIndex,
  removeFolder: RemoveFolder,
  selectFiles: SelectFiles,
  addFiles: AddFiles,
  removeFile: RemoveFile,
})

export const search = createSearchStore({
  search: Search,
  opened: (path) => void RecordOpened(path).catch(logFailure('search: could not record an opened file')),
})

export const dashboard = createDashboardStore({
  on: EventsOn,
  // Go's time.Time fields arrive as RFC 3339 strings; the generated model types them loosely.
  get: async () => (await GetDashboard()) as unknown as DashboardData,
  removeFromIndex: RemoveFromIndex,
  moveToTrash: MoveToTrash,
  forgetOpened: ForgetOpened,
  recordOpened: RecordOpened,
})

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
