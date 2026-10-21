import { tick } from 'svelte'
import { afterEach, describe, expect, it } from 'vitest'
import SegmentedControl from './SegmentedControl.svelte'

const options = [
  { value: 'system', label: 'System' },
  { value: 'nocturne', label: 'Nocturne' },
  { value: 'daylight', label: 'Daylight' },
]

function mount(value = 'system') {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new SegmentedControl({ target, props: { label: 'Theme', options, value } })
  const changes: string[] = []
  component.$on('change', (e: CustomEvent<string>) => changes.push(e.detail))
  const radios = () => [...target.querySelectorAll<HTMLButtonElement>('[role="radio"]')]
  return { component, target, changes, radios }
}

const key = (el: HTMLElement, name: string) => el.dispatchEvent(new KeyboardEvent('keydown', { key: name, bubbles: true }))

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SegmentedControl', () => {
  it('is a labelled radiogroup with only the current value checked and tabbable', () => {
    const { target, radios } = mount()
    expect(target.querySelector('[role="radiogroup"]')?.getAttribute('aria-label')).toBe('Theme')
    expect(radios().map((r) => r.getAttribute('aria-checked'))).toEqual(['true', 'false', 'false'])
    expect(radios().map((r) => r.tabIndex)).toEqual([0, -1, -1])
  })

  it('clicking another option dispatches change; clicking the current one does not', () => {
    const { radios, changes } = mount()
    radios()[0].click()
    radios()[2].click()
    expect(changes).toEqual(['daylight'])
  })

  it('arrow keys move the selection and wrap around', () => {
    const { radios, changes } = mount()
    key(radios()[0], 'ArrowRight')
    key(radios()[0], 'ArrowLeft')
    key(radios()[0], 'Enter')
    expect(changes).toEqual(['nocturne', 'daylight'])
  })

  it('follows the value prop', async () => {
    const { component, radios } = mount()
    component.$set({ value: 'nocturne' })
    await tick()
    expect(radios()[1].getAttribute('aria-checked')).toBe('true')
    expect(radios()[1].tabIndex).toBe(0)
  })
})
