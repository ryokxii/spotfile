import { describe, expect, it } from 'vitest'
import { formatBytes, truncateMiddle } from './format'

describe('formatBytes', () => {
  it.each([
    [0, '0 B'],
    [512, '512 B'],
    [1024, '1.0 KB'],
    [1536, '1.5 KB'],
    [2_150_000, '2.1 MB'],
    [5 * 1024 ** 3, '5.0 GB'],
    [1024 ** 4, '1024.0 GB'],
    [-5, '0 B'],
    [Number.NaN, '0 B'],
  ])('%s → %s', (bytes, text) => {
    expect(formatBytes(bytes)).toBe(text)
  })
})

describe('truncateMiddle', () => {
  it('leaves short paths alone', () => {
    expect(truncateMiddle('/a/b', 20)).toBe('/a/b')
  })

  it('cuts the middle and keeps the last segment whole', () => {
    expect(truncateMiddle('/Users/zafer/Documents/Research', 20)).toBe('/Users/zaf…/Research')
  })

  it('handles Windows separators', () => {
    expect(truncateMiddle('C:\\Users\\zafer\\Documents\\Research', 20)).toBe('C:\\Users\\z…\\Research')
  })

  it('falls back to a plain middle cut when the last segment is too long', () => {
    const out = truncateMiddle('/x/' + 'a'.repeat(30), 11)
    expect(out).toHaveLength(11)
    expect(out.startsWith('/x/aa')).toBe(true)
    expect(out).toContain('…')
  })
})
