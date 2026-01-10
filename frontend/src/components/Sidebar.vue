<script setup>
import { ref, onMounted } from 'vue'
import { GetAll, CreateWorkspace } from '../../wailsjs/wailsjs/go/delivery/WorkspaceHandler'

const workspaces = ref([])

const loadWorkspaces = async () => {
  workspaces.value = await GetAll()
}

const addNewWorkspace = async () => {
  const name = prompt("Nhập tên Workspace mới:")
  if (name) {
    await CreateWorkspace(name, "Mô tả từ UI")
    await loadWorkspaces()
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
      <div v-for="ws in workspaces" :key="ws.id"
           class="p-2 hover:bg-gray-800 rounded cursor-pointer text-sm mb-1 flex items-center gap-2 transition-colors">
        <span class="text-gray-500">📁</span>
        <span class="truncate">{{ ws.name }}</span>
      </div>
    </div>
  </aside>
</template>