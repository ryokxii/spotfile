// App-wide store instances wired to the Wails bindings. Components import from
// here; tests construct the stores directly with their own runtimes.

import { EngineStatus, IndexFolder, Search, SelectFolder } from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'
import { createEngineStore } from './engine'
import { createSearchStore } from './search'
import { createThemeStore } from './theme'

export const engine = createEngineStore({
  on: EventsOn,
  status: EngineStatus,
  selectFolder: SelectFolder,
  indexFolder: IndexFolder,
})

export const search = createSearchStore({ search: Search })

export const theme = createThemeStore(window.matchMedia('(prefers-color-scheme: light)'), document.documentElement)
