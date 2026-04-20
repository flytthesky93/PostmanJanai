<script setup>
import { ref, computed, watch } from 'vue'
import {
  PickCollectionFile,
  PreviewCollectionFile,
  ImportCollectionFile
} from '../../wailsjs/wailsjs/go/delivery/ImportHandler'

const props = defineProps({
  open: { type: Boolean, default: false }
})
const emit = defineEmits(['close', 'imported', 'console'])

const filePath = ref('')
const loading = ref(false)
const importing = ref(false)
const preview = ref(null)
const errorMessage = ref('')

const createEnvironment = ref(true)
const activateEnvironment = ref(false)

const previewStats = computed(() => {
  if (!preview.value) return null
  const counts = { folders: 0, requests: 0 }
  const walk = (items) => {
    if (!Array.isArray(items)) return
    for (const it of items) {
      if (it?.folder) {
        counts.folders += 1
        walk(it.folder.items)
      } else if (it?.request) {
        counts.requests += 1
      }
    }
  }
  walk(preview.value.root_items)
  return counts
})

const warnings = computed(() => {
  const list = preview.value?.warnings
  return Array.isArray(list) ? list : []
})

const variableCount = computed(() => {
  const list = preview.value?.variables
  return Array.isArray(list) ? list.length : 0
})

const fileLabel = computed(() => {
  const p = filePath.value
  if (!p) return ''
  const norm = p.replace(/\\/g, '/')
  const idx = norm.lastIndexOf('/')
  return idx >= 0 ? p.slice(idx + 1) : p
})

const resetState = () => {
  filePath.value = ''
  preview.value = null
  errorMessage.value = ''
  loading.value = false
  importing.value = false
  createEnvironment.value = true
  activateEnvironment.value = false
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      resetState()
    }
  }
)

const pickFile = async () => {
  errorMessage.value = ''
  try {
    const picked = await PickCollectionFile()
    if (!picked) return
    filePath.value = picked
    await loadPreview()
  } catch (e) {
    errorMessage.value = e?.message || String(e)
  }
}

const loadPreview = async () => {
  if (!filePath.value) return
  loading.value = true
  preview.value = null
  errorMessage.value = ''
  try {
    const parsed = await PreviewCollectionFile(filePath.value)
    preview.value = parsed || null
  } catch (e) {
    errorMessage.value = e?.message || String(e)
    preview.value = null
  } finally {
    loading.value = false
  }
}

const runImport = async () => {
  if (!filePath.value || importing.value) return
  importing.value = true
  errorMessage.value = ''
  try {
    const opts = {
      create_environment: createEnvironment.value && variableCount.value > 0,
      activate_environment: activateEnvironment.value && createEnvironment.value && variableCount.value > 0
    }
    const res = await ImportCollectionFile(filePath.value, opts)
    const summary = [
      `Imported into root folder "${res?.root_folder_name ?? '?'}"`,
      `folders=${res?.folders_created ?? 0}`,
      `requests=${res?.requests_created ?? 0}`
    ]
    if (res?.environment_name) {
      summary.push(`env="${res.environment_name}"`)
    }
    emit('console', `[Import] ${summary.join(' · ')}`)
    const res_warnings = Array.isArray(res?.warnings) ? res.warnings : []
    for (const w of res_warnings) {
      emit('console', `[Import] warn: ${w}`)
    }
    emit('imported', res)
    emit('close')
  } catch (e) {
    errorMessage.value = e?.message || String(e)
  } finally {
    importing.value = false
  }
}

const close = () => {
  if (importing.value) return
  emit('close')
}
</script>

<template>
  <Transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      style="background: rgba(0,0,0,0.6)"
      @click.self="close"
    >
      <div
        class="flex w-full max-w-2xl flex-col rounded-lg border border-gray-700 bg-[#1b1b1b] shadow-2xl"
        style="background: #1b1b1b; color: #e5e7eb; max-height: 90vh"
        role="dialog"
        aria-modal="true"
        aria-labelledby="import-modal-title"
      >
        <div class="flex shrink-0 items-center justify-between border-b border-gray-700 px-4 py-3">
          <h2 id="import-modal-title" class="text-sm font-semibold">Import collection</h2>
          <button
            type="button"
            class="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white"
            aria-label="Close"
            :disabled="importing"
            @click="close"
          >
            ✕
          </button>
        </div>

        <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4 py-3">
          <p class="text-xs text-gray-400" style="color: #9ca3af">
            Supports Postman Collection v2.1 / v2.0 (JSON), OpenAPI 3.x (JSON or YAML), Insomnia v4 export.
            The file is imported into a new root folder; existing root names are auto-renamed with " (2)", " (3)", …
          </p>

          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-3 py-1 text-xs font-semibold text-gray-200 hover:border-orange-500/50 hover:bg-gray-800"
              :disabled="loading || importing"
              @click="pickFile"
            >
              Choose file…
            </button>
            <span class="truncate text-xs text-gray-300" :title="filePath">
              {{ fileLabel || 'No file selected' }}
            </span>
          </div>

          <div v-if="loading" class="text-xs text-gray-400" style="color: #9ca3af">Parsing file…</div>

          <div v-if="errorMessage" class="rounded border border-red-700/60 bg-red-900/20 px-3 py-2 text-xs text-red-300">
            {{ errorMessage }}
          </div>

          <div v-if="preview && !loading" class="rounded border border-gray-700 bg-[#161616] p-3 text-xs">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="truncate text-sm font-semibold text-white" :title="preview.name">{{ preview.name }}</div>
                <div class="text-[10px] uppercase tracking-wide text-gray-500" style="color: #9ca3af">
                  {{ preview.format_label }}
                </div>
              </div>
              <div class="flex items-center gap-3 text-gray-300">
                <span>📁 {{ previewStats?.folders ?? 0 }}</span>
                <span>🔗 {{ previewStats?.requests ?? 0 }}</span>
                <span v-if="variableCount > 0">🌿 {{ variableCount }}</span>
              </div>
            </div>

            <div v-if="preview.description" class="mb-2 text-gray-400" style="color: #9ca3af">
              {{ preview.description }}
            </div>

            <div v-if="variableCount > 0" class="mb-2 space-y-1 border-t border-gray-800 pt-2">
              <label class="flex items-center gap-2">
                <input
                  type="checkbox"
                  class="h-3.5 w-3.5 accent-orange-500"
                  v-model="createEnvironment"
                />
                <span>Create environment "{{ preview.name }}" with {{ variableCount }} variable(s)</span>
              </label>
              <label class="flex items-center gap-2 pl-5">
                <input
                  type="checkbox"
                  class="h-3.5 w-3.5 accent-orange-500"
                  v-model="activateEnvironment"
                  :disabled="!createEnvironment"
                />
                <span :class="{ 'text-gray-500': !createEnvironment }">Activate the new environment immediately</span>
              </label>
            </div>

            <div v-if="warnings.length > 0" class="mt-2 border-t border-gray-800 pt-2">
              <div class="mb-1 text-[10px] font-semibold uppercase tracking-wide text-amber-400">
                Warnings ({{ warnings.length }})
              </div>
              <ul class="max-h-32 list-disc space-y-0.5 overflow-y-auto pl-5 text-[11px] text-amber-300">
                <li v-for="(w, idx) in warnings" :key="idx">{{ w }}</li>
              </ul>
            </div>
          </div>
        </div>

        <div class="flex shrink-0 items-center justify-end gap-2 border-t border-gray-700 px-4 py-3">
          <button
            type="button"
            class="rounded px-3 py-1 text-xs font-semibold text-gray-300 hover:text-white"
            :disabled="importing"
            @click="close"
          >
            Cancel
          </button>
          <button
            type="button"
            class="rounded bg-orange-500 px-3 py-1 text-xs font-semibold text-black transition-opacity disabled:opacity-50"
            :disabled="!preview || loading || importing"
            @click="runImport"
          >
            {{ importing ? 'Importing…' : 'Import' }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.12s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
