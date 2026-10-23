import { afterEach, describe, expect, it } from 'vitest'
import { locateChunkRange } from './textHighlight'

afterEach(() => {
  document.body.innerHTML = ''
})

function pre(text: string): HTMLElement {
  const el = document.createElement('pre')
  el.textContent = text
  document.body.append(el)
  return el
}

describe('locateChunkRange', () => {
  it('matches across whitespace differences by collapsing runs', () => {
    const range = locateChunkRange(pre('alpha   beta\ngamma delta'), 'alpha beta gamma')
    expect(range?.toString()).toBe('alpha   beta\ngamma')
  })

  it('finds a whitespace-free chunk in reformatted text only when told to ignore whitespace', () => {
    const root = pre('{\n  "name": "Spotfile",\n  "port": 8080\n}')
    const chunk = '{"name":"Spotfile","port":8080}'
    expect(locateChunkRange(root, chunk)).toBeNull()
    expect(locateChunkRange(root, chunk, { ignoreWhitespace: true })?.toString()).toBe('{\n  "name": "Spotfile",\n  "port": 8080\n}')
  })

  it('maps a match that spans several elements back to the DOM', () => {
    const root = document.createElement('div')
    root.innerHTML = '<div><span>"a"</span><span>: </span><span>1</span></div><div><span>"b"</span></div>'
    document.body.append(root)
    expect(locateChunkRange(root, '"a":1 "b"', { ignoreWhitespace: true })?.toString()).toBe('"a": 1"b"')
  })
})
