import { tick } from 'svelte'
import { afterEach, describe, expect, it } from 'vitest'
import StructuredView from './StructuredView.svelte'

afterEach(() => {
  document.body.innerHTML = ''
})

// '{"a":{"b":1},"c":[1,2]}' pretty-printed:
// 1 {            ← folds to 9
// 2   "a": {     ← folds to 4
// 3     "b": 1
// 4   },
// 5   "c": [     ← folds to 8
// 6     1,
// 7     2
// 8   ]
// 9 }
const JSON_SOURCE = '{"a":{"b":1},"c":[1,2]}'
const FORMATTED = '{\n  "a": {\n    "b": 1\n  },\n  "c": [\n    1,\n    2\n  ]\n}'

function mount(kind: 'json' | 'yaml', source: string) {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new StructuredView({ target, props: { kind, source } })
  const root = target.querySelector('.structured') as HTMLElement
  const lines = () => [...target.querySelectorAll('.line')] as HTMLElement[]
  const toggles = () => [...target.querySelectorAll('button.toggle')] as HTMLButtonElement[]
  return { component, root, lines, toggles }
}

describe('StructuredView', () => {
  it('renders highlighted lines whose text is exactly the source', () => {
    const { root, lines } = mount('json', JSON_SOURCE)
    expect(lines()).toHaveLength(9)
    expect([...root.querySelectorAll('.tok-key')].map((e) => e.textContent)).toEqual(['"a"', '"b"', '"c"'])
    expect([...root.querySelectorAll('.tok-number')].map((e) => e.textContent)).toEqual(['1', '1', '2'])
    expect(root.textContent).toBe(FORMATTED.split('\n').join(''))
    expect(lines()[4].dataset.line).toBe('5')
  })

  it('offers a toggle on every foldable line', () => {
    const { toggles, lines } = mount('json', JSON_SOURCE)
    expect(toggles()).toHaveLength(3)
    expect(toggles().map((t) => t.closest('.line')?.getAttribute('data-line'))).toEqual(['1', '2', '5'])
    expect(toggles().every((t) => t.getAttribute('aria-expanded') === 'true')).toBe(true)
    expect(lines()).toHaveLength(9)
  })

  it('collapses a block to a summary and expands it again', async () => {
    const { toggles, lines, root } = mount('json', JSON_SOURCE)
    toggles()[1].click()
    await tick()
    expect(lines()).toHaveLength(7)
    expect(lines()[1].textContent).toBe('  "a": { … 1 key },')
    expect(root.textContent).not.toContain('"b"')
    expect(toggles()[1].getAttribute('aria-expanded')).toBe('false')

    toggles()[1].click()
    await tick()
    expect(lines()).toHaveLength(9)
  })

  it('collapses and expands everything', async () => {
    const { component, lines } = mount('json', JSON_SOURCE)
    component.collapseAll()
    await tick()
    expect(lines().map((l) => l.textContent)).toEqual(['{ … 2 keys }'])
    component.expandAll()
    await tick()
    expect(lines()).toHaveLength(9)
  })

  it('folds YAML blocks with a line count', async () => {
    const { toggles, lines } = mount('yaml', 'server:\n  host: a\n  port: 80\nname: x')
    expect(toggles()).toHaveLength(1)
    toggles()[0].click()
    await tick()
    expect(lines().map((l) => l.textContent)).toEqual(['server: … 2 lines', 'name: x'])
  })
})
