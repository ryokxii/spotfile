import { get } from 'svelte/store'
import { describe, expect, it, vi } from 'vitest'
import { createEngineStore, initialEngineState, reduce, type EngineRuntime, type EngineState } from './engine'

function apply(...events: Parameters<typeof reduce>[1][]): EngineState {
  return events.reduce(reduce, initialEngineState)
}

describe('reduce', () => {
  it('starts not ready, idle, empty', () => {
    expect(initialEngineState).toMatchObject({ ready: false, error: '', indexing: null, documents: 0, chunks: 0 })
  })

  it('applies a status snapshot', () => {
    const s = apply({ type: 'status', status: { ready: true, error: '', documents: 17, chunks: 819 } })
    expect(s).toMatchObject({ ready: true, documents: 17, chunks: 819 })
  })

  it('engine:ready clears a previous engine error', () => {
    const s = apply({ type: 'engine:error', message: 'llama.cpp not found' }, { type: 'engine:ready' })
    expect(s.ready).toBe(true)
    expect(s.error).toBe('')
  })

  it('tracks indexing progress by distinct files and chunks', () => {
    const s = apply(
      { type: 'index:start', total: 3, dir: '/notes' },
      { type: 'index:prepared', totalChunks: 40 },
      { type: 'index:chunk', total: 1, path: '/notes/a.md' },
      { type: 'index:chunk', total: 2, path: '/notes/a.md' },
      { type: 'index:chunk', total: 3, path: '/notes/b.pdf' },
    )
    expect(s.indexing).toMatchObject({ files: 3, filesDone: 2, chunks: 40, chunksDone: 3, currentFile: '/notes/b.pdf' })
    expect(s.folder).toBe('/notes')
  })

  it('index:start clears stale engine and index errors (the engine must be up to index)', () => {
    const s = apply(
      { type: 'engine:error', message: 'boom' },
      { type: 'index:error', message: 'old' },
      { type: 'index:start', total: 1, dir: '/x' },
    )
    expect(s.error).toBe('')
    expect(s.indexError).toBe('')
  })

  it('index:error stops indexing and records the message', () => {
    const s = apply({ type: 'index:start', total: 2, dir: '/x' }, { type: 'index:error', message: 'engine init: llama.cpp not found' })
    expect(s.indexing).toBeNull()
    expect(s.indexError).toBe('engine init: llama.cpp not found')
  })

  it('index:done ends indexing and proves the engine is ready', () => {
    const s = apply({ type: 'index:start', total: 1, dir: '/x' }, { type: 'index:done', total: 12 })
    expect(s.indexing).toBeNull()
    expect(s.ready).toBe(true)
  })

  it('tracks watcher state and live re-indexing', () => {
    let s = apply({ type: 'watcher:started' }, { type: 'watcher:reindexing', path: '/x/a.md' })
    expect(s).toMatchObject({ watching: true, ready: true, reindexingFile: '/x/a.md' })
    s = reduce(s, { type: 'watcher:done' })
    expect(s.reindexingFile).toBe('')
  })

  it('never mutates the previous state', () => {
    const before = apply({ type: 'index:start', total: 2, dir: '/x' })
    const snapshot = JSON.stringify(before)
    reduce(before, { type: 'index:chunk', total: 1, path: '/x/a.md' })
    expect(JSON.stringify(before)).toBe(snapshot)
  })
})

function fakeRuntime(overrides: Partial<EngineRuntime> = {}) {
  const handlers = new Map<string, (...data: unknown[]) => void>()
  const runtime: EngineRuntime = {
    on: (event, cb) => {
      handlers.set(event, cb)
      return () => handlers.delete(event)
    },
    status: vi.fn(async () => ({ ready: true, error: '', documents: 5, chunks: 50 })),
    selectFolder: vi.fn(async () => '/picked'),
    indexFolder: vi.fn(async () => undefined),
    ...overrides,
  }
  return { runtime, emit: (event: string, ...data: unknown[]) => handlers.get(event)?.(...data), handlers }
}

describe('createEngineStore', () => {
  it('loads a status snapshot on start and applies runtime events', async () => {
    const { runtime, emit } = fakeRuntime()
    const engine = createEngineStore(runtime)
    await engine.start()
    expect(get(engine)).toMatchObject({ ready: true, documents: 5, chunks: 50 })

    emit('index:start', { total: 4, dir: '/docs' })
    expect(get(engine).indexing?.files).toBe(4)
  })

  it('subscribes to index:error, the event the old UI dropped', async () => {
    const { runtime, handlers } = fakeRuntime()
    await createEngineStore(runtime).start()
    expect(handlers.has('index:error')).toBe(true)
  })

  it('refreshes totals from the backend when indexing finishes', async () => {
    const status = vi
      .fn()
      .mockResolvedValueOnce({ ready: true, error: '', documents: 0, chunks: 0 })
      .mockResolvedValueOnce({ ready: true, error: '', documents: 9, chunks: 120 })
    const { runtime, emit } = fakeRuntime({ status })
    const engine = createEngineStore(runtime)
    await engine.start()
    emit('index:done', 120)
    await vi.waitFor(() => expect(get(engine).documents).toBe(9))
  })

  it('stop removes every listener', async () => {
    const { runtime, handlers } = fakeRuntime()
    const engine = createEngineStore(runtime)
    await engine.start()
    engine.stop()
    expect(handlers.size).toBe(0)
  })

  it('chooseFolder indexes the picked folder and ignores a cancelled dialog', async () => {
    const { runtime } = fakeRuntime()
    const engine = createEngineStore(runtime)
    await engine.chooseFolder()
    expect(runtime.indexFolder).toHaveBeenCalledWith('/picked')

    const cancelled = fakeRuntime({ selectFolder: vi.fn(async () => '') })
    await createEngineStore(cancelled.runtime).chooseFolder()
    expect(cancelled.runtime.indexFolder).not.toHaveBeenCalled()
  })

  it('chooseFolder surfaces a rejected IndexFolder as an index error', async () => {
    const { runtime } = fakeRuntime({ indexFolder: vi.fn(async () => Promise.reject(new Error('indexing is already in progress'))) })
    const engine = createEngineStore(runtime)
    await engine.chooseFolder()
    expect(get(engine).indexError).toBe('indexing is already in progress')
  })
})
