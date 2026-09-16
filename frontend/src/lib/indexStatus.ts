// The one-line index status shown under the search bar, derived from engine
// state. Precedence: errors → indexing → live update → starting → summary.

import type { EngineState } from '../stores/engine'
import { filename } from './path'

export type StatusTone = 'neutral' | 'active' | 'error'

export interface StatusLine {
  tone: StatusTone
  text: string
  progress: number | null // 0..1 while indexing with known totals
}

const count = (n: number, noun: string): string => `${n.toLocaleString('en-US')} ${noun}${n === 1 ? '' : 's'}`

const firstLine = (message: string): string => message.split('\n')[0]

export function statusLine(s: EngineState): StatusLine {
  if (s.error) return { tone: 'error', text: firstLine(s.error), progress: null }
  if (s.indexError) return { tone: 'error', text: firstLine(s.indexError), progress: null }

  if (s.indexing) {
    const { files, filesDone, chunks, chunksDone, currentFile } = s.indexing
    if (chunks === 0) return { tone: 'active', text: `Preparing ${count(files, 'file')}…`, progress: null }
    const progress = Math.min(1, chunksDone / chunks)
    const parts = [
      currentFile ? `Indexing ${filename(currentFile)}` : 'Indexing',
      `${Math.round(progress * 100)}%`,
      `${filesDone} of ${count(files, 'file')}`,
    ]
    return { tone: 'active', text: parts.join(' · '), progress }
  }

  if (s.reindexingFile) return { tone: 'active', text: `Updating ${filename(s.reindexingFile)}`, progress: null }
  if (!s.ready) return { tone: 'neutral', text: 'Starting search engine…', progress: null }
  if (s.documents === 0) return { tone: 'neutral', text: 'No folder indexed yet', progress: null }

  const parts = [count(s.documents, 'file'), count(s.chunks, 'chunk')]
  if (s.watching && s.folder) parts.push(`watching ${filename(s.folder)}`)
  return { tone: 'neutral', text: parts.join(' · '), progress: null }
}
