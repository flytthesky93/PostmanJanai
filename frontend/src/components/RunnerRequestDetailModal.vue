<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import JsonCodeMirror from './JsonCodeMirror.vue'

/**
 * RunnerRequestDetailModal — read-only inspector for a single request that
 * ran inside a folder run. Mirrors the look-and-feel of
 * `HistoryDetailModal.vue` (tabs, header, scroll layout) and now shares the
 * same data contract for raw request/response payloads (Phase 8.1):
 *   - request_headers_json, request_body  → resolved request snapshot
 *   - response_headers_json, response_body → response received from the wire
 * Captures + assertion outcomes still get their own "Tests" tab so the user
 * can see chained variables and pass/fail evidence side-by-side with the
 * raw payload.
 */

const props = defineProps({
  open: { type: Boolean, default: false },
  /** @type {import('vue').PropType<Record<string, any> | null>} */
  item: { type: Object, default: null },
  /** Position of the row in the run (1-based) for the header label. */
  index: { type: Number, default: 0 }
})

const emit = defineEmits(['close'])

const activeTab = ref('request')

watch(
  () => props.open,
  (o) => {
    if (o) activeTab.value = 'request'
  }
)

function prettyJsonText(raw) {
  if (raw == null || raw === '') return ''
  const s = String(raw)
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}

function headersJsonToLines(jsonStr) {
  if (jsonStr == null || String(jsonStr).trim() === '') return '(none)'
  try {
    const arr = JSON.parse(String(jsonStr))
    if (Array.isArray(arr)) {
      const lines = arr.map((h) => `${h.key ?? ''}: ${h.value ?? ''}`)
      return lines.length ? lines.join('\n') : '(none)'
    }
  } catch {
    /* fall through */
  }
  return String(jsonStr)
}

const requestHeadersText = computed(() => headersJsonToLines(props.item?.request_headers_json))
const responseHeadersText = computed(() => headersJsonToLines(props.item?.response_headers_json))
const requestBodyPretty = computed(() => prettyJsonText(props.item?.request_body))
const responseBodyPretty = computed(() => prettyJsonText(props.item?.response_body))

const hasRequestBody = computed(() => {
  const v = props.item?.request_body
  return v != null && String(v).trim() !== ''
})
const hasResponseBody = computed(() => {
  const v = props.item?.response_body
  return v != null && String(v).trim() !== ''
})

const assertions = computed(() => {
  const arr = props.item?.assertions
  return Array.isArray(arr) ? arr : []
})

const captures = computed(() => {
  const arr = props.item?.captures
  return Array.isArray(arr) ? arr : []
})

const assertionsPassed = computed(() => assertions.value.filter((a) => a.passed).length)
const assertionsFailed = computed(() => assertions.value.length - assertionsPassed.value)
const capturesOk = computed(() => captures.value.filter((c) => c.captured).length)
const capturesMissed = computed(() => captures.value.length - capturesOk.value)

const testsTabBadge = computed(() => {
  const total = assertions.value.length + captures.value.length
  if (total === 0) return ''
  const failed = assertionsFailed.value + capturesMissed.value
  return failed > 0 ? `${failed} failed` : `${total} ok`
})

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

function truncatePreview(value, max = 240) {
  if (value == null) return ''
  const s = String(value)
  if (s.length <= max) return s
  return s.slice(0, max) + '…'
}

function onKeydown(e) {
  if (e.key === 'Escape' && props.open) {
    e.preventDefault()
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="#app">
    <div
      v-if="open"
      class="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 px-3 py-6"
      role="dialog"
      aria-modal="true"
      aria-labelledby="runner-request-detail-title"
      @click.self="emit('close')"
    >
      <div
        class="flex max-h-[min(92vh,900px)] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-gray-600 bg-[#1f1f1f] shadow-xl"
        @click.stop
      >
        <div class="flex shrink-0 items-start justify-between gap-2 border-b border-gray-700 px-4 py-3">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h2 id="runner-request-detail-title" class="truncate text-sm font-semibold text-white">
                {{ item?.request_name || 'Request' }}
              </h2>
              <span v-if="index" class="rounded bg-gray-700/70 px-1.5 py-0.5 font-mono text-[10px] text-gray-200">
                #{{ index }}
              </span>
              <span
                v-if="item?.status"
                class="inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
                :class="statusBadgeClass(item.status)"
              >
                {{ item.status }}
              </span>
            </div>
            <p class="mt-0.5 truncate text-[11px] text-gray-500" :title="item?.url">
              <span class="font-mono text-orange-300">{{ item?.method || '—' }}</span>
              <span class="ml-1">{{ item?.url || '' }}</span>
            </p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-700 hover:text-white"
            aria-label="Close"
            @click="emit('close')"
          >
            Close
          </button>
        </div>

        <template v-if="item">
          <div class="shrink-0 border-b border-gray-800 px-4 pt-2">
            <div class="flex flex-wrap gap-1 text-xs font-semibold">
              <button
                type="button"
                class="rounded-t px-3 py-2"
                :class="activeTab === 'request' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
                @click="activeTab = 'request'"
              >
                Request
              </button>
              <button
                type="button"
                class="rounded-t px-3 py-2"
                :class="activeTab === 'response' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
                @click="activeTab = 'response'"
              >
                Response
              </button>
              <button
                type="button"
                class="flex items-center gap-1.5 rounded-t px-3 py-2"
                :class="activeTab === 'tests' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
                @click="activeTab = 'tests'"
              >
                Tests
                <span
                  v-if="testsTabBadge"
                  class="rounded border border-gray-700 px-1.5 py-0.5 text-[10px] font-normal"
                  :class="(assertionsFailed + capturesMissed) > 0 ? 'border-red-500/40 text-red-300' : 'border-emerald-500/40 text-emerald-300'"
                >
                  {{ testsTabBadge }}
                </span>
              </button>
            </div>
          </div>

          <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-4 text-xs">
            <template v-if="activeTab === 'request'">
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <span class="rounded bg-gray-700 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white">
                  {{ item.method || '—' }}
                </span>
                <span v-if="item.request_name" class="truncate text-[11px] text-gray-300">{{ item.request_name }}</span>
              </div>
              <div class="mb-3 break-all font-mono text-gray-300">{{ item.url || '—' }}</div>

              <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Request headers</div>
              <pre
                class="mb-4 max-h-40 overflow-auto rounded border border-gray-700 bg-[#121212] p-2 font-mono text-[11px] leading-relaxed text-gray-300 whitespace-pre-wrap break-words"
                >{{ requestHeadersText }}</pre
              >

              <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Request body</div>
              <div
                v-if="hasRequestBody"
                class="overflow-hidden rounded border border-gray-700"
                style="height: min(40vh, 320px); min-height: 120px"
              >
                <JsonCodeMirror :model-value="requestBodyPretty" :read-only="true" class="h-full min-h-0 flex-1" />
              </div>
              <pre
                v-else
                class="mb-4 rounded border border-gray-700 bg-[#121212] p-3 text-[11px] italic text-gray-500"
                >(no request body)</pre
              >

              <div class="mt-4 mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Run metadata</div>
              <ul class="space-y-1 rounded border border-gray-700 bg-[#121212] p-3 text-[11px] text-gray-300">
                <li>
                  <span class="text-gray-500">Saved request id:</span>
                  <span class="ml-1 font-mono">{{ item.request_id || '— (deleted or ad-hoc)' }}</span>
                </li>
                <li>
                  <span class="text-gray-500">Run id:</span>
                  <span class="ml-1 font-mono">{{ item.run_id || '—' }}</span>
                </li>
                <li v-if="item.sort_order != null">
                  <span class="text-gray-500">Order:</span>
                  <span class="ml-1 font-mono">#{{ item.sort_order }}</span>
                </li>
                <li v-if="item.created_at">
                  <span class="text-gray-500">Recorded at:</span>
                  <span class="ml-1">{{ item.created_at }}</span>
                </li>
              </ul>

              <p class="mt-3 text-[10px] italic leading-snug text-gray-500">
                Headers and body are stored after variable substitution — they reflect what was actually sent to the server, not the saved-request template.
              </p>
            </template>

            <template v-else-if="activeTab === 'response'">
              <div class="mb-3 flex flex-wrap items-center gap-3 text-gray-300">
                <span>
                  <span class="text-gray-500">Status:</span>
                  <span
                    class="ml-1 font-mono"
                    :class="item.status_code && item.status_code < 400 ? 'text-emerald-300' : 'text-red-300'"
                  >
                    {{ item.status_code || '—' }}
                  </span>
                </span>
                <span>
                  <span class="text-gray-500">Time:</span>
                  <span class="ml-1">{{ fmtDuration(item.duration_ms) }}</span>
                </span>
                <span>
                  <span class="text-gray-500">Size:</span>
                  <span class="ml-1">{{ fmtSize(item.response_size_bytes) }}</span>
                </span>
                <span class="ml-auto">
                  <span class="text-gray-500">Outcome:</span>
                  <span
                    class="ml-1 inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
                    :class="statusBadgeClass(item.status)"
                  >
                    {{ item.status }}
                  </span>
                </span>
              </div>

              <div v-if="item.error_message" class="mb-3 rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-[11px] text-red-200">
                <div class="mb-1 text-[10px] font-bold uppercase tracking-wider text-red-300">Error</div>
                <div class="whitespace-pre-wrap break-words font-mono">{{ item.error_message }}</div>
              </div>

              <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Response headers</div>
              <pre
                class="mb-4 max-h-40 overflow-auto rounded border border-gray-700 bg-[#121212] p-2 font-mono text-[11px] leading-relaxed text-gray-300 whitespace-pre-wrap break-words"
                >{{ responseHeadersText }}</pre
              >

              <div class="mb-2 flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">
                <span>Response body</span>
                <span
                  v-if="item.body_truncated"
                  class="rounded border border-amber-500/40 px-1.5 py-0.5 text-[10px] font-normal normal-case text-amber-300"
                  title="Response was larger than the executor max-body limit and got truncated when stored"
                >
                  truncated
                </span>
              </div>
              <div
                v-if="hasResponseBody"
                class="overflow-hidden rounded border border-gray-700"
                style="height: min(40vh, 320px); min-height: 120px"
              >
                <JsonCodeMirror :model-value="responseBodyPretty" :read-only="true" class="h-full min-h-0 flex-1" />
              </div>
              <pre
                v-else
                class="rounded border border-gray-700 bg-[#121212] p-3 text-[11px] italic text-gray-500"
                >(no response body)</pre
              >
            </template>

            <template v-else-if="activeTab === 'tests'">
              <div v-if="assertions.length === 0 && captures.length === 0" class="rounded border border-gray-700 bg-[#121212] p-4 text-center text-[11px] text-gray-500">
                No assertions or captures were configured for this request.
              </div>

              <div v-if="assertions.length" class="mb-4 rounded border border-gray-700 bg-[#121212]">
                <div class="flex items-center justify-between border-b border-gray-800 px-3 py-2 text-[10px] uppercase tracking-wider text-gray-500">
                  <span>Assertions ({{ assertions.length }})</span>
                  <span class="font-normal normal-case text-gray-500">
                    <span class="text-emerald-400">{{ assertionsPassed }}</span>
                    passed /
                    <span class="text-red-400">{{ assertionsFailed }}</span>
                    failed
                  </span>
                </div>
                <table class="w-full text-[11px]">
                  <thead class="text-gray-500">
                    <tr class="text-left">
                      <th class="px-3 py-1 font-medium">Pass</th>
                      <th class="px-3 py-1 font-medium">Name</th>
                      <th class="px-3 py-1 font-medium">Source</th>
                      <th class="px-3 py-1 font-medium">Op</th>
                      <th class="px-3 py-1 font-medium">Expected</th>
                      <th class="px-3 py-1 font-medium">Actual</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(a, ai) in assertions"
                      :key="`a-${ai}`"
                      class="border-t border-gray-800/60 align-top"
                    >
                      <td class="px-3 py-1">
                        <span :class="a.passed ? 'text-emerald-400' : 'text-red-400'">
                          {{ a.passed ? '✓' : '✗' }}
                        </span>
                      </td>
                      <td class="px-3 py-1 text-gray-200">{{ a.name || '—' }}</td>
                      <td class="px-3 py-1 text-gray-400">
                        {{ a.source }}<span v-if="a.expression"> · <span class="font-mono text-gray-500">{{ a.expression }}</span></span>
                      </td>
                      <td class="px-3 py-1 font-mono text-gray-400">{{ a.operator }}</td>
                      <td class="px-3 py-1 font-mono text-gray-300" :title="a.expected">{{ truncatePreview(a.expected, 80) }}</td>
                      <td class="px-3 py-1 font-mono text-gray-300" :title="a.actual">{{ truncatePreview(a.actual, 200) }}</td>
                    </tr>
                  </tbody>
                </table>
                <ul
                  v-if="assertions.some((a) => a.error_message)"
                  class="border-t border-gray-800 bg-[#0e0e0e] px-3 py-2 text-[10px] text-red-300"
                >
                  <li v-for="(a, ai) in assertions.filter((x) => x.error_message)" :key="`aerr-${ai}`">
                    <span class="font-semibold">{{ a.name || '(unnamed)' }}:</span> {{ a.error_message }}
                  </li>
                </ul>
              </div>

              <div v-if="captures.length" class="rounded border border-gray-700 bg-[#121212]">
                <div class="flex items-center justify-between border-b border-gray-800 px-3 py-2 text-[10px] uppercase tracking-wider text-gray-500">
                  <span>Captures ({{ captures.length }})</span>
                  <span class="font-normal normal-case text-gray-500">
                    <span class="text-sky-400">{{ capturesOk }}</span>
                    captured /
                    <span class="text-amber-400">{{ capturesMissed }}</span>
                    missed
                  </span>
                </div>
                <table class="w-full text-[11px]">
                  <thead class="text-gray-500">
                    <tr class="text-left">
                      <th class="px-3 py-1 font-medium">OK</th>
                      <th class="px-3 py-1 font-medium">Name</th>
                      <th class="px-3 py-1 font-medium">Source</th>
                      <th class="px-3 py-1 font-medium">Target</th>
                      <th class="px-3 py-1 font-medium">Value</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(c, ci) in captures"
                      :key="`c-${ci}`"
                      class="border-t border-gray-800/60 align-top"
                    >
                      <td class="px-3 py-1">
                        <span :class="c.captured ? 'text-sky-400' : 'text-amber-400'">
                          {{ c.captured ? '✓' : '○' }}
                        </span>
                      </td>
                      <td class="px-3 py-1 text-gray-200">{{ c.name || '—' }}</td>
                      <td class="px-3 py-1 text-gray-400">
                        {{ c.source }}<span v-if="c.expression"> · <span class="font-mono text-gray-500">{{ c.expression }}</span></span>
                      </td>
                      <td class="px-3 py-1 font-mono text-gray-400">
                        {{ c.target_scope }}<span v-if="c.target_variable">.{{ c.target_variable }}</span>
                      </td>
                      <td class="px-3 py-1 font-mono text-gray-300 break-all" :title="c.value">{{ truncatePreview(c.value, 200) }}</td>
                    </tr>
                  </tbody>
                </table>
                <ul
                  v-if="captures.some((c) => c.error_message)"
                  class="border-t border-gray-800 bg-[#0e0e0e] px-3 py-2 text-[10px] text-amber-300"
                >
                  <li v-for="(c, ci) in captures.filter((x) => x.error_message)" :key="`cerr-${ci}`">
                    <span class="font-semibold">{{ c.name || '(unnamed)' }}:</span> {{ c.error_message }}
                  </li>
                </ul>
              </div>
            </template>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>
