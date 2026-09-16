import { get } from 'svelte/store'
import { describe, expect, it } from 'vitest'
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

describe('createThemeStore', () => {
  it('follows the OS preference and stamps data-theme on the root element', () => {
    const root = document.createElement('html')
    const light = fakeQuery(true)
    const theme = createThemeStore(light.query, root)
    const stop = theme.subscribe(() => {})
    expect(get(theme)).toBe('daylight')
    expect(root.dataset.theme).toBe('daylight')
    stop()
  })

  it('switches live when the OS appearance changes', () => {
    const root = document.createElement('html')
    const os = fakeQuery(false)
    const theme = createThemeStore(os.query, root)
    const stop = theme.subscribe(() => {})
    expect(root.dataset.theme).toBe('nocturne')

    os.change(true)
    expect(get(theme)).toBe('daylight')
    expect(root.dataset.theme).toBe('daylight')
    stop()
  })

  it('stops listening once the last subscriber leaves', () => {
    const os = fakeQuery(false)
    const theme = createThemeStore(os.query, document.createElement('html'))
    const stop = theme.subscribe(() => {})
    expect(os.listeners.size).toBe(1)
    stop()
    expect(os.listeners.size).toBe(0)
  })
})
