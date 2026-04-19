<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import JsonCodeMirror from './JsonCodeMirror.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  /** @type {import('vue').PropType<Record<string, unknown> | null>} */
  item: { type: Object, default: null }
})

const emit = defineEmits(['close', 'delete'])

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
      return arr.map((h) => `${h.key ?? ''}: ${h.value ?? ''}`).join('\n')
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

function formatWhen(raw) {
  if (raw == null || raw === '') return '—'
  if (typeof raw === 'string') {
    const d = new Date(raw)
    return Number.isNaN(d.getTime()) ? raw : d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
  }
  if (typeof raw === 'object' && raw !== null) {
    try {
      const d = new Date(raw)
      if (!Number.isNaN(d.getTime())) {
        return d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
      }
    } catch {
      /* ignore */
    }
  }
  return String(raw)
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
      class="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 px-3 py-6"
      role="dialog"
      aria-modal="true"
      aria-labelledby="history-detail-title"
      @click.self="emit('close')"
    >
      <div
        class="flex max-h-[min(92vh,900px)] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-gray-600 bg-[#1f1f1f] shadow-xl"
        @click.stop
      >
        <div class="flex shrink-0 items-start justify-between gap-2 border-b border-gray-700 px-4 py-3">
          <div class="min-w-0">
            <h2 id="history-detail-title" class="text-sm font-semibold text-white">Request snapshot</h2>
            <p v-if="item" class="mt-0.5 text-[11px] text-gray-500">{{ formatWhen(item.created_at) }}</p>
            <p v-else class="mt-0.5 text-[11px] text-gray-500">Loading…</p>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <button
              v-if="item?.id && !loading"
              type="button"
              class="rounded px-2 py-1 text-xs text-red-300 hover:bg-red-900/40 hover:text-red-100"
              @click="emit('delete')"
            >
              Delete
            </button>
            <button
              type="button"
              class="shrink-0 rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-700 hover:text-white"
              aria-label="Close"
              @click="emit('close')"
            >
              Close
            </button>
          </div>
        </div>

        <div v-if="loading && !item" class="app-scrollbar flex min-h-[200px] items-center justify-center p-8 text-sm text-gray-400">
          Loading snapshot…
        </div>

        <template v-else-if="item">
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
          </div>
        </div>

        <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-4 text-xs">
          <template v-if="activeTab === 'request'">
            <div class="mb-3 flex flex-wrap items-center gap-2">
              <span
                class="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white"
                style="background: #374151"
                >{{ item.method }}</span>
            </div>
            <div class="mb-3 break-all font-mono text-gray-300">{{ item.url }}</div>
            <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Request headers</div>
            <pre
              class="mb-4 max-h-40 overflow-auto rounded border border-gray-700 bg-[#121212] p-2 font-mono text-[11px] leading-relaxed text-gray-300 whitespace-pre-wrap break-words"
              >{{ requestHeadersText }}</pre
            >
            <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Request body</div>
            <div class="overflow-hidden rounded border border-gray-700" style="height: min(40vh, 320px); min-height: 120px">
              <JsonCodeMirror :model-value="requestBodyPretty" :read-only="true" class="h-full min-h-0 flex-1" />
            </div>
          </template>

          <template v-else>
            <div class="mb-3 flex flex-wrap gap-3 text-gray-300">
              <span><span class="text-gray-500">Status:</span> {{ item.status_code }}</span>
              <span v-if="item.duration_ms != null"><span class="text-gray-500">Time:</span> {{ item.duration_ms }} ms</span>
              <span v-if="item.response_size_bytes != null"
                ><span class="text-gray-500">Size:</span> {{ item.response_size_bytes }} B</span
              >
            </div>
            <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Response headers</div>
            <pre
              class="mb-4 max-h-40 overflow-auto rounded border border-gray-700 bg-[#121212] p-2 font-mono text-[11px] leading-relaxed text-gray-300 whitespace-pre-wrap break-words"
              >{{ responseHeadersText }}</pre
            >
            <div class="mb-2 text-[10px] font-bold uppercase tracking-wider text-gray-500">Response body</div>
            <div class="overflow-hidden rounded border border-gray-700" style="height: min(40vh, 320px); min-height: 120px">
              <JsonCodeMirror :model-value="responseBodyPretty" :read-only="true" class="h-full min-h-0 flex-1" />
            </div>
          </template>
        </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>
