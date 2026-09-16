import { describe, expect, it } from 'vitest'
import { initialEngineState, type EngineState } from '../stores/engine'
import { statusLine } from './indexStatus'

const ready: EngineState = { ...initialEngineState, ready: true }

describe('statusLine', () => {
  it('shows the engine starting before it is ready', () => {
    expect(statusLine(initialEngineState)).toEqual({ tone: 'neutral', text: 'Starting search engine…', progress: null })
  })

  it('shows only the first line of an engine error, as an error', () => {
    const line = statusLine({ ...initialEngineState, error: 'llama.cpp not found — install it\nstack detail' })
    expect(line).toEqual({ tone: 'error', text: 'llama.cpp not found — install it', progress: null })
  })

  it('shows an index error', () => {
    expect(statusLine({ ...ready, indexError: 'no supported text or PDF files found in Empty' })).toMatchObject({
      tone: 'error',
      text: 'no supported text or PDF files found in Empty',
    })
  })

  it('prompts for a folder when nothing is indexed', () => {
    expect(statusLine(ready)).toMatchObject({ tone: 'neutral', text: 'No folder indexed yet' })
  })

  it('summarises the index, with singular/plural and thousands separators', () => {
    expect(statusLine({ ...ready, documents: 1, chunks: 1 }).text).toBe('1 file · 1 chunk')
    expect(statusLine({ ...ready, documents: 219, chunks: 12819 }).text).toBe('219 files · 12,819 chunks')
  })

  it('mentions the watched folder by name', () => {
    expect(statusLine({ ...ready, documents: 3, chunks: 30, watching: true, folder: '/Users/z/Notes' }).text).toBe(
      '3 files · 30 chunks · watching Notes',
    )
  })

  it('shows preparation before chunk totals are known', () => {
    const indexing = { files: 30, filesDone: 0, chunks: 0, chunksDone: 0, currentFile: '', seen: new Set<string>() }
    expect(statusLine({ ...ready, indexing })).toEqual({ tone: 'active', text: 'Preparing 30 files…', progress: null })
  })

  it('shows file, percentage and file count while indexing, capped at 100%', () => {
    const indexing = { files: 30, filesDone: 12, chunks: 200, chunksDone: 84, currentFile: '/n/notes.pdf', seen: new Set<string>() }
    expect(statusLine({ ...ready, indexing })).toEqual({
      tone: 'active',
      text: 'Indexing notes.pdf · 42% · 12 of 30 files',
      progress: 0.42,
    })
    const over = { ...indexing, chunksDone: 230 }
    expect(statusLine({ ...ready, indexing: over })).toMatchObject({ progress: 1 })
  })

  it('an index error outranks progress, and indexing outranks a live update', () => {
    const indexing = { files: 2, filesDone: 1, chunks: 10, chunksDone: 5, currentFile: '/a.md', seen: new Set<string>() }
    expect(statusLine({ ...ready, indexing, indexError: 'boom' }).tone).toBe('error')
    expect(statusLine({ ...ready, indexing, reindexingFile: '/b.md' }).text).toMatch(/^Indexing/)
    expect(statusLine({ ...ready, documents: 2, chunks: 9, reindexingFile: '/b.md' })).toEqual({
      tone: 'active',
      text: 'Updating b.md',
      progress: null,
    })
  })
})
