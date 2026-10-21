import { tick } from 'svelte'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import type { DashboardData, StaleFile } from '../../stores/dashboard'
import DashboardSearch from './DashboardSearch.svelte'
import DiskTile from './DiskTile.svelte'
import FreeSpaceTile from './FreeSpaceTile.svelte'
import RecentFilesTile from './RecentFilesTile.svelte'
import RecentSearchesTile from './RecentSearchesTile.svelte'
import StaleFileList from './StaleFileList.svelte'

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

const now = new Date(2026, 9, 21, 12, 0)
const minutesAgo = (m: number) => new Date(now.getTime() - m * 60_000).toISOString()

function mount<P extends Record<string, unknown>>(Component: any, props: P) {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new Component({ target, props })
  const events: [string, unknown][] = []
  for (const name of ['search', 'open', 'forget', 'remove', 'trash', 'showAll']) {
    component.$on(name, (e: CustomEvent) => events.push([name, e.detail]))
  }
  const button = (text: string) =>
    [...target.querySelectorAll('button')].find((b) => b.textContent?.trim() === text) as HTMLButtonElement
  return { target, events, button }
}

const data: DashboardData = {
  folders: [{ path: '/n', files: 412 }],
  indexBytes: 2_200_000,
  files: 412,
  chunks: 9860,
  types: [
    { type: 'Markdown', files: 121 },
    { type: 'PDF', files: 214 },
    { type: 'Text', files: 77 },
  ],
  lastUpdate: { path: '/n/meeting-notes.md', at: minutesAgo(3) },
  recentSearches: [],
  recentFiles: [],
  stale: [],
  staleBytes: 0,
  history: true,
  staleDays: 90,
}

const staleFile = (name: string, bytes: number): StaleFile => ({ path: `/n/${name}`, bytes, modified: minutesAgo(200 * 24 * 60) })

describe('DashboardSearch', () => {
  it('dispatches the trimmed query on Enter, and nothing for a blank one', () => {
    const { target, events } = mount(DashboardSearch, {})
    const input = target.querySelector('input') as HTMLInputElement
    input.value = '   '
    input.dispatchEvent(new Event('input'))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    input.value = '  grant deadlines '
    input.dispatchEvent(new Event('input'))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(events).toEqual([['search', 'grant deadlines']])
  })

  it('Ctrl+K focuses the search bar', () => {
    const { target } = mount(DashboardSearch, {})
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    expect(document.activeElement).toBe(target.querySelector('input'))
  })
})

describe('DiskTile', () => {
  it('shows the index size, counts, types in a fixed order and the last update', () => {
    const { target } = mount(DiskTile, { data, now })
    expect(target.querySelector('.value')?.textContent).toBe('2.1')
    expect(target.textContent).toContain('MB index')
    expect(target.textContent).toContain('412 files · 9,860 chunks')
    const legend = [...target.querySelectorAll('.legend li')].map((li) => li.textContent?.replace(/\s+/g, ' ').trim())
    expect(legend).toEqual(['PDF 214', 'Markdown 121', 'Text 77'])
    expect(target.textContent).toContain('meeting-notes.md')
    expect(target.textContent).toContain('3 min ago')
  })

  it('cuts a long last-updated filename with an ellipsis and keeps the full path in a tooltip', () => {
    const { target } = mount(DiskTile, { data, now })
    const name = target.querySelector('.well strong') as HTMLElement
    expect(name.classList.contains('truncate')).toBe(true)
    expect(name.closest('[title]')?.getAttribute('title')).toBe('/n/meeting-notes.md')
  })

  it('says so when nothing has been updated yet', () => {
    const { target } = mount(DiskTile, { data: { ...data, lastUpdate: null, types: [] }, now })
    expect(target.textContent).toContain('Nothing updated yet')
    expect(target.querySelector('.bar')).toBeNull()
  })
})

describe('RecentFilesTile', () => {
  const files = [
    { path: '/n/thesis.pdf', at: minutesAgo(120) },
    { path: '/n/notes.md', at: minutesAgo(30 * 60) },
  ]

  it('opens and forgets files', () => {
    const { target, events } = mount(RecentFilesTile, { files, history: true, now })
    ;(target.querySelector('.open') as HTMLButtonElement).click()
    ;(target.querySelector('.forget') as HTMLButtonElement).click()
    expect(events).toEqual([
      ['open', '/n/thesis.pdf'],
      ['forget', '/n/thesis.pdf'],
    ])
    expect(target.textContent).toContain('2h ago')
    expect(target.textContent).toContain('yesterday')
  })

  it('has quiet empty states', () => {
    expect(mount(RecentFilesTile, { files: [], history: true, now }).target.textContent).toContain('Files you open will appear here')
    expect(mount(RecentFilesTile, { files: [], history: false, now }).target.textContent).toContain('History is off')
  })
})

describe('RecentSearchesTile', () => {
  it('cuts a long search with an ellipsis and shows it whole in a tooltip', () => {
    const query = 'a very long search about every grant deadline this year and the next'
    const { button } = mount(RecentSearchesTile, { searches: [{ query, at: minutesAgo(5) }], history: true })
    const chip = button(query)
    expect(chip.classList.contains('truncate')).toBe(true)
    expect(chip.title).toBe(query)
  })

  it('runs a recent search again', () => {
    const { button, events } = mount(RecentSearchesTile, { searches: [{ query: 'invoice march', at: minutesAgo(5) }], history: true })
    button('invoice march').click()
    expect(events).toEqual([['search', 'invoice march']])
  })

  it('has quiet empty states', () => {
    expect(mount(RecentSearchesTile, { searches: [], history: true }).target.textContent).toContain('Your searches will appear here')
    expect(mount(RecentSearchesTile, { searches: [], history: false }).target.textContent).toContain('History is off')
  })
})

describe('StaleFileList', () => {
  const files = [staleFile('old-syllabus.pdf', 48_000), staleFile('draft.md', 12_000)]

  it('removes from the index straight away', () => {
    const { target, events } = mount(StaleFileList, { files, busy: false, errors: {}, now })
    ;(target.querySelector('.remove') as HTMLButtonElement).click()
    expect(events).toEqual([['remove', '/n/old-syllabus.pdf']])
  })

  it('moves to the Trash only after confirming', async () => {
    const { target, events, button } = mount(StaleFileList, { files, busy: false, errors: {}, now })
    ;(target.querySelector('.trash') as HTMLButtonElement).click()
    await tick()
    expect(events).toEqual([])
    expect(target.querySelector('dialog')?.open).toBe(true)
    expect(target.textContent).toContain('Move old-syllabus.pdf to the Trash?')
    button('Move to Trash').click()
    await tick()
    expect(events).toEqual([['trash', '/n/old-syllabus.pdf']])
  })

  it('shows a row error and disables actions while busy', () => {
    const { target } = mount(StaleFileList, { files, busy: true, errors: { '/n/draft.md': 'Nothing was deleted.' }, now })
    expect(target.textContent).toContain('Nothing was deleted.')
    expect((target.querySelector('.remove') as HTMLButtonElement).disabled).toBe(true)
    expect(target.textContent).toContain('6 months')
  })
})

describe('FreeSpaceTile', () => {
  it('lists the three largest and offers the rest', () => {
    const stale = ['a', 'b', 'c', 'd'].map((n, i) => staleFile(`${n}.md`, 4000 - i))
    const { target, events, button } = mount(FreeSpaceTile, { stale, staleBytes: 15_994, staleDays: 90, busy: false, errors: {}, now })
    expect(target.querySelectorAll('.stale-row')).toHaveLength(3)
    expect(target.textContent).toContain('4 files untouched for 90+ days')
    button('Show all 4 →').click()
    expect(events).toEqual([['showAll', null]])
  })

  it('says when nothing is stale', () => {
    const { target } = mount(FreeSpaceTile, { stale: [], staleBytes: 0, staleDays: 30, busy: false, errors: {}, now })
    expect(target.textContent).toContain('Nothing stale. Everything was used in the last 30 days.')
  })
})
