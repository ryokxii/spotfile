// Appearance preferences, saved by the Go backend in ~/.spotfile/settings.json.
// A change applies at once and rolls back if the save fails.

import { get, writable, type Readable } from 'svelte/store'

export type ThemePreference = 'system' | 'nocturne' | 'daylight'
export type MotionPreference = 'system' | 'reduce' | 'full'

export interface Appearance {
  theme: ThemePreference
  motion: MotionPreference
}

export interface SettingsState extends Appearance {
  error: string // last failed save, '' when none
}

export const defaultAppearance: Appearance = { theme: 'system', motion: 'system' }

// The backend surface the store needs; the app passes the Wails bindings.
export interface SettingsRuntime {
  get(): Promise<Appearance>
  setAppearance(theme: ThemePreference, motion: MotionPreference): Promise<void>
}

export interface SettingsStore extends Readable<SettingsState> {
  load(): Promise<void>
  setTheme(theme: ThemePreference): Promise<void>
  setMotion(motion: MotionPreference): Promise<void>
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function createSettingsStore(runtime: SettingsRuntime): SettingsStore {
  const store = writable<SettingsState>({ ...defaultAppearance, error: '' })

  async function save<K extends keyof Appearance>(key: K, value: Appearance[K]): Promise<void> {
    const previous = get(store)[key]
    if (previous === value) return
    store.update((s) => ({ ...s, [key]: value }))
    const { theme, motion } = get(store)
    try {
      await runtime.setAppearance(theme, motion)
      store.update((s) => ({ ...s, error: '' }))
    } catch (err) {
      const error = `Couldn't save — ${messageOf(err)}`
      // Roll back only if no newer choice has replaced this value meanwhile.
      store.update((s) => (s[key] === value ? { ...s, [key]: previous, error } : { ...s, error }))
    }
  }

  return {
    subscribe: store.subscribe,
    async load() {
      try {
        const saved = await runtime.get()
        store.set({ theme: saved.theme, motion: saved.motion, error: '' })
      } catch (err) {
        console.error('settings: could not load, using defaults', err)
      }
    },
    setTheme: (theme) => save('theme', theme),
    setMotion: (motion) => save('motion', motion),
  }
}
