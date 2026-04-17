<script setup>
import { ref, watch } from 'vue'
import FolderTreeNode from './FolderTreeNode.vue'

const props = defineProps({
  /** Selected root folder id (sidebar) — tree shows this folder's contents */
  rootFolderId: { type: String, default: null }
})

const emit = defineEmits(['open-saved-request', 'console'])

const treeRef = ref(null)

async function loadTree() {
  await treeRef.value?.load?.()
}

watch(
  () => props.rootFolderId,
  () => {
    loadTree()
  }
)

/** From Sidebar ⋮: create child folder under this root */
function openCreateChildFolderForRoot(rootId) {
  treeRef.value?.openCreateSubfolder?.(rootId)
}

/** From Sidebar ⋮: create request in root folder */
function openCreateRootRequestForFolder(rootId) {
  treeRef.value?.openCreateRequest?.(rootId)
}

defineExpose({ loadTree, openCreateChildFolderForRoot, openCreateRootRequestForFolder })
</script>

<template>
  <div v-if="rootFolderId" class="border-t border-gray-800 bg-[#1c1c1c]">
    <div class="app-scrollbar max-h-52 overflow-y-auto px-2 py-1.5 text-[11px]">
      <FolderTreeNode
        ref="treeRef"
        :folder-id="rootFolderId"
        :depth="0"
        @open-saved-request="(id) => emit('open-saved-request', id)"
        @console="(m) => emit('console', m)"
      />
    </div>
  </div>
</template>
