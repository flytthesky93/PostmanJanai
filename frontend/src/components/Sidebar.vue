<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { GetAll, CreateWorkspace, Update, Delete } from '../../wailsjs/wailsjs/go/delivery/WorkspaceHandler'
import { List as ListHistory } from '../../wailsjs/wailsjs/go/delivery/HistoryHandler'

const props = defineProps({
  /** Currently selected workspace id (UUID string), for linking sends to history. */
  activeWorkspaceId: { type: String, default: null }
})

const emit = defineEmits(['update:activeWorkspaceId'])

/** @type {import('vue').Ref<'workspaces' | 'history'>} */
const sidebarTab = ref('workspaces')

const workspaces = ref([])
/** Backend may return null; never let v-for / .length throw or the whole sidebar can go blank */
const workspaceList = computed(() => (Array.isArray(workspaces.value) ? workspaces.value : []))
const loading = ref(false)

const historyItems = ref([])
const historyLoading = ref(false)
const historyList = computed(() => (Array.isArray(historyItems.value) ? historyItems.value : []))
const submitting = ref(false)

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
    title: 'Create workspace',
    submitLabel: 'Create',
    target: null
  }
}

const openEditModal = (ws) => {
  if (!ws) return
  formName.value = ws.workspace_name || ''
  formDescription.value = ws.workspace_description || ''
  modalState.value = {
    show: true,
    mode: 'edit',
    title: 'Edit workspace',
    submitLabel: 'Save',
    target: ws
  }
}

const openDeleteModal = (ws) => {
  if (!ws) return
  modalState.value = {
    show: true,
    mode: 'delete',
    title: 'Delete workspace',
    submitLabel: 'Delete',
    target: ws
  }
}

const closeModal = (force = false) => {
  if (!force && submitting.value) return
  modalState.value.show = false
}

/** Per-workspace ⋮ menu: teleport + fixed so overflow does not clip */
const menuOpenForId = ref(null)
const menuStyle = ref({
  position: 'fixed',
  top: '0px',
  left: '0px',
  zIndex: 45
})

const menuTargetWs = computed(() => {
  const id = menuOpenForId.value
  if (id == null) return null
  return workspaceList.value.find((w) => w.id === id) ?? null
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
    const width = 168
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

const onDocumentPointerDown = (e) => {
  if (menuOpenForId.value == null) return
  const t = e.target
  if (t.closest?.('[data-ws-menu]')) return
  closeWorkspaceMenu()
}

onMounted(() => {
  /* Run after ⋮ click so toggle opens before document handler closes the menu */
  document.addEventListener('pointerdown', onDocumentPointerDown, false)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, false)
})

const selectWorkspace = (ws) => {
  if (!ws?.id) return
  emit('update:activeWorkspaceId', ws.id)
}

const syncSelectionAfterLoad = () => {
  const list = workspaceList.value
  const ids = new Set(list.map((w) => w.id))
  if (props.activeWorkspaceId && !ids.has(props.activeWorkspaceId)) {
    emit('update:activeWorkspaceId', list[0]?.id ?? null)
    return
  }
  if (!props.activeWorkspaceId && list.length > 0) {
    emit('update:activeWorkspaceId', list[0].id)
  }
}

const loadHistory = async () => {
  historyLoading.value = true
  try {
    const list = await ListHistory()
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

watch(sidebarTab, (tab) => {
  if (tab === 'history') {
    loadHistory()
  }
})

const loadWorkspaces = async () => {
  loading.value = true
  try {
    const list = await GetAll()
    workspaces.value = Array.isArray(list) ? list : []
    syncSelectionAfterLoad()
  } catch (error) {
    console.error('[Workspace] Load failed:', error)
    showToast('error', `Could not load workspaces: ${error?.message || error}`)
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
        showToast('warning', 'Workspace name cannot be empty')
        return
      }
      await CreateWorkspace({
        workspace_name: name,
        workspace_description: formDescription.value.trim()
      })
      showToast('success', 'Workspace created')
      closeModal(true)
      await loadWorkspaces()
      return
    }

    if (modalState.value.mode === 'edit') {
      const target = modalState.value.target
      const name = formName.value.trim()
      if (!target || !name) {
        showToast('warning', 'Invalid workspace data')
        return
      }
      await Update(target.id, name, formDescription.value.trim())
      showToast('success', 'Workspace updated')
      closeModal(true)
      await loadWorkspaces()
      return
    }

    if (modalState.value.mode === 'delete') {
      const target = modalState.value.target
      if (!target) {
        showToast('warning', 'Invalid workspace')
        return
      }
      await Delete(target.id)
      showToast('success', `Deleted workspace "${target.workspace_name}"`)
      if (props.activeWorkspaceId === target.id) {
        emit('update:activeWorkspaceId', null)
      }
      closeModal(true)
      await loadWorkspaces()
    }
  } catch (error) {
    console.error('[Workspace] Action failed:', error)
    const label = modalState.value.mode === 'create'
      ? 'Create'
      : (modalState.value.mode === 'edit' ? 'Update' : 'Delete')
    const msg = error?.message || String(error)
    if (msg.includes('WS_301') || msg.includes('already exists')) {
      showToast('warning', 'That workspace name is already in use. Choose another name.')
    } else {
      showToast('error', `${label} workspace failed: ${msg}`)
    }
    // Keep modal open to fix name / retry
  } finally {
    submitting.value = false
  }
}

onMounted(loadWorkspaces)

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

defineExpose({ refreshHistory })
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
          class="flex-1 px-2 py-2.5 text-xs font-semibold uppercase tracking-wide transition-colors"
          :class="
            sidebarTab === 'workspaces'
              ? 'border-b-2 border-orange-500 bg-[#1a1a1a] text-white'
              : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
          "
          @click="sidebarTab = 'workspaces'"
        >
          Workspaces
        </button>
        <button
          type="button"
          class="flex-1 px-2 py-2.5 text-xs font-semibold uppercase tracking-wide transition-colors"
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

      <template v-if="sidebarTab === 'workspaces'">
        <div class="flex min-h-0 flex-1 flex-col">
          <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-2" @scroll.passive="closeWorkspaceMenu">
            <div v-if="loading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Loading workspaces…</div>
            <div v-else-if="workspaceList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
              No workspaces yet.
            </div>
            <div
              v-for="ws in workspaceList"
              :key="ws.id"
              role="button"
              tabindex="0"
              class="mb-1 flex items-center gap-1 rounded p-2 text-sm transition-colors hover:bg-gray-800 group cursor-pointer"
              :class="{ 'bg-gray-800/80': activeWorkspaceId === ws.id }"
              @click="selectWorkspace(ws)"
              @keydown.enter.prevent="selectWorkspace(ws)"
            >
              <span class="text-gray-500 shrink-0">📁</span>
              <span class="min-w-0 flex-1 truncate pr-1">{{ ws.workspace_name }}</span>
              <button
                type="button"
                data-ws-menu
                class="shrink-0 rounded p-1.5 text-gray-400 hover:bg-gray-700 hover:text-white opacity-70 group-hover:opacity-100"
                style="min-width: 28px; line-height: 1"
                :aria-expanded="menuOpenForId === ws.id"
                aria-haspopup="menu"
                :aria-label="'Workspace actions ' + (ws.workspace_name || '')"
                @click.stop="toggleWorkspaceMenu(ws, $event)"
              >
                ⋮
              </button>
            </div>
          </div>
          <div class="flex shrink-0 justify-end border-t border-gray-800 bg-[#1c1c1c] px-2 py-1">
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-0.5 text-[10px] font-semibold text-orange-500 transition-colors hover:border-orange-500/50 hover:bg-gray-800"
              style="color: #f97316"
              aria-label="Add workspace"
              title="Add workspace"
              @click="openCreateModal"
            >
              Add workspace
            </button>
          </div>
        </div>
      </template>

      <template v-else>
        <div class="flex shrink-0 items-center justify-between border-b border-gray-800 px-3 py-2">
          <span class="text-xs text-gray-500" style="color: #9ca3af">Recent requests</span>
          <button
            type="button"
            class="text-xs font-medium text-orange-500 hover:underline"
            style="color: #f97316"
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
            class="mb-2 rounded border border-gray-700/90 bg-[#1a1a1a] p-2 text-left"
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
        v-if="menuOpenForId !== null && menuTargetWs"
        data-ws-menu
        class="min-w-[168px] rounded-md border border-gray-600 bg-[#2a2a2a] py-1 shadow-xl"
        :style="menuStyle"
        role="menu"
      >
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-gray-200 hover:bg-gray-700"
          @click="onEditFromMenu(menuTargetWs)"
        >
          Edit workspace
        </button>
        <button
          type="button"
          role="menuitem"
          class="w-full px-3 py-2 text-left text-sm text-red-400 hover:bg-gray-700 hover:text-red-300"
          @click="onDeleteFromMenu(menuTargetWs)"
        >
          Delete workspace
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
            Are you sure you want to delete workspace
            <span class="text-white font-semibold">"{{ modalState.target?.workspace_name }}"</span>?
          </p>
        </template>
        <template v-else>
          <label class="block text-xs text-gray-400 mb-1">Name</label>
          <input
            v-model="formName"
            type="text"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Workspace name"
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