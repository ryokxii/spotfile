import { afterEach, describe, expect, it } from 'vitest'
import ProfileButton from './ProfileButton.svelte'

afterEach(() => {
  document.body.innerHTML = ''
})

function mount(props: { name?: string; compact?: boolean; selected?: boolean }) {
  const target = document.createElement('div')
  document.body.append(target)
  new ProfileButton({ target, props })
  return target.querySelector('a') as HTMLAnchorElement
}

describe('ProfileButton', () => {
  it('links to Profile with initials and the name', () => {
    const link = mount({ name: 'Zafer Acar' })
    expect(link.getAttribute('href')).toBe('#/profile')
    expect(link.querySelector('.avatar')?.textContent).toBe('ZA')
    expect(link.textContent).toContain('Zafer Acar')
  })

  it('invites a name when there is none', () => {
    const link = mount({ name: '' })
    expect(link.textContent).toContain('Add your name')
  })

  it('shows only the initials in the compact rail, labelled for screen readers', () => {
    const link = mount({ name: 'Zafer Acar', compact: true })
    expect(link.textContent?.trim()).toBe('ZA')
    expect(link.getAttribute('aria-label')).toBe('Profile — Zafer Acar')
  })
})
