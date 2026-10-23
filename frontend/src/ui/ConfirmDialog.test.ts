import { tick } from 'svelte'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import ConfirmDialog from './ConfirmDialog.svelte'

beforeAll(() => {
  // jsdom has <dialog> but not its modal methods.
  HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement) {
    this.open = true
  }
  HTMLDialogElement.prototype.close = function (this: HTMLDialogElement) {
    this.open = false
  }
})

afterEach(() => {
  document.body.innerHTML = ''
})

function mount(open = false) {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new ConfirmDialog({ target, props: { open, title: 'Clear the search index?', confirmLabel: 'Clear index' } })
  const events: string[] = []
  component.$on('confirm', () => events.push('confirm'))
  component.$on('cancel', () => events.push('cancel'))
  const dialog = target.querySelector('dialog') as HTMLDialogElement
  const button = (text: string) => [...target.querySelectorAll('button')].find((b) => b.textContent?.trim() === text) as HTMLButtonElement
  return { component, dialog, events, button }
}

describe('ConfirmDialog', () => {
  it('opens modally with focus on Cancel, and closes when open turns false', async () => {
    const { component, dialog } = mount()
    expect(dialog.open).toBe(false)

    component.$set({ open: true })
    await tick()
    await tick()
    expect(dialog.open).toBe(true)
    expect(document.activeElement?.textContent?.trim()).toBe('Cancel')

    component.$set({ open: false })
    await tick()
    expect(dialog.open).toBe(false)
  })

  it('Cancel dispatches cancel only', async () => {
    const { button, events } = mount(true)
    await tick()
    button('Cancel').click()
    expect(events).toEqual(['cancel'])
  })

  it('Esc dispatches cancel and leaves closing to the open prop', async () => {
    const { dialog, events } = mount(true)
    await tick()
    const esc = new Event('cancel', { cancelable: true })
    dialog.dispatchEvent(esc)
    expect(events).toEqual(['cancel'])
    expect(esc.defaultPrevented).toBe(true)
  })

  it('the confirm button dispatches confirm', async () => {
    const { button, events } = mount(true)
    await tick()
    button('Clear index').click()
    expect(events).toEqual(['confirm'])
  })
})
