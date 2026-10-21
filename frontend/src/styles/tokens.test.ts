// @vitest-environment node
//
// Contrast floor from context/DESIGN.md §2, verified against the real token
// values in tokens.css rather than by eye.

import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { contrastRatio } from '../lib/theme/contrast'

const css = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '')

// Custom properties declared in the rule whose selector list contains `selector`.
function declarations(selector: string, source = css): Record<string, string> {
  const rule = /([^{}]+)\{([^}]*)\}/g
  for (let m = rule.exec(source); m; m = rule.exec(source)) {
    const selectors = m[1].split(',').map((s) => s.trim())
    if (!selectors.includes(selector)) continue
    const vars: Record<string, string> = {}
    for (const d of m[2].matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) vars[d[1]] = d[2].trim()
    return vars
  }
  throw new Error(`no rule for ${selector} in tokens.css`)
}

const THEMES = {
  nocturne: ":root[data-theme='nocturne']",
  daylight: ":root[data-theme='daylight']",
}
const PLANES = ['--color-surface-sunken', '--color-surface', '--color-surface-raised']
const TEXT = ['--color-ink', '--color-ink-secondary', '--color-ink-faint']
const SEMANTIC_TEXT = ['--color-primary', '--color-secondary', '--color-urgent']
const FLOOR = 4.5

describe.each(Object.entries(THEMES))('%s contrast floor', (_theme, selector) => {
  const vars = declarations(selector)

  it.each([...TEXT, ...SEMANTIC_TEXT].flatMap((fg) => PLANES.map((bg) => [fg, bg])))(
    '%s on %s ≥ 4.5:1',
    (fg, bg) => {
      expect(contrastRatio(vars[fg], vars[bg])).toBeGreaterThanOrEqual(FLOOR)
    },
  )

  it('--color-on-primary on a solid --color-primary fill ≥ 4.5:1', () => {
    expect(contrastRatio(vars['--color-on-primary'], vars['--color-primary'])).toBeGreaterThanOrEqual(FLOOR)
  })
})

describe('theme parity', () => {
  it('both themes define exactly the same colour tokens', () => {
    const nocturne = Object.keys(declarations(THEMES.nocturne)).sort()
    const daylight = Object.keys(declarations(THEMES.daylight)).sort()
    expect(daylight).toEqual(nocturne)
  })
})

const MOTION_TOKENS = [
  '--motion-instant',
  '--motion-quick',
  '--motion-normal',
  '--motion-slow',
  '--motion-stagger',
  '--motion-press',
  '--motion-rise',
  '--motion-spin',
]

describe('reduce motion', () => {
  const forced = () => declarations(":root[data-motion='reduce']")

  it('covers every motion token defined on :root', () => {
    const defined = Object.keys(declarations(':root')).filter((k) => k.startsWith('--motion-'))
    expect(defined.sort()).toEqual([...MOTION_TOKENS].sort())
  })

  it('the Settings override zeroes every motion token and the press scale', () => {
    const vars = forced()
    for (const token of MOTION_TOKENS) expect(vars[token], token).toMatch(/^0(ms|px)$/)
    expect(vars['--press-scale']).toBe('1')
  })

  it('the OS setting zeroes the same tokens unless Settings forces motion on', () => {
    const media = /@media \(prefers-reduced-motion: reduce\)\s*\{([\s\S]*?)\}\s*\}/.exec(css)
    expect(media).not.toBeNull()
    const os = declarations(":root:not([data-motion='full'])", `${media![1]}}`)
    expect(os).toEqual(forced())
  })
})
