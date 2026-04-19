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
import HistoryDetailModal from './HistoryDetailModal.vue'
import FolderTreeNode from './FolderTreeNode.vue'

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
          <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2" @scroll.passive="closeWorkspaceMenu">
            <div v-if="loading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading folders…</div>
            <div v-else-if="rootFolderList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
              No root folders yet.
            </div>
            <div v-for="ws in rootFolderList" v-else :key="ws.id" class="mb-1">
              <div
                role="button"
                tabindex="0"
                class="flex items-center gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800 group cursor-pointer"
                :class="{ 'bg-gray-800/80': activeRootFolderId === ws.id }"
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
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-gray-200 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
              aria-label="Import from cURL"
              title="Import as ad-hoc request (not tied to a saved item)"
              @click="openCurlImportModal"
            >
              Import cURL
            </button>
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
        <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2">
          <div v-if="historyLoading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading history…</div>
          <div v-else-if="historyList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
            No request history yet. Send a request to see it here.
          </div>
          <div
            v-for="h in historyList"
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
            <div class="mt-1 truncate text-xs text-gray-300" :title="h.url">{{ truncateMiddle(h.url, 52) }}</div>
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