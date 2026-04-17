<script setup>
import { ref } from 'vue'
import JsonCodeMirror from './JsonCodeMirror.vue'
import { PickFileForBody, ImportFromCurl } from '../../wailsjs/wailsjs/go/delivery/HTTPHandler'

const props = defineProps({
  /** Current workspace UUID for history (optional). */
  activeWorkspaceId: { type: String, default: null }
})

const emit = defineEmits(['send', 'console'])

const url = ref('')
const method = ref('GET')
const body = ref('')

/** none | raw | form_urlencoded | multipart */
const bodyMode = ref('raw')

/** @type {import('vue').Ref<Array<{ key: string, value: string }>>} */
const queryParams = ref([{ key: '', value: '' }])
/** @type {import('vue').Ref<Array<{ key: string, value: string }>>} */
const headers = ref([
  { key: 'Accept', value: 'application/json' },
  { key: '', value: '' }
])

/** form-urlencoded */
const formFields = ref([{ key: '', value: '' }])

/** multipart: text | file */
const multipartParts = ref([{ key: '', kind: 'text', value: '', file_path: '' }])

const activeTab = ref('params')

/** Step 1: choose import type. Step 2: cURL paste (when applicable). */
const importMenuOpen = ref(false)
const curlModalOpen = ref(false)
const curlText = ref(
  'curl -X GET "https://httpbin.org/get?hello=world"'
)

const openImportMenu = () => {
  importMenuOpen.value = true
}

const closeImportMenu = () => {
  importMenuOpen.value = false
}

const chooseImportCurl = () => {
  closeImportMenu()
  curlModalOpen.value = true
}

const chooseImportCollection = () => {
  emit('console', '[Import] Collection import is not available yet.')
  closeImportMenu()
}

const closeCurlModal = () => {
  curlModalOpen.value = false
}

function applyInputToForm(payload) {
  if (!payload) return
  method.value = (payload.method || 'GET').toUpperCase()
  const rawUrl = String(payload.url || '').trim()
  try {
    const u = new URL(rawUrl)
    if (u.search) {
      url.value = `${u.origin}${u.pathname}`
      const pairs = []
      u.searchParams.forEach((v, k) => pairs.push({ key: k, value: v }))
      queryParams.value = pairs.length ? pairs : [{ key: '', value: '' }]
    } else {
      url.value = rawUrl
      queryParams.value = [{ key: '', value: '' }]
    }
  } catch {
    url.value = rawUrl
    queryParams.value = [{ key: '', value: '' }]
  }

  const bm = payload.body_mode || 'none'
  bodyMode.value = bm
  body.value = payload.body || ''

  if (payload.headers?.length) {
    headers.value = payload.headers.map((h) => ({ key: h.key || '', value: h.value || '' }))
    if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
  } else {
    headers.value = [{ key: '', value: '' }]
  }

  if (payload.form_fields?.length) {
    formFields.value = payload.form_fields.map((f) => ({ key: f.key || '', value: f.value || '' }))
    if (formFields.value.length === 0) formFields.value.push({ key: '', value: '' })
  } else {
    formFields.value = [{ key: '', value: '' }]
  }

  if (payload.multipart_parts?.length) {
    multipartParts.value = payload.multipart_parts.map((p) => ({
      key: p.key || '',
      kind: p.kind === 'file' ? 'file' : 'text',
      value: p.value || '',
      file_path: p.file_path || ''
    }))
    if (multipartParts.value.length === 0) {
      multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
    }
  } else {
    multipartParts.value = [{ key: '', kind: 'text', value: '', file_path: '' }]
  }

  activeTab.value = bm === 'none' || bm === '' ? 'params' : 'body'
}

const applyCurlImport = async () => {
  const text = (curlText.value || '').trim()
  if (!text) {
    emit('console', '[cURL] Paste a command first.')
    return
  }
  try {
    const payload = await ImportFromCurl(text)
    applyInputToForm(payload)
    closeCurlModal()
    emit('console', '[Import] Request imported from cURL.')
  } catch (e) {
    emit('console', `[cURL] ${e?.message || String(e)}`)
  }
}

const addQueryRow = () => queryParams.value.push({ key: '', value: '' })
const removeQueryRow = (i) => {
  queryParams.value.splice(i, 1)
  if (queryParams.value.length === 0) queryParams.value.push({ key: '', value: '' })
}

const addHeaderRow = () => headers.value.push({ key: '', value: '' })
const removeHeaderRow = (i) => {
  headers.value.splice(i, 1)
  if (headers.value.length === 0) headers.value.push({ key: '', value: '' })
}

const addFormFieldRow = () => formFields.value.push({ key: '', value: '' })
const removeFormFieldRow = (i) => {
  formFields.value.splice(i, 1)
  if (formFields.value.length === 0) formFields.value.push({ key: '', value: '' })
}

const addMultipartField = () => {
  multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
}
const removeMultipartRow = (i) => {
  multipartParts.value.splice(i, 1)
  if (multipartParts.value.length === 0) {
    multipartParts.value.push({ key: '', kind: 'text', value: '', file_path: '' })
  }
}

const pickMultipartFile = async (i) => {
  try {
    const path = await PickFileForBody()
    if (path) {
      multipartParts.value[i].file_path = path
      multipartParts.value[i].kind = 'file'
    }
  } catch (e) {
    emit('console', `[File] ${e?.message || String(e)}`)
  }
}

const handleSend = () => {
  const q = queryParams.value.filter((p) => (p.key || '').trim() !== '')
  const h = headers.value.filter((p) => (p.key || '').trim() !== '')

  const payload = {
    method: method.value,
    url: url.value,
    query_params: q,
    headers: h,
    body_mode: bodyMode.value,
    body: bodyMode.value === 'raw' ? body.value : '',
    form_fields:
      bodyMode.value === 'form_urlencoded'
        ? formFields.value.filter((p) => (p.key || '').trim() !== '')
        : undefined,
    multipart_parts:
      bodyMode.value === 'multipart'
        ? multipartParts.value
            .filter((p) => (p.key || '').trim() !== '')
            .map((p) => ({
              key: p.key.trim(),
              kind: p.kind === 'file' ? 'file' : 'text',
              value: p.kind === 'text' ? p.value : '',
              file_path: p.kind === 'file' ? (p.file_path || '').trim() : ''
            }))
        : undefined
  }
  const ws = props.activeWorkspaceId
  if (typeof ws === 'string' && ws.trim() !== '') {
    payload.workspace_id = ws.trim()
  }
  emit('send', payload)
}

const formatJsonBody = () => {
  const raw = (body.value ?? '').trim()
  if (!raw) {
    return
  }
  try {
    const parsed = JSON.parse(raw)
    body.value = JSON.stringify(parsed, null, 2)
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'Invalid JSON.'
    emit('console', `[Format JSON] ${msg}`)
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden border-b border-gray-800 bg-[#212121]">
    <div class="flex shrink-0 items-center gap-2 p-3">
      <select
        v-model="method"
        class="rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm font-bold text-green-500 outline-none"
      >
        <option>GET</option>
        <option>POST</option>
        <option>PUT</option>
        <option>PATCH</option>
        <option>DELETE</option>
        <option>HEAD</option>
        <option>OPTIONS</option>
      </select>
      <input
        v-model="url"
        type="text"
        class="min-w-0 flex-1 rounded border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-gray-200 outline-none focus:border-orange-500"
        placeholder="https://api.example.com/v1/resource"
        @keydown.enter.prevent="handleSend"
      />
      <button
        type="button"
        class="shrink-0 rounded bg-orange-600 px-6 py-2 text-sm font-bold text-white transition-all hover:bg-orange-700 active:scale-95"
        @click="handleSend"
      >
        Send
      </button>
    </div>

    <div class="shrink-0 border-t border-gray-800/80 px-3 pt-2 pb-1">
      <div class="flex items-end justify-between gap-3">
        <div class="min-w-0 flex-1">
          <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Request</div>
          <div class="mt-1 flex flex-wrap gap-1 text-xs font-semibold">
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'params' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'params'"
            >
              Query
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'headers' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'headers'"
            >
              Headers
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'body' ? 'bg-[#181818] text-orange-400' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'body'"
            >
              Body
            </button>
          </div>
        </div>
        <button
          type="button"
          class="shrink-0 rounded px-2 py-1 text-[10px] font-semibold uppercase tracking-wide text-gray-500 transition-colors hover:bg-gray-800/80 hover:text-orange-400"
          title="Import request or collection"
          @click="openImportMenu"
        >
          Import
        </button>
      </div>
    </div>

    <div class="flex min-h-0 flex-1 flex-col overflow-hidden border-t border-gray-800 bg-[#181818]">
      <div v-show="activeTab === 'params'" class="app-scrollbar min-h-0 flex-1 overflow-auto p-3 text-sm">
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-2 pr-2 font-medium">Key</th>
              <th class="pb-2 pr-2 font-medium">Value</th>
              <th class="w-10 pb-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in queryParams" :key="'q-' + i">
              <td class="pr-2 pb-1 align-top">
                <input v-model="row.key" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
              </td>
              <td class="pr-2 pb-1 align-top">
                <input v-model="row.value" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
              </td>
              <td class="pb-1 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  aria-label="Remove param"
                  title="Remove row"
                  @click="removeQueryRow(i)"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </td>
            </tr>
            <tr>
              <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
              <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                  aria-label="Add query param"
                  title="Add param"
                  @click="addQueryRow"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-show="activeTab === 'headers'" class="app-scrollbar min-h-0 flex-1 overflow-auto p-3 text-sm">
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="text-left text-gray-500">
              <th class="pb-2 pr-2 font-medium">Key</th>
              <th class="pb-2 pr-2 font-medium">Value</th>
              <th class="w-10 pb-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in headers" :key="'h-' + i">
              <td class="pr-2 pb-1 align-top">
                <input v-model="row.key" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
              </td>
              <td class="pr-2 pb-1 align-top">
                <input v-model="row.value" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
              </td>
              <td class="pb-1 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                  aria-label="Remove header"
                  title="Remove row"
                  @click="removeHeaderRow(i)"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </td>
            </tr>
            <tr>
              <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
              <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                <button
                  type="button"
                  class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                  aria-label="Add header"
                  title="Add header"
                  @click="addHeaderRow"
                >
                  <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-show="activeTab === 'body'" class="flex min-h-0 flex-1 flex-col p-3" style="min-height: 80px">
        <div class="mb-2 flex shrink-0 flex-wrap items-center gap-2">
          <label class="text-xs text-gray-500">Body type</label>
          <select
            v-model="bodyMode"
            class="rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
          >
            <option value="none">None</option>
            <option value="raw">Raw / JSON</option>
            <option value="form_urlencoded">x-www-form-urlencoded</option>
            <option value="multipart">form-data (multipart)</option>
          </select>
        </div>

        <!-- Raw -->
        <template v-if="bodyMode === 'raw'">
          <div class="mb-2 flex shrink-0 flex-wrap items-center justify-between gap-2">
            <span class="text-xs text-gray-500">Content (JSON or raw text)</span>
            <button
              type="button"
              class="rounded border border-gray-600 bg-gray-800 px-3 py-1 text-xs font-medium text-gray-200 hover:border-orange-500 hover:text-orange-300"
              title="Pretty-print JSON"
              @click="formatJsonBody"
            >
              Format JSON
            </button>
          </div>
          <JsonCodeMirror v-model="body" class="min-h-0 flex-1" />
        </template>

        <!-- none -->
        <div v-else-if="bodyMode === 'none'" class="flex flex-1 items-center justify-center rounded border border-dashed border-gray-700 py-8 text-sm text-gray-600">
          No request body (e.g. GET or no payload).
        </div>

        <!-- urlencoded -->
        <div v-else-if="bodyMode === 'form_urlencoded'" class="app-scrollbar min-h-0 flex-1 overflow-auto text-sm">
          <p class="mb-2 text-[11px] text-gray-500">Content-Type: application/x-www-form-urlencoded (set automatically unless overridden in Headers).</p>
          <table class="w-full border-collapse text-xs">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="pb-2 pr-2 font-medium">Key</th>
                <th class="pb-2 pr-2 font-medium">Value</th>
                <th class="w-10 pb-2" />
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in formFields" :key="'f-' + i">
                <td class="pr-2 pb-1 align-top">
                  <input v-model="row.key" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
                </td>
                <td class="pr-2 pb-1 align-top">
                  <input v-model="row.value" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
                </td>
                <td class="pb-1 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                    title="Remove row"
                    @click="removeFormFieldRow(i)"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </td>
              </tr>
              <tr>
                <td class="border-t border-gray-800/50 p-0 pt-2" colspan="2"></td>
                <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                    aria-label="Add form field"
                    title="Add field"
                    @click="addFormFieldRow"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- multipart -->
        <div v-else-if="bodyMode === 'multipart'" class="app-scrollbar min-h-0 flex-1 overflow-auto text-sm">
          <p class="mb-2 text-[11px] text-gray-500">
            multipart/form-data — boundary is sent with the request. Files: pick a local path.
          </p>
          <table class="w-full border-collapse text-xs">
            <thead>
              <tr class="text-left text-gray-500">
                <th class="pb-2 pr-1 font-medium">Key</th>
                <th class="w-24 pb-2 pr-1 font-medium">Type</th>
                <th class="pb-2 pr-2 font-medium">Value / File</th>
                <th class="w-10 pb-2" />
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in multipartParts" :key="'m-' + i">
                <td class="pr-1 pb-1 align-top">
                  <input v-model="row.key" class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200" />
                </td>
                <td class="pr-1 pb-1 align-top">
                  <select v-model="row.kind" class="w-full rounded border border-gray-700 bg-gray-900 px-1 py-1 text-gray-200">
                    <option value="text">Text</option>
                    <option value="file">File</option>
                  </select>
                </td>
                <td class="pr-2 pb-1 align-top">
                  <input
                    v-if="row.kind === 'text'"
                    v-model="row.value"
                    class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1 text-gray-200"
                    placeholder="Value"
                  />
                  <div v-else class="flex flex-col gap-1">
                    <input
                      :value="row.file_path"
                      readonly
                      class="w-full rounded border border-gray-700 bg-gray-800/80 px-2 py-1 text-[11px] text-gray-400"
                      placeholder="No file selected"
                    />
                    <button
                      type="button"
                      class="self-start rounded border border-gray-600 bg-gray-800 px-2 py-0.5 text-[11px] text-orange-400 hover:border-orange-500"
                      @click="pickMultipartFile(i)"
                    >
                      Choose file…
                    </button>
                  </div>
                </td>
                <td class="pb-1 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-500/15 hover:text-red-400"
                    title="Remove row"
                    @click="removeMultipartRow(i)"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </td>
              </tr>
              <tr>
                <td class="border-t border-gray-800/50 p-0 pt-2" colspan="3"></td>
                <td class="border-t border-gray-800/50 p-0 pt-2 align-middle">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-orange-500/15 hover:text-orange-400"
                    aria-label="Add multipart field"
                    title="Add field"
                    @click="addMultipartField"
                  >
                    <svg class="h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

  <Teleport to="#app">
    <div
      v-if="importMenuOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="import-menu-title"
    >
      <div
        class="w-full max-w-md rounded-lg border border-gray-700 bg-[#1f1f1f] p-4 shadow-xl"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0 flex-1">
            <h3 id="import-menu-title" class="text-sm font-semibold text-white">Import</h3>
            <p class="mt-1 text-[11px] leading-relaxed text-gray-500">Choose what you want to bring into the app.</p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded p-1.5 text-lg leading-none text-gray-500 hover:bg-gray-800 hover:text-gray-200"
            aria-label="Close"
            @click="closeImportMenu"
          >
            ×
          </button>
        </div>
        <div class="mt-4 flex flex-col gap-2">
          <button
            type="button"
            class="rounded border border-gray-600 bg-[#2a2a2a] px-3 py-2.5 text-left transition-colors hover:border-orange-500/40 hover:bg-gray-800/90"
            @click="chooseImportCurl"
          >
            <span class="block text-sm font-medium text-gray-100">Request from cURL</span>
            <span class="mt-0.5 block text-[10px] text-gray-500">Paste a curl command (e.g. from browser DevTools).</span>
          </button>
          <button
            type="button"
            class="rounded border border-gray-700 bg-[#252525] px-3 py-2.5 text-left transition-colors hover:border-gray-600 hover:bg-[#2a2a2a]"
            @click="chooseImportCollection"
          >
            <span class="block text-sm font-medium text-gray-300">Collection</span>
            <span class="mt-0.5 block text-[10px] text-gray-500">Postman collection, OpenAPI, etc. — coming soon.</span>
          </button>
        </div>
        <div class="mt-4 flex justify-end">
          <button
            type="button"
            class="rounded px-3 py-1.5 text-xs text-gray-400 hover:bg-gray-800 hover:text-gray-200"
            @click="closeImportMenu"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <Teleport to="#app">
    <div
      v-if="curlModalOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="curl-import-title"
    >
      <div class="flex max-h-[85vh] w-full max-w-2xl flex-col rounded-lg border border-gray-700 bg-[#1f1f1f] shadow-lg">
        <div class="flex items-start justify-between gap-2 border-b border-gray-700 px-4 py-3">
          <div class="min-w-0 flex-1 pr-2">
            <h3 id="curl-import-title" class="text-sm font-semibold text-white">Import from cURL</h3>
            <p class="mt-1 text-[11px] leading-relaxed text-gray-500">
              Paste a full curl command (e.g. from browser DevTools → Copy as cURL). Supports -X, -H, -d, --data-urlencode, -F,
              -G, -d @file, -u (Basic auth).
            </p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded p-1.5 text-lg leading-none text-gray-500 hover:bg-gray-800 hover:text-gray-200"
            aria-label="Close"
            @click="closeCurlModal"
          >
            ×
          </button>
        </div>
        <div class="min-h-0 flex-1 overflow-hidden p-4">
          <textarea
            v-model="curlText"
            class="app-scrollbar h-80 w-full resize-y rounded border border-gray-700 bg-gray-900 px-3 py-2 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
            spellcheck="false"
            placeholder="curl https://httpbin.org/get"
          />
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
          <button
            type="button"
            class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600"
            @click="closeCurlModal"
          >
            Cancel
          </button>
          <button
            type="button"
            class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700"
            @click="applyCurlImport"
          >
            Import
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
