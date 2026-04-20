<script setup>
import { ref, watch, onMounted } from 'vue'
import { RenderSnippet, ListSnippetKinds } from '../../wailsjs/wailsjs/go/delivery/SnippetHandler'

const props = defineProps({
  /** Returns the same object shape as Wails `HTTPExecuteInput` (env not yet substituted — backend does that). */
  buildPayload: { type: Function, required: true }
})

const emit = defineEmits(['console'])

const expanded = ref(false)
const kinds = ref(/** @type {string[]} */ ([]))
const kind = ref('curl_bash')
const text = ref('')
const loading = ref(false)

const kindLabels = {
  curl_bash: 'cURL (bash)',
  curl_cmd: 'cURL (Windows cmd)',
  fetch_js: 'fetch (JavaScript)',
  axios_js: 'axios (JavaScript)',
  httpie: 'HTTPie'
}

function labelFor(k) {
  return kindLabels[k] || k
}

onMounted(async () => {
  try {
    const list = await ListSnippetKinds()
    kinds.value = Array.isArray(list) ? list : []
    if (kinds.value.length && !kinds.value.includes(kind.value)) {
      kind.value = kinds.value[0]
    }
  } catch (e) {
    emit('console', `[Snippet] ${e?.message || e}`)
  }
})

async function generate() {
  let payload
  try {
    payload = props.buildPayload()
  } catch (e) {
    emit('console', `[Snippet] ${e?.message || e}`)
    return
  }
  if (!payload || !(String(payload.url || '').trim())) {
    emit('console', '[Snippet] URL is required.')
    text.value = ''
    return
  }
  loading.value = true
  try {
    const out = await RenderSnippet(payload, kind.value)
    text.value = typeof out === 'string' ? out : String(out ?? '')
  } catch (e) {
    emit('console', `[Snippet] ${e?.message || e}`)
    text.value = ''
  } finally {
    loading.value = false
  }
}

watch(expanded, (open) => {
  if (open) generate()
})

watch(kind, () => {
  if (expanded.value) generate()
})

async function copy() {
  const s = text.value || ''
  if (!s) return
  try {
    await navigator.clipboard.writeText(s)
    emit('console', '[Snippet] Copied to clipboard.')
  } catch (e) {
    emit('console', `[Snippet] Copy failed: ${e?.message || e}`)
  }
}
</script>

<template>
  <div class="shrink-0 border-b border-gray-800 bg-[#1a1a1a] px-3 py-1.5">
    <button
      type="button"
      class="flex w-full items-center justify-between text-left text-[11px] font-semibold uppercase tracking-wide text-gray-500 hover:text-gray-300"
      aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <span>Code snippet</span>
      <span class="text-[10px]">{{ expanded ? '▼' : '▶' }}</span>
    </button>
    <div v-if="expanded" class="mt-2 space-y-2">
      <div class="flex flex-wrap items-center gap-2">
        <label class="sr-only" for="snippet-kind">Format</label>
        <select
          id="snippet-kind"
          v-model="kind"
          class="max-w-[220px] rounded border border-gray-700 bg-[#252525] px-2 py-1 text-xs text-gray-200 outline-none focus:border-orange-500/60"
        >
          <option v-for="k in kinds" :key="k" :value="k">{{ labelFor(k) }}</option>
        </select>
        <button
          type="button"
          class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-1 text-xs font-semibold text-gray-200 hover:border-orange-500/50 disabled:opacity-50"
          :disabled="loading"
          @click="generate"
        >
          Refresh
        </button>
        <button
          type="button"
          class="rounded border border-orange-600/50 bg-orange-600/20 px-2 py-1 text-xs font-semibold text-orange-300 hover:bg-orange-600/30 disabled:opacity-50"
          :disabled="!text || loading"
          @click="copy"
        >
          Copy
        </button>
        <span v-if="loading" class="text-[10px] text-gray-500">Generating…</span>
      </div>
      <textarea
        readonly
        :value="text"
        rows="8"
        class="w-full resize-y rounded border border-gray-700 bg-[#121212] px-2 py-1.5 font-mono text-[11px] leading-relaxed text-gray-200 outline-none focus:border-orange-500/50"
        spellcheck="false"
      />
    </div>
  </div>
</template>
