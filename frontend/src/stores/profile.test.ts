import { get } from 'svelte/store'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanName, createProfileStore, NAME_MAX, type Profile, type ProfileRuntime } from './profile'

function fakeRuntime(overrides: Partial<ProfileRuntime> = {}): ProfileRuntime {
  return {
    get: vi.fn(async (): Promise<Profile> => ({ name: '', history: true, staleDays: 90 })),
    setProfile: vi.fn(async () => undefined),
    setPrivacy: vi.fn(async () => undefined),
    clearHistory: vi.fn(async () => undefined),
    ...overrides,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('cleanName', () => {
  it('trims and caps at NAME_MAX characters', () => {
    expect(cleanName('  Zafer Acar  ')).toBe('Zafer Acar')
    expect(Array.from(cleanName('é'.repeat(NAME_MAX + 5)))).toHaveLength(NAME_MAX)
  })
})

describe('createProfileStore', () => {
  it('loads saved values', async () => {
    const profile = createProfileStore(fakeRuntime({ get: async () => ({ name: 'Zafer', history: false, staleDays: 30 }) }))
    await profile.load()
    expect(get(profile)).toEqual({ name: 'Zafer', history: false, staleDays: 30, nameError: '', privacyError: '' })
  })

  it('keeps defaults when loading fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const profile = createProfileStore(fakeRuntime({ get: () => Promise.reject(new Error('no bridge')) }))
    await profile.load()
    expect(get(profile)).toMatchObject({ name: '', history: true, staleDays: 90 })
  })

  it('saves a cleaned name and skips an unchanged one', async () => {
    const runtime = fakeRuntime()
    const profile = createProfileStore(runtime)
    await profile.setName('  Zafer  ')
    expect(get(profile).name).toBe('Zafer')
    expect(runtime.setProfile).toHaveBeenCalledWith('Zafer')
    await profile.setName('Zafer ')
    expect(runtime.setProfile).toHaveBeenCalledOnce()
  })

  it('restores the name when saving fails', async () => {
    const profile = createProfileStore(fakeRuntime({ setProfile: () => Promise.reject(new Error('disk full')) }))
    await profile.setName('Zafer')
    expect(get(profile)).toMatchObject({ name: '', nameError: "Couldn't save — disk full" })
  })

  it('saves both privacy values together', async () => {
    const runtime = fakeRuntime()
    const profile = createProfileStore(runtime)
    await profile.setStaleDays(180)
    await profile.setHistory(false)
    expect(runtime.setPrivacy).toHaveBeenNthCalledWith(1, true, 180)
    expect(runtime.setPrivacy).toHaveBeenNthCalledWith(2, false, 180)
  })

  it('restores privacy values when saving fails', async () => {
    const profile = createProfileStore(fakeRuntime({ setPrivacy: () => Promise.reject(new Error('disk full')) }))
    await profile.setHistory(false)
    expect(get(profile)).toMatchObject({ history: true, privacyError: "Couldn't save — disk full" })
  })

  it('reports a failed history clear', async () => {
    const profile = createProfileStore(fakeRuntime({ clearHistory: () => Promise.reject(new Error('locked')) }))
    await profile.clearHistory()
    expect(get(profile).privacyError).toBe("Couldn't clear history — locked")
  })
})
