<script setup>
import { ref, watch, computed, onMounted, onUnmounted, inject, nextTick } from 'vue'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import * as SavedRequestAPI from '../../wailsjs/wailsjs/go/delivery/SavedRequestHandler'

const props = defineProps({
  folderId: { type: String, required: true },
  depth: { type: Number, default: 0 }
})

const emit = defineEmits(['open-saved-request', 'console', 'saved-request', 'tree-changed'])

const folderTreeReload = inject('folderTreeReload', null)
/**
 * Reveal state provided by Sidebar: after a search result click we walk this
 * chain (root → target) and each node on the path auto-expands the next hop.
 * The node whose `folderId === chain[chain.length - 2]` also scrolls/flashes
 * the target child row; if the target is a folder AND `requestId` is set, the
 * target's own node flashes that request row once requests have loaded.
 */
const folderTreeReveal = inject('folderTreeReveal', null)

/** `folderId` currently flashing as a direct child of this node, or null. */
const flashChildFolderId = ref(null)
/** `requestId` currently flashing inside this node, or null. */
const flashRequestId = ref(null)
let flashChildTimer = null
let flashRequestTimer = null

function triggerChildFolderFlash(fid) {
  if (flashChildTimer) clearTimeout(flashChildTimer)
  flashChildFolderId.value = fid
  nextTick(() => {
    const el = rootEl.value?.querySelector?.(`[data-subfolder-row="${fid}"]`)
    if (el?.scrollIntoView) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  })
  flashChildTimer = setTimeout(() => {
    flashChildFolderId.value = null
    flashChildTimer = null
  }, 1500)
}

function triggerRequestFlash(rid) {
  if (flashRequestTimer) clearTimeout(flashRequestTimer)
  flashRequestId.value = rid
  nextTick(() => {
    const el = rootEl.value?.querySelector?.(`[data-request-row="${rid}"]`)
    if (el?.scrollIntoView) el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  })
  flashRequestTimer = setTimeout(() => {
    flashRequestId.value = null
    flashRequestTimer = null
  }, 1500)
}

const rootEl = ref(null)

/** Apply the current reveal state against this node's children/requests. Safe
 * to call repeatedly; no-op when nothing matches. */
function applyReveal() {
  const st = folderTreeReveal?.value
  if (!st || !Array.isArray(st.chain) || st.chain.length === 0) return
  const idx = st.chain.indexOf(props.folderId)
  if (idx < 0) return
  const nextHop = idx < st.chain.length - 1 ? st.chain[idx + 1] : null

  if (nextHop) {
    // Expand the chain's next folder so its own FolderTreeNode instance mounts
    // and can continue the reveal recursively.
    if (childFolders.value.some((f) => f.id === nextHop)) {
      if (!expandedFolderIds.value[nextHop]) {
        const cur = { ...expandedFolderIds.value }
        cur[nextHop] = true
        expandedFolderIds.value = cur
      }
      // If this hop IS the target, flash the child row we just expanded.
      if (idx + 1 === st.chain.length - 1 && st.targetId === nextHop) {
        triggerChildFolderFlash(nextHop)
      }
    }
  }

  if (props.folderId === st.targetId && st.requestId) {
    if (requests.value.some((r) => r.id === st.requestId)) {
      triggerRequestFlash(st.requestId)
    }
  }
}

function notifyFolderTreeReload(folderId) {
  const r = folderTreeReload
  if (!r?.value || folderId == null || folderId === '') return
  r.value = {
    targetId: folderId,
    tick: r.value.tick + 1
  }
}

const childFolders = ref([])
const requests = ref([])
const loading = ref(false)

/**
 * Shared across all FolderTreeNode instances and owned by Sidebar, so the
 * expand/collapse state survives mount/unmount (e.g. when a root is collapsed
 * then re-expanded) AND can be persisted to localStorage in one place.
 * Fallback to a local ref when no provider is found (e.g. tests/storybook).
 */
const expandedFolderIds = inject(
  'expandedFolderIds',
  ref(/** @type {Record<string, boolean>} */ ({}))
)

function isChildExpanded(folderId) {
  return !!expandedFolderIds.value[folderId]
}

function toggleChildExpand(folderId) {
  const cur = { ...expandedFolderIds.value }
  if (cur[folderId]) delete cur[folderId]
  else cur[folderId] = true
  expandedFolderIds.value = cur
}

/* ─────────────────────────── Inline rename ──────────────────────────────── */

/** Currently inline-renaming folder (child of this node), or null. */
const inlineRenameFolder = ref({ id: /** @type {string|null} */(null), name: '' })
const inlineRenameRequest = ref({ id: /** @type {string|null} */(null), name: '' })
/** Guard against re-entering submit via both Enter and blur. */
const inlineSubmitting = ref(false)

/** Delayed single-click for request rows only (double-click = inline rename). */
const CLICK_DOUBLE_WINDOW_MS = 220
let requestRowSingleClickTimer = null

function onFolderRowClick(f) {
  if (inlineRenameFolder.value.id === f.id) return
  toggleChildExpand(f.id)
}

function onRequestRowClick(r) {
  if (inlineRenameRequest.value.id === r.id) return
  if (requestRowSingleClickTimer) return
  requestRowSingleClickTimer = setTimeout(() => {
    requestRowSingleClickTimer = null
    openRequest(r.id)
  }, CLICK_DOUBLE_WINDOW_MS)
}

function onRequestRowDblClick(r) {
  if (requestRowSingleClickTimer) {
    clearTimeout(requestRowSingleClickTimer)
    requestRowSingleClickTimer = null
  }
  startInlineRequestRename(r)
}

function startInlineFolderRename(f) {
  if (!f?.id) return
  closeFolderRowMenu()
  closeRequestRowMenu()
  cancelInlineRequestRename()
  inlineRenameFolder.value = { id: f.id, name: f.name || '' }
  nextTick(() => {
    const el = rootEl.value?.querySelector?.(`[data-inline-folder-input="${f.id}"]`)
    if (el) {
      el.focus()
      try { el.select() } catch { /* ignore */ }
    }
  })
}

function cancelInlineFolderRename() {
  inlineRenameFolder.value = { id: null, name: '' }
}

async function submitInlineFolderRename() {
  if (inlineSubmitting.value) return
  const id = inlineRenameFolder.value.id
  const name = (inlineRenameFolder.value.name || '').trim()
  if (!id) return
  const target = childFolders.value.find((f) => f.id === id)
  if (!target) { cancelInlineFolderRename(); return }
  if (!name || name === target.name) { cancelInlineFolderRename(); return }
  inlineSubmitting.value = true
  try {
    await FolderAPI.UpdateFolder(id, name, target.description || '')
    cancelInlineFolderRename()
    await load()
  } catch (e) {
    const msg = e?.message || String(e)
    if (msg.includes('FOL_301') || msg.includes('already exists')) {
      emit('console', '[Folders] That folder name is already in use at this level.')
    } else {
      emit('console', `[Folders] ${msg}`)
    }
    cancelInlineFolderRename()
  } finally {
    inlineSubmitting.value = false
  }
}

async function startInlineRequestRename(r) {
  if (!r?.id) return
  closeFolderRowMenu()
  closeRequestRowMenu()
  cancelInlineFolderRename()
  inlineRenameRequest.value = { id: r.id, name: r.name || '' }
  await nextTick()
  const el = rootEl.value?.querySelector?.(`[data-inline-request-input="${r.id}"]`)
  if (el) {
    el.focus()
    try { el.select() } catch { /* ignore */ }
  }
}

function cancelInlineRequestRename() {
  inlineRenameRequest.value = { id: null, name: '' }
}

async function submitInlineRequestRename() {
  if (inlineSubmitting.value) return
  const id = inlineRenameRequest.value.id
  const name = (inlineRenameRequest.value.name || '').trim()
  if (!id) return
  const local = requests.value.find((r) => r.id === id)
  if (!local) { cancelInlineRequestRename(); return }
  if (!name || name === local.name) { cancelInlineRequestRename(); return }
  inlineSubmitting.value = true
  try {
    const full = await SavedRequestAPI.Get(id)
    if (!full) { cancelInlineRequestRename(); return }
    full.name = name
    await SavedRequestAPI.Update(full)
    cancelInlineRequestRename()
    await load()
    emit('saved-request')
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
    cancelInlineRequestRename()
  } finally {
    inlineSubmitting.value = false
  }
}

const PMJ_FOLDER = 'pmj/folder'
const PMJ_REQUEST = 'pmj/request'

/** Top ~38% of row = same level as this folder (parent = current node's folderId). Bottom = inside folder. */
const DND_SIBLING_RATIO = 0.38

/** While dragging, highlights drop target + mode for visual feedback */
const dndHover = ref(/** @type {{ folderId: string, mode: 'sibling' | 'into' } | null} */ (null))

/** Drop gaps between sibling folders: `before-<id>` or `append` */
const folderReorderGapHover = ref(/** @type {string|null} */ (null))

function dndModeFromPointer(el, e) {
  const r = el.getBoundingClientRect()
  const ratio = (e.clientY - r.top) / Math.max(r.height, 1)
  return ratio < DND_SIBLING_RATIO ? 'sibling' : 'into'
}

function clearDndHover() {
  dndHover.value = null
  folderReorderGapHover.value = null
}

function onFolderReorderGapDragOver(insertBeforeId, e) {
  e.preventDefault()
  try {
    e.dataTransfer.dropEffect = 'move'
  } catch {
    /* ignore */
  }
  folderReorderGapHover.value = insertBeforeId == null ? 'append' : `before-${insertBeforeId}`
}

function onFolderReorderGapDragLeave(e) {
  const rel = e.relatedTarget
  if (rel && e.currentTarget.contains(rel)) return
  folderReorderGapHover.value = null
}

async function onFolderReorderGapDrop(e, insertBeforeId) {
  e.preventDefault()
  e.stopPropagation()
  folderReorderGapHover.value = null
  const fromF = e.dataTransfer.getData(PMJ_FOLDER)
  if (!fromF) return
  try {
    const ib = insertBeforeId == null ? '' : insertBeforeId
    await FolderAPI.ReorderFolder(fromF, props.folderId, ib)
    emit('tree-changed')
    await load()
  } catch (err) {
    emit('console', `[Folders] ${err?.message || err}`)
  }
}

function onDragStartFolder(f, e) {
  if (!f?.id) return
  closeFolderRowMenu()
  closeRequestRowMenu()
  e.dataTransfer.setData(PMJ_FOLDER, f.id)
  e.dataTransfer.effectAllowed = 'move'
}

function onDragStartRequest(r, e) {
  if (!r?.id) return
  closeFolderRowMenu()
  closeRequestRowMenu()
  e.dataTransfer.setData(PMJ_REQUEST, r.id)
  e.dataTransfer.effectAllowed = 'move'
}

function onFolderDropRowDragOver(f, e) {
  e.preventDefault()
  try {
    e.dataTransfer.dropEffect = 'move'
  } catch {
    /* ignore */
  }
  const mode = dndModeFromPointer(e.currentTarget, e)
  dndHover.value = { folderId: f.id, mode }
}

function onFolderDropRowDragLeave(f, e) {
  const wrap = e.currentTarget
  const rel = e.relatedTarget
  if (rel && wrap.contains(rel)) return
  if (dndHover.value?.folderId === f.id) clearDndHover()
}

async function onFolderDropRowDrop(f, e) {
  e.preventDefault()
  const mode = dndHover.value?.folderId === f.id
    ? dndHover.value.mode
    : dndModeFromPointer(e.currentTarget, e)
  clearDndHover()

  const fromF = e.dataTransfer.getData(PMJ_FOLDER)
  const fromR = e.dataTransfer.getData(PMJ_REQUEST)
  if (!fromF && !fromR) return

  try {
    if (fromF) {
      if (mode === 'sibling') {
        if (fromF === f.id) return
        await FolderAPI.MoveFolder(fromF, props.folderId)
      } else {
        if (fromF === f.id) return
        await FolderAPI.MoveFolder(fromF, f.id)
      }
    } else if (fromR) {
      if (mode === 'sibling') {
        await SavedRequestAPI.MoveRequest(fromR, props.folderId)
      } else {
        await SavedRequestAPI.MoveRequest(fromR, f.id)
      }
    }
    emit('tree-changed')
    await load()
  } catch (err) {
    emit('console', `[Folders] ${err?.message || err}`)
  }
}

function onFolderDropRowDragEnd() {
  clearDndHover()
}

const folderModal = ref({
  open: false,
  mode: 'create',
  parentId: null,
  editId: '',
  name: '',
  description: ''
})

const requestModal = ref({
  open: false,
  parentFolderId: null,
  name: 'New request'
})

/** ⋮ menu cho từng subfolder (cùng format với root row trong Sidebar) */
const folderMenuOpenId = ref(null)
const folderMenuStyle = ref({
  position: 'fixed',
  top: '0px',
  left: '0px',
  zIndex: 50
})

const menuTargetFolder = computed(() => {
  const id = folderMenuOpenId.value
  if (!id) return null
  return childFolders.value.find((x) => x.id === id) ?? null
})

function closeFolderRowMenu() {
  folderMenuOpenId.value = null
}

function toggleFolderRowMenu(f, event) {
  event?.stopPropagation()
  if (!f) return
  closeRequestRowMenu()
  if (folderMenuOpenId.value === f.id) {
    closeFolderRowMenu()
    return
  }
  folderMenuOpenId.value = f.id
  const el = event?.currentTarget
  if (el && typeof el.getBoundingClientRect === 'function') {
    const r = el.getBoundingClientRect()
    const width = 220
    let left = r.right - width
    if (left < 8) left = 8
    if (left + width > window.innerWidth - 8) left = window.innerWidth - width - 8
    const top = r.bottom + 4
    folderMenuStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 50
    }
  }
}

function onFolderMenuNewFolder(f) {
  closeFolderRowMenu()
  openCreateSubfolder(f.id)
}

function onFolderMenuNewRequest(f) {
  closeFolderRowMenu()
  openCreateRequest(f.id)
}

function onFolderMenuRename(f) {
  closeFolderRowMenu()
  startInlineFolderRename(f)
}

async function onFolderMenuDuplicate(f) {
  closeFolderRowMenu()
  if (!f?.id) return
  try {
    await FolderAPI.DuplicateFolder(f.id)
    await load()
    emit('tree-changed')
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  }
}

function onFolderMenuEdit(f) {
  closeFolderRowMenu()
  openEditFolder(f)
}

function onFolderMenuDelete(f) {
  closeFolderRowMenu()
  openDeleteFolderModal(f)
}

const requestMenuOpenId = ref(null)
const requestMenuStyle = ref({
  position: 'fixed',
  top: '0px',
  left: '0px',
  zIndex: 50
})

const menuTargetRequest = computed(() => {
  const id = requestMenuOpenId.value
  if (!id) return null
  return requests.value.find((x) => x.id === id) ?? null
})

const renameRequestModal = ref({
  open: false,
  id: '',
  name: ''
})

function closeRequestRowMenu() {
  requestMenuOpenId.value = null
}

function toggleRequestRowMenu(r, event) {
  event?.stopPropagation()
  if (!r) return
  closeFolderRowMenu()
  if (requestMenuOpenId.value === r.id) {
    closeRequestRowMenu()
    return
  }
  requestMenuOpenId.value = r.id
  const el = event?.currentTarget
  if (el && typeof el.getBoundingClientRect === 'function') {
    const rect = el.getBoundingClientRect()
    const width = 200
    let left = rect.right - width
    if (left < 8) left = 8
    if (left + width > window.innerWidth - 8) left = window.innerWidth - width - 8
    const top = rect.bottom + 4
    requestMenuStyle.value = {
      position: 'fixed',
      top: `${top}px`,
      left: `${left}px`,
      zIndex: 50
    }
  }
}

function onRequestMenuRename(r) {
  closeRequestRowMenu()
  startInlineRequestRename(r)
}

async function onRequestMenuDuplicate(r) {
  closeRequestRowMenu()
  if (!r?.id) return
  try {
    const created = await SavedRequestAPI.Duplicate(r.id)
    await load()
    emit('saved-request')
    if (created?.id) emit('open-saved-request', created.id)
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
  }
}

async function openRenameRequest(r) {
  closeRequestRowMenu()
  try {
    const full = await SavedRequestAPI.Get(r.id)
    renameRequestModal.value = {
      open: true,
      id: r.id,
      name: (full?.name || r.name || '').trim()
    }
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
  }
}

function closeRenameRequestBackdrop() {
  renameRequestModal.value.open = false
}

async function submitRenameRequestModal() {
  const name = (renameRequestModal.value.name || '').trim()
  const id = renameRequestModal.value.id
  if (!name || !id) return
  try {
    const full = await SavedRequestAPI.Get(id)
    full.name = name
    await SavedRequestAPI.Update(full)
    renameRequestModal.value.open = false
    await load()
    emit('open-saved-request', id)
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
  }
}

function onRequestMenuDelete(r) {
  closeRequestRowMenu()
  openDeleteRequestModal(r)
}

function onDocumentPointerDownFolderMenu(e) {
  if (folderMenuOpenId.value == null && requestMenuOpenId.value == null) return
  const t = e.target
  if (t.closest?.('[data-folder-tree-menu]')) return
  if (t.closest?.('[data-request-tree-menu]')) return
  closeFolderRowMenu()
  closeRequestRowMenu()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDownFolderMenu, false)
  document.addEventListener('dragend', clearDndHover, false)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDownFolderMenu, false)
  document.removeEventListener('dragend', clearDndHover, false)
})

/** Dùng mousedown thay vì click để không đóng modal khi kéo chọn text rồi thả ở overlay */
function closeFolderModalBackdrop() {
  folderModal.value.open = false
}

function closeRequestModalBackdrop() {
  requestModal.value.open = false
}

async function load() {
  const fid = props.folderId
  if (!fid) {
    childFolders.value = []
    requests.value = []
    return
  }
  loading.value = true
  try {
    const [folders, reqs] = await Promise.all([
      FolderAPI.ListChildFolders(fid),
      SavedRequestAPI.ListByFolder(fid)
    ])
    childFolders.value = Array.isArray(folders) ? folders : []
    requests.value = Array.isArray(reqs) ? reqs : []
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.folderId,
  () => {
    load()
  },
  { immediate: true }
)

watch(
  () => folderTreeReload?.value?.tick,
  () => {
    const tid = folderTreeReload?.value?.targetId
    if (tid != null && tid !== '' && tid === props.folderId) {
      load()
    }
  }
)

/** Re-run reveal after each load (children list changed) and every time the
 * Sidebar publishes a new command (tick bump). */
watch(
  () => [
    folderTreeReveal?.value?.tick,
    childFolders.value.length,
    requests.value.length
  ],
  () => {
    applyReveal()
  },
  { immediate: true }
)

function openCreateSubfolder(parentId) {
  folderModal.value = {
    open: true,
    mode: 'create',
    parentId,
    editId: '',
    name: '',
    description: ''
  }
}

function openEditFolder(f) {
  folderModal.value = {
    open: true,
    mode: 'edit',
    parentId: null,
    editId: f.id,
    name: f.name,
    description: f.description || ''
  }
}

async function submitFolderModal() {
  const name = (folderModal.value.name || '').trim()
  if (!name) {
    emit('console', '[Folders] Name is required.')
    return
  }
  const wasCreate = folderModal.value.mode === 'create'
  const parentIdForReload = folderModal.value.parentId
  try {
    if (wasCreate) {
      const payload = {
        name,
        description: (folderModal.value.description || '').trim()
      }
      if (folderModal.value.parentId) {
        payload.parent_id = folderModal.value.parentId
      }
      await FolderAPI.CreateFolder(payload)
    } else {
      await FolderAPI.UpdateFolder(folderModal.value.editId, name, (folderModal.value.description || '').trim())
    }
    folderModal.value.open = false
    if (wasCreate) {
      if (parentIdForReload && parentIdForReload === props.folderId) {
        await load()
      } else if (parentIdForReload) {
        notifyFolderTreeReload(parentIdForReload)
      } else {
        await load()
      }
    } else {
      await load()
    }
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  }
}

const folderDeleteModal = ref({
  open: false,
  /** @type {{ id: string, name: string } | null} */
  target: null
})
const requestDeleteModal = ref({
  open: false,
  /** @type {{ id: string, name: string } | null} */
  target: null
})
const deleteActionLoading = ref(false)

function openDeleteFolderModal(f) {
  if (!f) return
  folderDeleteModal.value = { open: true, target: { id: f.id, name: f.name || '' } }
}

function closeFolderDeleteModal() {
  folderDeleteModal.value = { open: false, target: null }
}

async function confirmDeleteFolder() {
  const t = folderDeleteModal.value.target
  if (!t) return
  deleteActionLoading.value = true
  try {
    await FolderAPI.DeleteFolder(t.id)
    closeFolderDeleteModal()
    await load()
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  } finally {
    deleteActionLoading.value = false
  }
}

function openDeleteRequestModal(r) {
  if (!r) return
  requestDeleteModal.value = { open: true, target: { id: r.id, name: r.name || '' } }
}

function closeRequestDeleteModal() {
  requestDeleteModal.value = { open: false, target: null }
}

async function confirmDeleteRequest() {
  const t = requestDeleteModal.value.target
  if (!t) return
  deleteActionLoading.value = true
  try {
    await SavedRequestAPI.Delete(t.id)
    closeRequestDeleteModal()
    await load()
  } catch (e) {
    emit('console', `[Saved] ${e?.message || String(e)}`)
  } finally {
    deleteActionLoading.value = false
  }
}

function openCreateRequest(parentFolderId) {
  requestModal.value = { open: true, parentFolderId, name: 'New request' }
}

async function submitRequestModal() {
  const name = (requestModal.value.name || '').trim()
  const fid = requestModal.value.parentFolderId
  if (!name || !fid) return
  const targetFolderId = fid
  const dto = {
    folder_id: fid,
    name,
    method: 'GET',
    url: 'https://',
    body_mode: 'none',
    headers: [],
    query_params: [],
    form_fields: [],
    multipart_parts: []
  }
  try {
    const created = await SavedRequestAPI.Create(dto)
    requestModal.value.open = false
    if (targetFolderId === props.folderId) {
      await load()
    } else {
      notifyFolderTreeReload(targetFolderId)
    }
    if (created?.id) {
      emit('open-saved-request', created.id)
    }
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  }
}

function openRequest(id) {
  emit('open-saved-request', id)
}

defineExpose({ load, openCreateSubfolder, openCreateRequest })
</script>

<template>
  <div ref="rootEl" class="folder-tree-node text-[11px]">
    <div v-if="loading && depth === 0" class="py-1 text-gray-500">Loading…</div>

    <!-- Subfolders: click hàng (trừ ⋮) để thu/mở; hiệu ứng folder-tree-slide -->
    <div v-for="f in childFolders" :key="f.id" class="mb-0.5">
      <div
        class="min-h-[8px] flex-shrink-0 cursor-pointer rounded transition-colors"
        :class="{ 'bg-orange-500/40': folderReorderGapHover === `before-${f.id}` }"
        title="Drop folder to reorder — click toggles this folder"
        @click="onFolderRowClick(f)"
        @dragover.prevent="onFolderReorderGapDragOver(f.id, $event)"
        @dragleave="onFolderReorderGapDragLeave"
        @drop="onFolderReorderGapDrop($event, f.id)"
      />
      <div
        data-folder-drop-row
        class="relative flex min-h-[40px] flex-col rounded border border-transparent transition-colors"
        :class="{
          'border-t-2 border-orange-500 bg-orange-500/15':
            dndHover?.folderId === f.id && dndHover?.mode === 'sibling',
          'ring-1 ring-orange-500/60 bg-orange-500/15':
            dndHover?.folderId === f.id && dndHover?.mode === 'into',
          'pmj-reveal-flash': flashChildFolderId === f.id
        }"
        :title="
          dndHover?.folderId === f.id
            ? dndHover.mode === 'sibling'
              ? 'Drop: same level as this folder (sibling under parent)'
              : 'Drop: move inside this folder'
            : f.name
        "
        @dragover.capture.prevent="onFolderDropRowDragOver(f, $event)"
        @dragleave="onFolderDropRowDragLeave(f, $event)"
        @drop.capture="onFolderDropRowDrop(f, $event)"
      >
        <div
          v-if="dndHover?.folderId === f.id"
          class="flex shrink-0 justify-between gap-1 border-b border-orange-500/25 px-1 py-0.5 text-[9px] leading-tight text-orange-300/95"
        >
          <span
            :class="
              dndHover.mode === 'sibling' ? 'font-semibold text-orange-100' : 'opacity-35'
            "
            >▲ Same level</span>
          <span
            :class="dndHover.mode === 'into' ? 'font-semibold text-orange-100' : 'opacity-35'"
            >▼ Inside</span>
        </div>
        <div
          role="button"
          tabindex="0"
          :data-subfolder-row="f.id"
          draggable="true"
          class="flex min-h-[36px] flex-1 cursor-pointer items-stretch gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800 group"
          @click="onFolderRowClick(f)"
          @dragstart="onDragStartFolder(f, $event)"
          @dragend="onFolderDropRowDragEnd"
        >
          <div
            class="flex min-w-0 flex-1 items-center gap-1"
            @keydown.enter.prevent="inlineRenameFolder.id === f.id ? undefined : toggleChildExpand(f.id)"
          >
            <span class="text-gray-500 shrink-0">📁</span>
            <input
              v-if="inlineRenameFolder.id === f.id"
              :data-inline-folder-input="f.id"
              v-model="inlineRenameFolder.name"
              type="text"
              class="min-w-0 flex-1 rounded border border-orange-500/50 bg-gray-900 px-1 py-0.5 text-sm text-gray-200 outline-none"
              @click.stop
              @keydown.enter.prevent="submitInlineFolderRename"
              @keydown.escape.prevent="cancelInlineFolderRename"
              @blur="submitInlineFolderRename"
            />
            <span v-else class="min-w-0 flex-1 truncate pr-1 text-gray-200">{{ f.name }}</span>
          </div>
          <button
            type="button"
            draggable="false"
            data-folder-tree-menu
            class="shrink-0 rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-white opacity-70 group-hover:opacity-100"
            style="min-width: 28px; line-height: 1"
            :aria-expanded="folderMenuOpenId === f.id"
            aria-haspopup="menu"
            :aria-label="'Folder actions ' + (f.name || '')"
            @click.stop="toggleFolderRowMenu(f, $event)"
            @dblclick.stop
            @dragstart.stop
          >
            ⋮
          </button>
        </div>
      </div>
      <Transition name="folder-tree-slide">
        <div v-show="isChildExpanded(f.id)" class="ml-1 border-l border-gray-700/80 pl-2.5">
          <FolderTreeNode
            :folder-id="f.id"
            :depth="depth + 1"
            @open-saved-request="(id) => emit('open-saved-request', id)"
            @console="(m) => emit('console', m)"
            @saved-request="emit('saved-request')"
            @tree-changed="emit('tree-changed')"
          />
        </div>
      </Transition>
    </div>

    <div
      v-if="childFolders.length > 0"
      class="min-h-[8px] flex-shrink-0 rounded transition-colors"
      :class="{ 'bg-orange-500/40': folderReorderGapHover === 'append' }"
      title="Drop folder to move to end of list"
      @dragover.prevent="onFolderReorderGapDragOver(null, $event)"
      @dragleave="onFolderReorderGapDragLeave"
      @drop="onFolderReorderGapDrop($event, null)"
    />

    <!-- Requests: ⋮ → Rename / Delete -->
    <div
      v-for="r in requests"
      :key="r.id"
      :data-request-row="r.id"
      draggable="true"
      class="mb-0.5 flex min-w-0 items-center gap-0.5 py-0.5 group"
      :class="{ 'pmj-reveal-flash': flashRequestId === r.id }"
      @dragstart="onDragStartRequest(r, $event)"
    >
      <div
        class="min-w-0 flex-1 cursor-pointer text-left text-gray-400 hover:text-orange-300"
        :title="r.url"
        @click="onRequestRowClick(r)"
        @dblclick="onRequestRowDblClick(r)"
        @keydown.enter.prevent="inlineRenameRequest.id === r.id ? undefined : openRequest(r.id)"
      >
        <span class="font-mono text-[10px] text-gray-500">{{ r.method }}</span>
        <input
          v-if="inlineRenameRequest.id === r.id"
          :data-inline-request-input="r.id"
          v-model="inlineRenameRequest.name"
          type="text"
          class="ml-1 inline-block w-[min(100%,12rem)] rounded border border-orange-500/50 bg-gray-900 px-1 py-0.5 text-xs text-gray-200 outline-none"
          @click.stop
          @keydown.enter.prevent="submitInlineRequestRename"
          @keydown.escape.prevent="cancelInlineRequestRename"
          @blur="submitInlineRequestRename"
        />
        <span v-else class="ml-1">{{ r.name }}</span>
      </div>
      <button
        type="button"
        draggable="false"
        data-request-tree-menu
        class="shrink-0 rounded p-1 text-[10px] text-gray-500 opacity-70 hover:bg-gray-800 hover:text-gray-300 group-hover:opacity-100"
        style="min-width: 24px; line-height: 1"
        :aria-expanded="requestMenuOpenId === r.id"
        aria-haspopup="menu"
        :aria-label="'Request actions ' + (r.name || '')"
        @click.stop="toggleRequestRowMenu(r, $event)"
        @dblclick.stop
        @dragstart.stop
      >
        ⋮
      </button>
    </div>

    <Teleport to="#app">
      <div
        v-if="folderMenuOpenId !== null && menuTargetFolder"
        data-folder-tree-menu
        class="min-w-[220px] rounded-md border border-gray-600 bg-[#2a2a2a] py-1 shadow-xl"
        :style="folderMenuStyle"
        role="menu"
      >
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-orange-300 hover:bg-gray-700"
          @click="onFolderMenuNewFolder(menuTargetFolder)"
        >
          New Folder
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-orange-300 hover:bg-gray-700"
          @click="onFolderMenuNewRequest(menuTargetFolder)"
        >
          New Request
        </button>
        <div class="my-1 border-t border-gray-600" role="separator" />
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onFolderMenuRename(menuTargetFolder)"
        >
          Rename
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onFolderMenuDuplicate(menuTargetFolder)"
        >
          Duplicate
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onFolderMenuEdit(menuTargetFolder)"
        >
          Edit folder…
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-gray-700 hover:text-red-300"
          @click="onFolderMenuDelete(menuTargetFolder)"
        >
          Delete folder
        </button>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="requestMenuOpenId !== null && menuTargetRequest"
        data-request-tree-menu
        class="min-w-[180px] rounded-md border border-gray-600 bg-[#2a2a2a] py-1 shadow-xl"
        :style="requestMenuStyle"
        role="menu"
      >
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onRequestMenuRename(menuTargetRequest)"
        >
          Rename
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onRequestMenuDuplicate(menuTargetRequest)"
        >
          Duplicate
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-gray-700 hover:text-red-300"
          @click="onRequestMenuDelete(menuTargetRequest)"
        >
          Delete
        </button>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="folderModal.open"
        class="fixed inset-0 z-[55] flex items-center justify-center bg-black/50 px-4"
        role="presentation"
        @mousedown.self="closeFolderModalBackdrop"
      >
        <div class="w-full max-w-sm rounded-lg border border-gray-600 bg-[#1f1f1f] p-4 shadow-xl" role="dialog" aria-modal="true" @mousedown.stop>
          <h3 class="text-sm font-semibold text-white">
            {{ folderModal.mode === 'create' ? 'New folder' : 'Edit folder' }}
          </h3>
          <label class="mt-3 block text-[10px] text-gray-500">Name</label>
          <input
            v-model="folderModal.name"
            type="text"
            class="mt-1 w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-sm text-gray-200"
          />
          <label class="mt-2 block text-[10px] text-gray-500">Description</label>
          <textarea
            v-model="folderModal.description"
            rows="2"
            class="mt-1 w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-sm text-gray-200"
          />
          <div class="mt-3 flex justify-end gap-2">
            <button type="button" class="rounded px-3 py-1 text-xs text-gray-400 hover:bg-gray-700" @click="folderModal.open = false">
              Cancel
            </button>
            <button type="button" class="rounded bg-orange-600 px-3 py-1 text-xs font-semibold text-white hover:bg-orange-700" @click="submitFolderModal">
              OK
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="renameRequestModal.open"
        class="fixed inset-0 z-[55] flex items-center justify-center bg-black/50 px-4"
        role="presentation"
        @mousedown.self="closeRenameRequestBackdrop"
      >
        <div class="w-full max-w-sm rounded-lg border border-gray-600 bg-[#1f1f1f] p-4 shadow-xl" role="dialog" aria-modal="true" @mousedown.stop>
          <h3 class="text-sm font-semibold text-white">Rename request</h3>
          <label class="mt-3 block text-[10px] text-gray-500">Name</label>
          <input
            v-model="renameRequestModal.name"
            type="text"
            class="mt-1 w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-sm text-gray-200"
            @keydown.enter.prevent="submitRenameRequestModal"
          />
          <div class="mt-3 flex justify-end gap-2">
            <button type="button" class="rounded px-3 py-1 text-xs text-gray-400 hover:bg-gray-700" @click="renameRequestModal.open = false">
              Cancel
            </button>
            <button
              type="button"
              class="rounded bg-orange-600 px-3 py-1 text-xs font-semibold text-white hover:bg-orange-700"
              @click="submitRenameRequestModal"
            >
              Save
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="requestModal.open"
        class="fixed inset-0 z-[55] flex items-center justify-center bg-black/50 px-4"
        role="presentation"
        @mousedown.self="closeRequestModalBackdrop"
      >
        <div class="w-full max-w-sm rounded-lg border border-gray-600 bg-[#1f1f1f] p-4 shadow-xl" role="dialog" aria-modal="true" @mousedown.stop>
          <h3 class="text-sm font-semibold text-white">New saved request</h3>
          <label class="mt-3 block text-[10px] text-gray-500">Name</label>
          <input
            v-model="requestModal.name"
            type="text"
            class="mt-1 w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-sm text-gray-200"
            @keydown.enter.prevent="submitRequestModal"
          />
          <div class="mt-3 flex justify-end gap-2">
            <button type="button" class="rounded px-3 py-1 text-xs text-gray-400 hover:bg-gray-700" @click="requestModal.open = false">
              Cancel
            </button>
            <button type="button" class="rounded bg-orange-600 px-3 py-1 text-xs font-semibold text-white hover:bg-orange-700" @click="submitRequestModal">
              Create
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="folderDeleteModal.open"
        class="fixed inset-0 z-[56] flex items-center justify-center bg-black/50 px-4"
        role="presentation"
        data-folder-delete-confirm
        @mousedown.self="closeFolderDeleteModal"
      >
        <div class="w-full max-w-md rounded-lg border border-gray-600 bg-[#1f1f1f] shadow-xl" role="dialog" aria-modal="true" @mousedown.stop>
          <div class="border-b border-gray-700 px-4 py-3">
            <h3 class="text-sm font-semibold text-white">Delete folder</h3>
          </div>
          <div class="p-4">
            <p class="text-sm text-gray-300">
              Delete folder
              <span class="font-semibold text-white">"{{ folderDeleteModal.target?.name }}"</span>
              and everything inside? This cannot be undone — all nested folders and saved requests under it will be
              removed.
            </p>
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
            <button
              type="button"
              class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
              :disabled="deleteActionLoading"
              @click="closeFolderDeleteModal"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
              :disabled="deleteActionLoading"
              @click="confirmDeleteFolder"
            >
              {{ deleteActionLoading ? 'Working…' : 'Delete' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="#app">
      <div
        v-if="requestDeleteModal.open"
        class="fixed inset-0 z-[56] flex items-center justify-center bg-black/50 px-4"
        role="presentation"
        data-request-delete-confirm
        @mousedown.self="closeRequestDeleteModal"
      >
        <div class="w-full max-w-md rounded-lg border border-gray-600 bg-[#1f1f1f] shadow-xl" role="dialog" aria-modal="true" @mousedown.stop>
          <div class="border-b border-gray-700 px-4 py-3">
            <h3 class="text-sm font-semibold text-white">Delete saved request</h3>
          </div>
          <div class="p-4">
            <p class="text-sm text-gray-300">
              Remove saved request
              <span class="font-semibold text-white">"{{ requestDeleteModal.target?.name }}"</span>
              ? This cannot be undone.
            </p>
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
            <button
              type="button"
              class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
              :disabled="deleteActionLoading"
              @click="closeRequestDeleteModal"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
              :disabled="deleteActionLoading"
              @click="confirmDeleteRequest"
            >
              {{ deleteActionLoading ? 'Working…' : 'Delete' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
