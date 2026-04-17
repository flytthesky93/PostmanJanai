<script setup>
import { ref, watch } from 'vue'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import * as SavedRequestAPI from '../../wailsjs/wailsjs/go/delivery/SavedRequestHandler'

const props = defineProps({
  folderId: { type: String, required: true },
  depth: { type: Number, default: 0 }
})

const emit = defineEmits(['open-saved-request', 'console'])

const childFolders = ref([])
const requests = ref([])
const loading = ref(false)

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
  try {
    if (folderModal.value.mode === 'create') {
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
    await load()
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
    await load()
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
  <div class="folder-tree-node" :style="{ paddingLeft: depth > 0 ? '10px' : '0' }">
    <div v-if="loading && depth === 0" class="py-1 text-[11px] text-gray-500">Loading…</div>

    <div v-for="f in childFolders" :key="f.id" class="mb-1">
      <div
        class="flex items-center gap-1 rounded bg-gray-800/40 px-1 py-0.5"
        :style="{ marginLeft: depth > 0 ? '4px' : '0' }"
      >
        <span class="min-w-0 flex-1 truncate text-[11px] font-medium text-gray-300" :title="f.name">{{ f.name }}</span>
        <button
          type="button"
          class="shrink-0 text-[10px] text-gray-500 hover:text-orange-400"
          title="New subfolder"
          @click="openCreateSubfolder(f.id)"
        >
          +F
        </button>
        <button
          type="button"
          class="shrink-0 text-[10px] text-gray-500 hover:text-orange-400"
          title="New request in this folder"
          @click="openCreateRequest(f.id)"
        >
          +R
        </button>
        <button type="button" class="shrink-0 text-[10px] text-gray-500 hover:text-white" title="Edit" @click="openEditFolder(f)">
          ✎
        </button>
        <button type="button" class="shrink-0 text-[10px] text-gray-500 hover:text-red-400" title="Delete" @click="deleteFolder(f)">
          ×
        </button>
      </div>
      <FolderTreeNode :folder-id="f.id" :depth="depth + 1" @open-saved-request="(id) => emit('open-saved-request', id)" @console="(m) => emit('console', m)" />
    </div>

    <div
      v-for="r in requests"
      :key="r.id"
      class="flex items-center gap-1 py-0.5"
      :style="{ marginLeft: depth > 0 ? '8px' : '0' }"
    >
      <button
        type="button"
        class="min-w-0 flex-1 truncate text-left text-[11px] text-gray-400 hover:text-orange-300"
        :title="r.url"
        @click="openRequest(r.id)"
      >
        <span class="font-mono text-[10px] text-gray-500">{{ r.method }}</span>
        {{ r.name }}
      </button>
      <button type="button" class="shrink-0 text-[10px] text-gray-600 hover:text-red-400" @click="deleteRequest(r)">×</button>
    </div>

    <Teleport to="#app">
      <div
        v-if="folderModal.open"
        class="fixed inset-0 z-[55] flex items-center justify-center bg-black/50 px-4"
        @click.self="folderModal.open = false"
      >
        <div class="w-full max-w-sm rounded-lg border border-gray-600 bg-[#1f1f1f] p-4 shadow-xl" @click.stop>
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
        v-if="requestModal.open"
        class="fixed inset-0 z-[55] flex items-center justify-center bg-black/50 px-4"
        @click.self="requestModal.open = false"
      >
        <div class="w-full max-w-sm rounded-lg border border-gray-600 bg-[#1f1f1f] p-4 shadow-xl" @click.stop>
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
