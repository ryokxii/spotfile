// Profile and privacy preferences, saved by the Go backend in settings.json:
// the display name, whether history is kept, and when a file counts as stale.
// A change applies at once and rolls back if the save fails.

import { get, writable, type Readable } from 'svelte/store'

export type StaleDays = 30 | 90 | 180
export const STALE_DAY_CHOICES: StaleDays[] = [30, 90, 180]
export const NAME_MAX = 60

export interface Profile {
  name: string
  history: boolean
  staleDays: StaleDays
}

export interface ProfileState extends Profile {
  nameError: string // last failed name save, '' when none
  privacyError: string // last failed privacy save or history clear
}

export const defaultProfile: Profile = { name: '', history: true, staleDays: 90 }

// The backend surface the store needs; the app passes the Wails bindings.
export interface ProfileRuntime {
  get(): Promise<Profile>
  setProfile(name: string): Promise<void>
  setPrivacy(history: boolean, staleDays: number): Promise<void>
  clearHistory(): Promise<void>
}

export interface ProfileStore extends Readable<ProfileState> {
  load(): Promise<void>
  setName(name: string): Promise<void>
  setHistory(on: boolean): Promise<void>
  setStaleDays(days: StaleDays): Promise<void>
  clearHistory(): Promise<void>
}

// Trimmed and capped at NAME_MAX characters (not UTF-16 units).
export function cleanName(raw: string): string {
  return Array.from(raw.trim()).slice(0, NAME_MAX).join('').trim()
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const saveFailed = (err: unknown): string => `Couldn't save — ${messageOf(err)}`

export function createProfileStore(runtime: ProfileRuntime): ProfileStore {
  const store = writable<ProfileState>({ ...defaultProfile, nameError: '', privacyError: '' })

  async function setPrivacy(change: Partial<Pick<Profile, 'history' | 'staleDays'>>): Promise<void> {
    const before = get(store)
    const next = { ...before, ...change }
    if (next.history === before.history && next.staleDays === before.staleDays) return
    store.set(next)
    try {
      await runtime.setPrivacy(next.history, next.staleDays)
      store.update((s) => ({ ...s, privacyError: '' }))
    } catch (err) {
      // Roll back only if no newer choice has replaced these values meanwhile.
      store.update((s) =>
        s.history === next.history && s.staleDays === next.staleDays
          ? { ...s, history: before.history, staleDays: before.staleDays, privacyError: saveFailed(err) }
          : { ...s, privacyError: saveFailed(err) },
      )
    }
  }

  return {
    subscribe: store.subscribe,
    async load() {
      try {
        const saved = await runtime.get()
        store.update((s) => ({ ...s, name: saved.name, history: saved.history, staleDays: saved.staleDays }))
      } catch (err) {
        console.error('profile: could not load, using defaults', err)
      }
    },
    async setName(raw) {
      const name = cleanName(raw)
      const previous = get(store).name
      if (name === previous) return
      store.update((s) => ({ ...s, name }))
      try {
        await runtime.setProfile(name)
        store.update((s) => ({ ...s, nameError: '' }))
      } catch (err) {
        store.update((s) => (s.name === name ? { ...s, name: previous, nameError: saveFailed(err) } : { ...s, nameError: saveFailed(err) }))
      }
    },
    setHistory: (history) => setPrivacy({ history }),
    setStaleDays: (staleDays) => setPrivacy({ staleDays }),
    async clearHistory() {
      try {
        await runtime.clearHistory()
        store.update((s) => ({ ...s, privacyError: '' }))
      } catch (err) {
        store.update((s) => ({ ...s, privacyError: `Couldn't clear history — ${messageOf(err)}` }))
      }
    },
  }
}
