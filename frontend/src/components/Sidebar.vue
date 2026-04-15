<script setup>
import { ref, onMounted } from 'vue'
import { GetAll, CreateWorkspace, Update, Delete } from '../../wailsjs/wailsjs/go/delivery/WorkspaceHandler'

const workspaces = ref([])
const loading = ref(false)

const loadWorkspaces = async () => {
  loading.value = true
  try {
    workspaces.value = await GetAll()
  } catch (error) {
    console.error('[Workspace] Load failed:', error)
    alert(`Không tải được danh sách workspace: ${error?.message || error}`)
  } finally {
    loading.value = false
  }
}

const addNewWorkspace = async () => {
  const name = prompt('Nhập tên Workspace mới:')
  if (name && name.trim()) {
    try {
      await CreateWorkspace({
        workspace_name: name.trim(),
        workspace_description: 'Mô tả từ UI'
      })
      await loadWorkspaces()
    } catch (error) {
      console.error('[Workspace] Create failed:', error)
      alert(`Tạo workspace thất bại: ${error?.message || error}`)
    }
  }
}

const editWorkspace = async (ws) => {
  const nextName = prompt('Tên workspace mới:', ws.workspace_name || '')
  if (!nextName || !nextName.trim()) return

  const nextDesc = prompt('Mô tả workspace:', ws.workspace_description || '')
  try {
    await Update(ws.id, nextName.trim(), (nextDesc || '').trim())
    await loadWorkspaces()
  } catch (error) {
    console.error('[Workspace] Update failed:', error)
    alert(`Cập nhật workspace thất bại: ${error?.message || error}`)
  }
}

const deleteWorkspace = async (ws) => {
  const confirmed = confirm(`Xóa workspace "${ws.workspace_name}"?`)
  if (!confirmed) return
  try {
    await Delete(ws.id)
    await loadWorkspaces()
  } catch (error) {
    console.error('[Workspace] Delete failed:', error)
    alert(`Xóa workspace thất bại: ${error?.message || error}`)
  }
}

onMounted(loadWorkspaces)
</script>

<template>
  <aside class="w-64 border-r border-gray-800 bg-[#212121] flex flex-col">
    <div class="p-4 border-b border-gray-800 flex justify-between items-center">
      <span class="font-bold text-white uppercase text-xs tracking-widest">Workspaces</span>
      <button @click="addNewWorkspace" class="hover:bg-gray-700 px-2 rounded text-lg text-orange-500 font-bold">+</button>
    </div>
    <div class="flex-1 overflow-y-auto p-2">
      <div v-if="loading" class="text-xs text-gray-500 p-2">Đang tải workspace...</div>
      <div v-for="ws in workspaces" :key="ws.id"
           class="p-2 hover:bg-gray-800 rounded text-sm mb-1 flex items-center gap-2 transition-colors">
        <span class="text-gray-500">📁</span>
        <span class="truncate flex-1">{{ ws.workspace_name }}</span>
        <button @click="editWorkspace(ws)" class="text-xs text-gray-400 hover:text-white">Sửa</button>
        <button @click="deleteWorkspace(ws)" class="text-xs text-red-400 hover:text-red-300">Xóa</button>
      </div>
    </div>
  </aside>
</template>