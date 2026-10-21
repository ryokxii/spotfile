// The Dashboard's figures and history from the Go backend, reloaded when the
// index or the history changes.

import { writable, type Readable } from 'svelte/store'
import type { FolderStat } from './engine'

export interface TypeCount {
  type: string
  files: number
}

export interface RecentSearch {
  query: string
  at: string // RFC 3339
}

export interface RecentFile {
  path: string
  at: string // RFC 3339
}

export interface StaleFile {
  path: string
  bytes: number // estimated share of the index
  modified: string // RFC 3339
}

export interface DashboardData {
  folders: FolderStat[]
  indexBytes: number
  files: number
  chunks: number
  types: TypeCount[]
  lastUpdate: RecentFile | null
  recentSearches: RecentSearch[]
  recentFiles: RecentFile[]
  stale: StaleFile[] // largest first
  staleBytes: number
  history: boolean
  staleDays: number
}

export type DashboardPhase = 'loading' | 'ready' | 'error'

export interface DashboardState {
  phase: DashboardPhase
  data: DashboardData | null
  error: string
}

// The backend surface the store needs; the app passes the Wails bindings.
export interface DashboardRuntime {
  on(event: string, callback: (...data: unknown[]) => void): () => void
  get(): Promise<DashboardData>
  removeFromIndex(path: string): Promise<void>
  moveToTrash(path: string): Promise<void>
  forgetOpened(path: string): Promise<void>
  recordOpened(path: string): Promise<void>
}

export interface DashboardStore extends Readable<DashboardState> {
  start(): Promise<void>
  stop(): void
  reload(): Promise<void>
  // Row actions resolve to an error message, or '' on success.
  removeFromIndex(path: string): Promise<string>
  moveToTrash(path: string): Promise<string>
  forget(path: string): Promise<string>
  opened(path: string): void
}

// Events after which the figures may have changed.
export const RELOAD_EVENTS = ['index:done', 'watcher:done', 'index:cleared', 'index:changed', 'activity:changed']
// A burst of events (a folder of watcher updates) reloads once.
export const RELOAD_DELAY_MS = 250

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function createDashboardStore(runtime: DashboardRuntime): DashboardStore {
  const store = writable<DashboardState>({ phase: 'loading', data: null, error: '' })
  let unsubscribers: (() => void)[] = []
  let timer: ReturnType<typeof setTimeout> | undefined
  let latest = 0 // only the most recent load may write

  async function reload(): Promise<void> {
    const id = ++latest
    try {
      const data = await runtime.get()
      if (id === latest) store.set({ phase: 'ready', data, error: '' })
    } catch (err) {
      if (id !== latest) return
      // Figures already shown stay; only an empty Dashboard shows the error state.
      store.update((s) => (s.data ? { ...s, error: messageOf(err) } : { phase: 'error', data: null, error: messageOf(err) }))
    }
  }

  function scheduleReload(): void {
    clearTimeout(timer)
    timer = setTimeout(() => void reload(), RELOAD_DELAY_MS)
  }

  async function act(call: () => Promise<void>): Promise<string> {
    try {
      await call()
      return ''
    } catch (err) {
      return messageOf(err)
    }
  }

  return {
    subscribe: store.subscribe,
    async start() {
      unsubscribers = RELOAD_EVENTS.map((event) => runtime.on(event, scheduleReload))
      await reload()
    },
    stop() {
      clearTimeout(timer)
      unsubscribers.forEach((off) => off())
      unsubscribers = []
    },
    reload,
    removeFromIndex: (path) => act(() => runtime.removeFromIndex(path)),
    moveToTrash: (path) => act(() => runtime.moveToTrash(path)),
    forget: (path) => act(() => runtime.forgetOpened(path)),
    opened(path) {
      runtime.recordOpened(path).catch((err) => console.error('dashboard: could not record an opened file', err))
    },
  }
}
