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

const assertionList = computed(() =>
  Array.isArray(props.result?.assertions) ? props.result.assertions : []
)
const captureList = computed(() =>
  Array.isArray(props.result?.captures) ? props.result.captures : []
)
const scriptConsoleList = computed(() =>
  Array.isArray(props.result?.script_console) ? props.result.script_console : []
)
const scriptTestList = computed(() =>
  Array.isArray(props.result?.script_tests) ? props.result.script_tests : []
)
const assertionPassCount = computed(() => assertionList.value.filter((a) => a.passed).length)
const assertionFailCount = computed(() => assertionList.value.length - assertionPassCount.value)
const scriptTestPassCount = computed(() => scriptTestList.value.filter((t) => t.passed).length)
const scriptTestFailCount = computed(() => scriptTestList.value.length - scriptTestPassCount.value)
const hasTests = computed(
  () =>
    assertionList.value.length > 0 ||
    captureList.value.length > 0 ||
    scriptConsoleList.value.length > 0 ||
    scriptTestList.value.length > 0
)
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col overflow-hidden bg-[#181818]">
    <div v-if="loading" class="shrink-0 px-3 pt-3 text-sm text-orange-400">Sending…</div>

    <!-- Errors are logged to the console panel instead of this header. -->

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
            <button
              v-if="hasTests"
              type="button"
              class="rounded-t px-3 py-2"
              :class="activeTab === 'tests' ? 'text-white border-b-2 border-orange-500' : 'text-gray-500 hover:text-gray-300'"
              @click="activeTab = 'tests'"
            >
              Results
              <span
                v-if="assertionList.length || scriptTestList.length"
                class="ml-1.5 inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-semibold"
                :class="
                  assertionFailCount === 0 && scriptTestFailCount === 0
                    ? 'bg-emerald-500/15 text-emerald-300'
                    : 'bg-red-500/15 text-red-300'
                "
              >
                <template v-if="assertionList.length">{{ assertionPassCount }}/{{ assertionList.length }} rules</template>
                <template v-if="assertionList.length && scriptTestList.length"> · </template>
                <template v-if="scriptTestList.length">{{ scriptTestPassCount }}/{{ scriptTestList.length }} scripts</template>
              </span>
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
      <div
        v-else-if="activeTab === 'tests' && result"
        class="app-scrollbar h-full overflow-auto p-2 text-xs"
      >
        <section v-if="scriptConsoleList.length" class="mb-3">
          <h4 class="mb-1 text-[10px] font-bold uppercase tracking-wider text-gray-500">
            Script console ({{ scriptConsoleList.length }})
          </h4>
          <ul class="space-y-0.5 font-mono text-[11px]">
            <li
              v-for="(ln, i) in scriptConsoleList"
              :key="'sc' + i"
              class="rounded border border-gray-800 bg-[#141414] px-2 py-1"
              :class="{
                'text-gray-300': ln.level === 'log' || ln.level === 'info' || ln.level === 'debug',
                'text-amber-300': ln.level === 'warn',
                'text-red-300': ln.level === 'error'
              }"
            >
              <span class="text-[10px] uppercase text-gray-500">{{ ln.level || 'log' }}</span>
              <span class="ml-2 whitespace-pre-wrap break-words">{{ ln.message }}</span>
            </li>
          </ul>
        </section>
        <section v-if="scriptTestList.length" class="mb-3">
          <h4 class="mb-1 text-[10px] font-bold uppercase tracking-wider text-gray-500">
            Script tests ({{ scriptTestPassCount }}/{{ scriptTestList.length }})
          </h4>
          <ul class="space-y-1">
            <li
              v-for="(st, i) in scriptTestList"
              :key="'st' + i"
              class="rounded border px-2 py-1.5"
              :class="st.passed ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-red-500/30 bg-red-500/5'"
            >
              <div class="font-semibold" :class="st.passed ? 'text-emerald-300' : 'text-red-300'">
                {{ st.passed ? 'PASS' : 'FAIL' }} · {{ st.name || '(unnamed)' }}
              </div>
              <div v-if="!st.passed && st.detail" class="mt-0.5 break-words font-mono text-[11px] text-red-300">{{ st.detail }}</div>
            </li>
          </ul>
        </section>
        <section v-if="assertionList.length" class="mb-3">
          <h4 class="mb-1 text-[10px] font-bold uppercase tracking-wider text-gray-500">
            Assertions ({{ assertionPassCount }}/{{ assertionList.length }})
          </h4>
          <ul class="space-y-1">
            <li
              v-for="(a, i) in assertionList"
              :key="'a' + i"
              class="rounded border px-2 py-1.5"
              :class="a.passed ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-red-500/30 bg-red-500/5'"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-semibold" :class="a.passed ? 'text-emerald-300' : 'text-red-300'">
                  {{ a.passed ? 'PASS' : 'FAIL' }} · {{ a.name || '(unnamed)' }}
                </span>
                <span class="text-[10px] text-gray-500">{{ a.source }} {{ a.operator }}</span>
              </div>
              <div v-if="!a.passed && a.error_message" class="mt-0.5 break-words font-mono text-[11px] text-red-300">{{ a.error_message }}</div>
              <div v-if="a.actual" class="mt-0.5 truncate font-mono text-[11px] text-gray-400" :title="a.actual">
                actual: {{ a.actual }}
              </div>
            </li>
          </ul>
        </section>
        <section v-if="captureList.length">
          <h4 class="mb-1 text-[10px] font-bold uppercase tracking-wider text-gray-500">
            Captures ({{ captureList.length }})
          </h4>
          <ul class="space-y-1">
            <li
              v-for="(c, i) in captureList"
              :key="'c' + i"
              class="rounded border px-2 py-1.5"
              :class="c.error_message ? 'border-amber-500/30 bg-amber-500/5' : 'border-gray-700 bg-[#141414]'"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-semibold" :class="c.error_message ? 'text-amber-300' : 'text-gray-200'">
                  {{ c.name || '(unnamed)' }}
                </span>
                <span class="text-[10px] text-gray-500">{{ c.source }} → {{ c.target_scope }}.{{ c.target_variable }}</span>
              </div>
              <div v-if="c.error_message" class="mt-0.5 font-mono text-[11px] text-amber-300">{{ c.error_message }}</div>
              <div v-else-if="c.value" class="mt-0.5 truncate font-mono text-[11px] text-gray-400" :title="c.value">
                {{ c.value }}
              </div>
            </li>
          </ul>
        </section>
      </div>
      <div v-else-if="!loading" class="flex h-full min-h-[120px] items-center justify-center italic text-gray-600">
        Press Send to see the response…
      </div>
    </div>
  </div>
</template>
