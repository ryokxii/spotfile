import { get, readable, writable } from 'svelte/store'
import { describe, expect, it } from 'vitest'
import type { ThemePreference } from './settings'
import { createThemeStore, type ColorSchemeQuery } from './theme'

function fakeQuery(prefersLight: boolean) {
  const listeners = new Set<(e: { matches: boolean }) => void>()
  const query: ColorSchemeQuery = {
    matches: prefersLight,
    addEventListener: (_type, cb) => listeners.add(cb),
    removeEventListener: (_type, cb) => listeners.delete(cb),
  }
  return {
    query,
    listeners,
    change(light: boolean) {
      query.matches = light
      listeners.forEach((cb) => cb({ matches: light }))
    },
  }
}

const system = () => readable<ThemePreference>('system')

describe('createThemeStore', () => {
  it('follows the OS preference and stamps data-theme on the root element', () => {
    const root = document.createElement('html')
    const theme = createThemeStore(fakeQuery(true).query, root, system())
    const stop = theme.subscribe(() => undefined)
    expect(get(theme)).toBe('daylight')
    expect(root.dataset.theme).toBe('daylight')
    stop()
  })

  it('switches live when the OS appearance changes', () => {
    const root = document.createElement('html')
    const os = fakeQuery(false)
    const theme = createThemeStore(os.query, root, system())
    const stop = theme.subscribe(() => undefined)
    expect(root.dataset.theme).toBe('nocturne')

    os.change(true)
    expect(get(theme)).toBe('daylight')
    expect(root.dataset.theme).toBe('daylight')
    stop()
  })

  it('a pinned theme ignores the OS', () => {
    const root = document.createElement('html')
    const os = fakeQuery(false)
    const theme = createThemeStore(os.query, root, readable<ThemePreference>('daylight'))
    const stop = theme.subscribe(() => undefined)
    expect(root.dataset.theme).toBe('daylight')

    os.change(false)
    expect(get(theme)).toBe('daylight')
    stop()
  })

  it('re-applies when the preference changes', () => {
    const root = document.createElement('html')
    const preference = writable<ThemePreference>('system')
    const theme = createThemeStore(fakeQuery(false).query, root, preference)
    const stop = theme.subscribe(() => undefined)

    preference.set('daylight')
    expect(root.dataset.theme).toBe('daylight')
    preference.set('system')
    expect(root.dataset.theme).toBe('nocturne')
    stop()
  })

  it('stops listening once the last subscriber leaves', () => {
    const os = fakeQuery(false)
    const theme = createThemeStore(os.query, document.createElement('html'), system())
    const stop = theme.subscribe(() => undefined)
    expect(os.listeners.size).toBe(1)
    stop()
    expect(os.listeners.size).toBe(0)
  })
})
