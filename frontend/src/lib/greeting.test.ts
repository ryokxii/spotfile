import { describe, expect, it } from 'vitest'
import { greeting, initials, partOfDay } from './greeting'

describe('partOfDay', () => {
  it.each([
    [4, 'evening'],
    [5, 'morning'],
    [11, 'morning'],
    [12, 'afternoon'],
    [17, 'afternoon'],
    [18, 'evening'],
    [23, 'evening'],
  ])('%s h → %s', (hour, part) => {
    expect(partOfDay(hour)).toBe(part)
  })
})

describe('greeting', () => {
  it('uses the first name', () => {
    expect(greeting(20, 'Zafer Acar')).toBe('Good evening, Zafer')
  })

  it('has no name part without a name', () => {
    expect(greeting(9, '   ')).toBe('Good morning')
  })
})

describe('initials', () => {
  it.each([
    ['Zafer Acar', 'ZA'],
    ['zafer', 'Z'],
    ['Ana María de la Cruz', 'AC'],
    ['  élise   ', 'É'],
    ['', ''],
  ])('%s → %s', (name, letters) => {
    expect(initials(name)).toBe(letters)
  })
})
