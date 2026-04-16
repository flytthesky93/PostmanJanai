<script setup>
import { computed, ref } from 'vue'
import JsonCodeMirror from './JsonCodeMirror.vue'

const props = defineProps({
  result: { type: Object, default: null },
  loading: { type: Boolean, default: false }
})

const activeTab = ref('preview')

const prettyBody = computed(() => {
  const raw = props.result?.response_body
  if (raw == null || raw === '') return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
})

const headersText = computed(() => {
  const list = props.result?.response_headers
  if (!Array.isArray(list) || list.length === 0) return ''
  return list.map((h) => `${h.key}: ${h.value}`).join('\n')
})

const summaryParts = computed(() => {
  const r = props.result
  if (!r) return []
  const parts = []
  if (r.status_code != null && r.status_code !== 0) {
    parts.push({ label: 'Status', value: String(r.status_code) })
  }
  if (r.duration_ms != null) {
    parts.push({ label: 'Time', value: `${r.duration_ms} ms` })
  }
  if (r.response_size_bytes != null) {
    parts.push({ label: 'Size', value: `${r.response_size_bytes} B` })
  }
  if (r.body_truncated) {
    parts.push({ label: '', value: 'Body truncated (limit)' })
  }
  return parts
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col overflow-hidden bg-[#181818]">
    <div v-if="loading" class="shrink-0 px-3 pt-3 text-sm text-orange-400">Sending…</div>

    <div
      v-if="result?.error_message"
      class="shrink-0 break-all px-3 pt-2 font-mono text-sm"
      :class="result.status_code ? 'text-amber-400' : 'text-red-400'"
    >
      {{ result.error_message }}
    </div>

    <!-- Meta row + tabs (Postman-style) -->
    <div class="shrink-0 border-b border-gray-800 px-3 pt-2">
      <div class="flex flex-wrap items-end justify-between gap-x-4 gap-y-2">
        <div class="min-w-0 flex-1">
          <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Response</div>
          <div class="mt-1 flex flex-wrap gap-1 text-xs font-semibold">
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'preview' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'preview'"
            >
              Preview
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'headers' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'headers'"
            >
              Headers
            </button>
          </div>
        </div>
        <div
          v-if="summaryParts.length"
          class="flex shrink-0 flex-wrap items-center justify-end gap-x-3 gap-y-1 pb-1 text-xs text-gray-400"
        >
          <span v-for="(p, i) in summaryParts" :key="i">
            <template v-if="p.label"><span class="text-gray-500">{{ p.label }}:</span> {{ p.value }}</template>
            <template v-else><span class="text-amber-500">{{ p.value }}</span></template>
          </span>
        </div>
      </div>
    </div>

    <div
      class="min-h-0 flex-1 overflow-hidden bg-[#121212] p-2 font-mono text-xs shadow-inner"
    >
      <div
        v-if="activeTab === 'preview' && (result || loading)"
        class="relative flex h-full min-h-0 flex-1 flex-col"
      >
        <JsonCodeMirror :model-value="prettyBody" :read-only="true" class="min-h-0 flex-1" />
        <div
          v-if="!loading && result && !prettyBody"
          class="pointer-events-none absolute inset-0 flex items-center justify-center text-sm italic text-gray-600"
        >
          (empty)
        </div>
      </div>
      <pre
        v-else-if="activeTab === 'headers' && result"
        class="app-scrollbar h-full overflow-auto break-words whitespace-pre-wrap p-1 leading-relaxed text-gray-300"
        >{{ headersText || '(no headers)' }}</pre
      >
      <div v-else-if="!loading" class="flex h-full min-h-[120px] items-center justify-center italic text-gray-600">
        Press Send to see the response…
      </div>
    </div>
  </div>
</template>
