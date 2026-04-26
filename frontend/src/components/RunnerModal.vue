<script setup>
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import * as RunnerAPI from '../../wailsjs/wailsjs/go/delivery/RunnerHandler'
import * as EnvAPI from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'
import * as FolderAPI from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import { EventsOn, EventsOff } from '../../wailsjs/wailsjs/runtime/runtime'
import RunnerRequestDetailModal from './RunnerRequestDetailModal.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  folderId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'console'])

/** Folder dropdown options (flattened tree). */
const folderOptions = ref(/** @type {Array<{ id: string, label: string }>} */ ([]))
const folderId = ref('')
/** Environment dropdown options. */
const envOptions = ref(/** @type {Array<{ id: string, name: string }>} */ ([]))
const envId = ref('')

const stopOnFail = ref(false)
const notes = ref('')

/** UI phases:
 *  - "idle" — config form visible, ready to launch.
 *  - "running" — request progress streaming via Wails events.
 *  - "done" — final report rendered.
 */
const phase = ref('idle')

/** Live progress stream (one entry per request done). */
const progress = ref(/** @type {Array<Record<string, any>>} */ ([]))
/** Final result (RunnerRunDetail). */
const result = ref(/** @type {Record<string, any> | null} */ (null))
/** Active run id (set on runner:started). */
const runId = ref('')

/** Recent runs panel (folded by default to keep modal focused). */
const recentOpen = ref(false)
const recentRuns = ref(/** @type {Array<Record<string, any>>} */ ([]))
const recentLoading = ref(false)

/**
 * Detail panel state — clicking a request row in the run table opens a
 * modal (RunnerRequestDetailModal) showing tabs for Request / Response /
 * Tests, mirroring the request-history snapshot UX. We track the row id and
 * its 1-based position so the header can show "#order".
 */
const detailRow = ref(/** @type {Record<string, any> | null} */ (null))
const detailIndex = ref(0)

/**
 * Confirmation modal for deleting a recent run — replaces native window.confirm
 * with the same overlay style used elsewhere in the app (Folder/Request delete).
 */
const deleteConfirm = ref({
  open: false,
  /** @type {{ id: string, label: string } | null} */
  target: null,
  loading: false
})

const totals = computed(() => {
  const r = result.value
  const total = r?.total_count ?? progress.value.length
  const passed = r?.passed_count ?? progress.value.filter((p) => p.status === 'passed').length
  const failed = r?.failed_count ?? progress.value.filter((p) => p.status === 'failed').length
  const errored = r?.error_count ?? progress.value.filter((p) => p.status === 'errored').length
  return { total, passed, failed, errored }
})

let unsubStarted = null
let unsubRequest = null
let unsubFinished = null

function bindEvents() {
  unsubStarted = EventsOn('runner:started', (payload) => {
    runId.value = payload?.run_id || ''
  })
  unsubRequest = EventsOn('runner:request', (payload) => {
    const row = payload?.request
    if (!row) return
    progress.value = [...progress.value, row]
  })
  unsubFinished = EventsOn('runner:finished', async (payload) => {
    const id = payload?.run_id || runId.value
    if (id) {
      try {
        const detail = await RunnerAPI.GetRun(id)
        result.value = detail
        // Re-hydrate from the persisted detail so request/response snapshots
        // are present even if the live progress stream was skipped or the
        // browser tab was sleeping when individual `runner:request` events
        // landed. The DB is the source of truth post-run.
        if (Array.isArray(detail?.requests) && detail.requests.length > 0) {
          progress.value = detail.requests
        }
      } catch (e) {
        emit('console', `[Runner] Could not load run detail: ${e?.message || String(e)}`)
      }
    }
    phase.value = 'done'
    refreshRecent()
  })
}

function unbindEvents() {
  if (unsubStarted) {
    try { unsubStarted() } catch { /* noop */ }
    unsubStarted = null
  }
  if (unsubRequest) {
    try { unsubRequest() } catch { /* noop */ }
    unsubRequest = null
  }
  if (unsubFinished) {
    try { unsubFinished() } catch { /* noop */ }
    unsubFinished = null
  }
  try {
    EventsOff('runner:started')
    EventsOff('runner:request')
    EventsOff('runner:finished')
  } catch { /* noop */ }
}

async function loadFolderOptions() {
  folderOptions.value = []
  try {
    const roots = await FolderAPI.ListRootFolders()
    const list = Array.isArray(roots) ? roots : []
    const out = []
    async function walk(id, label) {
      const children = await FolderAPI.ListChildFolders(id)
      const ch = Array.isArray(children) ? children : []
      for (const c of ch) {
        const lbl = `${label} / ${c.name}`
        out.push({ id: c.id, label: lbl })
        await walk(c.id, lbl)
      }
    }
    for (const r of list) {
      out.push({ id: r.id, label: r.name })
      await walk(r.id, r.name)
    }
    folderOptions.value = out
  } catch (e) {
    emit('console', `[Runner] Could not load folders: ${e?.message || String(e)}`)
  }
}

async function loadEnvOptions() {
  envOptions.value = []
  try {
    const list = await EnvAPI.List()
    const arr = Array.isArray(list) ? list : []
    envOptions.value = arr.map((e) => ({ id: e.id, name: e.name }))
    const active = arr.find((e) => e.is_active)
    if (active && !envId.value) envId.value = String(active.id)
  } catch (e) {
    emit('console', `[Runner] Could not load environments: ${e?.message || String(e)}`)
  }
}

async function refreshRecent() {
  recentLoading.value = true
  try {
    const list = await RunnerAPI.ListRecentRuns(20)
    recentRuns.value = Array.isArray(list) ? list : []
  } catch (e) {
    emit('console', `[Runner] Could not list recent runs: ${e?.message || String(e)}`)
  } finally {
    recentLoading.value = false
  }
}

watch(
  () => props.open,
  async (v) => {
    if (!v) return
    await loadFolderOptions()
    await loadEnvOptions()
    if (props.folderId && folderOptions.value.some((o) => o.id === props.folderId)) {
      folderId.value = props.folderId
    } else if (!folderId.value && folderOptions.value.length > 0) {
      folderId.value = folderOptions.value[0].id
    }
    refreshRecent()
  },
  { immediate: false }
)

watch(
  () => props.folderId,
  (v) => {
    if (v && folderOptions.value.some((o) => o.id === v)) folderId.value = v
  }
)

onMounted(() => {
  bindEvents()
})

onUnmounted(() => {
  unbindEvents()
})

function resetRun() {
  progress.value = []
  result.value = null
  runId.value = ''
  closeRequestDetail()
}

async function launch() {
  if (!folderId.value) {
    emit('console', '[Runner] Choose a folder to run.')
    return
  }
  resetRun()
  phase.value = 'running'
  try {
    await RunnerAPI.RunFolder({
      folder_id: folderId.value,
      environment_id: envId.value || null,
      stop_on_fail: !!stopOnFail.value,
      notes: notes.value || ''
    })
  } catch (e) {
    phase.value = 'idle'
    emit('console', `[Runner] ${e?.message || String(e)}`)
  }
}

async function cancel() {
  try {
    await RunnerAPI.CancelRun()
    emit('console', '[Runner] Cancel requested.')
  } catch (e) {
    emit('console', `[Runner] Cancel failed: ${e?.message || String(e)}`)
  }
}

function close() {
  if (phase.value === 'running') {
    emit('console', '[Runner] Cancel the run before closing.')
    return
  }
  emit('close')
}

async function loadRecent(id) {
  if (!id) return
  try {
    const detail = await RunnerAPI.GetRun(id)
    result.value = detail
    progress.value = Array.isArray(detail?.requests) ? detail.requests : []
    runId.value = id
    phase.value = 'done'
    closeRequestDetail()
    // Collapse the recent panel so the loaded report uses the available space —
    // otherwise a long history list keeps the table cramped at the top.
    recentOpen.value = false
  } catch (e) {
    emit('console', `[Runner] Could not load run: ${e?.message || String(e)}`)
  }
}

async function exportReport(format) {
  const id = runId.value || result.value?.id
  if (!id) {
    emit('console', '[Runner] Run a folder first to export.')
    return
  }
  try {
    const path = await RunnerAPI.ExportRunReport(id, format)
    if (path) emit('console', `[Runner] Report saved: ${path}`)
  } catch (e) {
    emit('console', `[Runner] Export failed: ${e?.message || String(e)}`)
  }
}

function requestDeleteRecent(run) {
  if (!run?.id) return
  const label = run.folder_name
    ? `${run.folder_name} · ${run.started_at || ''}`
    : (run.started_at || run.id)
  deleteConfirm.value = {
    open: true,
    target: { id: run.id, label },
    loading: false
  }
}

function closeDeleteConfirm() {
  if (deleteConfirm.value.loading) return
  deleteConfirm.value = { open: false, target: null, loading: false }
}

async function confirmDeleteRecent() {
  const t = deleteConfirm.value.target
  if (!t?.id) return
  deleteConfirm.value = { ...deleteConfirm.value, loading: true }
  try {
    await RunnerAPI.DeleteRun(t.id)
    if (runId.value === t.id) {
      resetRun()
      phase.value = 'idle'
    }
    deleteConfirm.value = { open: false, target: null, loading: false }
    refreshRecent()
  } catch (e) {
    emit('console', `[Runner] Delete failed: ${e?.message || String(e)}`)
    deleteConfirm.value = { ...deleteConfirm.value, loading: false }
  }
}

function openRequestDetail(row, index) {
  if (!row) return
  detailRow.value = row
  detailIndex.value = (index ?? row.sort_order ?? 0) || 0
}

function closeRequestDetail() {
  detailRow.value = null
  detailIndex.value = 0
}

function statusBadgeClass(status) {
  switch (status) {
    case 'passed':
      return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
    case 'failed':
      return 'bg-red-500/15 text-red-300 border-red-500/30'
    case 'errored':
      return 'bg-amber-500/15 text-amber-300 border-amber-500/30'
    case 'skipped':
      return 'bg-gray-500/15 text-gray-300 border-gray-500/30'
    default:
      return 'bg-gray-500/15 text-gray-300 border-gray-500/30'
  }
}

function fmtDuration(ms) {
  if (ms == null || isNaN(ms)) return '—'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

function fmtSize(bytes) {
  if (bytes == null || isNaN(bytes)) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}
</script>

<template>
  <Teleport to="#app">
    <div
      v-if="open"
      class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 px-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="runner-modal-title"
    >
      <div class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-lg border border-gray-700 bg-[#1a1a1a] shadow-2xl">
        <div class="flex shrink-0 items-center justify-between gap-3 border-b border-gray-700 bg-[#212121] px-4 py-3">
          <div>
            <h2 id="runner-modal-title" class="text-sm font-semibold text-white">Collection Runner</h2>
            <p class="mt-0.5 text-[11px] text-gray-500">
              Run all saved requests in a folder sequentially. Captures chain values into the active environment or in-memory bag for later requests.
            </p>
          </div>
          <button
            type="button"
            class="rounded border border-gray-600 bg-gray-800 px-2 py-1 text-xs text-gray-200 hover:border-orange-500/50 disabled:opacity-50"
            :disabled="phase === 'running'"
            @click="close"
          >
            Close
          </button>
        </div>

        <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
          <!-- Config form -->
          <div class="grid shrink-0 grid-cols-1 gap-3 md:grid-cols-2">
            <div>
              <label class="mb-1 block text-[10px] font-medium uppercase tracking-wide text-gray-500">Folder</label>
              <select
                v-model="folderId"
                :disabled="phase === 'running'"
                class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
              >
                <option v-for="opt in folderOptions" :key="opt.id" :value="opt.id">{{ opt.label }}</option>
              </select>
            </div>
            <div>
              <label class="mb-1 block text-[10px] font-medium uppercase tracking-wide text-gray-500">Environment</label>
              <select
                v-model="envId"
                :disabled="phase === 'running'"
                class="w-full rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
              >
                <option value="">No environment</option>
                <option v-for="opt in envOptions" :key="opt.id" :value="opt.id">{{ opt.name }}</option>
              </select>
            </div>
            <div class="md:col-span-2 flex flex-wrap items-center gap-3">
              <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-300">
                <input v-model="stopOnFail" type="checkbox" :disabled="phase === 'running'" class="rounded border-gray-600" />
                Stop on first failure
              </label>
              <input
                v-model="notes"
                type="text"
                :disabled="phase === 'running'"
                placeholder="Notes (optional)"
                class="flex-1 rounded border border-gray-700 bg-gray-900 px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
              />
            </div>
          </div>

          <div class="mt-3 flex shrink-0 items-center gap-2">
            <button
              v-if="phase !== 'running'"
              type="button"
              class="rounded bg-orange-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
              :disabled="!folderId"
              @click="launch"
            >
              {{ phase === 'done' ? 'Run again' : 'Run folder' }}
            </button>
            <button
              v-else
              type="button"
              class="rounded border border-red-500/40 bg-red-500/10 px-4 py-1.5 text-xs font-semibold text-red-300 hover:bg-red-500/20"
              @click="cancel"
            >
              Cancel run
            </button>
            <span v-if="phase === 'running'" class="text-[11px] text-gray-400">
              Running… {{ progress.length }} done
            </span>
            <div v-if="phase === 'done'" class="ml-auto flex items-center gap-1">
              <button
                type="button"
                class="rounded border border-gray-600 bg-[#2a2a2a] px-3 py-1.5 text-[11px] font-semibold text-gray-200 hover:border-orange-500/50 hover:text-orange-300"
                title="Save report as JSON"
                @click="exportReport('json')"
              >
                Export JSON
              </button>
              <button
                type="button"
                class="rounded border border-gray-600 bg-[#2a2a2a] px-3 py-1.5 text-[11px] font-semibold text-gray-200 hover:border-orange-500/50 hover:text-orange-300"
                title="Save report as Markdown"
                @click="exportReport('md')"
              >
                Export Markdown
              </button>
            </div>
          </div>

          <!-- Totals -->
          <div v-if="phase !== 'idle'" class="mt-3 shrink-0 rounded border border-gray-700 bg-[#141414] px-3 py-2 text-xs">
            <div class="flex flex-wrap items-center gap-3">
              <span class="text-gray-500">Total: <span class="text-gray-200">{{ totals.total }}</span></span>
              <span class="text-emerald-400">Passed: {{ totals.passed }}</span>
              <span class="text-red-400">Failed: {{ totals.failed }}</span>
              <span class="text-amber-400">Errored: {{ totals.errored }}</span>
              <span v-if="result?.duration_ms != null" class="ml-auto text-gray-500">
                {{ fmtDuration(result.duration_ms) }}
              </span>
            </div>
          </div>

          <!-- Stream / report -->
          <div class="mt-3 min-h-0 flex-1 overflow-hidden rounded border border-gray-800 bg-[#141414]">
            <div class="app-scrollbar h-full overflow-auto p-2">
              <div v-if="progress.length === 0 && phase === 'idle'" class="px-2 py-6 text-center text-[11px] text-gray-500">
                Configure and click <span class="text-orange-400">Run folder</span> to start.
              </div>
              <table v-else class="w-full text-xs">
                <thead class="sticky top-0 bg-[#141414] text-gray-500">
                  <tr class="text-left">
                    <th class="px-2 py-1 font-medium">#</th>
                    <th class="px-2 py-1 font-medium">Status</th>
                    <th class="px-2 py-1 font-medium">Method</th>
                    <th class="px-2 py-1 font-medium">Request</th>
                    <th class="px-2 py-1 font-medium">Code</th>
                    <th class="px-2 py-1 font-medium">Duration</th>
                    <th class="px-2 py-1 font-medium">Size</th>
                    <th class="px-2 py-1 font-medium">Tests</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(row, i) in progress"
                    :key="row.id || i"
                    class="cursor-pointer border-t border-gray-800/50 align-top hover:bg-gray-800/40"
                    role="button"
                    tabindex="0"
                    title="Click to view request detail"
                    @click="openRequestDetail(row, row.sort_order ?? i + 1)"
                    @keydown.enter.prevent="openRequestDetail(row, row.sort_order ?? i + 1)"
                    @keydown.space.prevent="openRequestDetail(row, row.sort_order ?? i + 1)"
                  >
                    <td class="px-2 py-1 text-gray-500">{{ row.sort_order ?? i + 1 }}</td>
                    <td class="px-2 py-1">
                      <span
                        class="inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
                        :class="statusBadgeClass(row.status)"
                      >
                        {{ row.status }}
                      </span>
                    </td>
                    <td class="px-2 py-1 font-mono text-orange-300">{{ row.method }}</td>
                    <td class="px-2 py-1">
                      <div class="font-medium text-gray-200">{{ row.request_name || '—' }}</div>
                      <div class="truncate text-[10px] text-gray-500" :title="row.url">{{ row.url }}</div>
                      <div v-if="row.error_message" class="mt-0.5 text-[10px] text-red-400">{{ row.error_message }}</div>
                    </td>
                    <td class="px-2 py-1 font-mono text-gray-300">{{ row.status_code || '—' }}</td>
                    <td class="px-2 py-1 text-gray-300">{{ fmtDuration(row.duration_ms) }}</td>
                    <td class="px-2 py-1 text-gray-300">{{ fmtSize(row.response_size_bytes) }}</td>
                    <td class="px-2 py-1 text-[10px]">
                      <span v-if="(row.assertions || []).length === 0 && (row.captures || []).length === 0" class="text-gray-600">—</span>
                      <span v-else class="inline-flex items-center gap-1">
                        <span
                          v-if="(row.assertions || []).length"
                          class="rounded border border-gray-700 px-1 py-0.5 text-gray-400"
                          :title="`${(row.assertions || []).filter((a) => a.passed).length} passed / ${(row.assertions || []).length} assertions`"
                        >
                          <span class="text-emerald-400">{{ (row.assertions || []).filter((a) => a.passed).length }}</span>
                          /
                          <span class="text-gray-300">{{ (row.assertions || []).length }}</span>
                        </span>
                        <span
                          v-if="(row.captures || []).length"
                          class="rounded border border-gray-700 px-1 py-0.5 text-gray-400"
                          :title="`${(row.captures || []).filter((c) => c.captured).length} captured / ${(row.captures || []).length} captures`"
                        >
                          <span class="text-sky-400">{{ (row.captures || []).filter((c) => c.captured).length }}</span>
                          /
                          <span class="text-gray-300">{{ (row.captures || []).length }}</span>
                        </span>
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Recent runs -->
          <div class="mt-3 shrink-0 rounded border border-gray-800 bg-[#141414]">
            <button
              type="button"
              class="flex w-full items-center justify-between px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-gray-400 hover:text-orange-300"
              @click="recentOpen = !recentOpen"
            >
              Recent runs ({{ recentRuns.length }})
              <span class="text-gray-500">{{ recentOpen ? '▲' : '▼' }}</span>
            </button>
            <div v-if="recentOpen" class="border-t border-gray-800/80 px-2 py-2">
              <div v-if="recentLoading" class="px-2 py-2 text-[11px] text-gray-500">Loading…</div>
              <div v-else-if="recentRuns.length === 0" class="px-2 py-2 text-[11px] text-gray-500">No runs yet.</div>
              <div v-else class="space-y-1">
                <div
                  v-for="r in recentRuns"
                  :key="r.id"
                  class="flex items-center gap-2 rounded px-2 py-1 text-[11px] hover:bg-gray-800"
                >
                  <span class="font-mono text-gray-500">{{ r.started_at }}</span>
                  <span class="font-medium text-gray-200">{{ r.folder_name || 'Folder' }}</span>
                  <span class="text-gray-500">·</span>
                  <span class="text-emerald-400">{{ r.passed_count }}</span>
                  /
                  <span class="text-red-400">{{ r.failed_count }}</span>
                  /
                  <span class="text-amber-400">{{ r.error_count }}</span>
                  <span class="ml-auto text-gray-500">{{ fmtDuration(r.duration_ms) }}</span>
                  <button
                    type="button"
                    class="rounded border border-gray-700 px-2 py-0.5 text-[10px] text-orange-300 hover:border-orange-500/50"
                    @click="loadRecent(r.id)"
                  >
                    Open
                  </button>
                  <button
                    type="button"
                    class="rounded border border-gray-700 px-2 py-0.5 text-[10px] text-red-400 hover:border-red-500/50"
                    @click="requestDeleteRecent(r)"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <RunnerRequestDetailModal
      :open="detailRow !== null"
      :item="detailRow"
      :index="detailIndex"
      @close="closeRequestDetail"
    />

    <!-- Delete-run confirmation: same overlay style as folder/request delete -->
    <div
      v-if="deleteConfirm.open"
      class="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 px-4"
      role="presentation"
      data-runner-delete-confirm
      @mousedown.self="closeDeleteConfirm"
    >
      <div
        class="w-full max-w-md rounded-lg border border-gray-600 bg-[#1f1f1f] shadow-xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="runner-delete-title"
        @mousedown.stop
      >
        <div class="border-b border-gray-700 px-4 py-3">
          <h3 id="runner-delete-title" class="text-sm font-semibold text-white">Delete run history</h3>
        </div>
        <div class="p-4 text-sm text-gray-300">
          <p>
            Remove this runner history entry?
          </p>
          <p v-if="deleteConfirm.target?.label" class="mt-2 truncate font-medium text-white" :title="deleteConfirm.target.label">
            {{ deleteConfirm.target.label }}
          </p>
          <p class="mt-2 text-[11px] text-gray-500">
            This deletes the saved summary and all per-request results for the run. It cannot be undone.
          </p>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-700 px-4 py-3">
          <button
            type="button"
            class="rounded bg-gray-700 px-3 py-1.5 text-xs text-white hover:bg-gray-600 disabled:opacity-50"
            :disabled="deleteConfirm.loading"
            @click="closeDeleteConfirm"
          >
            Cancel
          </button>
          <button
            type="button"
            class="rounded bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
            :disabled="deleteConfirm.loading"
            @click="confirmDeleteRecent"
          >
            {{ deleteConfirm.loading ? 'Deleting…' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
