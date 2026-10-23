// Highlighting and folding for JSON and YAML in the file viewer. Pure functions:
// tokens keep every source character, line by line, so the rendered text is
// exactly the file's text and search highlighting can still find chunks in it.

export type StructuredKind = 'json' | 'yaml'
export type TokenKind = 'key' | 'string' | 'number' | 'literal' | 'punct' | 'comment' | 'plain'

export interface Token {
  kind: TokenKind
  text: string
}

// A foldable block: its first and last line (0-based, inclusive) and what to
// show in place of the hidden lines.
export interface Fold {
  start: number
  end: number
  label: string
}

export interface Structured {
  lines: Token[][]
  folds: Fold[] // sorted by start, at most one per start line
}

const KIND_BY_EXT: Record<string, StructuredKind> = {
  '.json': 'json',
  '.jsonc': 'json',
  '.yaml': 'yaml',
  '.yml': 'yaml',
}

// Which structured viewer renders a path, or null for any other file.
export function structuredKindFor(path: string): StructuredKind | null {
  const dot = path.lastIndexOf('.')
  const slash = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  if (dot <= slash) return null
  return KIND_BY_EXT[path.slice(dot).toLowerCase()] ?? null
}

const plural = (n: number, word: string): string => `${n} ${word}${n === 1 ? '' : 's'}`

// Pretty-prints a minified JSON file (one line that parses). Anything else —
// already formatted, with comments, or invalid — is returned as written.
export function formatJson(source: string): string {
  const trimmed = source.trim()
  if (!trimmed || trimmed.includes('\n')) return source
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return source
  }
}

// Ordered JSON(C) token rules, tried at each position; the first match wins.
const JSON_RULES: [TokenKind | 'newline', RegExp][] = [
  ['newline', /\n/y],
  ['plain', /[ \t\r]+/y],
  ['comment', /\/\/[^\n]*/y],
  ['comment', /\/\*[\s\S]*?(?:\*\/|$)/y],
  ['string', /"(?:[^"\\\n]|\\.)*"?/y],
  ['number', /-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/y],
  ['literal', /(?:true|false|null)\b/y],
  ['punct', /[{}[\],:]/y],
  ['plain', /[^\s"{}[\],:/]+|\//y],
]

// Splits JSON (comments allowed) into tokens per line. A string followed by
// ':' on its line is a key.
export function tokenizeJson(source: string): Token[][] {
  const lines: Token[][] = [[]]
  const push = (kind: TokenKind, text: string) => {
    text.split('\n').forEach((part, n) => {
      if (n > 0) lines.push([])
      if (part) lines[lines.length - 1].push({ kind, text: part })
    })
  }
  let i = 0
  while (i < source.length) {
    for (const [kind, rule] of JSON_RULES) {
      rule.lastIndex = i
      const match = rule.exec(source)
      if (!match || match[0] === '') continue
      if (kind === 'newline') lines.push([])
      else push(kind, match[0])
      i += match[0].length
      break
    }
  }
  return lines.map(markKeys)
}

function markKeys(line: Token[]): Token[] {
  return line.map((token, i) => {
    if (token.kind !== 'string') return token
    const next = line.slice(i + 1).find((t) => t.kind !== 'plain')
    return next?.kind === 'punct' && next.text === ':' ? { ...token, kind: 'key' } : token
  })
}

interface OpenBracket {
  char: '{' | '['
  line: number
  commas: number
  hasContent: boolean
}

// Multi-line objects and arrays, labelled with how many keys or items they hold.
// When several brackets open on one line, the outermost fold is kept.
export function jsonFolds(lines: Token[][]): Fold[] {
  const stack: OpenBracket[] = []
  const byStart = new Map<number, Fold>()
  lines.forEach((line, lineIndex) => {
    for (const token of line) {
      if (token.kind === 'plain' || token.kind === 'comment') continue
      const top = stack[stack.length - 1]
      if (token.kind === 'punct' && (token.text === '{' || token.text === '[')) {
        if (top) top.hasContent = true
        stack.push({ char: token.text, line: lineIndex, commas: 0, hasContent: false })
      } else if (token.kind === 'punct' && (token.text === '}' || token.text === ']')) {
        const open = stack.pop()
        if (!open || lineIndex === open.line) continue
        const count = open.hasContent ? open.commas + 1 : 0
        const fold = { start: open.line, end: lineIndex, label: plural(count, open.char === '{' ? 'key' : 'item') }
        const existing = byStart.get(fold.start)
        if (!existing || fold.end >= existing.end) byStart.set(fold.start, fold)
      } else if (token.kind === 'punct' && token.text === ',') {
        if (top) top.commas++
      } else if (top && !(token.kind === 'punct' && token.text === ':')) {
        top.hasContent = true
      }
    }
  })
  return [...byStart.values()].sort((a, b) => a.start - b.start)
}

const YAML_KEY = /^((?:"(?:[^"\\]|\\.)*")|(?:'(?:[^']|'')*')|(?:[^\s#'"{}[\],&*!|>%@`][^#:]*?))(\s*)(:)(?=\s|$)/
const YAML_NUMBER = /^[-+]?(?:\d[\d_]*(?:\.\d*)?(?:[eE][-+]?\d+)?|\.\d+|0x[0-9a-fA-F]+|0o[0-7]+|\.inf|\.nan)$/i
const YAML_LITERAL = /^(?:true|false|yes|no|on|off|null|~)$/i
const YAML_BLOCK = /^[|>][-+0-9]*(?=\s|$)/
const YAML_QUOTED = /^(?:"(?:[^"\\]|\\.)*"?|'(?:[^']|'')*'?)/
const YAML_PROPERTY = /^[&*!][^\s,{}[\]]*/

const indentOf = (line: string): number => line.length - line.trimStart().length

// Splits YAML into tokens per line: keys, scalars, comments, list dashes and
// block scalar bodies. Flow collections ({…}, […]) stay plain.
export function tokenizeYaml(source: string): Token[][] {
  let blockScalarIndent: number | null = null // inside `key: |` while lines are deeper than this
  return source.split('\n').map((line) => {
    const tokens: Token[] = []
    const push = (kind: TokenKind, text: string) => text && tokens.push({ kind, text })
    const indent = indentOf(line)

    if (blockScalarIndent !== null) {
      if (line.trim() === '') return line ? [{ kind: 'plain', text: line }] : []
      if (indent > blockScalarIndent) {
        push('plain', line.slice(0, indent))
        push('string', line.slice(indent))
        return tokens
      }
      blockScalarIndent = null
    }

    push('plain', line.slice(0, indent))
    let rest = line.slice(indent)
    if (rest.startsWith('#')) {
      push('comment', rest)
      return tokens
    }
    const marker = /^(?:---|\.\.\.)(?=\s|$)/.exec(rest)
    if (marker) {
      push('punct', marker[0])
      push('plain', rest.slice(marker[0].length))
      return tokens
    }
    for (let dash = /^-(\s+|$)/.exec(rest); dash; dash = /^-(\s+|$)/.exec(rest)) {
      push('punct', '-')
      push('plain', dash[1])
      rest = rest.slice(dash[0].length)
    }
    const key = YAML_KEY.exec(rest)
    if (key) {
      push('key', key[1])
      push('plain', key[2])
      push('punct', key[3])
      rest = rest.slice(key[0].length)
    }
    if (tokenizeYamlValue(rest, push)) blockScalarIndent = indent
    return tokens
  })
}

// Tokenizes what follows a key or dash. Returns true when it opens a block scalar.
function tokenizeYamlValue(value: string, push: (kind: TokenKind, text: string) => void): boolean {
  let rest = value
  const take = (kind: TokenKind, length: number) => {
    push(kind, rest.slice(0, length))
    rest = rest.slice(length)
  }
  const space = () => take('plain', rest.length - rest.trimStart().length)

  space()
  for (let prop = YAML_PROPERTY.exec(rest); prop; prop = YAML_PROPERTY.exec(rest)) {
    take('punct', prop[0].length)
    space()
  }
  if (rest.startsWith('#')) {
    take('comment', rest.length)
    return false
  }
  const block = YAML_BLOCK.exec(rest)
  if (block) {
    take('punct', block[0].length)
    space()
    take(rest.startsWith('#') ? 'comment' : 'plain', rest.length)
    return true
  }
  const quoted = YAML_QUOTED.exec(rest)
  if (quoted) {
    take('string', quoted[0].length)
  } else if (rest && !/^[{[]/.test(rest)) {
    const commentAt = rest.search(/\s#/)
    const scalar = (commentAt === -1 ? rest : rest.slice(0, commentAt)).trimEnd()
    take(YAML_NUMBER.test(scalar) ? 'number' : YAML_LITERAL.test(scalar) ? 'literal' : 'string', scalar.length)
  }
  space()
  take(rest.startsWith('#') ? 'comment' : 'plain', rest.length)
  return false
}

// Indented YAML blocks: a line followed by more-indented lines, or a `key:`
// followed by `- ` items at its own indent. Blank lines inside are included,
// trailing ones aren't; comment lines never start a fold.
export function yamlFolds(lines: string[]): Fold[] {
  const folds: Fold[] = []
  lines.forEach((line, start) => {
    const content = line.trim()
    if (content === '' || content.startsWith('#')) return
    const indent = indentOf(line)
    const keyWithoutValue = /:\s*(?:#.*)?$/.test(content) && !content.startsWith('- ')
    let end = start
    for (let j = start + 1; j < lines.length; j++) {
      const next = lines[j]
      if (next.trim() === '') continue
      const nextIndent = indentOf(next)
      const listItem = nextIndent === indent && /^-(\s|$)/.test(next.trimStart())
      if (nextIndent > indent || (keyWithoutValue && listItem)) end = j
      else break
    }
    if (end > start) folds.push({ start, end, label: plural(end - start, 'line') })
  })
  return folds
}

// Everything the viewer needs for one file.
export function parseStructured(kind: StructuredKind, source: string): Structured {
  if (kind === 'json') {
    const lines = tokenizeJson(formatJson(source))
    return { lines, folds: jsonFolds(lines) }
  }
  return { lines: tokenizeYaml(source), folds: yamlFolds(source.split('\n')) }
}
