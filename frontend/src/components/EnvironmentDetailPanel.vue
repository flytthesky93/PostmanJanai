<script setup>
import { ref, watch } from 'vue'
import * as EnvAPI from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'

const props = defineProps({
  environmentId: { type: String, default: null }
})

const emit = defineEmits(['console', 'saved', 'deleted'])

const loading = ref(false)
const savingMeta = ref(false)
const savingVars = ref(false)
const name = ref('')
const description = ref('')
/** @type {import('vue').Ref<Array<{ key: string, value: string, enabled: boolean }>>} */
const variables = ref([{ key: '', value: '', enabled: true }])

const deleteOpen = ref(false)
const deleting = ref(false)

function showToast(msg) {
  emit('console', msg)
}

function resetForm() {
  name.value = ''
  description.value = ''
  variables.value = [{ key: '', value: '', enabled: true }]
}

async function load() {
  const id = props.environmentId
  if (!id) {
    resetForm()
    return
  }
  loading.value = true
  try {
    const full = await EnvAPI.Get(id)
    if (!full || typeof full !== 'object') {
      resetForm()
      return
    }
    name.value = full.name || ''
    description.value = full.description || ''
    const vars = Array.isArray(full.variables) ? full.variables : []
    if (vars.length === 0) {
      variables.value = [{ key: '', value: '', enabled: true }]
    } else {
      variables.value = vars.map((v) => ({
        key: v.key || '',
        value: v.value ?? '',
        enabled: v.enabled !== false
      }))
    }
  } catch (e) {
    const msg = e?.message || String(e)
    showToast(`[Env] Could not load: ${msg}`)
    resetForm()
  } finally {
    loading.value = false
  }
}

watch(
  () => props.environmentId,
  () => {
    load()
  },
  { immediate: true }
)

async function saveMeta() {
  const id = props.environmentId
  if (!id) return
  const n = name.value.trim()
  if (!n) {
    showToast('[Env] Name is required.')
    return
  }
  savingMeta.value = true
  try {
    await EnvAPI.UpdateMeta(id, n, description.value.trim())
    showToast('[Env] Saved name & description.')
    emit('saved')
  } catch (e) {
    const msg = e?.message || String(e)
    if (msg.includes('ENV_602') || msg.includes('already exists')) {
      showToast('[Env] That name is already used. Choose another.')
    } else {
      showToast(`[Env] Save failed: ${msg}`)
    }
  } finally {
    savingMeta.value = false
  }
}

function addVariableRow() {
  variables.value.push({ key: '', value: '', enabled: true })
}

function removeVariableRow(index) {
  const next = variables.value.filter((_, i) => i !== index)
  variables.value = next.length ? next : [{ key: '', value: '', enabled: true }]
}

async function saveVariables() {
  const id = props.environmentId
  if (!id) return
  const keys = new Set()
  for (const row of variables.value) {
    const k = String(row.key || '').trim()
    if (!k) continue
    const lower = k.toLowerCase()
    if (keys.has(lower)) {
      showToast(`[Env] Duplicate variable key: ${k}`)
      return
    }
    keys.add(lower)
  }
  const payload = variables.value
    .map((row, i) => ({
      key: String(row.key || '').trim(),
      value: row.value ?? '',
      enabled: row.enabled !== false,
      sort_order: i
    }))
    .filter((r) => r.key !== '')

  savingVars.value = true
  try {
    await EnvAPI.SaveVariables(id, payload)
    showToast('[Env] Variables saved.')
    emit('saved')
    await load()
  } catch (e) {
    const msg = e?.message || String(e)
    if (msg.includes('ENV_603')) {
      showToast('[Env] Duplicate key: each variable name must be unique.')
    } else {
      showToast(`[Env] Could not save variables: ${msg}`)
    }
  } finally {
    savingVars.value = false
  }
}

async function confirmDelete() {
  const id = props.environmentId
  if (!id) return
  deleting.value = true
  try {
    await EnvAPI.Delete(id)
    showToast(`[Env] Deleted "${name.value || 'environment'}".`)
    deleteOpen.value = false
    emit('deleted')
  } catch (e) {
    showToast(`[Env] Delete failed: ${e?.message || e}`)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div
    class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-[#1c1c1c]"
    style="min-height: 0"
  >
    <div class="shrink-0 border-b border-[#2a2a2a] bg-[#212121] px-4 py-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
          <h2 class="text-sm font-semibold text-white">Environment</h2>
          <p v-if="!environmentId" class="mt-1 text-xs text-gray-500">Select an environment from the sidebar.</p>
          <p v-else class="mt-1 truncate text-xs text-gray-400" :title="name">{{ name || '…' }}</p>
        </div>
        <div v-if="environmentId" class="flex shrink-0 flex-wrap items-center gap-2">
          <button
            type="button"
            class="rounded border border-red-800/80 bg-red-950/40 px-3 py-1.5 text-xs font-semibold text-red-300 hover:bg-red-900/50"
            @click="deleteOpen = true"
          >
            Delete
          </button>
        </div>
      </div>
    </div>

    <div v-if="environmentId" class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-4">
      <div v-if="loading" class="text-xs text-gray-500">Loading…</div>
      <template v-else>
        <div class="mb-6 rounded border border-gray-700/90 bg-[#1a1a1a] p-4">
          <div class="mb-3 text-xs font-semibold uppercase tracking-wide text-gray-500">Details</div>
          <label class="mb-1 block text-xs text-gray-400">Name</label>
          <input
            v-model="name"
            type="text"
            class="mb-3 w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Environment name"
          />
          <label class="mb-1 block text-xs text-gray-400">Description</label>
          <textarea
            v-model="description"
            rows="2"
            class="mb-3 w-full rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
            placeholder="Optional"
          />
          <button
            type="button"
            class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
            :disabled="savingMeta"
            @click="saveMeta"
          >
            {{ savingMeta ? 'Saving…' : 'Save details' }}
          </button>
        </div>

        <div class="rounded border border-gray-700/90 bg-[#1a1a1a] p-4">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
            <span class="text-xs font-semibold uppercase tracking-wide text-gray-500">Variables</span>
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-1 text-[11px] font-semibold text-orange-400 hover:border-orange-500/50"
              @click="addVariableRow"
            >
              Add row
            </button>
          </div>
          <div class="space-y-2">
            <div
              v-for="(row, idx) in variables"
              :key="idx"
              class="flex flex-wrap items-center gap-2 rounded border border-gray-800 bg-[#141414] p-2"
            >
              <label class="flex shrink-0 cursor-pointer items-center gap-1.5 text-[11px] text-gray-400">
                <input v-model="row.enabled" type="checkbox" class="rounded border-gray-600" />
                On
              </label>
              <input
                v-model="row.key"
                type="text"
                class="min-w-[120px] flex-1 rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
                placeholder="KEY"
              />
              <input
                v-model="row.value"
                type="text"
                class="min-w-[160px] flex-[2] rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
                placeholder="value"
              />
              <button
                type="button"
                class="shrink-0 rounded px-2 py-1 text-xs text-gray-500 hover:bg-gray-800 hover:text-red-300"
                title="Remove row"
                @click="removeVariableRow(idx)"
              >
                ✕
              </button>
            </div>
          </div>
          <button
            type="button"
            class="mt-4 rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
            :disabled="savingVars"
            @click="saveVariables"
          >
            {{ savingVars ? 'Saving…' : 'Save variables' }}
          </button>
        </div>
      </template>
    </div>

    <Teleport to="#app">
      <div
        v-if="deleteOpen"
        class="fixed inset-0 z-[45] flex items-center justify-center bg-black/50 px-4"
        role="dialog"
        aria-modal="true"
      >
        <div class="w-full max-w-md rounded-lg border border-gray-700 bg-[#1f1f1f] shadow-lg">
          <div class="border-b border-gray-700 px-4 py-3">
            <h3 class="text-sm font-semibold text-white">Delete environment</h3>
          </div>
          <div class="p-4 text-sm text-gray-300">
            Delete
            <span class="font-semibold text-white">{{ name || 'this environment' }}</span>
            and all its variables? This cannot be undone.
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
            <button
              type="button"
              class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
              :disabled="deleting"
              @click="deleteOpen = false"
            >
              Cancel
            </button>
            <button
              type="button"
              class="rounded bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
              :disabled="deleting"
              @click="confirmDelete"
            >
              {{ deleting ? 'Deleting…' : 'Delete' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
