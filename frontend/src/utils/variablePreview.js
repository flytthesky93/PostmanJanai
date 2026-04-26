const VAR_PATTERN = /\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}/g

export function hasVariables(text) {
  VAR_PATTERN.lastIndex = 0
  return VAR_PATTERN.test(String(text || ''))
}

export function buildVariableMap(rows = []) {
  const out = {}
  for (const row of Array.isArray(rows) ? rows : []) {
    const key = String(row?.key || '').trim()
    if (!key || row?.enabled === false) continue
    out[key] = {
      value: row?.value == null ? '' : String(row.value),
      secret: row?.kind === 'secret'
    }
  }
  return out
}

export function previewVariables(text, rows = []) {
  const input = String(text || '')
  const map = buildVariableMap(rows)
  let unresolved = 0
  let secretCount = 0
  VAR_PATTERN.lastIndex = 0
  const output = input.replace(VAR_PATTERN, (token, key) => {
    const row = map[String(key || '').trim()]
    if (!row) {
      unresolved += 1
      return token
    }
    if (row.secret) {
      secretCount += 1
      return '***'
    }
    return row.value
  })
  return {
    output,
    unresolved,
    secretCount,
    changed: output !== input
  }
}
