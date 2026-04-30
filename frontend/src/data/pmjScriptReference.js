/**
 * PMJ scripting — Help tab copy + CodeMirror completions (mirror internal/service/pmj_runtime.go).
 * Canonical API: pmj · alias: pm
 */

import { snippetCompletion } from '@codemirror/autocomplete'

/** Rows for Help → Scripting tab (Vietnamese). */
export const PMJ_SCRIPT_HELP_ROWS = [
  {
    component: 'pmj / pm',
    example: 'pmj · pm',
    pre: true,
    post: true,
    description:
      'Namespace API chính. `pm` là alias của `pmj`. Postman collections dùng `pm.*`; export/import có thể chuyển đổi.'
  },
  {
    component: 'console',
    example: 'console.log("msg")',
    pre: true,
    post: true,
    description: 'Đưa text vào Response → Results → Script console (log · info · warn · error · debug).'
  },
  {
    component: 'environment',
    example: "pmj.environment.get('K') · .set · .unset",
    pre: true,
    post: true,
    description: 'Biến môi trường đang active. `set` ghi DB + cache phiên; `unset` xóa trong env active.'
  },
  {
    component: 'variables',
    example: "pmj.variables.get('K') · .set · .unset",
    pre: true,
    post: true,
    description: 'Biến session (runner / một lần Send): chỉ trong phiên, không persist ra disk.'
  },
  {
    component: 'request',
    example: 'pmj.request.method · url · headers · body.raw',
    pre: true,
    post: false,
    description: 'Pre-request: chỉnh request sẽ gửi (`headers.get` / `headers.add`; body là `body.raw`).'
  },
  {
    component: 'response',
    example: 'pmj.response.url · .code · headers · text() · json()',
    pre: false,
    post: true,
    description:
      'Post-response: `url` = URL gửi thật (sau ghép query — `final_url` backend), `code`, `responseTime` (ms), `headers.get(h)`, `text()`, `json()` (parse lỗi → throw). Giá trị `url` trong body JSON dùng `pmj.response.json().url`.'
  },
  {
    component: 'test / expect',
    example: 'pmj.test("tên", () => { pmj.expect(x).to.equal(1) })',
    pre: false,
    post: true,
    description:
      'Subset Chai nhỏ: `.equal`, `.eql`, `.include`, `.exist`, `.to.have.status(n)`, `.be.ok` / `true` / `false`, v.v.'
  },
  {
    component: 'sendRequest',
    example: 'pmj.sendRequest({ url, method, body, headers, query }, (err, res) => {})',
    pre: true,
    post: true,
    description:
      'HTTP lồng trong script — chỉ kiểu callback (không có Promise). `res.code`, `.text()`, `.json()`, `.headers`; xem luôn tham số `err`.'
  },
  {
    component: '{{VAR}} trong chuỗi',
    example: "'Bearer {{TOKEN}}'",
    pre: true,
    post: false,
    description:
      'Vẫn highlight `{{VAR}}`; thay sau khi pre-request trong pipeline như các field khác của request.'
  }
]

function splitSuffix(suffix) {
  if (!suffix) return { segments: [], partial: '' }
  const trailDot = suffix.endsWith('.')
  const core = trailDot ? suffix.slice(0, -1) : suffix
  const parts = core.split('.').filter((p) => p !== '')
  if (trailDot) {
    return { segments: parts, partial: '' }
  }
  if (parts.length === 0) return { segments: [], partial: '' }
  const partial = parts[parts.length - 1]
  const segments = parts.slice(0, -1)
  return { segments, partial }
}

function o(label, extra = {}) {
  return {
    label,
    type: extra.type || 'text',
    detail: extra.info,
    info: extra.info,
    apply: extra.apply !== undefined ? extra.apply : label
  }
}

function filterOpts(list, partial) {
  if (!partial) return list
  const pl = String(partial).toLowerCase()
  return list.filter((x) => {
    const lbl = typeof x.label === 'string' ? x.label.toLowerCase() : ''
    return lbl.startsWith(pl) || completionLabelStem(x).startsWith(pl)
  })
}

/**
 * @param {'pre' | 'post'} phase
 * @param {string[]} segments
 */
function listForSegments(phase, segments) {
  const key = segments.join('.')

  if (key === '') {
    if (phase === 'pre') {
      return [
        o('request', { type: 'namespace', info: '.method · .url · .headers · .body.raw' }),
        o('environment', { type: 'namespace', info: '.get · .set · .unset — active env' }),
        o('variables', { type: 'namespace', info: '.get · .set · .unset — session memory' }),
        o('sendRequest', {
          type: 'function',
          apply: 'sendRequest({ url: "", method: "GET" }, (err, res) => {\n  \n})',
          info: 'Nested HTTP (callback)'
        })
      ]
    }
    return [
      o('response', { type: 'namespace', info: '.url · .code · .headers · text() · json()' }),
      snippetCompletion("test('${1:name}', () => {\n  ${}\n})", {
        label: 'test',
        detail: 'Named test block',
        type: 'function'
      }),
      snippetCompletion('expect(${1:actual})', {
        label: 'expect',
        detail: 'Start assertion',
        type: 'function'
      }),
      o('environment', { type: 'namespace', info: '.get · .set · .unset — active env' }),
      o('variables', { type: 'namespace', info: '.get · .set · .unset — session memory' }),
      o('sendRequest', {
        type: 'function',
        apply: 'sendRequest({ url: "", method: "GET" }, (err, res) => {\n  \n})',
        info: 'Nested HTTP (callback)'
      })
    ]
  }

  if (key === 'environment' || key === 'variables') {
    return [
      o('get', { type: 'method', apply: 'get("")', info: '(key) → string' }),
      o('set', { type: 'method', apply: 'set("", "")', info: '(key, value)' }),
      o('unset', { type: 'method', apply: 'unset("")', info: '(key)' })
    ]
  }

  if (phase === 'pre' && key === 'request') {
    return [
      o('method', { type: 'property', info: 'HTTP method string' }),
      o('url', { type: 'property', info: 'URL string' }),
      o('headers', { type: 'namespace', info: '.get · .add' }),
      o('body', { type: 'namespace', info: '.raw' })
    ]
  }

  if (phase === 'pre' && key === 'request.headers') {
    return [
      o('get', { type: 'method', apply: 'get("")', info: '(header name) → value' }),
      o('add', { type: 'method', apply: 'add("", "")', info: '(name, value)' })
    ]
  }

  if (phase === 'pre' && key === 'request.body') {
    return [o('raw', { type: 'property', info: 'Raw body text' })]
  }

  if (phase === 'post' && key === 'response') {
    return [
      o('url', { type: 'property', info: 'Resolved request URL (same as final_url after Send)' }),
      o('code', { type: 'property', info: 'Status code number' }),
      o('responseTime', { type: 'property', info: 'Duration ms' }),
      o('headers', { type: 'namespace', info: '.get(name)' }),
      snippetCompletion('text()', { label: 'text', detail: 'Body as string', type: 'method' }),
      snippetCompletion('json()', { label: 'json', detail: 'Parse JSON body', type: 'method' })
    ]
  }

  if (phase === 'post' && key === 'response.headers') {
    return [o('get', { type: 'method', apply: 'get("")', info: '(header name)' })]
  }

  return []
}

const GLOBAL_ITEMS = [
  snippetCompletion('console.log(${msg})', { label: 'console.log', detail: '→ Script console', type: 'function' }),
  snippetCompletion('console.info(${msg})', { label: 'console.info', type: 'function' }),
  snippetCompletion('console.warn(${msg})', { label: 'console.warn', type: 'function' }),
  snippetCompletion('console.error(${msg})', { label: 'console.error', type: 'function' }),
  snippetCompletion('JSON.stringify(${value})', { label: 'JSON.stringify', type: 'function' }),
  snippetCompletion('JSON.parse(${text})', { label: 'JSON.parse', type: 'function' }),
  snippetCompletion('pmj.', { label: 'pmj.', detail: 'API pmj', type: 'keyword' }),
  snippetCompletion('pm.', { label: 'pm.', detail: 'Alias pm', type: 'keyword' })
]

function completionLabelStem(x) {
  const lbl = x && typeof x.label === 'string' ? x.label : ''
  let s = lbl.replace(/\.$/, '').trim().toLowerCase()
  const paren = s.indexOf('(')
  if (paren !== -1) {
    s = s.slice(0, paren).trim()
  }
  return s
}

/**
 * @param {'pre' | 'post'} phase
 */
export function createPmjCompletionSource(phase) {
  return (context) => {
    const line = context.state.doc.lineAt(context.pos)
    const before = line.text.slice(0, context.pos - line.from)

    const pmMatch = before.match(/\b(pmj|pm)\.([\s\S]*)$/i)
    if (pmMatch) {
      const suffix = (pmMatch[2] || '').replace(/\s/g, '')
      const { segments, partial } = splitSuffix(suffix)
      const all = listForSegments(phase, segments)
      const opts = filterOpts(all, partial)
      if (opts.length === 0 && !context.explicit) return null
      return {
        from: context.pos - partial.length,
        to: context.pos,
        filter: false,
        options: opts.length ? opts : all
      }
    }

    const w = context.matchBefore(/\w+$/)
    const hasWord = !!(w && w.from < w.to)
    const word = hasWord ? context.state.doc.sliceString(w.from, context.pos) : ''

    if (!context.explicit && !hasWord) return null

    if (context.explicit && !hasWord) {
      return {
        from: context.pos,
        to: context.pos,
        filter: false,
        options: GLOBAL_ITEMS.slice()
      }
    }

    const opts = GLOBAL_ITEMS.filter((item) =>
      completionLabelStem(item).startsWith(word.toLowerCase())
    )
    if (!opts.length && !context.explicit) return null

    return {
      from: w.from,
      to: context.pos,
      filter: false,
      options: opts.length ? opts : GLOBAL_ITEMS.slice()
    }
  }
}
