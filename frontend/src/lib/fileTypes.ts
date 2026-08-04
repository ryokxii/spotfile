// Single source of truth for how a search result is previewed.
//
// The backend (Go's isIndexable) owns *what* gets indexed; the frontend only
// decides *how* to render a result it received. Since every result is already
// an indexed file, anything that isn't a PDF or Markdown document renders as
// plain text — so there's no need to duplicate the backend's extension list.

export type ViewerKind = 'pdf' | 'markdown' | 'text'

export interface ResultLike {
  docPath: string
  pageNum: number
  text: string
}

export interface PreviewTarget {
  kind: ViewerKind
  path: string
  page: number
  highlight: string
}

const MARKDOWN_EXTENSIONS = new Set(['.md', '.markdown'])

function extensionOf(path: string): string {
  const dot = path.lastIndexOf('.')
  return dot === -1 ? '' : path.slice(dot).toLowerCase()
}

// Which viewer renders a given file path.
export function viewerKindFor(path: string): ViewerKind {
  const ext = extensionOf(path)
  if (ext === '.pdf') return 'pdf'
  if (MARKDOWN_EXTENSIONS.has(ext)) return 'markdown'
  return 'text'
}

// Build the preview-pane target for a search result.
export function previewTargetFor(result: ResultLike): PreviewTarget {
  return {
    kind: viewerKindFor(result.docPath),
    path: result.docPath,
    page: result.pageNum > 0 ? result.pageNum : 1,
    highlight: result.text,
  }
}
