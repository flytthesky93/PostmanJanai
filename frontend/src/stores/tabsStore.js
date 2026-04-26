import { reactive, computed, watch } from 'vue'

const STORAGE_KEY = 'pmj.tabs.v1'
const MAX_TABS = 20

/**
 * Snapshot of the RequestPanel reactive state. JSON-serializable.
 * Kept flat so that shallow merge + diff work well.
 * @typedef {Object} RequestSnapshot
 * @property {string} url
 * @property {string} method
 * @property {string} body
 * @property {string} bodyMode - none | raw | xml | form_urlencoded | multipart
 * @property {boolean} bodyRawEditor
 * @property {Array<{ key: string, value: string }>} queryParams
 * @property {Array<{ key: string, value: string, locked?: boolean }>} headers
 * @property {Array<{ key: string, value: string }>} formFields
 * @property {Array<{ key: string, kind: 'text'|'file', value: string, file_path: string }>} multipartParts
 * @property {string} activeTab - params | headers | auth | body
 * @property {string} authType - none | bearer | basic | apikey
 * @property {string} authBearerToken
 * @property {string} authUsername
 * @property {string} authPassword
 * @property {string} authApiKey
 * @property {string} authApiKeyName
 * @property {string} authApiKeyIn - header | query
 * @property {boolean} insecureSkipVerify
 * @property {string|null} savedRequestId
 * @property {string|null} savedFolderId
 * @property {string} savedRequestLabel
 */

/**
 * A single tab in the editor.
 * @typedef {Object} TabState
 * @property {string} id
 * @property {RequestSnapshot} snapshot - live state (what RequestPanel currently shows / would show)
 * @property {RequestSnapshot} baseline - state we compare against for dirty tracking
 * @property {Object|null} response - HTTPExecute result (not persisted)
 * @property {boolean} loading - transient (not persisted)
 */

let idCounter = 0
function newTabId() {
  idCounter += 1
  return `tab-${Date.now().toString(36)}-${idCounter}`
}

export function emptySnapshot() {
  return {
    url: '',
    method: 'GET',
    body: '',
    bodyMode: 'raw',
    bodyRawEditor: false,
    queryParams: [{ key: '', value: '' }],
    headers: [
      { key: 'Accept', value: 'application/json' },
      { key: '', value: '' }
    ],
    formFields: [{ key: '', value: '' }],
    multipartParts: [{ key: '', kind: 'text', value: '', file_path: '' }],
    activeTab: 'params',
    authType: 'none',
    authBearerToken: '',
    authUsername: '',
    authPassword: '',
    authApiKey: '',
    authApiKeyName: '',
    authApiKeyIn: 'header',
    insecureSkipVerify: false,
    savedRequestId: null,
    savedFolderId: null,
    savedRequestLabel: ''
  }
}

function clone(snap) {
  return snap ? JSON.parse(JSON.stringify(snap)) : emptySnapshot()
}

function canonicalForDiff(snap) {
  if (!snap) return ''
  // ignore bodyRawEditor + activeTab UI toggles — they don't affect outgoing request
  const copy = clone(snap)
  copy.bodyRawEditor = false
  copy.activeTab = 'params'
  return JSON.stringify(copy)
}

/**
 * Build a snapshot from a SavedRequestFull DTO.
 * @param {Object} dto
 * @returns {RequestSnapshot}
 */
export function snapshotFromSavedRequest(dto) {
  const s = emptySnapshot()
  if (!dto) return s
  s.url = String(dto.url || '')
  s.method = (dto.method || 'GET').toUpperCase()
  const bm = dto.body_mode || 'none'
  s.bodyMode = bm
  s.body = bm === 'raw' || bm === 'xml' ? (dto.raw_body != null ? String(dto.raw_body) : '') : ''

  if (dto.headers?.length) {
    s.headers = dto.headers.map((h) => ({ key: h.key || '', value: h.value || '' }))
    if (s.headers.length === 0) s.headers = [{ key: '', value: '' }]
  } else {
    s.headers = [{ key: '', value: '' }]
  }
  if (dto.query_params?.length) {
    s.queryParams = dto.query_params.map((p) => ({ key: p.key || '', value: p.value || '' }))
    if (s.queryParams.length === 0) s.queryParams = [{ key: '', value: '' }]
  }
  if (dto.form_fields?.length) {
    s.formFields = dto.form_fields.map((f) => ({ key: f.key || '', value: f.value || '' }))
    if (s.formFields.length === 0) s.formFields = [{ key: '', value: '' }]
  }
  if (dto.multipart_parts?.length) {
    s.multipartParts = dto.multipart_parts.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    }))
    if (s.multipartParts.length === 0) {
      s.multipartParts = [{ key: '', kind: 'text', value: '', file_path: '' }]
    }
  }

  const a = dto.auth || null
  if (a && typeof a === 'object') {
    s.authType = String(a.type || 'none').toLowerCase().trim() || 'none'
    s.authBearerToken = a.bearer_token != null ? String(a.bearer_token) : ''
    s.authUsername = a.username != null ? String(a.username) : ''
    s.authPassword = a.password != null ? String(a.password) : ''
    s.authApiKey = a.api_key != null ? String(a.api_key) : ''
    s.authApiKeyName = a.api_key_name != null ? String(a.api_key_name) : ''
    s.authApiKeyIn = String(a.api_key_in || 'header').toLowerCase() === 'query' ? 'query' : 'header'
  }

  s.insecureSkipVerify = !!dto.insecure_skip_verify
  s.savedRequestId = dto.id || null
  s.savedFolderId = dto.folder_id || null
  s.savedRequestLabel = dto.name || 'Request'
  s.activeTab = bm === 'none' || bm === '' ? 'params' : 'body'
  return s
}

/**
 * Build a snapshot from an import-cURL payload (always ad-hoc).
 * @param {Object} payload
 * @returns {RequestSnapshot}
 */
export function snapshotFromCurlPayload(payload) {
  const s = emptySnapshot()
  if (!payload) return s
  s.method = (payload.method || 'GET').toUpperCase()
  const rawUrl = String(payload.url || '').trim()
  try {
    const u = new URL(rawUrl)
    if (u.search) {
      s.url = `${u.origin}${u.pathname}`
      const pairs = []
      u.searchParams.forEach((v, k) => pairs.push({ key: k, value: v }))
      s.queryParams = pairs.length ? pairs : [{ key: '', value: '' }]
    } else {
      s.url = rawUrl
    }
  } catch {
    s.url = rawUrl
  }

  const bm = payload.body_mode || 'none'
  s.bodyMode = bm
  s.body = payload.body || ''

  if (payload.headers?.length) {
    s.headers = payload.headers.map((h) => ({ key: h.key || '', value: h.value || '' }))
  }
  if (s.headers.length === 0) s.headers = [{ key: '', value: '' }]
  if (payload.form_fields?.length) {
    s.formFields = payload.form_fields.map((f) => ({ key: f.key || '', value: f.value || '' }))
  }
  if (payload.multipart_parts?.length) {
    s.multipartParts = payload.multipart_parts.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    }))
  }

  s.activeTab = bm === 'none' || bm === '' ? 'params' : 'body'
  s.insecureSkipVerify = false
  s.savedRequestId = null
  s.savedFolderId = null
  s.savedRequestLabel = ''
  return s
}

function deriveTitle(snap) {
  if (!snap) return 'New Request'
  if (snap.savedRequestLabel) return snap.savedRequestLabel
  const raw = (snap.url || '').trim()
  if (!raw) return 'New Request'
  try {
    const u = new URL(/^https?:\/\//i.test(raw) ? raw : `https://${raw}`)
    const parts = u.pathname.split('/').filter(Boolean)
    const last = parts.length ? parts[parts.length - 1] : u.host
    return last.length > 32 ? last.slice(0, 30) + '…' : last
  } catch {
    return raw.length > 32 ? raw.slice(0, 30) + '…' : raw || 'New Request'
  }
}

function makeTab(snapshot) {
  const snap = snapshot ? clone(snapshot) : emptySnapshot()
  return {
    id: newTabId(),
    snapshot: snap,
    baseline: clone(snap),
    response: null,
    loading: false
  }
}

const state = reactive({
  tabs: [],
  activeTabId: null,
  hydrated: false
})

function findTab(id) {
  return state.tabs.find((t) => t.id === id) || null
}

function findSavedTab(savedRequestId) {
  if (!savedRequestId) return null
  return state.tabs.find((t) => t.snapshot?.savedRequestId === savedRequestId) || null
}

function persistDebounced() {
  if (!state.hydrated) return
  if (persistDebounced._timer) clearTimeout(persistDebounced._timer)
  persistDebounced._timer = setTimeout(persistNow, 200)
}

function persistNow() {
  try {
    const payload = {
      activeTabId: state.activeTabId,
      tabs: state.tabs.map((t) => ({
        id: t.id,
        snapshot: t.snapshot,
        baseline: t.baseline
      }))
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
  } catch {
    /* quota / unavailable — ignore */
  }
}

function ensureTabId() {
  if (!state.activeTabId && state.tabs.length > 0) {
    state.activeTabId = state.tabs[0].id
  }
}

function openBlank() {
  if (state.tabs.length >= MAX_TABS) {
    return null
  }
  const tab = makeTab(null)
  state.tabs.push(tab)
  state.activeTabId = tab.id
  persistDebounced()
  return tab
}

function openSavedRequest(dto) {
  if (!dto?.id) return null
  const existing = findSavedTab(String(dto.id))
  if (existing) {
    state.activeTabId = existing.id
    const fresh = snapshotFromSavedRequest(dto)
    existing.snapshot = fresh
    existing.baseline = clone(fresh)
    persistDebounced()
    return existing
  }
  if (state.tabs.length >= MAX_TABS) return null
  const snap = snapshotFromSavedRequest(dto)
  const tab = makeTab(snap)
  state.tabs.push(tab)
  state.activeTabId = tab.id
  persistDebounced()
  return tab
}

function openAdhocFromPayload(payload) {
  const snap = snapshotFromCurlPayload(payload)
  const active = findTab(state.activeTabId)
  // If active tab is a brand-new blank ad-hoc (no url, no savedRequestId, baseline==snapshot), reuse it.
  if (
    active &&
    !active.snapshot.savedRequestId &&
    !active.snapshot.url &&
    canonicalForDiff(active.snapshot) === canonicalForDiff(active.baseline)
  ) {
    active.snapshot = snap
    active.baseline = clone(snap)
    persistDebounced()
    return active
  }
  if (state.tabs.length >= MAX_TABS) return null
  const tab = makeTab(snap)
  state.tabs.push(tab)
  state.activeTabId = tab.id
  persistDebounced()
  return tab
}

function activateTab(id) {
  const t = findTab(id)
  if (!t) return
  state.activeTabId = t.id
  persistDebounced()
}

function closeTab(id) {
  const idx = state.tabs.findIndex((t) => t.id === id)
  if (idx < 0) return
  const wasActive = state.tabs[idx].id === state.activeTabId
  state.tabs.splice(idx, 1)
  if (state.tabs.length === 0) {
    state.activeTabId = null
  } else if (wasActive) {
    const next = state.tabs[Math.min(idx, state.tabs.length - 1)]
    state.activeTabId = next.id
  }
  persistDebounced()
}

/** Update the active tab's snapshot (called by RequestPanel after state mutations). */
function updateActiveSnapshot(snapshot) {
  const t = findTab(state.activeTabId)
  if (!t || !snapshot) return
  t.snapshot = clone(snapshot)
  persistDebounced()
}

/** Mark the active tab's current snapshot as baseline (after successful save). */
function markActiveBaseline() {
  const t = findTab(state.activeTabId)
  if (!t) return
  t.baseline = clone(t.snapshot)
  persistDebounced()
}

/** After an ad-hoc tab is saved into a folder, upgrade the snapshot with the new savedRequestId. */
function promoteActiveToSaved(dto) {
  const t = findTab(state.activeTabId)
  if (!t || !dto?.id) return
  t.snapshot = snapshotFromSavedRequest(dto)
  t.baseline = clone(t.snapshot)
  persistDebounced()
}

function setActiveResponse(response) {
  const t = findTab(state.activeTabId)
  if (!t) return
  t.response = response
  // response is not persisted
}

function setActiveLoading(loading) {
  const t = findTab(state.activeTabId)
  if (!t) return
  t.loading = !!loading
}

function setTabResponse(id, response) {
  const t = findTab(id)
  if (!t) return
  t.response = response
}

function setTabLoading(id, loading) {
  const t = findTab(id)
  if (!t) return
  t.loading = !!loading
}

function restore() {
  if (state.hydrated) return
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed?.tabs)) {
        state.tabs = parsed.tabs
          .filter((t) => t && t.id && t.snapshot)
          .slice(0, MAX_TABS)
          .map((t) => ({
            id: t.id,
            snapshot: { ...emptySnapshot(), ...t.snapshot },
            baseline: { ...emptySnapshot(), ...(t.baseline || t.snapshot) },
            response: null,
            loading: false
          }))
        state.activeTabId = parsed.activeTabId || null
        ensureTabId()
      }
    }
  } catch {
    state.tabs = []
    state.activeTabId = null
  }
  state.hydrated = true
}

const activeTab = computed(() => findTab(state.activeTabId))

const tabsMeta = computed(() =>
  state.tabs.map((t) => ({
    id: t.id,
    title: deriveTitle(t.snapshot),
    method: t.snapshot?.method || 'GET',
    kind: t.snapshot?.savedRequestId ? 'saved' : 'adhoc',
    insecureTLS: !!t.snapshot?.insecureSkipVerify,
    dirty: canonicalForDiff(t.snapshot) !== canonicalForDiff(t.baseline),
    savedRequestId: t.snapshot?.savedRequestId || null
  }))
)

// Defensive: re-persist when tabs array mutates structurally (add/remove/reorder).
watch(
  () => state.tabs.length,
  () => persistDebounced()
)

export function useTabsStore() {
  return {
    state,
    activeTab,
    tabsMeta,
    restore,
    openBlank,
    openSavedRequest,
    openAdhocFromPayload,
    activateTab,
    closeTab,
    updateActiveSnapshot,
    markActiveBaseline,
    promoteActiveToSaved,
    setActiveResponse,
    setActiveLoading,
    setTabResponse,
    setTabLoading
  }
}
