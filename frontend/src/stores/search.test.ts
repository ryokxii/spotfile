import { get } from 'svelte/store'
import { describe, expect, it, vi } from 'vitest'
import { createSearchStore, resultKey, type SearchResult } from './search'

const result = (docPath: string, pageNum = 0, chunkIdx = 0): SearchResult => ({
  docPath,
  pageNum,
  chunkIdx,
  text: `text of ${docPath}`,
  score: 0.8,
})

describe('createSearchStore', () => {
  it('ignores blank queries', async () => {
    const api = { search: vi.fn() }
    const search = createSearchStore(api)
    await search.submit('   ')
    expect(api.search).not.toHaveBeenCalled()
    expect(get(search).phase).toBe('idle')
  })

  it('moves idle → loading → results and keeps the trimmed query', async () => {
    let resolve!: (r: SearchResult[]) => void
    const api = { search: vi.fn(() => new Promise<SearchResult[]>((r) => (resolve = r))) }
    const search = createSearchStore(api)

    const pending = search.submit('  midterm date ')
    expect(get(search)).toMatchObject({ phase: 'loading', query: 'midterm date' })
    expect(api.search).toHaveBeenCalledWith('midterm date', 5)

    resolve([result('/a.pdf', 2)])
    await pending
    expect(get(search).phase).toBe('results')
    expect(get(search).results).toHaveLength(1)
  })

  it('treats a null response as no results', async () => {
    const search = createSearchStore({ search: async () => null as unknown as SearchResult[] })
    await search.submit('q')
    expect(get(search)).toMatchObject({ phase: 'results', results: [] })
  })

  it('records the backend error message', async () => {
    const search = createSearchStore({ search: async () => Promise.reject(new Error('engine not initialised')) })
    await search.submit('q')
    expect(get(search)).toMatchObject({ phase: 'error', error: 'engine not initialised' })
  })

  it('a newer search wins over a slower earlier one', async () => {
    const resolvers: ((r: SearchResult[]) => void)[] = []
    const search = createSearchStore({ search: () => new Promise<SearchResult[]>((r) => resolvers.push(r)) })

    const first = search.submit('first')
    const second = search.submit('second')
    resolvers[1]([result('/second.md')])
    await second
    resolvers[0]([result('/first.md')])
    await first

    expect(get(search).query).toBe('second')
    expect(get(search).results[0].docPath).toBe('/second.md')
  })

  it('opens and closes a preview, and a new search closes it', async () => {
    const search = createSearchStore({ search: async () => [result('/a.pdf', 3)] })
    await search.submit('q')
    search.open(get(search).results[0])
    expect(get(search).preview).toMatchObject({ kind: 'pdf', path: '/a.pdf', page: 3, key: resultKey(result('/a.pdf', 3)) })

    search.close()
    expect(get(search).preview).toBeNull()

    search.open(get(search).results[0])
    await search.submit('again')
    expect(get(search).preview).toBeNull()
  })

  it('reset returns to idle', async () => {
    const search = createSearchStore({ search: async () => [result('/a.md')] })
    await search.submit('q')
    search.reset()
    expect(get(search)).toMatchObject({ phase: 'idle', query: '', results: [], preview: null })
  })
})

describe('resultKey', () => {
  it('distinguishes chunks with the same index on different pages', () => {
    expect(resultKey(result('/a.pdf', 1, 0))).not.toBe(resultKey(result('/a.pdf', 2, 0)))
  })
})
