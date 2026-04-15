<script setup>
import { ref, onMounted, computed } from 'vue'
import { GetAll, CreateWorkspace, Update, Delete } from '../../wailsjs/wailsjs/go/delivery/WorkspaceHandler'

const workspaces = ref([])
/** Backend may return null; never let v-for / .length throw or the whole sidebar can go blank */
const workspaceList = computed(() => (Array.isArray(workspaces.value) ? workspaces.value : []))
const loading = ref(false)
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
    title: 'Tạo Workspace',
    submitLabel: 'Tạo',
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
    title: 'Cập nhật Workspace',
    submitLabel: 'Lưu',
    target: ws
  }
}

const openDeleteModal = (ws) => {
  if (!ws) return
  modalState.value = {
    show: true,
    mode: 'delete',
    title: 'Xóa Workspace',
    submitLabel: 'Xóa',
    target: ws
  }
}

const closeModal = (force = false) => {
  if (!force && submitting.value) return
  modalState.value.show = false
}

const loadWorkspaces = async () => {
  loading.value = true
  try {
    const list = await GetAll()
    workspaces.value = Array.isArray(list) ? list : []
  } catch (error) {
    console.error('[Workspace] Load failed:', error)
    showToast('error', `Không tải được danh sách workspace: ${error?.message || error}`)
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
        showToast('warning', 'Tên workspace không được để trống')
        return
      }
      await CreateWorkspace({
        workspace_name: name,
        workspace_description: formDescription.value.trim()
      })
      showToast('success', 'Tạo workspace thành công')
      closeModal(true)
      await loadWorkspaces()
      return
    }

    if (modalState.value.mode === 'edit') {
      const target = modalState.value.target
      const name = formName.value.trim()
      if (!target || !name) {
        showToast('warning', 'Thông tin workspace không hợp lệ')
        return
      }
      await Update(target.id, name, formDescription.value.trim())
      showToast('success', 'Cập nhật workspace thành công')
      closeModal(true)
      await loadWorkspaces()
      return
    }

    if (modalState.value.mode === 'delete') {
      const target = modalState.value.target
      if (!target) {
        showToast('warning', 'Workspace không hợp lệ')
        return
      }
      await Delete(target.id)
      showToast('success', `Đã xóa workspace "${target.workspace_name}"`)
      closeModal(true)
      await loadWorkspaces()
    }
  } catch (error) {
    console.error('[Workspace] Action failed:', error)
    const label = modalState.value.mode === 'create'
      ? 'Tạo'
      : (modalState.value.mode === 'edit' ? 'Cập nhật' : 'Xóa')
    showToast('error', `${label} workspace thất bại: ${error?.message || error}`)
    closeModal(true)
  } finally {
    submitting.value = false
  }
}

onMounted(loadWorkspaces)
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
      <div class="flex shrink-0 items-center justify-between border-b border-gray-800 p-4">
        <span class="text-xs font-bold uppercase tracking-widest text-white" style="color: #ffffff">Workspaces</span>
        <button
          type="button"
          @click="openCreateModal"
          class="rounded px-2 text-lg font-bold text-orange-500 hover:bg-gray-700"
          style="color: #f97316"
        >
          +
        </button>
      </div>
      <div class="min-h-0 flex-1 overflow-y-auto p-2">
        <div v-if="loading" class="p-2 text-xs text-gray-500" style="color: #9ca3af">Đang tải workspace...</div>
        <div v-else-if="workspaceList.length === 0" class="p-2 text-xs text-gray-500" style="color: #9ca3af">
          Chưa có workspace nào.
        </div>
        <div
          v-for="ws in workspaceList"
          :key="ws.id"
          class="mb-1 flex items-center gap-2 rounded p-2 text-sm transition-colors hover:bg-gray-800"
        >
          <span class="text-gray-500">📁</span>
          <span class="flex-1 truncate">{{ ws.workspace_name }}</span>
          <button type="button" @click="openEditModal(ws)" class="text-xs text-gray-400 hover:text-white">Sửa</button>
          <button type="button" @click="openDeleteModal(ws)" class="text-xs text-red-400 hover:text-red-300">Xóa</button>
        </div>
      </div>
    </aside>

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
            Bạn có chắc muốn xóa workspace
            <span class="text-white font-semibold">"{{ modalState.target?.workspace_name }}"</span>?
          </p>
        </template>
        <template v-else>
          <label class="block text-xs text-gray-400 mb-1">Tên workspace</label>
          <input
            v-model="formName"
            type="text"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Nhập tên workspace"
          />
          <label class="block text-xs text-gray-400 mt-3 mb-1">Mô tả</label>
          <textarea
            v-model="formDescription"
            rows="3"
            class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Mô tả workspace (tuỳ chọn)"
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
          Hủy
        </button>
        <button
          @click="submitModal"
          :disabled="submitting"
          class="px-3 py-1.5 rounded text-xs text-white disabled:opacity-50"
          :class="modalState.mode === 'delete' ? 'bg-red-600 hover:bg-red-700' : 'bg-orange-600 hover:bg-orange-700'"
        >
          {{ submitting ? 'Đang xử lý...' : modalState.submitLabel }}
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