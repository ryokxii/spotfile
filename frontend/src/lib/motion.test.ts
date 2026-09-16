import { afterEach, describe, expect, it } from 'vitest'
import { duration } from './motion'

afterEach(() => {
  document.documentElement.removeAttribute('style')
})

describe('duration', () => {
  it('reads a millisecond motion token from the root element', () => {
    document.documentElement.style.setProperty('--motion-normal', '240ms')
    expect(duration('normal')).toBe(240)
  })

  it('returns 0 when reduced motion zeroes the token', () => {
    document.documentElement.style.setProperty('--motion-slow', '0ms')
    expect(duration('slow')).toBe(0)
  })

  it('returns 0 for a missing or malformed token rather than NaN', () => {
    expect(duration('quick')).toBe(0)
    document.documentElement.style.setProperty('--motion-quick', 'fast')
    expect(duration('quick')).toBe(0)
  })
})
