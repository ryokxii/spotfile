// Engine and indexing state: one reducer over every backend event, so a view
// that unmounts on navigation never misses progress, and every emitted event
// has exactly one listener.

import { writable, type Readable } from 'svelte/store'

export interface EngineStatus {
  ready: boolean
  error: string
  documents: number
  chunks: number
}

export interface IndexProgress {
  files: number
  filesDone: number
  chunks: number // estimated total; 0 until index:prepared
  chunksDone: number
  currentFile: string
  seen: ReadonlySet<string>
}

export interface EngineState {
  ready: boolean
  error: string // engine failed to start
  indexError: string
  indexing: IndexProgress | null
  reindexingFile: string // watcher re-index in flight, '' when none
  watching: boolean
  documents: number
  chunks: number
  folder: string
}

export type EngineEvent =
  | { type: 'status'; status: EngineStatus }
  | { type: 'engine:ready' }
  | { type: 'engine:error'; message: string }
  | { type: 'index:start'; total: number; dir: string }
  | { type: 'index:prepared'; totalChunks: number }
  | { type: 'index:chunk'; total: number; path: string }
  | { type: 'index:done'; total: number }
  | { type: 'index:error'; message: string }
  | { type: 'watcher:started' }
  | { type: 'watcher:reindexing'; path: string }
  | { type: 'watcher:done' }

export const initialEngineState: EngineState = {
  ready: false,
  error: '',
  indexError: '',
  indexing: null,
  reindexingFile: '',
  watching: false,
  documents: 0,
  chunks: 0,
  folder: '',
}

export function reduce(state: EngineState, event: EngineEvent): EngineState {
  switch (event.type) {
    case 'status':
      return { ...state, ...event.status }
    case 'engine:ready':
      return { ...state, ready: true, error: '' }
    case 'engine:error':
      return { ...state, ready: false, error: event.message }
    case 'index:start':
      return {
        ...state,
        error: '', // indexing only starts once the engine is up
        indexError: '',
        folder: event.dir || state.folder,
        indexing: { files: event.total, filesDone: 0, chunks: 0, chunksDone: 0, currentFile: '', seen: new Set() },
      }
    case 'index:prepared':
      return state.indexing ? { ...state, indexing: { ...state.indexing, chunks: event.totalChunks } } : state
    case 'index:chunk': {
      if (!state.indexing) return state
      const { seen } = state.indexing
      const isNew = !seen.has(event.path)
      return {
        ...state,
        indexing: {
          ...state.indexing,
          chunksDone: event.total,
          currentFile: event.path,
          seen: isNew ? new Set([...seen, event.path]) : seen,
          filesDone: isNew ? seen.size + 1 : state.indexing.filesDone,
        },
      }
    }
    case 'index:done':
      return { ...state, ready: true, indexing: null }
    case 'index:error':
      return { ...state, indexing: null, indexError: event.message }
    case 'watcher:started':
      return { ...state, ready: true, watching: true }
    case 'watcher:reindexing':
      return { ...state, reindexingFile: event.path }
    case 'watcher:done':
      return { ...state, reindexingFile: '' }
  }
}

// The backend surface the store needs; the app passes the Wails bindings.
export interface EngineRuntime {
  on(event: string, callback: (...data: unknown[]) => void): () => void
  status(): Promise<EngineStatus>
  selectFolder(): Promise<string>
  indexFolder(dir: string): Promise<void>
}

export interface EngineStore extends Readable<EngineState> {
  start(): Promise<void>
  stop(): void
  chooseFolder(): Promise<void>
}

interface ProgressPayload {
  total?: number
  dir?: string
  totalChunks?: number
  path?: string
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function createEngineStore(runtime: EngineRuntime): EngineStore {
  const { subscribe, update } = writable(initialEngineState)
  const dispatch = (event: EngineEvent) => update((s) => reduce(s, event))
  let unsubscribers: (() => void)[] = []

  async function refreshStatus(): Promise<void> {
    try {
      dispatch({ type: 'status', status: await runtime.status() })
    } catch (err) {
      dispatch({ type: 'engine:error', message: messageOf(err) })
    }
  }

  const listeners: Record<string, (data: unknown) => void> = {
    'engine:ready': () => dispatch({ type: 'engine:ready' }),
    'engine:error': (msg) => dispatch({ type: 'engine:error', message: String(msg) }),
    'index:start': (d) => {
      const p = d as ProgressPayload
      dispatch({ type: 'index:start', total: p.total ?? 0, dir: p.dir ?? '' })
    },
    'index:prepared': (d) => dispatch({ type: 'index:prepared', totalChunks: (d as ProgressPayload).totalChunks ?? 0 }),
    'index:chunk': (d) => {
      const p = d as ProgressPayload
      dispatch({ type: 'index:chunk', total: p.total ?? 0, path: p.path ?? '' })
    },
    'index:done': (total) => {
      dispatch({ type: 'index:done', total: Number(total) || 0 })
      void refreshStatus() // authoritative document/chunk totals after de-duplication
    },
    'index:error': (msg) => dispatch({ type: 'index:error', message: String(msg) }),
    'watcher:started': () => dispatch({ type: 'watcher:started' }),
    'watcher:reindexing': (d) => dispatch({ type: 'watcher:reindexing', path: (d as ProgressPayload).path ?? '' }),
    'watcher:done': () => {
      dispatch({ type: 'watcher:done' })
      void refreshStatus()
    },
  }

  return {
    subscribe,
    async start() {
      unsubscribers = Object.entries(listeners).map(([event, cb]) => runtime.on(event, cb))
      await refreshStatus() // catch up on anything emitted before the listeners existed
    },
    stop() {
      unsubscribers.forEach((off) => off())
      unsubscribers = []
    },
    async chooseFolder() {
      try {
        const dir = await runtime.selectFolder()
        if (!dir) return
        await runtime.indexFolder(dir)
      } catch (err) {
        dispatch({ type: 'index:error', message: messageOf(err) })
      }
    },
  }
}
