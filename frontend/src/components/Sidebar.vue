<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick, provide } from 'vue'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import * as EnvAPI from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'
import {
  List as ListHistory,
  Get as GetHistoryDetail,
  Delete as DeleteHistoryEntry
} from '../../wailsjs/wailsjs/go/delivery/HistoryHandler'
import { ImportFromCurl } from '../../wailsjs/wailsjs/go/delivery/HTTPHandler'
import { SearchTree as SearchTreeAPI } from '../../wailsjs/wailsjs/go/delivery/SearchHandler'
import HistoryDetailModal from './HistoryDetailModal.vue'
import FolderTreeNode from './FolderTreeNode.vue'
import ImportCollectionModal from './ImportCollectionModal.vue'
import HighlightText from './HighlightText.vue'

const props = defineProps({
  /** Selected root folder id (UUID) — history + tree scope */
  activeRootFolderId: { type: String, default: null }
})

const emit = defineEmits([
  'update:activeRootFolderId',
  'open-saved-request',
  'open-environment',
  'main-workspace-request',
  'environments-changed',
  'environment-deleted',
  'console',
  'apply-curl-import'
])

/** Đồng bộ UI sau khi tạo folder/request trong folder con: instance đúng `folderId` sẽ gọi load() */
const folderTreeReload = ref({ targetId: null, tick: 0 })
provide('folderTreeReload', folderTreeReload)

/**
 * Ra lệnh cho cây folder "lộ diện" một node cụ thể sau search.
 * - `chain`: [rootId, ..., targetFolderId] hoặc [rootId, ..., parentOfRequest]
 * - `targetId`: node cuối trong chain — được scroll + flash highlight
 * - `requestId`: khi click request hit, flash thêm hàng request đó sau khi lộ folder cha
 * - `tick`: tăng mỗi lần ra lệnh để các FolderTreeNode nhận re-trigger
 */
const folderTreeReveal = ref({ chain: [], targetId: null, requestId: null, tick: 0 })
provide('folderTreeReveal', folderTreeReveal)

/** @type {import('vue').Ref<'folders' | 'env' | 'history'>} */
const sidebarTab = ref('folders')

const rootFolders = ref([])
/** Backend may return null; never let v-for / .length throw or the whole sidebar can go blank */
const rootFolderList = computed(() => (Array.isArray(rootFolders.value) ? rootFolders.value : []))
const loading = ref(false)

const historyItems = ref([])
const historyLoading = ref(false)
const historyList = computed(() => (Array.isArray(historyItems.value) ? historyItems.value : []))
const submitting = ref(false)

const historyDetailOpen = ref(false)
const historyDetailLoading = ref(false)
/** @type {import('vue').Ref<Record<string, unknown> | null>} */
const historyDetailItem = ref(null)

const openHistoryDetail = async (h) => {
  if (!h?.id) return
  historyDetailOpen.value = true
  historyDetailLoading.value = true
  historyDetailItem.value = null
  try {
    const full = await GetHistoryDetail(String(h.id))
    historyDetailItem.value = full && typeof full === 'object' ? { ...full } : full
  } catch (error) {
    console.error('[History] Detail failed:', error)
    showToast('error', `Could not load history detail: ${error?.message || error}`)
    closeHistoryDetail()
  } finally {
    historyDetailLoading.value = false
  }
}

const closeHistoryDetail = () => {
  historyDetailOpen.value = false
  historyDetailItem.value = null
  historyDetailLoading.value = false
}

const curlModalOpen = ref(false)
const curlText = ref('curl -X GET "https://httpbin.org/get?hello=world"')

const importModalOpen = ref(false)

const openImportCollectionModal = () => {
  importModalOpen.value = true
}

const closeImportCollectionModal = () => {
  importModalOpen.value = false
}

/** After a successful import, refresh the Folders panel and auto-select the new root folder. */
const onCollectionImported = async (result) => {
  try {
    await loadRootFolders()
  } catch (error) {
    console.error('[Import] Refresh failed:', error)
  }
  if (result?.root_folder_id) {
    emit('update:activeRootFolderId', result.root_folder_id)
  }
  if (result?.environment_id) {
    try {
      await loadEnvironments()
    } catch (error) {
      console.error('[Import] Env refresh failed:', error)
    }
    emit('environments-changed')
  }
  const label = result?.root_folder_name || 'collection'
  showToast('success', `Imported "${label}" (${result?.requests_created ?? 0} requests)`)
}

/** Environments tab */
const envItems = ref([])
const envLoading = ref(false)
const envList = computed(() => (Array.isArray(envItems.value) ? envItems.value : []))

const envModalState = ref({
  show: false,
  mode: 'create',
  title: '',
  submitLabel: '',
  target: null
})
const envFormName = ref('')
const envFormDescription = ref('')
const envSubmitting = ref(false)

const menuOpenForEnvId = ref(null)
const envMenuStyle = ref({
  position: 'fixed',
  top: '0px',
  left: '0px',
  zIndex: 45
})

const openCurlImportModal = () => {
  curlModalOpen.value = true
}

const closeCurlModal = () => {
  curlModalOpen.value = false
}

const applyCurlImport = async () => {
  const text = (curlText.value || '').trim()
  if (!text) {
    emit('console', '[cURL] Paste a command first.')
    return
  }
  try {
    const payload = await ImportFromCurl(text)
    let plain = payload
    if (payload && typeof payload === 'object') {
      try {
        plain = JSON.parse(JSON.stringify(payload))
      } catch {
        plain = payload
      }
    }
    emit('apply-curl-import', plain)
    closeCurlModal()
    emit('console', '[Import] cURL loaded as a new ad-hoc request. Use Save in the request panel to store it in a folder.')
  } catch (e) {
    emit('console', `[cURL] ${e?.message || String(e)}`)
  }
}

const onDeleteHistoryFromModal = async () => {
  const id = historyDetailItem.value?.id
  if (!id) return
  try {
    await DeleteHistoryEntry(String(id))
    showToast('success', 'Removed from history')
    closeHistoryDetail()
    await loadHistory()
  } catch (error) {
    console.error('[History] Delete failed:', error)
    showToast('error', `Could not delete history: ${error?.message || error}`)
  }
}

const toast = ref({
  show: false,
  type: 'info',
  message: ''
})
let toastTimer = null

const modalState = ref({
  show: false,
  mode: 'create',
  title: '',
  submitLabel: '',
  target: null
})

const formName = ref('')
const formDescription = ref('')

const showToast = (type, message) => {
  if (toastTimer) {
    clearTimeout(toastTimer)
    toastTimer = null
  }
  toast.value = {
    show: true,
    type,
    message
  }
  toastTimer = setTimeout(() => {
    toast.value.show = false
  }, 3000)
}

const openCreateModal = () => {
  formName.value = ''
  formDescription.value = ''
  modalState.value = {
    show: true,
    mode: 'create',
    title: 'Create root folder',
    submitLabel: 'Create',
    target: null
  }
}

const openEditModal = (ws) => {
  if (!ws) return
  formName.value = ws.name || ''
  formDescription.value = ws.description || ''
  modalState.value = {
    show: true,
    mode: 'edit',
    title: 'Edit folder',
    submitLabel: 'Save',
    target: ws
  }
}

const openDeleteModal = (ws) => {
  if (!ws) return
  modalState.value = {
    show: true,
    mode: 'delete',
    title: 'Delete folder',
    submitLabel: 'Delete',
    target: ws
  }
}

const closeModal = (force = false) => {
  if (!force && submitting.value) return
  modalState.value.show = false
}

/** Per-folder ⋮ menu: teleport + fixed so overflow does not clip */
const menuOpenForId = ref(null)
const menuStyle = ref({
  position: 'fixed',
  top: '0px',
  left: '0px',
  zIndex: 45
})

/**
 * Giữ instance FolderTreeNode theo root id.
 * Không dùng ref()/reactive ở đây: gán trong callback `:ref` mỗi lần mount sẽ kích hoạt re-render
 * liên tục (vòng lặp) và có thể làm WebView/Wails treo — cửa sổ không hiện, process vẫn chạy.
 */
const folderTreeRefById = new Map()

function setFolderTreeRef(wsId, el) {
  if (el) folderTreeRefById.set(wsId, el)
  else folderTreeRefById.delete(wsId)
}

/** Chỉ các root được mở rõ (mặc định: thu gọn — không có key = đóng). */
const rootTreeExpanded = ref(/** @type {Record<string, boolean>} */ ({}))

function isRootTreeExpanded(wsId) {
  return !!rootTreeExpanded.value[wsId]
}

function toggleRootTree(wsId) {
  const next = { ...rootTreeExpanded.value }
  if (next[wsId]) delete next[wsId]
  else next[wsId] = true
  rootTreeExpanded.value = next
}

/**
 * Root row: mở/đóng cây 1 click.
 * - Root đang chọn: click = toggle thu/mở.
 * - Root khác đang chọn + cây root này đang mở: 1 click chỉ thu cây (giữ selection), tránh phải click 2 lần.
 * - Root khác + cây đang đóng: chọn root + mở cây.
 */
function onRootFolderRowClick(ws) {
  if (!ws?.id) return
  const isActive = props.activeRootFolderId === ws.id
  const expanded = isRootTreeExpanded(ws.id)

  if (!isActive) {
    if (expanded) {
      const next = { ...rootTreeExpanded.value }
      delete next[ws.id]
      rootTreeExpanded.value = next
      return
    }
    emit('update:activeRootFolderId', ws.id)
    const next = { ...rootTreeExpanded.value }
    next[ws.id] = true
    rootTreeExpanded.value = next
    return
  }

  toggleRootTree(ws.id)
}

const menuTargetWs = computed(() => {
  const id = menuOpenForId.value
  if (id == null) return null
  return rootFolderList.value.find((w) => w.id === id) ?? null
})

const closeWorkspaceMenu = () => {
  menuOpenForId.value = null
}

const toggleWorkspaceMenu = (ws, event) => {
  event?.stopPropagation()
  if (!ws) return
  if (menuOpenForId.value === ws.id) {
    closeWorkspaceMenu()
    return
  }
  menuOpenForId.value = ws.id
  const el = event?.currentTarget
  if (el && typeof el.getBoundingClientRect === 'function') {
    const r = el.getBoundingClientRect()
    const width = 220
    let left = r.right - width
    if (left < 8) left = 8
    if (left + width > window.innerWidth - 8) left = window.innerWidth - width - 8
    const top = r.bottom + 4
    menuStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 45
    }
  }
}

const onEditFromMenu = (ws) => {
  closeWorkspaceMenu()
  openEditModal(ws)
}

const onDeleteFromMenu = (ws) => {
  closeWorkspaceMenu()
  openDeleteModal(ws)
}

/** Select root folder then open nested-folder / request modal for that row */
const onNewFolderFromMenu = async (ws) => {
  closeWorkspaceMenu()
  if (!ws?.id) return
  emit('update:activeRootFolderId', ws.id)
  await nextTick()
  await nextTick()
  folderTreeRefById.get(ws.id)?.openCreateSubfolder?.(ws.id)
}

const onNewRootRequestFromMenu = async (ws) => {
  closeWorkspaceMenu()
  if (!ws?.id) return
  emit('update:activeRootFolderId', ws.id)
  await nextTick()
  await nextTick()
  folderTreeRefById.get(ws.id)?.openCreateRequest?.(ws.id)
}

const onDocumentPointerDown = (e) => {
  if (menuOpenForId.value != null) {
    const t = e.target
    if (!t.closest?.('[data-ws-menu]')) {
      closeWorkspaceMenu()
    }
  }
  if (menuOpenForEnvId.value != null) {
    const t = e.target
    if (!t.closest?.('[data-env-menu]')) {
      closeEnvMenu()
    }
  }
}

onMounted(() => {
  /* Run after ⋮ click so toggle opens before document handler closes the menu */
  document.addEventListener('pointerdown', onDocumentPointerDown, false)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, false)
})

const syncSelectionAfterLoad = () => {
  const list = rootFolderList.value
  const ids = new Set(list.map((w) => w.id))
  if (props.activeRootFolderId && !ids.has(props.activeRootFolderId)) {
    emit('update:activeRootFolderId', list[0]?.id ?? null)
    return
  }
  if (!props.activeRootFolderId && list.length > 0) {
    emit('update:activeRootFolderId', list[0].id)
  }
}

const loadHistory = async () => {
  historyLoading.value = true
  try {
    const list = await ListHistory('')
    historyItems.value = Array.isArray(list) ? list : []
  } catch (error) {
    console.error('[History] Load failed:', error)
    historyItems.value = []
    showToast('error', `Could not load history: ${error?.message || error}`)
  } finally {
    historyLoading.value = false
  }
}

/** Called after a request is sent so the History tab stays fresh. */
const refreshHistory = async () => {
  await loadHistory()
}

const loadEnvironments = async () => {
  envLoading.value = true
  try {
    const list = await EnvAPI.List()
    envItems.value = Array.isArray(list) ? list : []
  } catch (error) {
    console.error('[Env] Load failed:', error)
    envItems.value = []
    showToast('error', `Could not load environments: ${error?.message || error}`)
  } finally {
    envLoading.value = false
  }
}

const closeEnvMenu = () => {
  menuOpenForEnvId.value = null
}

const toggleEnvMenu = (item, event) => {
  event?.stopPropagation()
  if (!item?.id) return
  if (menuOpenForEnvId.value === item.id) {
    closeEnvMenu()
    return
  }
  menuOpenForEnvId.value = item.id
  const el = event?.currentTarget
  if (el && typeof el.getBoundingClientRect === 'function') {
    const r = el.getBoundingClientRect()
    const width = 200
    let left = r.right - width
    if (left < 8) left = 8
    if (left + width > window.innerWidth - 8) left = window.innerWidth - width - 8
    const top = r.bottom + 4
    envMenuStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 45
    }
  }
}

const menuTargetEnv = computed(() => {
  const id = menuOpenForEnvId.value
  if (id == null) return null
  return envList.value.find((e) => e.id === id) ?? null
})

const openCreateEnvModal = () => {
  closeEnvMenu()
  envFormName.value = ''
  envFormDescription.value = ''
  envModalState.value = {
    show: true,
    mode: 'create',
    title: 'New environment',
    submitLabel: 'Create',
    target: null
  }
}

const openEditEnvModal = (item) => {
  if (!item) return
  closeEnvMenu()
  envFormName.value = item.name || ''
  envFormDescription.value = item.description || ''
  envModalState.value = {
    show: true,
    mode: 'edit',
    title: 'Edit environment',
    submitLabel: 'Save',
    target: item
  }
}

const openDeleteEnvModal = (item) => {
  if (!item) return
  closeEnvMenu()
  envModalState.value = {
    show: true,
    mode: 'delete',
    title: 'Delete environment',
    submitLabel: 'Delete',
    target: item
  }
}

const closeEnvModal = (force = false) => {
  if (!force && envSubmitting.value) return
  envModalState.value.show = false
}

const submitEnvModal = async () => {
  try {
    envSubmitting.value = true
    if (envModalState.value.mode === 'create') {
      const name = envFormName.value.trim()
      if (!name) {
        showToast('warning', 'Environment name cannot be empty')
        return
      }
      await EnvAPI.Create(name, envFormDescription.value.trim())
      showToast('success', 'Environment created')
      closeEnvModal(true)
      await loadEnvironments()
      emit('environments-changed')
      return
    }
    if (envModalState.value.mode === 'edit') {
      const target = envModalState.value.target
      const name = envFormName.value.trim()
      if (!target?.id || !name) {
        showToast('warning', 'Invalid environment')
        return
      }
      await EnvAPI.UpdateMeta(target.id, name, envFormDescription.value.trim())
      showToast('success', 'Environment updated')
      closeEnvModal(true)
      await loadEnvironments()
      emit('environments-changed')
      return
    }
    if (envModalState.value.mode === 'delete') {
      const target = envModalState.value.target
      if (!target?.id) {
        showToast('warning', 'Invalid environment')
        return
      }
      await EnvAPI.Delete(target.id)
      showToast('success', `Deleted "${target.name}"`)
      emit('environment-deleted', target.id)
      closeEnvModal(true)
      await loadEnvironments()
      emit('environments-changed')
    }
  } catch (error) {
    console.error('[Env] Action failed:', error)
    const label =
      envModalState.value.mode === 'create'
        ? 'Create'
        : envModalState.value.mode === 'edit'
          ? 'Update'
          : 'Delete'
    const msg = error?.message || String(error)
    if (msg.includes('ENV_602') || msg.includes('already exists')) {
      showToast('warning', 'That environment name is already in use.')
    } else {
      showToast('error', `${label} failed: ${msg}`)
    }
  } finally {
    envSubmitting.value = false
  }
}

watch(sidebarTab, (tab) => {
  if (tab === 'history') {
    loadHistory()
  }
  if (tab === 'env') {
    loadEnvironments()
  }
  if (tab === 'folders' || tab === 'history') {
    emit('main-workspace-request')
  }
})

/* ─────────────────────────── Folders search ────────────────────────────── */

const folderSearchInput = ref('')
/** The debounced, trimmed query actually sent to the backend. */
const folderSearchQuery = ref('')
const folderSearchLoading = ref(false)
const folderSearchError = ref('')
/** @type {import('vue').Ref<{ folders: any[], requests: any[], truncated: boolean } | null>} */
const folderSearchResults = ref(null)

const folderSearchActive = computed(() => folderSearchQuery.value.trim().length > 0)

let folderSearchDebounceTimer = null
let folderSearchRequestToken = 0

function scheduleFolderSearch(raw) {
  const next = String(raw ?? '').trim()
  if (folderSearchDebounceTimer) {
    clearTimeout(folderSearchDebounceTimer)
    folderSearchDebounceTimer = null
  }
  if (!next) {
    folderSearchQuery.value = ''
    folderSearchResults.value = null
    folderSearchError.value = ''
    folderSearchLoading.value = false
    return
  }
  folderSearchDebounceTimer = setTimeout(() => {
    folderSearchDebounceTimer = null
    runFolderSearch(next)
  }, 250)
}

async function runFolderSearch(query) {
  folderSearchQuery.value = query
  folderSearchError.value = ''
  folderSearchLoading.value = true
  const myToken = ++folderSearchRequestToken
  try {
    const res = await SearchTreeAPI(query, 100)
    // Discard stale responses so a slow request can't overwrite a newer one.
    if (myToken !== folderSearchRequestToken) return
    folderSearchResults.value = res && typeof res === 'object'
      ? {
          folders: Array.isArray(res.folders) ? res.folders : [],
          requests: Array.isArray(res.requests) ? res.requests : [],
          truncated: !!res.truncated
        }
      : { folders: [], requests: [], truncated: false }
  } catch (error) {
    if (myToken !== folderSearchRequestToken) return
    console.error('[Search] SearchTree failed:', error)
    folderSearchError.value = error?.message || String(error)
    folderSearchResults.value = { folders: [], requests: [], truncated: false }
  } finally {
    if (myToken === folderSearchRequestToken) {
      folderSearchLoading.value = false
    }
  }
}

function onFolderSearchInput(e) {
  const v = e?.target?.value ?? ''
  folderSearchInput.value = v
  scheduleFolderSearch(v)
}

function clearFolderSearch() {
  folderSearchInput.value = ''
  scheduleFolderSearch('')
}

/** Id of the root folder row to flash briefly after a search reveal whose
 * target happens to BE the root. Nested folder / request flashes are handled
 * inside FolderTreeNode via the provided `folderTreeReveal` state. */
const flashRootFolderId = ref(null)
let flashRootTimer = null

/** Reveal a node deep in the tree: activate its root, expand the root row,
 * close the search overlay, and broadcast a reveal command so each nested
 * FolderTreeNode on the chain auto-expands the correct child (and the leaf
 * scrolls into view with a brief flash highlight). */
function revealInTree({ chain, targetId, requestId }) {
  if (!Array.isArray(chain) || chain.length === 0 || !targetId) return
  const rootId = chain[0]
  emit('update:activeRootFolderId', rootId)
  const next = { ...rootTreeExpanded.value }
  next[rootId] = true
  rootTreeExpanded.value = next
  folderTreeReveal.value = {
    chain: chain.slice(),
    targetId,
    requestId: requestId || null,
    tick: (folderTreeReveal.value?.tick || 0) + 1
  }
  clearFolderSearch()

  // Special case: user searched a root folder itself — the row lives in THIS
  // component's template, not inside FolderTreeNode, so we flash it here.
  if (targetId === rootId && !requestId) {
    if (flashRootTimer) clearTimeout(flashRootTimer)
    flashRootFolderId.value = rootId
    nextTick(() => {
      const el = document.querySelector(`[data-root-folder-row="${rootId}"]`)
      if (el && typeof el.scrollIntoView === 'function') {
        el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
      }
    })
    flashRootTimer = setTimeout(() => {
      flashRootFolderId.value = null
      flashRootTimer = null
    }, 1500)
  }
}

function onFolderSearchHit(hit) {
  if (!hit?.id) return
  // ancestor_ids goes [root, ..., self]; fall back to [root_id, self] when
  // the backend couldn't reconstruct (shouldn't happen but keep UI usable).
  const chain = Array.isArray(hit.ancestor_ids) && hit.ancestor_ids.length > 0
    ? hit.ancestor_ids
    : [hit.root_id, hit.id].filter(Boolean)
  revealInTree({ chain, targetId: hit.id })
}

function onRequestSearchHit(hit) {
  if (!hit?.id) return
  // The target folder to expand is the request's parent folder; requestId
  // tells the leaf node to flash that specific request row.
  const chain = Array.isArray(hit.ancestor_ids) && hit.ancestor_ids.length > 0
    ? hit.ancestor_ids
    : [hit.root_id, hit.folder_id].filter(Boolean)
  revealInTree({ chain, targetId: hit.folder_id, requestId: hit.id })
  emit('open-saved-request', hit.id)
}

/* ─────────────────────────── History filter ────────────────────────────── */

/** Free-text filter (matches URL, method, status code as substring). */
const historyFilterText = ref('')
/** Empty set = "all methods"; else whitelist. */
const historyMethodFilter = ref(/** @type {Set<string>} */(new Set()))
/** Empty set = "all status groups"; else whitelist of '2xx' | '3xx' | '4xx' | '5xx' | 'other'. */
const historyStatusFilter = ref(/** @type {Set<string>} */(new Set()))
const historyDateFrom = ref('')
const historyDateTo = ref('')
const historyFilterPanelOpen = ref(false)

const historyMethodChoices = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const historyStatusChoices = [
  { id: '2xx', label: '2xx' },
  { id: '3xx', label: '3xx' },
  { id: '4xx', label: '4xx' },
  { id: '5xx', label: '5xx' },
  { id: 'other', label: 'Other' }
]

function toggleHistoryMethod(m) {
  const set = new Set(historyMethodFilter.value)
  if (set.has(m)) set.delete(m)
  else set.add(m)
  historyMethodFilter.value = set
}

function toggleHistoryStatus(group) {
  const set = new Set(historyStatusFilter.value)
  if (set.has(group)) set.delete(group)
  else set.add(group)
  historyStatusFilter.value = set
}

function statusGroupOf(code) {
  const c = Number(code)
  if (c >= 200 && c < 300) return '2xx'
  if (c >= 300 && c < 400) return '3xx'
  if (c >= 400 && c < 500) return '4xx'
  if (c >= 500 && c < 600) return '5xx'
  return 'other'
}

function parseDateInput(v, endOfDay = false) {
  const s = String(v ?? '').trim()
  if (!s) return null
  // <input type="date"> gives 'YYYY-MM-DD' — interpret as local day.
  const d = new Date(endOfDay ? `${s}T23:59:59.999` : `${s}T00:00:00.000`)
  return Number.isNaN(d.getTime()) ? null : d
}

function parseRowDate(raw) {
  if (raw == null || raw === '') return null
  const d = new Date(raw)
  return Number.isNaN(d.getTime()) ? null : d
}

const historyFilterActive = computed(() => {
  return (
    (historyFilterText.value || '').trim().length > 0 ||
    historyMethodFilter.value.size > 0 ||
    historyStatusFilter.value.size > 0 ||
    !!historyDateFrom.value ||
    !!historyDateTo.value
  )
})

function clearHistoryFilters() {
  historyFilterText.value = ''
  historyMethodFilter.value = new Set()
  historyStatusFilter.value = new Set()
  historyDateFrom.value = ''
  historyDateTo.value = ''
}

const filteredHistoryList = computed(() => {
  const rows = historyList.value
  if (!historyFilterActive.value) return rows
  const text = historyFilterText.value.trim().toLowerCase()
  const methodSet = historyMethodFilter.value
  const statusSet = historyStatusFilter.value
  const from = parseDateInput(historyDateFrom.value, false)
  const to = parseDateInput(historyDateTo.value, true)
  return rows.filter((h) => {
    if (methodSet.size > 0 && !methodSet.has(String(h.method || '').toUpperCase())) return false
    if (statusSet.size > 0 && !statusSet.has(statusGroupOf(h.status_code))) return false
    if (from || to) {
      const d = parseRowDate(h.created_at)
      if (!d) return false
      if (from && d < from) return false
      if (to && d > to) return false
    }
    if (text) {
      const hay = [h.url, h.method, String(h.status_code ?? '')].filter(Boolean).join(' ').toLowerCase()
      if (!hay.includes(text)) return false
    }
    return true
  })
})

const loadRootFolders = async () => {
  loading.value = true
  try {
    const list = await FolderAPI.ListRootFolders()
    rootFolders.value = Array.isArray(list) ? list : []
    syncSelectionAfterLoad()
  } catch (error) {
    console.error('[Folder] Load failed:', error)
    showToast('error', `Could not load folders: ${error?.message || error}`)
  } finally {
    loading.value = false
  }
}

const submitModal = async () => {
  try {
    submitting.value = true
    if (modalState.value.mode === 'create') {
      const name = formName.value.trim()
      if (!name) {
        showToast('warning', 'Folder name cannot be empty')
        return
      }
      await FolderAPI.CreateFolder({
        name,
        description: formDescription.value.trim()
      })
      showToast('success', 'Folder created')
      closeModal(true)
      await loadRootFolders()
      return
    }

    if (modalState.value.mode === 'edit') {
      const target = modalState.value.target
      const name = formName.value.trim()
      if (!target || !name) {
        showToast('warning', 'Invalid folder data')
        return
      }
      await FolderAPI.UpdateFolder(target.id, name, formDescription.value.trim())
      showToast('success', 'Folder updated')
      closeModal(true)
      await loadRootFolders()
      return
    }

    if (modalState.value.mode === 'delete') {
      const target = modalState.value.target
      if (!target) {
        showToast('warning', 'Invalid folder')
        return
      }
      await FolderAPI.DeleteFolder(target.id)
      showToast('success', `Deleted folder "${target.name}"`)
      if (props.activeRootFolderId === target.id) {
        emit('update:activeRootFolderId', null)
      }
      closeModal(true)
      await loadRootFolders()
    }
  } catch (error) {
    console.error('[Folder] Action failed:', error)
    const label = modalState.value.mode === 'create'
      ? 'Create'
      : (modalState.value.mode === 'edit' ? 'Update' : 'Delete')
    const msg = error?.message || String(error)
    if (msg.includes('FOL_301') || msg.includes('already exists')) {
      showToast('warning', 'That folder name is already in use here. Choose another name.')
    } else {
      showToast('error', `${label} folder failed: ${msg}`)
    }
    // Keep modal open to fix name / retry
  } finally {
    submitting.value = false
  }
}

onMounted(loadRootFolders)

function truncateMiddle(s, max) {
  if (s == null || s === '') return ''
  const str = String(s)
  if (str.length <= max) return str
  const half = Math.max(1, Math.floor((max - 1) / 2))
  return str.slice(0, half) + '…' + str.slice(str.length - half)
}

function formatHistoryTime(raw) {
  if (raw == null || raw === '') return '—'
  if (typeof raw === 'string') {
    const d = new Date(raw)
    return Number.isNaN(d.getTime())
      ? raw
      : d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
  }
  if (typeof raw === 'object' && raw !== null) {
    try {
      const d = new Date(raw)
      if (!Number.isNaN(d.getTime())) {
        return d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
      }
    } catch {
      /* ignore */
    }
  }
  return String(raw)
}

function statusBadgeClass(code) {
  const c = Number(code)
  if (c === 0) return 'bg-gray-600 text-gray-100'
  if (c >= 200 && c < 300) return 'bg-emerald-900/90 text-emerald-200'
  if (c >= 300 && c < 400) return 'bg-blue-900/90 text-blue-200'
  if (c >= 400 && c < 500) return 'bg-amber-900/90 text-amber-200'
  if (c >= 500) return 'bg-red-900/90 text-red-200'
  return 'bg-gray-700 text-gray-200'
}

defineExpose({
  refreshHistory,
  refreshEnvironments: loadEnvironments,
  refreshCatalog: () => {
    for (const node of folderTreeRefById.values()) {
      node?.load?.()
    }
  }
})
</script>

<template>
  <!-- Explicit flex + size: do not rely on Tailwind alone inside Wails WebView -->
  <div
    class="relative flex min-h-0 min-w-0 flex-col"
    style="height: 100%; width: 100%; overflow: hidden; display: flex; flex-direction: column; box-sizing: border-box; background: #212121; color: #e5e7eb"
  >
    <aside
      class="flex min-h-0 flex-1 flex-col border-r border-gray-800 bg-[#212121]"
      style="flex: 1 1 0; min-height: 0; background: #212121"
    >
      <div class="flex shrink-0 border-b border-gray-800">
        <button
          type="button"
          class="flex-1 px-1.5 py-2.5 text-[10px] font-semibold uppercase tracking-wide transition-colors sm:text-xs"
          :class="
            sidebarTab === 'folders'
              ? 'border-b-2 border-orange-500 bg-[#1a1a1a] text-white'
              : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
          "
          @click="sidebarTab = 'folders'"
        >
          Folders
        </button>
        <button
          type="button"
          class="flex-1 px-1.5 py-2.5 text-[10px] font-semibold uppercase tracking-wide transition-colors sm:text-xs"
          :class="
            sidebarTab === 'env'
              ? 'border-b-2 border-orange-500 bg-[#1a1a1a] text-white'
              : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
          "
          @click="sidebarTab = 'env'"
        >
          Env
        </button>
        <button
          type="button"
          class="flex-1 px-1.5 py-2.5 text-[10px] font-semibold uppercase tracking-wide transition-colors sm:text-xs"
          :class="
            sidebarTab === 'history'
              ? 'border-b-2 border-orange-500 bg-[#1a1a1a] text-white'
              : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
          "
          @click="sidebarTab = 'history'"
        >
          History
        </button>
      </div>

      <template v-if="sidebarTab === 'folders'">
        <div class="flex min-h-0 flex-1 flex-col">
          <div class="relative shrink-0 border-b border-gray-800 px-2 py-2">
            <input
              :value="folderSearchInput"
              type="search"
              placeholder="Search folders & requests…"
              aria-label="Search folders and saved requests"
              class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1 pr-7 text-xs text-gray-200 placeholder:text-gray-500 focus:border-orange-500/60 focus:outline-none"
              @input="onFolderSearchInput"
            />
            <button
              v-if="folderSearchInput"
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 rounded px-1 text-xs text-gray-500 hover:text-gray-200"
              aria-label="Clear search"
              title="Clear"
              @click="clearFolderSearch"
            >
              ✕
            </button>
          </div>

          <div v-if="folderSearchActive" class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
            <div v-if="folderSearchLoading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
              Searching…
            </div>
            <div v-else-if="folderSearchError" class="p-2 text-xs text-red-400">
              {{ folderSearchError }}
            </div>
            <template v-else-if="folderSearchResults">
              <div
                v-if="folderSearchResults.folders.length === 0 && folderSearchResults.requests.length === 0"
                class="p-2 text-xs text-gray-500"
                style="color: #9ca3af"
              >
                No matches for “{{ folderSearchQuery }}”.
              </div>
              <template v-else>
                <div
                  v-if="folderSearchResults.folders.length > 0"
                  class="mb-1 px-1 text-[9px] font-semibold uppercase tracking-wider text-gray-500"
                >Folders · {{ folderSearchResults.folders.length }}</div>
                <div
                  v-for="hit in folderSearchResults.folders"
                  :key="'f:' + hit.id"
                  role="button"
                  tabindex="0"
                  class="mb-0.5 flex cursor-pointer items-start gap-1 rounded p-1.5 text-xs transition-colors hover:bg-gray-800"
                  :title="hit.path && hit.path.length ? hit.path.join(' / ') + ' / ' + hit.name : hit.name"
                  @click="onFolderSearchHit(hit)"
                  @keydown.enter.prevent="onFolderSearchHit(hit)"
                >
                  <span class="shrink-0 text-gray-500">📁</span>
                  <div class="min-w-0 flex-1">
                    <div class="truncate text-gray-200">
                      <HighlightText :text="hit.name" :query="folderSearchQuery" />
                    </div>
                    <div v-if="hit.path && hit.path.length" class="truncate text-[10px] text-gray-500">
                      {{ hit.path.join(' / ') }}
                    </div>
                  </div>
                </div>

                <div
                  v-if="folderSearchResults.requests.length > 0"
                  class="mb-1 mt-2 px-1 text-[9px] font-semibold uppercase tracking-wider text-gray-500"
                >Requests · {{ folderSearchResults.requests.length }}</div>
                <div
                  v-for="hit in folderSearchResults.requests"
                  :key="'r:' + hit.id"
                  role="button"
                  tabindex="0"
                  class="mb-0.5 flex cursor-pointer items-start gap-1 rounded p-1.5 text-xs transition-colors hover:bg-gray-800"
                  :title="hit.url"
                  @click="onRequestSearchHit(hit)"
                  @keydown.enter.prevent="onRequestSearchHit(hit)"
                >
                  <span class="shrink-0 font-mono text-[9px] text-gray-500 pt-0.5">{{ hit.method }}</span>
                  <div class="min-w-0 flex-1">
                    <div class="truncate text-gray-200">
                      <HighlightText :text="hit.name" :query="folderSearchQuery" />
                    </div>
                    <div class="truncate text-[10px] text-gray-500">
                      <HighlightText :text="hit.url" :query="folderSearchQuery" />
                    </div>
                    <div v-if="hit.path && hit.path.length" class="truncate text-[10px] text-gray-500">
                      {{ hit.path.join(' / ') }}
                    </div>
                  </div>
                </div>

                <div v-if="folderSearchResults.truncated" class="mt-2 px-1 text-[10px] text-gray-500">
                  Showing first {{ folderSearchResults.folders.length + folderSearchResults.requests.length }} matches — refine the query for more precise results.
                </div>
              </template>
            </template>
          </div>

          <div v-else class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2" @scroll.passive="closeWorkspaceMenu">
            <div v-if="loading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading folders…</div>
            <div v-else-if="rootFolderList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
              No root folders yet.
            </div>
            <div v-for="ws in rootFolderList" v-else :key="ws.id" class="mb-1">
              <div
                role="button"
                tabindex="0"
                :data-root-folder-row="ws.id"
                class="flex items-center gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800 group cursor-pointer"
                :class="{
                  'bg-gray-800/80': activeRootFolderId === ws.id,
                  'pmj-reveal-flash': flashRootFolderId === ws.id
                }"
                @click="onRootFolderRowClick(ws)"
                @keydown.enter.prevent="onRootFolderRowClick(ws)"
              >
                <span class="text-gray-500 shrink-0">📁</span>
                <span class="min-w-0 flex-1 truncate pr-1">{{ ws.name }}</span>
                <button
                  type="button"
                  data-ws-menu
                  class="shrink-0 rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-white opacity-70 group-hover:opacity-100"
                  style="min-width: 28px; line-height: 1"
                  :aria-expanded="menuOpenForId === ws.id"
                  aria-haspopup="menu"
                  :aria-label="'Folder actions ' + (ws.name || '')"
                  @click.stop="toggleWorkspaceMenu(ws, $event)"
                >
                  ⋮
                </button>
              </div>
              <!-- Giống FolderTreeNode: v-show trên con trực tiếp của Transition để enter/leave chạy mượt -->
              <Transition name="folder-tree-slide">
                <div
                  v-show="isRootTreeExpanded(ws.id)"
                  class="mt-1 border-l border-gray-700/90 pl-2.5 ml-1.5 text-[11px] text-gray-300"
                >
                  <FolderTreeNode
                    :ref="(el) => setFolderTreeRef(ws.id, el)"
                    :folder-id="ws.id"
                    :depth="0"
                    @open-saved-request="(id) => emit('open-saved-request', id)"
                    @console="(msg) => emit('console', msg)"
                  />
                </div>
              </Transition>
            </div>
          </div>
          <div class="flex shrink-0 flex-wrap items-center justify-between gap-1 border-t border-gray-800 bg-[#1c1c1c] px-2 py-1">
            <div class="flex flex-wrap items-center gap-1">
              <button
                type="button"
                class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-gray-200 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
                aria-label="Import collection"
                title="Import Postman / OpenAPI / Insomnia collection into a new root folder"
                @click="openImportCollectionModal"
              >
                Import
              </button>
              <button
                type="button"
                class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-gray-200 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
                aria-label="Import from cURL"
                title="Import as ad-hoc request (not tied to a saved item)"
                @click="openCurlImportModal"
              >
                cURL
              </button>
            </div>
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-orange-500 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
              style="color: #f97316"
              aria-label="Add root folder"
              title="Add root folder"
              @click="openCreateModal"
            >
              Add folder
            </button>
          </div>
        </div>
      </template>

      <template v-else-if="sidebarTab === 'env'">
        <div class="flex min-h-0 flex-1 flex-col">
          <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-gray-800 px-3 py-2">
            <span class="min-w-0 text-xs text-gray-500" style="color: #9ca3af">Environments</span>
            <button
              type="button"
              class="shrink-0 rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-orange-500 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
              style="color: #f97316"
              @click="openCreateEnvModal"
            >
              Add
            </button>
          </div>
          <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2" @scroll.passive="closeEnvMenu">
            <div v-if="envLoading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading…</div>
            <div v-else-if="envList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
              No environments yet. Create one to store variables.
            </div>
            <div v-for="item in envList" v-else :key="item.id" class="mb-1">
              <div
                role="button"
                tabindex="0"
                class="group flex cursor-pointer items-center gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800"
                :class="{ 'bg-gray-800/80 ring-1 ring-orange-500/30': item.is_active }"
                @click="emit('open-environment', item.id)"
                @keydown.enter.prevent="emit('open-environment', item.id)"
              >
                <span class="shrink-0 text-gray-500">🌿</span>
                <span class="min-w-0 flex-1 truncate pr-1">{{ item.name }}</span>
                <span
                  v-if="item.is_active"
                  class="shrink-0 rounded bg-emerald-900/80 px-1 py-0.5 text-[9px] font-bold uppercase text-emerald-200"
                  >Active</span>
                <button
                  type="button"
                  data-env-menu
                  class="shrink-0 rounded p-1.5 text-gray-400 opacity-70 hover:bg-gray-700 hover:text-white group-hover:opacity-100"
                  style="min-width: 28px; line-height: 1"
                  :aria-expanded="menuOpenForEnvId === item.id"
                  aria-haspopup="menu"
                  :aria-label="'Environment actions ' + (item.name || '')"
                  @click.stop="toggleEnvMenu(item, $event)"
                >
                  ⋮
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template v-else-if="sidebarTab === 'history'">
        <div class="flex shrink-0 items-center justify-between gap-2 border-b border-gray-800 px-3 py-2">
          <span class="min-w-0 text-xs text-gray-500" style="color: #9ca3af">Recent requests</span>
          <div class="flex items-center gap-1">
            <button
              type="button"
              class="shrink-0 rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold transition-colors hover:border-orange-500/50 hover:bg-gray-800"
              :class="historyFilterPanelOpen || historyFilterActive ? 'text-orange-300' : 'text-gray-300'"
              aria-label="Toggle history filters"
              title="Method / status / URL / date filters"
              @click="historyFilterPanelOpen = !historyFilterPanelOpen"
            >
              Filters<span v-if="historyFilterActive" class="ml-1 inline-block h-1.5 w-1.5 rounded-full bg-orange-400"></span>
            </button>
            <button
              type="button"
              class="shrink-0 rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-orange-500 transition-colors hover:border-orange-500/50 hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-50"
              style="color: #f97316"
              aria-label="Refresh history"
              title="Refresh history"
              :disabled="historyLoading"
              @click="loadHistory"
            >
              Refresh
            </button>
          </div>
        </div>

        <div class="shrink-0 border-b border-gray-800 px-3 py-2">
          <div class="relative">
            <input
              v-model="historyFilterText"
              type="search"
              placeholder="Filter by URL / method / status…"
              aria-label="Filter history"
              class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1 pr-7 text-xs text-gray-200 placeholder:text-gray-500 focus:border-orange-500/60 focus:outline-none"
            />
            <button
              v-if="historyFilterText"
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 rounded px-1 text-xs text-gray-500 hover:text-gray-200"
              aria-label="Clear filter text"
              title="Clear"
              @click="historyFilterText = ''"
            >
              ✕
            </button>
          </div>
        </div>

        <div v-if="historyFilterPanelOpen" class="shrink-0 border-b border-gray-800 bg-[#1a1a1a] px-3 py-2 space-y-2">
          <div>
            <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-500">Method</div>
            <div class="flex flex-wrap gap-1">
              <button
                v-for="m in historyMethodChoices"
                :key="m"
                type="button"
                class="rounded border px-1.5 py-0.5 text-[10px] font-mono transition-colors"
                :class="historyMethodFilter.has(m)
                  ? 'border-orange-500/60 bg-orange-500/20 text-orange-200'
                  : 'border-gray-700 bg-[#2a2a2a] text-gray-400 hover:border-gray-500'"
                @click="toggleHistoryMethod(m)"
              >{{ m }}</button>
            </div>
          </div>
          <div>
            <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-500">Status</div>
            <div class="flex flex-wrap gap-1">
              <button
                v-for="g in historyStatusChoices"
                :key="g.id"
                type="button"
                class="rounded border px-1.5 py-0.5 text-[10px] font-mono transition-colors"
                :class="historyStatusFilter.has(g.id)
                  ? 'border-orange-500/60 bg-orange-500/20 text-orange-200'
                  : 'border-gray-700 bg-[#2a2a2a] text-gray-400 hover:border-gray-500'"
                @click="toggleHistoryStatus(g.id)"
              >{{ g.label }}</button>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex-1 min-w-[120px]">
              <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-500">From</div>
              <input
                v-model="historyDateFrom"
                type="date"
                class="w-full rounded border border-gray-700 bg-[#2a2a2a] px-1.5 py-0.5 text-[11px] text-gray-200 focus:border-orange-500/60 focus:outline-none"
              />
            </div>
            <div class="flex-1 min-w-[120px]">
              <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-gray-500">To</div>
              <input
                v-model="historyDateTo"
                type="date"
                class="w-full rounded border border-gray-700 bg-[#2a2a2a] px-1.5 py-0.5 text-[11px] text-gray-200 focus:border-orange-500/60 focus:outline-none"
              />
            </div>
          </div>
          <div class="flex items-center justify-between pt-1">
            <span class="text-[10px] text-gray-500">
              {{ historyFilterActive ? `${filteredHistoryList.length} / ${historyList.length} match` : `${historyList.length} total` }}
            </span>
            <button
              v-if="historyFilterActive"
              type="button"
              class="rounded px-2 py-0.5 text-[10px] text-gray-400 hover:bg-gray-800 hover:text-white"
              @click="clearHistoryFilters"
            >
              Clear all
            </button>
          </div>
        </div>

        <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
          <div v-if="historyLoading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading history…</div>
          <div v-else-if="historyList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
            No request history yet. Send a request to see it here.
          </div>
          <div v-else-if="filteredHistoryList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
            No history entries match the current filter.
          </div>
          <div
            v-for="h in filteredHistoryList"
            :key="h.id"
            role="button"
            tabindex="0"
            class="mb-2 cursor-pointer rounded border border-gray-700/90 bg-[#1a1a1a] p-2 text-left transition-colors hover:border-orange-500/40 hover:bg-[#222]"
            title="View request & response snapshot"
            @click="openHistoryDetail(h)"
            @keydown.enter.prevent="openHistoryDetail(h)"
          >
            <div class="flex flex-wrap items-center gap-1.5">
              <span
                class="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white"
                style="background: #374151"
                >{{ h.method }}</span>
              <span
                class="rounded px-1.5 py-0.5 font-mono text-[10px] font-semibold"
                :class="statusBadgeClass(h.status_code)"
                >{{ h.status_code }}</span>
              <span v-if="h.duration_ms != null" class="text-[10px] text-gray-500">{{ h.duration_ms }} ms</span>
            </div>
            <div class="mt-1 truncate text-xs text-gray-300" :title="h.url">
              <HighlightText :text="truncateMiddle(h.url, 52)" :query="historyFilterText" />
            </div>
            <div class="mt-1 text-[10px] text-gray-500">{{ formatHistoryTime(h.created_at) }}</div>
          </div>
        </div>
      </template>
    </aside>

    <Teleport to="#app">
      <div
        v-if="menuOpenForEnvId !== null && menuTargetEnv"
        data-env-menu
        class="min-w-[200px] rounded-md border border-gray-600 bg-[#2a2a2a] py-1 shadow-xl"
        :style="envMenuStyle"
        role="menu"
      >
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="openEditEnvModal(menuTargetEnv)"
        >
          Edit
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-gray-700 hover:text-red-300"
          @click="openDeleteEnvModal(menuTargetEnv)"
        >
          Delete
        </button>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="menuOpenForId !== null && menuTargetWs"
        data-ws-menu
        class="min-w-[220px] rounded-md border border-gray-600 bg-[#2a2a2a] py-1 shadow-xl"
        :style="menuStyle"
        role="menu"
      >
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-orange-300 hover:bg-gray-700"
          @click="onNewFolderFromMenu(menuTargetWs)"
        >
          New Folder
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-orange-300 hover:bg-gray-700"
          @click="onNewRootRequestFromMenu(menuTargetWs)"
        >
          New Request
        </button>
        <div class="my-1 border-t border-gray-600" role="separator" />
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onEditFromMenu(menuTargetWs)"
        >
          Edit folder
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-gray-700 hover:text-red-300"
          @click="onDeleteFromMenu(menuTargetWs)"
        >
          Delete folder
        </button>
      </div>
    </Teleport>

    <!-- Teleport to body + position:fixed on body breaks stacking in some WebView2 builds; keep overlays under #app -->
    <Teleport to="#app">
  <div v-if="modalState.show" class="fixed inset-0 z-40 bg-black/50 flex items-center justify-center px-4">
    <div class="w-full max-w-md bg-[#1f1f1f] border border-gray-700 rounded-lg shadow-lg">
      <div class="px-4 py-3 border-b border-gray-700">
        <h3 class="text-sm font-semibold text-white">{{ modalState.title }}</h3>
      </div>

      <div class="p-4">
        <template v-if="modalState.mode === 'delete'">
          <p class="text-sm text-gray-300">
            Delete folder
            <span class="font-semibold text-white">"{{ modalState.target?.name }}"</span>
            and everything inside? This cannot be undone — all subfolders and saved requests in this tree will be
            removed.
          </p>
        </template>
        <template v-else>
          <label class="block text-xs text-gray-400 mb-1">Name</label>
          <input
            v-model="formName"
            type="text"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Folder name"
          />
          <label class="block text-xs text-gray-400 mt-3 mb-1">Description</label>
          <textarea
            v-model="formDescription"
            rows="3"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Description (optional)"
          />
        </template>
      </div>

      <div class="px-4 py-3 border-t border-gray-700 flex justify-end gap-2">
        <button
          type="button"
          @click="() => closeModal()"
          :disabled="submitting"
          class="px-3 py-1.5 rounded bg-gray-700 hover:bg-gray-600 text-xs text-white disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          @click="submitModal"
          :disabled="submitting"
          class="px-3 py-1.5 rounded text-xs text-white disabled:opacity-50"
          :class="modalState.mode === 'delete' ? 'bg-red-600 hover:bg-red-700' : 'bg-orange-600 hover:bg-orange-700'"
        >
          {{ submitting ? 'Working…' : modalState.submitLabel }}
        </button>
      </div>
    </div>
  </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="curlModalOpen"
        class="fixed inset-0 z-[58] flex items-center justify-center bg-black/50 px-4"
        role="dialog"
        aria-modal="true"
        aria-labelledby="sidebar-curl-import-title"
      >
        <div class="flex max-h-[85vh] w-full max-w-2xl flex-col rounded-lg border border-gray-700 bg-[#1f1f1f] shadow-lg">
          <div class="flex items-start justify-between gap-2 border-b border-gray-700 px-4 py-3">
            <div class="min-w-0 flex-1 pr-2">
              <h3 id="sidebar-curl-import-title" class="text-sm font-semibold text-white">Import from cURL</h3>
              <p class="mt-1 text-[11px] leading-relaxed text-gray-500">
                Creates an <span class="text-gray-400">ad-hoc</span> request in the editor. Use <span class="text-gray-400">Save</span> there to store it in a folder.
              </p>
            </div>
            <button
              type="button"
              class="shrink-0 rounded p-1.5 text-lg leading-none text-gray-500 hover:bg-gray-800 hover:text-gray-200"
              aria-label="Close"
              @click="closeCurlModal"
            >
              ×
            </button>
          </div>
          <div class="min-h-0 flex-1 overflow-hidden p-4">
            <textarea
              v-model="curlText"
              class="app-scrollbar h-80 w-full resize-y rounded border border-gray-700 bg-gray-900 px-3 py-2 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
              spellcheck="false"
              placeholder="curl https://httpbin.org/get"
            />
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
            <button
              type="button"
              class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600"
              @click="closeCurlModal"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700"
              @click="applyCurlImport"
            >
              Import
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <HistoryDetailModal
      :open="historyDetailOpen"
      :loading="historyDetailLoading"
      :item="historyDetailItem"
      @close="closeHistoryDetail"
      @delete="onDeleteHistoryFromModal"
    />

    <Teleport to="#app">
      <div
        v-if="envModalState.show"
        class="fixed inset-0 z-[42] flex items-center justify-center bg-black/50 px-4"
      >
        <div class="w-full max-w-md rounded-lg border border-gray-700 bg-[#1f1f1f] shadow-lg">
          <div class="border-b border-gray-700 px-4 py-3">
            <h3 class="text-sm font-semibold text-white">{{ envModalState.title }}</h3>
          </div>
          <div class="p-4">
            <template v-if="envModalState.mode === 'delete'">
              <p class="text-sm text-gray-300">
                Delete
                <span class="font-semibold text-white">"{{ envModalState.target?.name }}"</span>
                and all variables? This cannot be undone.
              </p>
            </template>
            <template v-else>
              <label class="mb-1 block text-xs text-gray-400">Name</label>
              <input
                v-model="envFormName"
                type="text"
                class="w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
                placeholder="Environment name"
              />
              <label class="mt-3 mb-1 block text-xs text-gray-400">Description</label>
              <textarea
                v-model="envFormDescription"
                rows="2"
                class="w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
                placeholder="Optional"
              />
            </template>
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
            <button
              type="button"
              class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
              :disabled="envSubmitting"
              @click="() => closeEnvModal()"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded px-3 py-1.5 text-xs text-white disabled:opacity-50"
              :class="
                envModalState.mode === 'delete' ? 'bg-red-600 hover:bg-red-700' : 'bg-orange-600 hover:bg-orange-700'
              "
              :disabled="envSubmitting"
              @click="submitEnvModal"
            >
              {{ envSubmitting ? 'Working…' : envModalState.submitLabel }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <ImportCollectionModal
      :open="importModalOpen"
      @close="closeImportCollectionModal"
      @imported="onCollectionImported"
      @console="(msg) => emit('console', msg)"
    />

    <div v-if="toast.show" class="pointer-events-none fixed bottom-4 right-4 z-50">
      <div
        class="pointer-events-auto px-4 py-2 rounded shadow text-sm border"
        :class="{
          'bg-emerald-900/90 border-emerald-500 text-emerald-100': toast.type === 'success',
          'bg-red-900/90 border-red-500 text-red-100': toast.type === 'error',
          'bg-amber-900/90 border-amber-500 text-amber-100': toast.type === 'warning',
          'bg-slate-800/90 border-slate-500 text-slate-100': toast.type === 'info'
        }"
      >
        {{ toast.message }}
      </div>
    </div>
  </div>
</template>

<style>
/*
 * Brief pulse used to point the user to a folder/request row that was just
 * revealed via search. Kept global (no scoped) so FolderTreeNode rows can
 * share the same class name.
 */
@keyframes pmj-reveal-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(249, 115, 22, 0.55); background-color: rgba(249, 115, 22, 0.22); }
  60%  { box-shadow: 0 0 0 4px rgba(249, 115, 22, 0);  background-color: rgba(249, 115, 22, 0.14); }
  100% { box-shadow: 0 0 0 0 rgba(249, 115, 22, 0);    background-color: transparent; }
}
.pmj-reveal-flash {
  animation: pmj-reveal-pulse 1.4s ease-out 1;
  border-radius: 0.25rem;
}
</style>