import { tick } from 'svelte'
import { afterEach, describe, expect, it } from 'vitest'
import NameField from './NameField.svelte'

afterEach(() => {
  document.body.innerHTML = ''
})

function mount(value = '') {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new NameField({ target, props: { value } })
  const saved: string[] = []
  component.$on('save', (e: CustomEvent<string>) => saved.push(e.detail))
  const input = target.querySelector('input') as HTMLInputElement
  const type = (text: string) => {
    input.focus()
    input.value = text
    input.dispatchEvent(new Event('input'))
  }
  const key = (k: string) => input.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true }))
  return { component, input, saved, type, key }
}

describe('NameField', () => {
  it('saves the trimmed name on blur', async () => {
    const { input, saved, type } = mount()
    type('  Zafer Acar  ')
    input.blur()
    await tick()
    expect(saved).toEqual(['Zafer Acar'])
  })

  it('saves on Enter', async () => {
    const { saved, type, key } = mount()
    type('Zafer')
    key('Enter')
    await tick()
    expect(saved).toEqual(['Zafer'])
  })

  it('does not save an unchanged name', async () => {
    const { input, saved, type } = mount('Zafer')
    type('Zafer ')
    input.blur()
    await tick()
    expect(saved).toEqual([])
  })

  it('Escape restores the saved name without saving', async () => {
    const { input, saved, type, key } = mount('Zafer')
    type('Somebody else')
    key('Escape')
    await tick()
    expect(saved).toEqual([])
    expect(input.value).toBe('Zafer')
  })

  it('limits typing to 60 characters', () => {
    const { input } = mount()
    expect(input.maxLength).toBe(60)
  })
})
