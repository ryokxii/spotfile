import { describe, expect, it } from 'vitest'
import { contrastRatio, parseHex } from './contrast'

describe('parseHex', () => {
  it('parses 6-digit hex case-insensitively', () => {
    expect(parseHex('#F7BC63')).toEqual({ r: 247, g: 188, b: 99 })
    expect(parseHex('#f7bc63')).toEqual({ r: 247, g: 188, b: 99 })
  })

  it('rejects anything that is not #RRGGBB', () => {
    expect(() => parseHex('F7BC63')).toThrow()
    expect(() => parseHex('#FFF')).toThrow()
    expect(() => parseHex('rgba(0,0,0,1)')).toThrow()
  })
})

describe('contrastRatio', () => {
  it('is 21:1 for black on white, in either order', () => {
    expect(contrastRatio('#000000', '#FFFFFF')).toBeCloseTo(21, 5)
    expect(contrastRatio('#FFFFFF', '#000000')).toBeCloseTo(21, 5)
  })

  it('is 1:1 for identical colours', () => {
    expect(contrastRatio('#101319', '#101319')).toBeCloseTo(1, 5)
  })

  it('matches the WCAG reference value for #767676 on white (4.54:1)', () => {
    expect(contrastRatio('#767676', '#FFFFFF')).toBeCloseTo(4.54, 2)
  })
})
