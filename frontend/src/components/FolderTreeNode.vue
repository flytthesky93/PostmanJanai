<script setup>
import { ref, watch, computed, onMounted, onUnmounted, inject } from 'vue'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import * as SavedRequestAPI from '../../wailsjs/wailsjs/go/delivery/SavedRequestHandler'

const props = defineProps({
  folderId: { type: String, required: true },
  depth: { type: Number, default: 0 }
})

const emit = defineEmits(['open-saved-request', 'console'])

const folderTreeReload = inject('folderTreeReload', null)

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

/** true = đã thu gọn (ẩn cây con + request của folder đó) */
const collapsedChildIds = ref(/** @type {Record<string, boolean>} */ ({}))

function isChildCollapsed(folderId) {
  return !!collapsedChildIds.value[folderId]
}

function toggleChildCollapse(folderId) {
  const cur = { ...collapsedChildIds.value }
  if (cur[folderId]) delete cur[folderId]
  else cur[folderId] = true
  collapsedChildIds.value = cur
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

function onFolderMenuEdit(f) {
  closeFolderRowMenu()
  openEditFolder(f)
}

function onFolderMenuDelete(f) {
  closeFolderRowMenu()
  deleteFolder(f)
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
  deleteRequest(r)
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
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDownFolderMenu, false)
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

async function deleteFolder(f) {
  if (!window.confirm(`Delete folder "${f.name}" and everything inside?`)) return
  try {
    await FolderAPI.DeleteFolder(f.id)
    await load()
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
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

async function deleteRequest(r) {
  if (!window.confirm(`Delete saved request "${r.name}"?`)) return
  try {
    await SavedRequestAPI.Delete(r.id)
    await load()
  } catch (e) {
    emit('console', `[Folders] ${e?.message || String(e)}`)
  }
}

defineExpose({ load, openCreateSubfolder, openCreateRequest })
</script>

<template>
  <div class="folder-tree-node text-[11px]">
    <div v-if="loading && depth === 0" class="py-1 text-gray-500">Loading…</div>

    <!-- Subfolders: click hàng (trừ ⋮) để thu/mở; hiệu ứng folder-tree-slide -->
    <div v-for="f in childFolders" :key="f.id" class="mb-0.5">
      <div
        role="button"
        tabindex="0"
        class="flex cursor-pointer items-center gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800 group"
        :title="f.name"
        @click="toggleChildCollapse(f.id)"
        @keydown.enter.prevent="toggleChildCollapse(f.id)"
      >
        <span class="text-gray-500 shrink-0">📁</span>
        <span class="min-w-0 flex-1 truncate pr-1 text-gray-200">{{ f.name }}</span>
        <button
          type="button"
          data-folder-tree-menu
          class="shrink-0 rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-white opacity-70 group-hover:opacity-100"
          style="min-width: 28px; line-height: 1"
          :aria-expanded="folderMenuOpenId === f.id"
          aria-haspopup="menu"
          :aria-label="'Folder actions ' + (f.name || '')"
          @click.stop="toggleFolderRowMenu(f, $event)"
        >
          ⋮
        </button>
      </div>
      <Transition name="folder-tree-slide">
        <div v-show="!isChildCollapsed(f.id)" class="ml-1 border-l border-gray-700/80 pl-2.5">
          <FolderTreeNode
            :folder-id="f.id"
            :depth="depth + 1"
            @open-saved-request="(id) => emit('open-saved-request', id)"
            @console="(m) => emit('console', m)"
          />
        </div>
      </Transition>
    </div>

    <!-- Requests: ⋮ → Rename / Delete -->
    <div
      v-for="r in requests"
      :key="r.id"
      class="mb-0.5 flex min-w-0 items-center gap-0.5 py-0.5 group"
    >
      <button
        type="button"
        class="min-w-0 flex-1 truncate text-left text-gray-400 hover:text-orange-300"
        :title="r.url"
        @click="openRequest(r.id)"
      >
        <span class="font-mono text-[10px] text-gray-500">{{ r.method }}</span>
        {{ r.name }}
      </button>
      <button
        type="button"
        data-request-tree-menu
        class="shrink-0 rounded p-1 text-[10px] text-gray-500 opacity-70 hover:bg-gray-800 hover:text-gray-300 group-hover:opacity-100"
        style="min-width: 24px; line-height: 1"
        :aria-expanded="requestMenuOpenId === r.id"
        aria-haspopup="menu"
        :aria-label="'Request actions ' + (r.name || '')"
        @click.stop="toggleRequestRowMenu(r, $event)"
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
          @click="onFolderMenuEdit(menuTargetFolder)"
        >
          Edit folder
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
          @click="openRenameRequest(menuTargetRequest)"
        >
          Rename
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
  </div>
</template>
