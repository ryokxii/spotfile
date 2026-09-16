// Search query, results and the open preview. Lives outside the view so it
// survives navigating away from Search and back.

import { writable, type Readable } from 'svelte/store'
import { previewTargetFor, type PreviewTarget } from '../lib/fileTypes'

export interface SearchResult {
  docPath: string
  chunkIdx: number
  pageNum: number
  text: string
  score: number
}

export type SearchPhase = 'idle' | 'loading' | 'results' | 'error'

export interface Preview extends PreviewTarget {
  key: string // resultKey of the result being previewed
}

export interface SearchState {
  query: string
  phase: SearchPhase
  results: SearchResult[]
  error: string
  preview: Preview | null
}

export interface SearchApi {
  search(query: string, topK: number): Promise<SearchResult[]>
}

export interface SearchStore extends Readable<SearchState> {
  submit(query: string): Promise<void>
  open(result: SearchResult): void
  close(): void
  reset(): void
}

export const TOP_K = 5

const idle: SearchState = { query: '', phase: 'idle', results: [], error: '', preview: null }

// chunkIdx is only unique within a page, so key on path + page + chunk.
export function resultKey(r: SearchResult): string {
  return `${r.docPath}|${r.pageNum}|${r.chunkIdx}`
}

export function createSearchStore(api: SearchApi): SearchStore {
  const { subscribe, set, update } = writable<SearchState>(idle)
  let latest = 0 // only the most recent submit may write results

  return {
    subscribe,
    async submit(raw) {
      const query = raw.trim()
      if (!query) return
      const id = ++latest
      set({ ...idle, query, phase: 'loading' })
      try {
        const results = (await api.search(query, TOP_K)) ?? []
        if (id === latest) update((s) => ({ ...s, phase: 'results', results }))
      } catch (err) {
        if (id === latest) update((s) => ({ ...s, phase: 'error', error: err instanceof Error ? err.message : String(err) }))
      }
    },
    open(result) {
      update((s) => ({ ...s, preview: { ...previewTargetFor(result), key: resultKey(result) } }))
    },
    close() {
      update((s) => ({ ...s, preview: null }))
    },
    reset() {
      latest++
      set(idle)
    },
  }
}
