// Follows the OS light/dark appearance and stamps data-theme on the root
// element, which selects the Nocturne or Daylight tokens in tokens.css.
// A manual override arrives with the Settings page.

import { readable, type Readable } from 'svelte/store'

export type ThemeName = 'nocturne' | 'daylight'

export interface ColorSchemeQuery {
  matches: boolean
  addEventListener(type: 'change', listener: (e: { matches: boolean }) => void): void
  removeEventListener(type: 'change', listener: (e: { matches: boolean }) => void): void
}

export function themeFor(prefersLight: boolean): ThemeName {
  return prefersLight ? 'daylight' : 'nocturne'
}

export function createThemeStore(query: ColorSchemeQuery, root: HTMLElement): Readable<ThemeName> {
  return readable<ThemeName>(themeFor(query.matches), (set) => {
    const apply = (prefersLight: boolean) => {
      const theme = themeFor(prefersLight)
      root.dataset.theme = theme
      set(theme)
    }
    const onChange = (e: { matches: boolean }) => apply(e.matches)

    apply(query.matches)
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  })
}
