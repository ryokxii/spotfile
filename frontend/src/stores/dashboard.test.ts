import { get } from 'svelte/store'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createDashboardStore, RELOAD_DELAY_MS, type DashboardData, type DashboardRuntime } from './dashboard'

const data = (files: number): DashboardData => ({
  folders: [{ path: '/n', files }],
  indexBytes: 2048,
  files,
  chunks: files * 10,
  types: [{ type: 'PDF', files }],
  lastUpdate: null,
  recentSearches: [],
  recentFiles: [],
  stale: [],
  staleBytes: 0,
  history: true,
  staleDays: 90,
})

function fakeRuntime(overrides: Partial<DashboardRuntime> = {}) {
  const handlers = new Map<string, () => void>()
  const runtime: DashboardRuntime = {
    on: (event, cb) => {
      handlers.set(event, cb as () => void)
      return () => handlers.delete(event)
    },
    get: vi.fn(async () => data(3)),
    removeFromIndex: vi.fn(async () => undefined),
    moveToTrash: vi.fn(async () => undefined),
    forgetOpened: vi.fn(async () => undefined),
    recordOpened: vi.fn(async () => undefined),
    ...overrides,
  }
  return { runtime, emit: (event: string) => handlers.get(event)?.(), handlers }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('createDashboardStore', () => {
  it('loads on start', async () => {
    const { runtime } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    expect(get(dashboard).phase).toBe('loading')
    await dashboard.start()
    expect(get(dashboard)).toMatchObject({ phase: 'ready', error: '' })
    expect(get(dashboard).data?.files).toBe(3)
  })

  it('shows an error when the first load fails', async () => {
    const { runtime } = fakeRuntime({ get: () => Promise.reject(new Error('bridge down')) })
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    expect(get(dashboard)).toEqual({ phase: 'error', data: null, error: 'bridge down' })
  })

  it('keeps the figures shown when a reload fails', async () => {
    const get_ = vi.fn().mockResolvedValueOnce(data(3)).mockRejectedValueOnce(new Error('bridge down'))
    const dashboard = createDashboardStore(fakeRuntime({ get: get_ }).runtime)
    await dashboard.start()
    await dashboard.reload()
    expect(get(dashboard)).toMatchObject({ phase: 'ready', error: 'bridge down' })
    expect(get(dashboard).data?.files).toBe(3)
  })

  it.each(['index:done', 'watcher:done', 'index:cleared', 'index:changed', 'activity:changed'])(
    'reloads after %s',
    async (event) => {
      const { runtime, emit } = fakeRuntime()
      const dashboard = createDashboardStore(runtime)
      await dashboard.start()
      emit(event)
      await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
      expect(runtime.get).toHaveBeenCalledTimes(2)
    },
  )

  it('reloads once for a burst of events', async () => {
    const { runtime, emit } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    emit('watcher:done')
    emit('index:changed')
    emit('activity:changed')
    await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
    expect(runtime.get).toHaveBeenCalledTimes(2)
  })

  it('stops listening on stop', async () => {
    const { runtime, emit, handlers } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    dashboard.stop()
    emit('index:changed')
    await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
    expect(runtime.get).toHaveBeenCalledOnce()
    expect(handlers.size).toBe(0)
  })

  it('row actions resolve to an error message, or empty on success', async () => {
    const { runtime } = fakeRuntime({ moveToTrash: () => Promise.reject(new Error('refused')) })
    const dashboard = createDashboardStore(runtime)
    expect(await dashboard.removeFromIndex('/n/a.md')).toBe('')
    expect(runtime.removeFromIndex).toHaveBeenCalledWith('/n/a.md')
    expect(await dashboard.moveToTrash('/n/a.md')).toBe('refused')
    expect(await dashboard.forget('/n/a.md')).toBe('')
    expect(runtime.forgetOpened).toHaveBeenCalledWith('/n/a.md')
  })

  it('records opened files and only logs a failure', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const { runtime } = fakeRuntime({ recordOpened: vi.fn(() => Promise.reject(new Error('locked'))) })
    createDashboardStore(runtime).opened('/n/a.md')
    await vi.runAllTimersAsync()
    expect(runtime.recordOpened).toHaveBeenCalledWith('/n/a.md')
    expect(error).toHaveBeenCalled()
  })
})
