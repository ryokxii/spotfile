import { describe, expect, it } from 'vitest'
import { age, ago } from './time'

const now = new Date(2026, 9, 21, 12, 0) // Wed 21 Oct 2026, local time
const minutesAgo = (m: number) => new Date(now.getTime() - m * 60_000)

describe('ago', () => {
  it.each([
    [0.5, 'just now'],
    [3, '3 min ago'],
    [59, '59 min ago'],
    [60, '1h ago'],
    [23 * 60, '23h ago'],
    [30 * 60, 'yesterday'],
    [3 * 24 * 60, 'Sun'],
    [20 * 24 * 60, 'Oct 1'],
  ])('%s minutes → %s', (m, text) => {
    expect(ago(minutesAgo(m), now)).toBe(text)
  })
})

describe('age', () => {
  it.each([
    [1, '1 day'],
    [45, '45 days'],
    [59, '59 days'],
    [61, '2 months'],
    [430, '14 months'],
  ])('%s days → %s', (days, text) => {
    expect(age(minutesAgo(days * 24 * 60), now)).toBe(text)
  })
})
