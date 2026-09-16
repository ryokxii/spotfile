// Motion tokens for Svelte transitions, which take durations in JavaScript.
// Reading the CSS custom property means prefers-reduced-motion (which zeroes
// the tokens in tokens.css) applies to JS transitions too.

export type MotionToken = 'instant' | 'quick' | 'normal' | 'slow' | 'stagger'

export function duration(token: MotionToken, root: HTMLElement = document.documentElement): number {
  const raw = getComputedStyle(root).getPropertyValue(`--motion-${token}`).trim()
  const ms = /^(\d+(?:\.\d+)?)ms$/.exec(raw)
  return ms ? Number(ms[1]) : 0
}

// Matches --ease-standard / --ease-enter (Flutter easeOutCubic / easeOutExpo)
// for Svelte's JS easing functions.
export function easeOutCubic(t: number): number {
  return 1 - (1 - t) ** 3
}

export function easeOutExpo(t: number): number {
  return t === 1 ? 1 : 1 - 2 ** (-10 * t)
}
