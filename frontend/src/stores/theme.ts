// Resolves the theme preference against the OS light/dark appearance and
// stamps data-theme on the root element, which selects the Nocturne or
// Daylight tokens in tokens.css.

import { derived, readable, type Readable } from 'svelte/store'
import type { ThemePreference } from './settings'

export type ThemeName = 'nocturne' | 'daylight'

export interface ColorSchemeQuery {
  matches: boolean
  addEventListener(type: 'change', listener: (e: { matches: boolean }) => void): void
  removeEventListener(type: 'change', listener: (e: { matches: boolean }) => void): void
}

export function themeFor(preference: ThemePreference, prefersLight: boolean): ThemeName {
  if (preference !== 'system') return preference
  return prefersLight ? 'daylight' : 'nocturne'
}

// Whether the OS prefers light, updating live while subscribed.
function osPrefersLight(query: ColorSchemeQuery): Readable<boolean> {
  return readable(query.matches, (set) => {
    const onChange = (e: { matches: boolean }) => set(e.matches)
    set(query.matches)
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  })
}

export function createThemeStore(
  query: ColorSchemeQuery,
  root: HTMLElement,
  preference: Readable<ThemePreference>,
): Readable<ThemeName> {
  return derived([preference, osPrefersLight(query)], ([pref, prefersLight]) => {
    const theme = themeFor(pref, prefersLight)
    root.dataset.theme = theme // applied here so every change reaches the DOM
    return theme
  })
}
