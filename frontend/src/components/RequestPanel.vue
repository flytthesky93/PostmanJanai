<script setup>
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import formatXml from 'xml-formatter'
import JsonCodeMirror from './JsonCodeMirror.vue'
import EnvVarMirrorField from './EnvVarMirrorField.vue'
import SnippetPanel from './SnippetPanel.vue'
import { PickFileForBody } from '../../wailsjs/wailsjs/go/delivery/HTTPHandler'
import * as SavedRequestAPI from '../../wailsjs/wailsjs/go/delivery/SavedRequestHandler'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'

const props = defineProps({
  /** Selected root folder UUID for history (optional). */
  activeRootFolderId: { type: String, default: null },
  /** Keys from active environment (enabled) — {{var}} UI + CodeMirror highlights. */
  declaredEnvKeys: { type: Array, default: () => [] },
  /** key → current value in active env (hover popover). */
  activeEnvValues: { type: Object, default: () => ({}) }
})

const emit = defineEmits([
  'send',
  'console',
  'saved-request',
  'patch-active-env-value',
  'snapshot-change',
  'baseline-committed',
  'promote-to-saved'
])

const url = ref('')
const method = ref('GET')
const body = ref('')

/** none | raw | xml | form_urlencoded | multipart */
const bodyMode = ref('raw')

/** When true, body is edited with EnvVarMirrorField (e.g. after double-click {{var}} in CodeMirror). */
const bodyRawEditor = ref(false)

/** Ref to JsonCodeMirror — read live doc on Send so {{env}} in body is not lost if v-model lags. */
const bodyCodeMirrorRef = ref(null)

function liveRawOrXmlBodyText() {
  const bm = bodyMode.value
  if (bm !== 'raw' && bm !== 'xml') return ''
  if (bodyRawEditor.value) return body.value ?? ''
  const fromCm = bodyCodeMirrorRef.value?.getDocText?.()
  if (typeof fromCm === 'string') return fromCm
  return body.value ?? ''
}

watch(bodyMode, () => {
  bodyRawEditor.value = false
})

const envKeysForFields = computed(() =>
  Array.isArray(props.declaredEnvKeys) ? props.declaredEnvKeys : []
)

const envValuesForFields = computed(() =>
  props.activeEnvValues && typeof props.activeEnvValues === 'object' ? props.activeEnvValues : {}
)

function forwardPatchEnvValue(p) {
  emit('patch-active-env-value', p)
}

/** @type {import('vue').Ref<Array<{ key: string, value: string }>>} */
const queryParams = ref([{ key: '', value: '' }])
/** @type {import('vue').Ref<Array<{ key: string, value: string, locked?: boolean }>>} */
const headers = ref([
  { key: 'Accept', value: 'application/json' },
  { key: '', value: '' }
])

function stripContentTypeHeaders(rows) {
  return rows.filter((h) => !/^content-type$/i.test(String(h.key || '').trim()))
}

function formUrlencodedLockedHeader() {
  return { key: 'Content-Type', value: 'application/x-www-form-urlencoded', locked: true }
}

watch(bodyMode, (mode, prev) => {
  if (mode === 'form_urlencoded') {
    headers.value = [formUrlencodedLockedHeader(), ...stripContentTypeHeaders(headers.value)]
  } else if (prev === 'form_urlencoded') {
    headers.value = headers.value.filter((h) => !h.locked)
  }
})

/** form-urlencoded */
const formFields = ref([{ key: '', value: '' }])

/** multipart: text | file */
const multipartParts = ref([{ key: '', kind: 'text', value: '', file_path: '' }])

const activeTab = ref('params')

/** none | bearer | basic | apikey — merged after env substitution on Send. */
const authType = ref('none')
const authBearerToken = ref('')
const authUsername = ref('')
const authPassword = ref('')
const authApiKey = ref('')
const authApiKeyName = ref('')
const authApiKeyIn = ref('header')
/** Dev/corp only: disable TLS certificate verification for this request (persisted on saved requests). */
const insecureSkipVerify = ref(false)

function buildAuthPayload() {
  const t = (authType.value || 'none').toLowerCase().trim()
  if (!t || t === 'none') return undefined
  return {
    type: t,
    bearer_token: authBearerToken.value ?? '',
    username: authUsername.value ?? '',
    password: authPassword.value ?? '',
    api_key: authApiKey.value ?? '',
    api_key_name: authApiKeyName.value ?? '',
    api_key_in: authApiKeyIn.value === 'query' ? 'query' : 'header'
  }
}

function syncAuthFromPayload(a) {
  if (!a || typeof a !== 'object') {
    authType.value = 'none'
    authBearerToken.value = ''
    authUsername.value = ''
    authPassword.value = ''
    authApiKey.value = ''
    authApiKeyName.value = ''
    authApiKeyIn.value = 'header'
    return
  }
  authType.value = String(a.type || 'none').toLowerCase().trim() || 'none'
  authBearerToken.value = a.bearer_token != null ? String(a.bearer_token) : ''
  authUsername.value = a.username != null ? String(a.username) : ''
  authPassword.value = a.password != null ? String(a.password) : ''
  authApiKey.value = a.api_key != null ? String(a.api_key) : ''
  authApiKeyName.value = a.api_key_name != null ? String(a.api_key_name) : ''
  authApiKeyIn.value = String(a.api_key_in || 'header').toLowerCase() === 'query' ? 'query' : 'header'
}

/** When set, Send includes request_id; Save updates DB. */
const savedRequestId = ref(null)
/** Folder that owns the saved request */
const savedFolderId = ref(null)
const savedRequestLabel = ref('')

/** Áp payload từ Import cURL (sidebar): luôn ad-hoc, không gắn saved request. */
function applyImportPayload(payload) {
  if (!payload) return
  applyInputToForm(payload)
}

function applyInputToForm(payload) {
  if (!payload) return
  insecureSkipVerify.value = false
  savedRequestId.value = null
  savedFolderId.value = null
  savedRequestLabel.value = ''
  method.value = (payload.method || 'GET').toUpperCase()
  const rawUrl = String(payload.url || '').trim()
  try {
    const u = new URL(rawUrl)
    if (u.search) {
      url.value = `${u.origin}${u.pathname}`
      const pairs = []
      u.searchParams.forEach((v, k) => pairs.push({ key: k, value: v }))
      queryParams.value = pairs.length ? pairs : [{ key: '', value: '' }]
    } else {
      url.value = rawUrl
      queryParams.value = [{ key: '', value: '' }]
    }
  } catch {
    url.value = rawUrl
    queryParams.value = [{ key: '', value: '' }]
  }

  const bm = payload.body_mode || 'none'
  body.value = payload.body || ''

  if (payload.headers?.length) {
    headers.value = payload.headers.map((h) => ({ key: h.key || '', value: h.value || '' }))
    if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
  } else {
    headers.value = [{ key: '', value: '' }]
  }
  if (bm === 'form_urlencoded') {
    headers.value = [formUrlencodedLockedHeader(), ...stripContentTypeHeaders(headers.value)]
  }

  bodyMode.value = bm

  if (payload.form_fields?.length) {
    formFields.value = payload.form_fields.map((f) => ({ key: f.key || '', value: f.value || '' }))
    if (formFields.value.length === 0) formFields.value.push({ key: '', value: '' })
  } else {
    formFields.value = [{ key: '', value: '' }]
  }

  if (payload.multipart_parts?.length) {
    multipartParts.value = payload.multipart_parts.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    }))
    if (multipartParts.value.length === 0) {
      multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
    }
  } else {
    multipartParts.value = [{ key: '', kind: 'text', value: '', file_path: '' }]
  }

  syncAuthFromPayload(payload.auth)

  activeTab.value = bm === 'none' || bm === '' ? 'params' : 'body'
}

const saveAdhocModalOpen = ref(false)
const saveAdhocName = ref('')
const saveAdhocFolderId = ref('')
/** @type {import('vue').Ref<Array<{ id: string, label: string }>>} */
const saveAdhocFolderOptions = ref([])
const saveAdhocFoldersLoading = ref(false)
const saveAdhocSubmitting = ref(false)

function suggestRequestName() {
  const raw = (url.value || '').trim()
  if (!raw) return 'New request'
  try {
    const u = new URL(/^https?:\/\//i.test(raw) ? raw : `https://${raw}`)
    const parts = u.pathname.split('/').filter(Boolean)
    const last = parts.length ? parts[parts.length - 1] : ''
    if (last) return last.length > 48 ? last.slice(0, 45) + '…' : last
    return u.host || 'New request'
  } catch {
    return 'New request'
  }
}

async function loadSaveAdhocFolderOptions() {
  saveAdhocFoldersLoading.value = true
  saveAdhocFolderOptions.value = []
  try {
    const roots = await FolderAPI.ListRootFolders()
    const list = Array.isArray(roots) ? roots : []
    const out = []
    async function walk(folderId, pathLabel) {
      const children = await FolderAPI.ListChildFolders(folderId)
      const ch = Array.isArray(children) ? children : []
      for (const c of ch) {
        const label = `${pathLabel} / ${c.name}`
        out.push({ id: c.id, label })
        await walk(c.id, label)
      }
    }
    for (const r of list) {
      out.push({ id: r.id, label: r.name })
      await walk(r.id, r.name)
    }
    saveAdhocFolderOptions.value = out
  } catch (e) {
    emit('console', `[Save] Could not load folders: ${e?.message || String(e)}`)
  } finally {
    saveAdhocFoldersLoading.value = false
  }
}

async function openSaveAdhocModal() {
  await loadSaveAdhocFolderOptions()
  if (saveAdhocFolderOptions.value.length === 0) {
    emit('console', '[Save] Create a folder in the sidebar before saving a request.')
    return
  }
  saveAdhocName.value = suggestRequestName()
  const preferred = props.activeRootFolderId && saveAdhocFolderOptions.value.some((o) => o.id === props.activeRootFolderId)
  saveAdhocFolderId.value = preferred ? props.activeRootFolderId : saveAdhocFolderOptions.value[0].id
  saveAdhocModalOpen.value = true
}

function closeSaveAdhocModal() {
  saveAdhocModalOpen.value = false
}

function buildNewSavedRequestDto() {
  const q = queryParams.value.filter((p) => (p.key || '').trim() !== '')
  const h = headers.value.filter((p) => (p.key || '').trim() !== '')
  const dto = {
    folder_id: (saveAdhocFolderId.value || '').trim(),
    name: (saveAdhocName.value || '').trim() || 'Untitled',
    method: method.value,
    url: url.value,
    body_mode: bodyMode.value,
    headers: h,
    query_params: q,
    form_fields:
      bodyMode.value === 'form_urlencoded'
        ? formFields.value.filter((p) => (p.key || '').trim() !== '')
        : [],
    multipart_parts:
      bodyMode.value === 'multipart'
        ? multipartParts.value
            .filter((p) => (p.key || '').trim() !== '')
            .map((p) => ({
              key: p.key.trim(),
              kind: p.kind === 'file' ? 'file' : 'text',
              value: p.kind === 'text' ? p.value : '',
              file_path: p.kind === 'file' ? (p.file_path || '').trim() : ''
            }))
        : []
  }
  if (bodyMode.value === 'raw' || bodyMode.value === 'xml') {
    dto.raw_body = liveRawOrXmlBodyText()
  }
  const ap = buildAuthPayload()
  if (ap) dto.auth = ap
  return dto
}

async function submitSaveAdhoc() {
  const fid = (saveAdhocFolderId.value || '').trim()
  const name = (saveAdhocName.value || '').trim()
  if (!fid) {
    emit('console', '[Save] Choose a folder.')
    return
  }
  if (!name) {
    emit('console', '[Save] Enter a request name.')
    return
  }
  saveAdhocSubmitting.value = true
  try {
    const created = await SavedRequestAPI.Create(buildNewSavedRequestDto())
    closeSaveAdhocModal()
    if (created) {
      loadFromSavedRequest(created)
      emit('promote-to-saved', created)
    }
    emit('console', `[Saved] "${name}" added to library.`)
    emit('saved-request')
  } catch (e) {
    const msg = e?.message || String(e)
    if (msg.includes('REQ_') || msg.toLowerCase().includes('already') || msg.toLowerCase().includes('exists')) {
      emit('console', '[Save] A request with that name already exists in this folder. Choose another name.')
    } else {
      emit('console', `[Save] ${msg}`)
    }
  } finally {
    saveAdhocSubmitting.value = false
  }
}

const addQueryRow = () => queryParams.value.push({ key: '', value: '' })
const removeQueryRow = (i) => {
  queryParams.value.splice(i, 1)
  if (queryParams.value.length === 0) queryParams.value.push({ key: '', value: '' })
}

const addHeaderRow = () => headers.value.push({ key: '', value: '' })
const removeHeaderRow = (i) => {
  if (headers.value[i]?.locked) return
  headers.value.splice(i, 1)
  if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
}

const addFormFieldRow = () => formFields.value.push({ key: '', value: '' })
const removeFormFieldRow = (i) => {
  formFields.value.splice(i, 1)
  if (formFields.value.length === 0) formFields.value.push({ key: '', value: '' })
}

const addMultipartField = () => {
  multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
}
const removeMultipartRow = (i) => {
  multipartParts.value.splice(i, 1)
  if (multipartParts.value.length === 0) {
    multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
  }
}

const pickMultipartFile = async (i) => {
  try {
    const path = await PickFileForBody()
    if (path) {
      multipartParts.value[i].file_path = path
      multipartParts.value[i].kind = 'file'
    }
  } catch (e) {
    emit('console', `[File] ${e?.message || String(e)}`)
  }
}

function validateUrlBeforeSend() {
  const raw = (url.value || '').trim()
  if (!raw) {
    emit('console', '[Request] URL is required before sending.')
    return false
  }
  const toParse = /^https?:\/\//i.test(raw) ? raw : `https://${raw}`
  try {
    const u = new URL(toParse)
    if (!u.hostname) {
      emit('console', '[Request] URL must include a host.')
      return false
    }
  } catch {
    emit('console', '[Request] Invalid URL. Check the address and try again.')
    return false
  }
  return true
}

/** Same shape as Wails `HTTPExecuteInput` — used by Send and snippet generator. */
function buildHttpExecutePayload() {
  const q = queryParams.value.filter((p) => (p.key || '').trim() !== '')
  const h = headers.value.filter((p) => (p.key || '').trim() !== '')

  const payload = {
    method: method.value,
    url: url.value,
    insecure_skip_verify: !!insecureSkipVerify.value,
    query_params: q,
    headers: h,
    body_mode: bodyMode.value,
    body: bodyMode.value === 'raw' || bodyMode.value === 'xml' ? liveRawOrXmlBodyText() : '',
    form_fields:
      bodyMode.value === 'form_urlencoded'
        ? formFields.value.filter((p) => (p.key || '').trim() !== '')
        : undefined,
    multipart_parts:
      bodyMode.value === 'multipart'
        ? multipartParts.value
            .filter((p) => (p.key || '').trim() !== '')
            .map((p) => ({
              key: p.key.trim(),
              kind: p.kind === 'file' ? 'file' : 'text',
              value: p.kind === 'text' ? p.value : '',
              file_path: p.kind === 'file' ? (p.file_path || '').trim() : ''
            }))
        : undefined
  }
  const root = props.activeRootFolderId
  if (typeof root === 'string' && root.trim() !== '') {
    payload.root_folder_id = root.trim()
  }
  if (savedRequestId.value) {
    payload.request_id = savedRequestId.value
  }
  const ap = buildAuthPayload()
  if (ap) payload.auth = ap
  return payload
}

const handleSend = () => {
  if (!validateUrlBeforeSend()) return
  emit('send', buildHttpExecutePayload())
}

/** Load a persisted request from SavedRequestFull (Wails). */
function applySavedRequestDto(dto) {
  if (!dto) return
  method.value = (dto.method || 'GET').toUpperCase()
  url.value = String(dto.url || '').trim()
  const bm = dto.body_mode || 'none'
  body.value = bm === 'raw' || bm === 'xml' ? (dto.raw_body != null ? String(dto.raw_body) : '') : ''

  if (dto.headers?.length) {
    headers.value = dto.headers.map((h) => ({ key: h.key || '', value: h.value || '' }))
    if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
  } else {
    headers.value = [{ key: '', value: '' }]
  }
  if (bm === 'form_urlencoded') {
    headers.value = [formUrlencodedLockedHeader(), ...stripContentTypeHeaders(headers.value)]
  }

  if (dto.query_params?.length) {
    queryParams.value = dto.query_params.map((p) => ({ key: p.key || '', value: p.value || '' }))
    if (queryParams.value.length === 0) queryParams.value.push({ key: '', value: '' })
  } else {
    queryParams.value = [{ key: '', value: '' }]
  }

  if (dto.form_fields?.length) {
    formFields.value = dto.form_fields.map((f) => ({ key: f.key || '', value: f.value || '' }))
    if (formFields.value.length === 0) formFields.value.push({ key: '', value: '' })
  } else {
    formFields.value = [{ key: '', value: '' }]
  }

  if (dto.multipart_parts?.length) {
    multipartParts.value = dto.multipart_parts.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    }))
    if (multipartParts.value.length === 0) {
      multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
    }
  } else {
    multipartParts.value = [{ key: '', kind: 'text', value: '', file_path: '' }]
  }

  bodyMode.value = bm
  syncAuthFromPayload(dto.auth)
  insecureSkipVerify.value = !!dto.insecure_skip_verify
  activeTab.value = bm === 'none' || bm === '' ? 'params' : 'body'
}

function loadFromSavedRequest(dto) {
  if (!dto) return
  savedRequestId.value = dto.id
  savedFolderId.value = dto.folder_id || null
  savedRequestLabel.value = dto.name || 'Request'
  applySavedRequestDto(dto)
}

function buildSavedRequestFull() {
  const q = queryParams.value.filter((p) => (p.key || '').trim() !== '')
  const h = headers.value.filter((p) => (p.key || '').trim() !== '')
  const full = {
    id: savedRequestId.value,
    folder_id: (savedFolderId.value || '').trim(),
    name: (savedRequestLabel.value || '').trim() || 'Untitled',
    method: method.value,
    url: url.value,
    insecure_skip_verify: !!insecureSkipVerify.value,
    body_mode: bodyMode.value,
    headers: h,
    query_params: q,
    form_fields:
      bodyMode.value === 'form_urlencoded'
        ? formFields.value.filter((p) => (p.key || '').trim() !== '')
        : [],
    multipart_parts:
      bodyMode.value === 'multipart'
        ? multipartParts.value
            .filter((p) => (p.key || '').trim() !== '')
            .map((p) => ({
              key: p.key.trim(),
              kind: p.kind === 'file' ? 'file' : 'text',
              value: p.kind === 'text' ? p.value : '',
              file_path: p.kind === 'file' ? (p.file_path || '').trim() : ''
            }))
        : []
  }
  if (bodyMode.value === 'raw' || bodyMode.value === 'xml') {
    full.raw_body = liveRawOrXmlBodyText()
  }
  const ap = buildAuthPayload()
  if (ap) full.auth = ap
  return full
}

async function saveSavedRequest() {
  if (!savedRequestId.value) return
  try {
    await SavedRequestAPI.Update(buildSavedRequestFull())
    emit('console', `[Saved] "${(savedRequestLabel.value || '').trim() || 'Request'}" updated.`)
    emit('saved-request')
    emit('baseline-committed')
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
  }
}

/** Lưu request: saved → Update; ad-hoc → hỏi folder + tên */
function onGlobalKeydown(e) {
  if (!(e.ctrlKey || e.metaKey)) return
  if (String(e.key || '').toLowerCase() !== 's') return
  e.preventDefault()
  if (savedRequestId.value) {
    saveSavedRequest()
    return
  }
  openSaveAdhocModal()
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown, true)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown, true)
})

/** Guard snapshot-change emissions while we are nạp lại state từ bên ngoài (hydrate). */
let hydrating = false
let snapshotEmitTimer = null

function captureSnapshot() {
  return {
    url: url.value,
    method: method.value,
    body: body.value,
    bodyMode: bodyMode.value,
    bodyRawEditor: bodyRawEditor.value,
    queryParams: queryParams.value.map((p) => ({ key: p.key || '', value: p.value || '' })),
    headers: headers.value.map((h) => {
      const row = { key: h.key || '', value: h.value || '' }
      if (h.locked) row.locked = true
      return row
    }),
    formFields: formFields.value.map((f) => ({ key: f.key || '', value: f.value || '' })),
    multipartParts: multipartParts.value.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    })),
    activeTab: activeTab.value,
    authType: authType.value,
    authBearerToken: authBearerToken.value,
    authUsername: authUsername.value,
    authPassword: authPassword.value,
    authApiKey: authApiKey.value,
    authApiKeyName: authApiKeyName.value,
    authApiKeyIn: authApiKeyIn.value,
    insecureSkipVerify: insecureSkipVerify.value,
    savedRequestId: savedRequestId.value,
    savedFolderId: savedFolderId.value,
    savedRequestLabel: savedRequestLabel.value
  }
}

function hydrate(snap) {
  if (!snap) return
  hydrating = true
  try {
    method.value = (snap.method || 'GET').toUpperCase()
    url.value = snap.url || ''
    body.value = snap.body || ''
    bodyMode.value = snap.bodyMode || 'none'
    bodyRawEditor.value = !!snap.bodyRawEditor
    queryParams.value = Array.isArray(snap.queryParams) && snap.queryParams.length
      ? snap.queryParams.map((p) => ({ key: p.key || '', value: p.value || '' }))
      : [{ key: '', value: '' }]
    const rawHeaders = Array.isArray(snap.headers) && snap.headers.length
      ? snap.headers.map((h) => {
          const row = { key: h.key || '', value: h.value || '' }
          if (h.locked) row.locked = true
          return row
        })
      : [{ key: '', value: '' }]
    if (bodyMode.value === 'form_urlencoded') {
      headers.value = [formUrlencodedLockedHeader(), ...stripContentTypeHeaders(rawHeaders)]
    } else {
      headers.value = rawHeaders.filter((h) => !h.locked)
      if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
    }
    formFields.value = Array.isArray(snap.formFields) && snap.formFields.length
      ? snap.formFields.map((f) => ({ key: f.key || '', value: f.value || '' }))
      : [{ key: '', value: '' }]
    multipartParts.value = Array.isArray(snap.multipartParts) && snap.multipartParts.length
      ? snap.multipartParts.map((p) => ({
          key: p.key || '',
          kind: p.kind === 'file' ? 'file' : 'text',
          value: p.value || '',
          file_path: p.file_path || ''
        }))
      : [{ key: '', kind: 'text', value: '', file_path: '' }]
    activeTab.value = snap.activeTab || 'params'
    authType.value = (snap.authType || 'none').toLowerCase()
    authBearerToken.value = snap.authBearerToken || ''
    authUsername.value = snap.authUsername || ''
    authPassword.value = snap.authPassword || ''
    authApiKey.value = snap.authApiKey || ''
    authApiKeyName.value = snap.authApiKeyName || ''
    authApiKeyIn.value = snap.authApiKeyIn === 'query' ? 'query' : 'header'
    insecureSkipVerify.value = !!snap.insecureSkipVerify
    savedRequestId.value = snap.savedRequestId || null
    savedFolderId.value = snap.savedFolderId || null
    savedRequestLabel.value = snap.savedRequestLabel || ''
  } finally {
    // release hydrating flag next tick so the deep watcher settles first
    setTimeout(() => {
      hydrating = false
    }, 0)
  }
}

function scheduleSnapshotEmit() {
  if (hydrating) return
  if (snapshotEmitTimer) clearTimeout(snapshotEmitTimer)
  snapshotEmitTimer = setTimeout(() => {
    snapshotEmitTimer = null
    emit('snapshot-change', captureSnapshot())
  }, 80)
}

watch(
  [
    url, method, body, bodyMode, bodyRawEditor,
    queryParams, headers, formFields, multipartParts, activeTab,
    authType, authBearerToken, authUsername, authPassword,
    authApiKey, authApiKeyName, authApiKeyIn,
    insecureSkipVerify,
    savedRequestId, savedFolderId, savedRequestLabel
  ],
  scheduleSnapshotEmit,
  { deep: true }
)

defineExpose({
  loadFromSavedRequest,
  applySavedRequestDto,
  saveSavedRequest,
  applyImportPayload,
  snapshot: captureSnapshot,
  hydrate,
  buildHttpExecutePayload
})

const formatJsonBody = () => {
  const raw = (liveRawOrXmlBodyText() ?? '').trim()
  if (!raw) {
    return
  }
  try {
    const parsed = JSON.parse(raw)
    body.value = JSON.stringify(parsed, null, 2)
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'Invalid JSON.'
    emit('console', `[Format JSON] ${msg}`)
  }
}

const formatXmlBody = () => {
  const raw = (liveRawOrXmlBodyText() ?? '').trim()
  if (!raw) {
    return
  }
  try {
    body.value = formatXml(raw, { indentation: '  ', collapseContent: true, lineSeparator: '\n' })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    emit('console', `[Format XML] ${msg}`)
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden border-b border-gray-800 bg-[#212121]">
    <div class="flex shrink-0 items-center gap-2 p-3">
      <select
        v-model="method"
        class="rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm font-bold text-green-500 outline-none"
      >
        <option>GET</option>
        <option>POST</option>
        <option>PUT</option>
        <option>PATCH</option>
        <option>DELETE</option>
        <option>HEAD</option>
        <option>OPTIONS</option>
      </select>
      <EnvVarMirrorField
        v-model="url"
        wrapper-class="min-w-0 flex-1"
        placeholder="https://api.example.com/v1/resource"
        :declared-keys="envKeysForFields"
        :env-values="envValuesForFields"
        @patch-env-value="forwardPatchEnvValue"
        @keydown.enter.prevent="handleSend"
      />
      <button
        v-if="!savedRequestId"
        type="button"
        class="shrink-0 rounded border border-gray-600 bg-[#2a2a2a] px-4 py-2 text-sm font-semibold text-gray-200 transition-colors hover:border-orange-500/50 hover:text-orange-300"
        title="Save into a folder (⌘/Ctrl+S)"
        @click="openSaveAdhocModal"
      >
        Save…
      </button>
      <button
        type="button"
        class="shrink-0 rounded bg-orange-600 px-6 py-2 text-sm font-bold text-white transition-all hover:bg-orange-700 active:scale-95"
        @click="handleSend"
      >
        Send
      </button>
    </div>

    <SnippetPanel :build-payload="buildHttpExecutePayload" @console="(m) => emit('console', m)" />

    <div class="shrink-0 border-t border-gray-800/80 px-3 pt-2 pb-1">
      <div class="flex items-end justify-between gap-3">
        <div class="min-w-0 flex-1">
          <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Request</div>
          <div class="mt-1 flex flex-wrap gap-1 text-xs font-semibold">
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'params' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'params'"
            >
              Query
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'headers' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'headers'"
            >
              Headers
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'auth' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'auth'"
            >
              Auth
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'body' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'body'"
            >
              Body
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="flex min-h-0 flex-1 flex-col overflow-hidden border-t border-gray-800 bg-[#181818]">
      <div v-show="activeTab === 'params'" class="app-scrollbar min-h-0 flex-1 overflow-auto p-3 text-sm">
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-2 pr-2 font-medium">Key</th>
              <th class="pb-2 pr-2 font-medium">Value</th>
              <th class="w-10 pb-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in queryParams" :key="'q-' + i">
              <td class="pr-2 pb-1 align-top">
                <EnvVarMirrorField
                  v-model="row.key"
                  wrapper-class="w-full"
                  :declared-keys="envKeysForFields"
                  :env-values="envValuesForFields"
                  @patch-env-value="forwardPatchEnvValue"
                />
              </td>
              <td class="pr-2 pb-1 align-top">
                <EnvVarMirrorField
                  v-model="row.value"
                  wrapper-class="w-full"
                  :declared-keys="envKeysForFields"
                  :env-values="envValuesForFields"
                  @patch-env-value="forwardPatchEnvValue"
                />
              </td>
              <td class="pb-1 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  aria-label="Remove param"
                  title="Remove row"
                  @click="removeQueryRow(i)"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </td>
            </tr>
            <tr>
              <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
              <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                  aria-label="Add query param"
                  title="Add param"
                  @click="addQueryRow"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-show="activeTab === 'headers'" class="app-scrollbar min-h-0 flex-1 overflow-auto p-3 text-sm">
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-2 pr-2 font-medium">Key</th>
              <th class="pb-2 pr-2 font-medium">Value</th>
              <th class="w-10 pb-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in headers" :key="'h-' + i">
              <td class="pr-2 pb-1 align-top">
                <EnvVarMirrorField
                  v-if="!row.locked"
                  v-model="row.key"
                  wrapper-class="w-full"
                  :declared-keys="envKeysForFields"
                  :env-values="envValuesForFields"
                  @patch-env-value="forwardPatchEnvValue"
                />
                <input
                  v-else
                  :value="row.key"
                  readonly
                  tabindex="-1"
                  title="Set automatically for x-www-form-urlencoded body"
                  class="w-full cursor-default rounded border border-gray-600/80 bg-gray-800/90 px-2 py-1 text-gray-300"
                />
              </td>
              <td class="pr-2 pb-1 align-top">
                <EnvVarMirrorField
                  v-if="!row.locked"
                  v-model="row.value"
                  wrapper-class="w-full"
                  :declared-keys="envKeysForFields"
                  :env-values="envValuesForFields"
                  @patch-env-value="forwardPatchEnvValue"
                />
                <input
                  v-else
                  :value="row.value"
                  readonly
                  tabindex="-1"
                  title="Set automatically for x-www-form-urlencoded body"
                  class="w-full cursor-default rounded border border-gray-600/80 bg-gray-800/90 px-2 py-1 text-gray-300"
                />
              </td>
              <td class="pb-1 align-middle">
                <button
                  v-if="!row.locked"
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  aria-label="Remove header"
                  title="Remove row"
                  @click="removeHeaderRow(i)"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
                <span v-else class="inline-flex h-8 w-8 shrink-0 items-center justify-center text-gray-600" title="Managed by Body type">
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                </span>
              </td>
            </tr>
            <tr>
              <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
              <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                  aria-label="Add header"
                  title="Add header"
                  @click="addHeaderRow"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-show="activeTab === 'auth'" class="app-scrollbar min-h-0 flex-1 overflow-auto p-3 text-sm">
        <p class="mb-3 text-xs text-gray-500">
          Applied after env placeholders (double-brace variables) are resolved. Bearer/Basic replace any existing
          <span class="font-mono text-gray-400">Authorization</span> header. API Key adds or replaces the named header or query key.
        </p>
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <label class="text-xs text-gray-500">Type</label>
          <select
            v-model="authType"
            class="rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
          >
            <option value="none">No auth</option>
            <option value="bearer">Bearer token</option>
            <option value="basic">Basic</option>
            <option value="apikey">API Key</option>
          </select>
        </div>
        <div v-if="authType === 'bearer'" class="space-y-2">
          <label class="block text-xs font-medium text-gray-500">Token</label>
          <EnvVarMirrorField
            v-model="authBearerToken"
            wrapper-class="w-full max-w-xl"
            placeholder="token or {{var}}"
            :declared-keys="envKeysForFields"
            :env-values="envValuesForFields"
            @patch-env-value="forwardPatchEnvValue"
          />
        </div>
        <div v-else-if="authType === 'basic'" class="max-w-xl space-y-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">Username</label>
            <EnvVarMirrorField
              v-model="authUsername"
              wrapper-class="w-full"
              :declared-keys="envKeysForFields"
              :env-values="envValuesForFields"
              @patch-env-value="forwardPatchEnvValue"
            />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">Password</label>
            <EnvVarMirrorField
              v-model="authPassword"
              wrapper-class="w-full"
              :declared-keys="envKeysForFields"
              :env-values="envValuesForFields"
              @patch-env-value="forwardPatchEnvValue"
            />
          </div>
        </div>
        <div v-else-if="authType === 'apikey'" class="max-w-xl space-y-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">Key name</label>
            <EnvVarMirrorField
              v-model="authApiKeyName"
              wrapper-class="w-full"
              placeholder="e.g. X-API-Key"
              :declared-keys="envKeysForFields"
              :env-values="envValuesForFields"
              @patch-env-value="forwardPatchEnvValue"
            />
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-500">Key value</label>
            <EnvVarMirrorField
              v-model="authApiKey"
              wrapper-class="w-full"
              :declared-keys="envKeysForFields"
              :env-values="envValuesForFields"
              @patch-env-value="forwardPatchEnvValue"
            />
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <label class="text-xs text-gray-500">Add to</label>
            <select
              v-model="authApiKeyIn"
              class="rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
            >
              <option value="header">Header</option>
              <option value="query">Query string</option>
            </select>
          </div>
        </div>

        <div class="mt-4 rounded border border-red-500/25 bg-red-500/5 p-3">
          <label class="flex cursor-pointer items-start gap-2 text-xs text-red-200">
            <input v-model="insecureSkipVerify" type="checkbox" class="mt-0.5" />
            <span>
              <span class="font-semibold">TLS: skip certificate verification</span>
              <span class="block text-[11px] text-red-200/80">
                Only use behind corporate TLS inspection or for local dev. Traffic may be vulnerable to MITM.
              </span>
            </span>
          </label>
        </div>
      </div>

      <div v-show="activeTab === 'body'" class="flex min-h-0 flex-1 flex-col p-3" style="min-height: 80px">
        <div class="mb-2 flex shrink-0 flex-wrap items-center justify-between gap-2">
          <div class="flex flex-wrap items-center gap-2">
            <label class="text-xs text-gray-500">Body type</label>
            <select
              v-model="bodyMode"
              class="rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
            >
              <option value="none">None</option>
              <option value="raw">Raw / JSON</option>
              <option value="xml">XML</option>
              <option value="form_urlencoded">x-www-form-urlencoded</option>
              <option value="multipart">form-data (multipart)</option>
            </select>
          </div>
          <div v-if="bodyMode === 'raw' || bodyMode === 'xml'" class="flex shrink-0 items-center gap-2">
            <button
              v-if="bodyMode === 'raw'"
              type="button"
              class="rounded border border-gray-600 bg-gray-800 px-3 py-1 text-xs font-medium text-gray-200 hover:border-orange-500 hover:text-orange-300"
              title="Pretty-print JSON"
              @click="formatJsonBody"
            >
              Format JSON
            </button>
            <button
              v-else
              type="button"
              class="rounded border border-gray-600 bg-gray-800 px-3 py-1 text-xs font-medium text-gray-200 hover:border-orange-500 hover:text-orange-300"
              title="Pretty-print XML"
              @click="formatXmlBody"
            >
              Format XML
            </button>
          </div>
        </div>

        <!-- Raw (JSON) or XML -->
        <template v-if="bodyMode === 'raw' || bodyMode === 'xml'">
          <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-1 overflow-hidden">
            <div v-if="bodyRawEditor" class="flex min-h-0 min-w-0 flex-1 flex-col gap-1 overflow-hidden">
              <div class="flex shrink-0 justify-end">
                <button
                  type="button"
                  class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-orange-400 hover:border-orange-500/50"
                  @click="bodyRawEditor = false"
                >
                  Syntax highlight
                </button>
              </div>
              <EnvVarMirrorField
                v-model="body"
                multiline
                :rows="14"
                wrapper-class="min-h-0 min-w-0 flex-1 overflow-hidden"
                :declared-keys="envKeysForFields"
                :env-values="envValuesForFields"
                @patch-env-value="forwardPatchEnvValue"
              />
            </div>
            <JsonCodeMirror
              v-else
              ref="bodyCodeMirrorRef"
              v-model="body"
              class="min-h-0 min-w-0 flex-1"
              :language="bodyMode === 'xml' ? 'xml' : 'json'"
              :placeholder="bodyMode === 'xml' ? 'Content (XML)' : 'Content (JSON or raw text)'"
              :declared-env-keys="envKeysForFields"
              :env-values="envValuesForFields"
              @request-raw-edit="bodyRawEditor = true"
              @patch-env-value="forwardPatchEnvValue"
            />
          </div>
        </template>

        <!-- none -->
        <div v-else-if="bodyMode === 'none'" class="flex flex-1 items-center justify-center rounded border border-dashed border-gray-700 py-8 text-sm text-gray-600">
          No request body (e.g. GET or no payload).
        </div>

        <!-- urlencoded -->
        <div v-else-if="bodyMode === 'form_urlencoded'" class="app-scrollbar min-h-0 flex-1 overflow-auto text-sm">
          <table class="w-full border-collapse text-xs">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="pb-2 pr-2 font-medium">Key</th>
                <th class="pb-2 pr-2 font-medium">Value</th>
                <th class="w-10 pb-2" />
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in formFields" :key="'f-' + i">
                <td class="pr-2 pb-1 align-top">
                  <EnvVarMirrorField
                    v-model="row.key"
                    wrapper-class="w-full"
                    :declared-keys="envKeysForFields"
                    :env-values="envValuesForFields"
                    @patch-env-value="forwardPatchEnvValue"
                  />
                </td>
                <td class="pr-2 pb-1 align-top">
                  <EnvVarMirrorField
                    v-model="row.value"
                    wrapper-class="w-full"
                    :declared-keys="envKeysForFields"
                    :env-values="envValuesForFields"
                    @patch-env-value="forwardPatchEnvValue"
                  />
                </td>
                <td class="pb-1 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                    title="Remove row"
                    @click="removeFormFieldRow(i)"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </td>
              </tr>
              <tr>
                <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
                <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                    aria-label="Add form field"
                    title="Add field"
                    @click="addFormFieldRow"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- multipart -->
        <div v-else-if="bodyMode === 'multipart'" class="app-scrollbar min-h-0 flex-1 overflow-auto text-sm">
          <table class="w-full border-collapse text-xs">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="pb-2 pr-1 font-medium">Key</th>
                <th class="w-24 pb-2 pr-1 font-medium">Type</th>
                <th class="pb-2 pr-2 font-medium">Value / File</th>
                <th class="w-10 pb-2" />
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in multipartParts" :key="'m-' + i">
                <td class="pr-1 pb-1 align-top">
                  <EnvVarMirrorField
                    v-model="row.key"
                    wrapper-class="w-full"
                    :declared-keys="envKeysForFields"
                    :env-values="envValuesForFields"
                    @patch-env-value="forwardPatchEnvValue"
                  />
                </td>
                <td class="pr-1 pb-1 align-top">
                  <select v-model="row.kind" class="w-full rounded border border-gray-700 bg-gray-900 px-1 py-1 text-gray-200">
                    <option value="text">Text</option>
                    <option value="file">File</option>
                  </select>
                </td>
                <td class="pr-2 pb-1 align-top">
                  <EnvVarMirrorField
                    v-if="row.kind === 'text'"
                    v-model="row.value"
                    wrapper-class="w-full"
                    placeholder="Value"
                    :declared-keys="envKeysForFields"
                    :env-values="envValuesForFields"
                    @patch-env-value="forwardPatchEnvValue"
                  />
                  <div v-else class="flex flex-col gap-1">
                    <EnvVarMirrorField
                      v-model="row.file_path"
                      wrapper-class="w-full"
                      placeholder="Path or {{var}}"
                      :declared-keys="envKeysForFields"
                      :env-values="envValuesForFields"
                      @patch-env-value="forwardPatchEnvValue"
                    />
                    <button
                      type="button"
                      class="self-start rounded border border-gray-600 bg-gray-800 px-2 py-0.5 text-[11px] text-orange-400 hover:border-orange-500"
                      @click="pickMultipartFile(i)"
                    >
                      Choose file…
                    </button>
                  </div>
                </td>
                <td class="pb-1 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                    title="Remove row"
                    @click="removeMultipartRow(i)"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </td>
              </tr>
              <tr>
                <td class="border-t border-gray-800/50 p-0 pt-2" colspan="3"></td>
                <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                    aria-label="Add multipart field"
                    title="Add field"
                    @click="addMultipartField"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

  <Teleport to="#app">
    <div
      v-if="saveAdhocModalOpen"
      class="fixed inset-0 z-[57] flex items-center justify-center bg-black/50 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="save-adhoc-title"
    >
      <div class="w-full max-w-md rounded-lg border border-gray-700 bg-[#1f1f1f] shadow-xl">
        <div class="border-b border-gray-700 px-4 py-3">
          <h3 id="save-adhoc-title" class="text-sm font-semibold text-white">Save request to folder</h3>
          <p class="mt-1 text-[11px] text-gray-500">Choose where to store this request in your library.</p>
        </div>
        <div class="space-y-3 p-4">
          <div>
            <label class="mb-1 block text-[10px] font-medium uppercase tracking-wide text-gray-500">Folder</label>
            <select
              v-model="saveAdhocFolderId"
              :disabled="saveAdhocFoldersLoading || saveAdhocFolderOptions.length === 0"
              class="w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            >
              <option v-for="opt in saveAdhocFolderOptions" :key="opt.id" :value="opt.id">{{ opt.label }}</option>
            </select>
            <p v-if="saveAdhocFoldersLoading" class="mt-1 text-[10px] text-gray-500">Loading folders…</p>
          </div>
          <div>
            <label class="mb-1 block text-[10px] font-medium uppercase tracking-wide text-gray-500">Request name</label>
            <input
              v-model="saveAdhocName"
              type="text"
              class="w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
              placeholder="My API call"
              @keydown.enter.prevent="submitSaveAdhoc"
            />
          </div>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
          <button
            type="button"
            class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
            :disabled="saveAdhocSubmitting"
            @click="closeSaveAdhocModal"
          >
            Cancel
          </button>
          <button
            type="button"
            class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
            :disabled="saveAdhocSubmitting || saveAdhocFoldersLoading"
            @click="submitSaveAdhoc"
          >
            {{ saveAdhocSubmitting ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
