import { describe, expect, it } from 'vitest'
import {
  formatJson,
  jsonFolds,
  parseStructured,
  structuredKindFor,
  tokenizeJson,
  tokenizeYaml,
  yamlFolds,
  type Token,
} from './structured'

// The visible (non-whitespace) tokens of a line as [kind, text] pairs.
const visible = (line: Token[]) => line.filter((t) => t.text.trim() !== '').map((t) => [t.kind, t.text])
const textOf = (lines: Token[][]) => lines.map((l) => l.map((t) => t.text).join('')).join('\n')

describe('structuredKindFor', () => {
  it.each([
    ['/n/a.json', 'json'],
    ['/n/a.JSONC', 'json'],
    ['/n/a.yaml', 'yaml'],
    ['/n/a.yml', 'yaml'],
    ['/n/a.toml', null],
    ['/n/json', null],
  ])('%s → %s', (path, kind) => {
    expect(structuredKindFor(path)).toBe(kind)
  })
})

describe('formatJson', () => {
  it('pretty-prints a single-line file that parses', () => {
    expect(formatJson('{"a":1,"b":[true,null]}\n')).toBe('{\n  "a": 1,\n  "b": [\n    true,\n    null\n  ]\n}')
  })

  it('keeps formatted, commented and invalid files as written', () => {
    for (const source of ['{\n"a":1}', '{"a":1} // note', '{"a":', '']) {
      expect(formatJson(source)).toBe(source)
    }
  })
})

describe('tokenizeJson', () => {
  it('keeps every character, line by line', () => {
    const source = '{\n  "a": "x", // c\n  "b": [1, -2.5e3]\n}'
    expect(textOf(tokenizeJson(source))).toBe(source)
  })

  it('tells keys from string values and colours numbers, literals and punctuation', () => {
    const [line] = tokenizeJson('{"name": "Spotfile", "port": 8080, "on": true, "x": null}')
    expect(visible(line)).toEqual([
      ['punct', '{'],
      ['key', '"name"'],
      ['punct', ':'],
      ['string', '"Spotfile"'],
      ['punct', ','],
      ['key', '"port"'],
      ['punct', ':'],
      ['number', '8080'],
      ['punct', ','],
      ['key', '"on"'],
      ['punct', ':'],
      ['literal', 'true'],
      ['punct', ','],
      ['key', '"x"'],
      ['punct', ':'],
      ['literal', 'null'],
      ['punct', '}'],
    ])
  })

  it('handles escaped quotes and comments, including multi-line ones', () => {
    const lines = tokenizeJson('{"q": "say \\"hi\\"", // trailing\n/* one\ntwo */ "k": 1}')
    expect(visible(lines[0])).toContainEqual(['string', '"say \\"hi\\""'])
    expect(visible(lines[0])).toContainEqual(['comment', '// trailing'])
    expect(visible(lines[1])).toEqual([['comment', '/* one']])
    expect(visible(lines[2])[0]).toEqual(['comment', 'two */'])
    expect(visible(lines[2])).toContainEqual(['key', '"k"'])
  })
})

describe('jsonFolds', () => {
  it('folds multi-line objects and arrays with counts', () => {
    const source = '{\n  "a": {\n    "x": 1,\n    "y": 2\n  },\n  "list": [\n    1,\n    2,\n    3\n  ],\n  "empty": {},\n  "one": [\n    {}\n  ]\n}'
    const folds = jsonFolds(tokenizeJson(source))
    expect(folds).toEqual([
      { start: 0, end: 14, label: '4 keys' },
      { start: 1, end: 4, label: '2 keys' },
      { start: 5, end: 9, label: '3 items' },
      { start: 11, end: 13, label: '1 item' },
    ])
  })

  it('keeps the outer fold when two brackets open on one line', () => {
    const folds = jsonFolds(tokenizeJson('[{\n  "a": 1\n}]'))
    expect(folds).toEqual([{ start: 0, end: 2, label: '1 item' }])
  })

  it('ignores brackets inside strings and comments', () => {
    const folds = jsonFolds(tokenizeJson('{\n  "a": "{[", // ]}\n  "b": 1\n}'))
    expect(folds).toEqual([{ start: 0, end: 3, label: '2 keys' }])
  })
})

describe('tokenizeYaml', () => {
  it('keeps every character, line by line', () => {
    const source = 'server:\n  host: "a#b" # note\n  ports:\n    - 80\n'
    expect(textOf(tokenizeYaml(source))).toBe(source)
  })

  it('colours keys, scalars, comments and list dashes', () => {
    const lines = tokenizeYaml('name: Spotfile\nport: 8080\nenabled: yes\nnothing: ~\nquoted: "a # b" # real comment\n- item\n# whole line')
    expect(visible(lines[0])).toEqual([['key', 'name'], ['punct', ':'], ['string', 'Spotfile']])
    expect(visible(lines[1])).toEqual([['key', 'port'], ['punct', ':'], ['number', '8080']])
    expect(visible(lines[2])).toEqual([['key', 'enabled'], ['punct', ':'], ['literal', 'yes']])
    expect(visible(lines[3])).toEqual([['key', 'nothing'], ['punct', ':'], ['literal', '~']])
    expect(visible(lines[4])).toEqual([['key', 'quoted'], ['punct', ':'], ['string', '"a # b"'], ['comment', '# real comment']])
    expect(visible(lines[5])).toEqual([['punct', '-'], ['string', 'item']])
    expect(visible(lines[6])).toEqual([['comment', '# whole line']])
  })

  it('treats a block scalar body as a string', () => {
    const lines = tokenizeYaml('script: |\n  echo "a: b" # not a comment\n  done\nnext: 1')
    expect(visible(lines[0])).toEqual([['key', 'script'], ['punct', ':'], ['punct', '|']])
    expect(visible(lines[1])).toEqual([['string', 'echo "a: b" # not a comment']])
    expect(visible(lines[3])[0]).toEqual(['key', 'next'])
  })

  it('reads list items that start with a key', () => {
    const [line] = tokenizeYaml('  - name: web')
    expect(visible(line)).toEqual([['punct', '-'], ['key', 'name'], ['punct', ':'], ['string', 'web']])
  })
})

describe('yamlFolds', () => {
  it('folds indented blocks, lists under a key at the same indent, and skips blank tails', () => {
    const source = [
      'server:', //       0
      '  host: a', //     1
      '', //              2
      '  ports:', //      3
      '  - 80', //        4
      '  - 443', //       5
      '', //              6
      'items:', //        7
      '- name: web', //   8
      '  port: 1', //     9
      '- name: db', //    10
      '# end', //         11
    ].join('\n')
    expect(yamlFolds(source.split('\n'))).toEqual([
      { start: 0, end: 5, label: '5 lines' },
      { start: 3, end: 5, label: '2 lines' },
      { start: 7, end: 10, label: '3 lines' },
      { start: 8, end: 9, label: '1 line' },
    ])
  })

  it('never folds from a comment', () => {
    expect(yamlFolds(['# heading', '  indented: 1'])).toEqual([])
  })
})

describe('parseStructured', () => {
  it('formats, tokenizes and folds JSON', () => {
    const parsed = parseStructured('json', '{"a":[1,2]}')
    expect(parsed.lines).toHaveLength(6)
    expect(parsed.folds[0]).toEqual({ start: 0, end: 5, label: '1 key' })
  })

  it('tokenizes and folds YAML as written', () => {
    const parsed = parseStructured('yaml', 'a:\n  b: 1\n')
    expect(parsed.lines).toHaveLength(3)
    expect(parsed.folds).toEqual([{ start: 0, end: 1, label: '1 line' }])
  })
})
