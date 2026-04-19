<script setup>
import { ref, watch, nextTick } from 'vue'
import * as EnvAPI from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'

const props = defineProps({
  environmentId: { type: String, default: null }
})

const emit = defineEmits(['console', 'saved', 'deleted'])

const loading = ref(false)
const savingVars = ref(false)
/** Display name only (edit via sidebar ⋮ → Edit). */
const name = ref('')
/** @type {import('vue').Ref<Array<{ key: string, value: string, enabled: boolean }>>} */
const variables = ref([{ key: '', value: '', enabled: true }])

const deleteOpen = ref(false)
const deleting = ref(false)
/** Scroll container for variable rows (scroll to new row on Add). */
const variablesScrollEl = ref(/** @type {HTMLElement | null} */ (null))

function showToast(msg) {
  emit('console', msg)
}

function resetForm() {
  name.value = ''
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

async function addVariableRow() {
  variables.value.push({ key: '', value: '', enabled: true })
  await nextTick()
  const el = variablesScrollEl.value
  if (!el) return
  if (typeof el.scrollTo === 'function') {
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
  } else {
    el.scrollTop = el.scrollHeight
  }
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

    <div
      v-if="environmentId"
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden p-4"
    >
      <div v-if="loading" class="text-xs text-gray-500">Loading…</div>
      <template v-else>
        <div
          class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded border border-gray-700/90 bg-[#1a1a1a]"
        >
          <div
            class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-gray-800/80 px-4 py-3"
          >
            <span class="text-xs font-semibold uppercase tracking-wide text-gray-500">Variables</span>
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-1 text-[11px] font-semibold text-orange-400 hover:border-orange-500/50"
              @click="addVariableRow"
            >
              Add row
            </button>
          </div>
          <div
            ref="variablesScrollEl"
            class="app-scrollbar min-h-0 min-w-0 flex-1 overflow-y-auto p-4"
          >
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
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  aria-label="Remove variable row"
                  title="Remove row"
                  @click="removeVariableRow(idx)"
                >
                  <svg
                    class="h-4 w-4"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </div>
          </div>
          <div class="shrink-0 border-t border-gray-800/80 px-4 py-3">
            <button
              type="button"
              class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
              :disabled="savingVars"
              @click="saveVariables"
            >
              {{ savingVars ? 'Saving…' : 'Save variables' }}
            </button>
          </div>
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
