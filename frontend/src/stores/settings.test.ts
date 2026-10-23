import { get } from 'svelte/store'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSettingsStore, type Appearance, type SettingsRuntime } from './settings'

function fakeRuntime(overrides: Partial<SettingsRuntime> = {}): SettingsRuntime {
  return {
    get: vi.fn(async (): Promise<Appearance> => ({ theme: 'system', motion: 'system' })),
    setAppearance: vi.fn(async () => undefined),
    ...overrides,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('createSettingsStore', () => {
  it('starts on defaults and loads saved values', async () => {
    const settings = createSettingsStore(fakeRuntime({ get: async () => ({ theme: 'daylight', motion: 'reduce' }) }))
    expect(get(settings)).toEqual({ theme: 'system', motion: 'system', error: '' })
    await settings.load()
    expect(get(settings)).toEqual({ theme: 'daylight', motion: 'reduce', error: '' })
  })

  it('keeps defaults when loading fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const settings = createSettingsStore(fakeRuntime({ get: () => Promise.reject(new Error('no bridge')) }))
    await settings.load()
    expect(get(settings)).toEqual({ theme: 'system', motion: 'system', error: '' })
  })

  it('applies a change immediately and saves both preferences', async () => {
    let finish!: () => void
    const runtime = fakeRuntime({ setAppearance: vi.fn(() => new Promise<void>((resolve) => (finish = resolve))) })
    const settings = createSettingsStore(runtime)

    const pending = settings.setTheme('nocturne')
    expect(get(settings).theme).toBe('nocturne')
    expect(runtime.setAppearance).toHaveBeenCalledWith('nocturne', 'system')

    finish()
    await pending
    expect(get(settings).error).toBe('')
  })

  it('rolls back and reports the error when saving fails', async () => {
    const settings = createSettingsStore(fakeRuntime({ setAppearance: () => Promise.reject(new Error('disk full')) }))
    await settings.setMotion('reduce')
    expect(get(settings)).toEqual({ theme: 'system', motion: 'system', error: "Couldn't save — disk full" })
  })

  it('clears the error after the next successful save', async () => {
    const setAppearance = vi.fn().mockRejectedValueOnce(new Error('disk full')).mockResolvedValue(undefined)
    const settings = createSettingsStore(fakeRuntime({ setAppearance }))
    await settings.setTheme('daylight')
    await settings.setTheme('daylight')
    expect(get(settings)).toMatchObject({ theme: 'daylight', error: '' })
  })

  it('does not roll back over a newer choice', async () => {
    let fail!: (err: Error) => void
    const setAppearance = vi
      .fn()
      .mockImplementationOnce(() => new Promise<void>((_, reject) => (fail = reject)))
      .mockResolvedValue(undefined)
    const settings = createSettingsStore(fakeRuntime({ setAppearance }))

    const first = settings.setTheme('nocturne')
    await settings.setTheme('daylight')
    fail(new Error('slow disk'))
    await first
    expect(get(settings).theme).toBe('daylight')
  })

  it('skips saving an unchanged value', async () => {
    const runtime = fakeRuntime()
    await createSettingsStore(runtime).setTheme('system')
    expect(runtime.setAppearance).not.toHaveBeenCalled()
  })
})
