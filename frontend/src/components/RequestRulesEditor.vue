<script setup>
import { ref, watch, computed } from 'vue'
import * as RuleAPI from '../../wailsjs/wailsjs/go/delivery/RuleHandler'

const props = defineProps({
  /** The owning saved request UUID (null when ad-hoc / unsaved). */
  requestId: { type: String, default: null },
  /**
   * "captures" — JSONPath / header / status / regex extraction → env var or memory.
   * "assertions" — pass/fail rules evaluated after the response arrives.
   */
  mode: { type: String, required: true, validator: (v) => v === 'captures' || v === 'assertions' }
})

const emit = defineEmits(['console'])

const loading = ref(false)
const saving = ref(false)
const dirty = ref(false)

/** Capture rows: { name, source, expression, target_scope, target_variable, enabled, sort_order, _isNew? } */
const captureRows = ref([])
/** Assertion rows: { name, source, expression, operator, expected, enabled, sort_order, _isNew? } */
const assertionRows = ref([])

const captureSourceOptions = [
  { value: 'json_body', label: 'JSON body (JSONPath)' },
  { value: 'header', label: 'Response header' },
  { value: 'status', label: 'Status code' },
  { value: 'regex_body', label: 'Regex body' }
]
const captureScopeOptions = [
  { value: 'environment', label: 'Active environment' },
  { value: 'memory', label: 'In-memory (run only)' }
]
const assertionSourceOptions = [
  { value: 'status', label: 'Status code' },
  { value: 'header', label: 'Response header' },
  { value: 'json_body', label: 'JSON body (JSONPath)' },
  { value: 'regex_body', label: 'Regex body' },
  { value: 'duration_ms', label: 'Duration (ms)' },
  { value: 'response_size_bytes', label: 'Response size (bytes)' }
]
const assertionOperatorOptions = [
  { value: 'eq', label: 'equals' },
  { value: 'neq', label: 'not equals' },
  { value: 'contains', label: 'contains' },
  { value: 'not_contains', label: 'not contains' },
  { value: 'gt', label: '>' },
  { value: 'lt', label: '<' },
  { value: 'gte', label: '>=' },
  { value: 'lte', label: '<=' },
  { value: 'regex', label: 'regex' },
  { value: 'exists', label: 'exists' },
  { value: 'not_exists', label: 'not exists' }
]

function expressionPlaceholder(source) {
  switch (source) {
    case 'json_body':
      return '$.data.token'
    case 'header':
      return 'X-Request-Id'
    case 'regex_body':
      return 'token=([A-Za-z0-9]+)'
    default:
      return ''
  }
}

const expressionDisabled = (source) =>
  source === 'status' || source === 'duration_ms' || source === 'response_size_bytes'
const expectedDisabled = (operator) => operator === 'exists' || operator === 'not_exists'

function blankCaptureRow() {
  return {
    name: '',
    source: 'json_body',
    expression: '',
    target_scope: 'environment',
    target_variable: '',
    enabled: true,
    sort_order: 0,
    _isNew: true
  }
}
function blankAssertionRow() {
  return {
    name: '',
    source: 'status',
    expression: '',
    operator: 'eq',
    expected: '',
    enabled: true,
    sort_order: 0,
    _isNew: true
  }
}

async function load() {
  const id = props.requestId
  if (!id) {
    captureRows.value = []
    assertionRows.value = []
    dirty.value = false
    return
  }
  loading.value = true
  try {
    if (props.mode === 'captures') {
      const rows = await RuleAPI.ListCaptures(id)
      captureRows.value = (Array.isArray(rows) ? rows : []).map((r) => ({
        name: r.name || '',
        source: r.source || 'json_body',
        expression: r.expression || '',
        target_scope: r.target_scope || 'environment',
        target_variable: r.target_variable || '',
        enabled: r.enabled !== false,
        sort_order: typeof r.sort_order === 'number' ? r.sort_order : 0
      }))
    } else {
      const rows = await RuleAPI.ListAssertions(id)
      assertionRows.value = (Array.isArray(rows) ? rows : []).map((r) => ({
        name: r.name || '',
        source: r.source || 'status',
        expression: r.expression || '',
        operator: r.operator || 'eq',
        expected: r.expected != null ? String(r.expected) : '',
        enabled: r.enabled !== false,
        sort_order: typeof r.sort_order === 'number' ? r.sort_order : 0
      }))
    }
    dirty.value = false
  } catch (e) {
    emit('console', `[Rules] Could not load: ${e?.message || String(e)}`)
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.requestId, props.mode],
  () => {
    load()
  },
  { immediate: true }
)

function markDirty() {
  dirty.value = true
}

function addRow() {
  if (props.mode === 'captures') {
    captureRows.value.push(blankCaptureRow())
  } else {
    assertionRows.value.push(blankAssertionRow())
  }
  markDirty()
}

function removeRow(i) {
  if (props.mode === 'captures') {
    captureRows.value.splice(i, 1)
  } else {
    assertionRows.value.splice(i, 1)
  }
  markDirty()
}

function moveRow(i, dir) {
  const list = props.mode === 'captures' ? captureRows.value : assertionRows.value
  const j = i + dir
  if (j < 0 || j >= list.length) return
  const tmp = list[i]
  list[i] = list[j]
  list[j] = tmp
  markDirty()
}

const buildCapturePayload = () =>
  captureRows.value
    .map((r, i) => ({
      name: String(r.name || '').trim(),
      source: String(r.source || 'json_body'),
      expression: String(r.expression || ''),
      target_scope: r.target_scope === 'memory' ? 'memory' : 'environment',
      target_variable: String(r.target_variable || '').trim(),
      enabled: r.enabled !== false,
      sort_order: i
    }))
    .filter((r) => r.name && r.target_variable)

const buildAssertionPayload = () =>
  assertionRows.value
    .map((r, i) => ({
      name: String(r.name || '').trim(),
      source: String(r.source || 'status'),
      expression: String(r.expression || ''),
      operator: String(r.operator || 'eq'),
      expected: String(r.expected || ''),
      enabled: r.enabled !== false,
      sort_order: i
    }))
    .filter((r) => r.name && r.operator)

async function save() {
  const id = props.requestId
  if (!id) {
    emit('console', '[Rules] Save the request first to attach rules.')
    return
  }
  saving.value = true
  try {
    if (props.mode === 'captures') {
      const out = await RuleAPI.SaveCaptures(id, buildCapturePayload())
      captureRows.value = (Array.isArray(out) ? out : []).map((r) => ({
        name: r.name || '',
        source: r.source || 'json_body',
        expression: r.expression || '',
        target_scope: r.target_scope || 'environment',
        target_variable: r.target_variable || '',
        enabled: r.enabled !== false,
        sort_order: typeof r.sort_order === 'number' ? r.sort_order : 0
      }))
    } else {
      const out = await RuleAPI.SaveAssertions(id, buildAssertionPayload())
      assertionRows.value = (Array.isArray(out) ? out : []).map((r) => ({
        name: r.name || '',
        source: r.source || 'status',
        expression: r.expression || '',
        operator: r.operator || 'eq',
        expected: r.expected != null ? String(r.expected) : '',
        enabled: r.enabled !== false,
        sort_order: typeof r.sort_order === 'number' ? r.sort_order : 0
      }))
    }
    dirty.value = false
    emit('console', `[Rules] Saved ${props.mode}.`)
  } catch (e) {
    emit('console', `[Rules] ${e?.message || String(e)}`)
  } finally {
    saving.value = false
  }
}

const isCapture = computed(() => props.mode === 'captures')
const headerHint = computed(() =>
  isCapture.value
    ? 'Run after each response. JSON values are auto-stringified; status writes the numeric code.'
    : 'Evaluated after the response arrives. Failed assertions surface inline and in the Runner report.'
)

defineExpose({ save, reload: load, dirty: () => dirty.value })
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <div v-if="!requestId" class="flex flex-1 items-center justify-center text-xs text-gray-500">
      Save this request first to attach {{ isCapture ? 'capture rules' : 'tests' }}.
    </div>
    <template v-else>
      <div class="shrink-0 border-b border-gray-800/80 px-3 py-2 text-[11px] text-gray-500">
        {{ headerHint }}
      </div>
      <div v-if="loading" class="flex flex-1 items-center justify-center text-xs text-gray-500">
        Loading…
      </div>
      <div
        v-else
        class="app-scrollbar min-h-0 flex-1 overflow-auto px-3 py-3"
      >
        <!-- CAPTURES -->
        <div v-if="isCapture" class="space-y-2">
          <div
            v-for="(row, i) in captureRows"
            :key="'cap-' + i"
            class="rounded border border-gray-800 bg-[#141414] p-2"
          >
            <div class="flex flex-wrap items-start gap-2">
              <label class="flex shrink-0 cursor-pointer items-center gap-1 pt-1.5 text-[11px] text-gray-400">
                <input v-model="row.enabled" type="checkbox" class="rounded border-gray-600" @change="markDirty" />
                On
              </label>
              <div class="min-w-[140px] flex-1">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Name</label>
                <input
                  v-model="row.name"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  placeholder="capture token"
                  @input="markDirty"
                />
              </div>
              <div class="min-w-[150px] shrink-0">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Source</label>
                <select
                  v-model="row.source"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  @change="markDirty"
                >
                  <option v-for="opt in captureSourceOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
              <div class="min-w-[180px] flex-[2]">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Expression</label>
                <input
                  v-model="row.expression"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500 disabled:cursor-not-allowed disabled:opacity-40"
                  :placeholder="expressionPlaceholder(row.source)"
                  :disabled="expressionDisabled(row.source)"
                  @input="markDirty"
                />
              </div>
              <div class="min-w-[140px] shrink-0">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Target scope</label>
                <select
                  v-model="row.target_scope"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  @change="markDirty"
                >
                  <option v-for="opt in captureScopeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
              <div class="min-w-[140px] flex-1">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Target variable</label>
                <input
                  v-model="row.target_variable"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
                  placeholder="ACCESS_TOKEN"
                  @input="markDirty"
                />
              </div>
              <div class="flex shrink-0 items-end gap-1 pt-4">
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-300"
                  title="Move up"
                  :disabled="i === 0"
                  @click="moveRow(i, -1)"
                >
                  ↑
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-300"
                  title="Move down"
                  :disabled="i === captureRows.length - 1"
                  @click="moveRow(i, 1)"
                >
                  ↓
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  title="Remove"
                  @click="removeRow(i)"
                >
                  ✕
                </button>
              </div>
            </div>
          </div>
          <div v-if="captureRows.length === 0" class="rounded border border-dashed border-gray-700 px-3 py-6 text-center text-[11px] text-gray-500">
            No capture rules yet — click <span class="text-orange-400">Add capture</span>.
          </div>
        </div>

        <!-- ASSERTIONS -->
        <div v-else class="space-y-2">
          <div
            v-for="(row, i) in assertionRows"
            :key="'ast-' + i"
            class="rounded border border-gray-800 bg-[#141414] p-2"
          >
            <div class="flex flex-wrap items-start gap-2">
              <label class="flex shrink-0 cursor-pointer items-center gap-1 pt-1.5 text-[11px] text-gray-400">
                <input v-model="row.enabled" type="checkbox" class="rounded border-gray-600" @change="markDirty" />
                On
              </label>
              <div class="min-w-[140px] flex-1">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Name</label>
                <input
                  v-model="row.name"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  placeholder="status is 200"
                  @input="markDirty"
                />
              </div>
              <div class="min-w-[160px] shrink-0">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Source</label>
                <select
                  v-model="row.source"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  @change="markDirty"
                >
                  <option v-for="opt in assertionSourceOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
              <div class="min-w-[180px] flex-[2]">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Expression</label>
                <input
                  v-model="row.expression"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500 disabled:cursor-not-allowed disabled:opacity-40"
                  :placeholder="expressionPlaceholder(row.source)"
                  :disabled="expressionDisabled(row.source)"
                  @input="markDirty"
                />
              </div>
              <div class="min-w-[120px] shrink-0">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Operator</label>
                <select
                  v-model="row.operator"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                  @change="markDirty"
                >
                  <option v-for="opt in assertionOperatorOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
              <div class="min-w-[160px] flex-1">
                <label class="mb-1 block text-[10px] uppercase tracking-wide text-gray-500">Expected</label>
                <input
                  v-model="row.expected"
                  type="text"
                  class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500 disabled:cursor-not-allowed disabled:opacity-40"
                  placeholder="200"
                  :disabled="expectedDisabled(row.operator)"
                  @input="markDirty"
                />
              </div>
              <div class="flex shrink-0 items-end gap-1 pt-4">
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-300"
                  title="Move up"
                  :disabled="i === 0"
                  @click="moveRow(i, -1)"
                >
                  ↑
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-300"
                  title="Move down"
                  :disabled="i === assertionRows.length - 1"
                  @click="moveRow(i, 1)"
                >
                  ↓
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 w-7 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  title="Remove"
                  @click="removeRow(i)"
                >
                  ✕
                </button>
              </div>
            </div>
          </div>
          <div v-if="assertionRows.length === 0" class="rounded border border-dashed border-gray-700 px-3 py-6 text-center text-[11px] text-gray-500">
            No assertions yet — click <span class="text-orange-400">Add assertion</span>.
          </div>
        </div>
      </div>
      <div class="flex shrink-0 items-center justify-between gap-2 border-t border-gray-800/80 px-3 py-2">
        <button
          type="button"
          class="rounded border border-gray-600 bg-[#2a2a2a] px-3 py-1.5 text-[11px] font-semibold text-orange-400 hover:border-orange-500/50"
          @click="addRow"
        >
          {{ isCapture ? '+ Add capture' : '+ Add assertion' }}
        </button>
        <div class="flex items-center gap-2">
          <span v-if="dirty" class="text-[11px] text-yellow-400">Unsaved changes</span>
          <button
            type="button"
            class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
            :disabled="saving || !dirty"
            @click="save"
          >
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </div>
    </template>
  </div>
</template>
