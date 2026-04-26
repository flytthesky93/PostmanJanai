<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { SearchTree } from '../../wailsjs/wailsjs/go/delivery/SearchHandler'
import { List as ListHistory } from '../../wailsjs/wailsjs/go/delivery/HistoryHandler'
import { List as ListEnvironments } from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'

const props = defineProps({
  open: { type: Boolean, default: false }
})

const emit = defineEmits([
  'close',
  'open-saved-request',
  'open-folder-hit',
  'open-environment',
  'new-tab',
  'new-folder',
  'import-collection',
  'import-curl',
  'new-environment',
  'open-settings',
  'console'
])

const query = ref('')
const selectedIndex = ref(0)
const loading = ref(false)
const searchResult = ref(null)
const histories = ref([])
const environments = ref([])
const inputRef = ref(null)

const commands = [
  { id: 'new-tab', type: 'command', label: 'New request tab', hint: 'Ctrl+T', action: 'new-tab' },
  { id: 'new-folder', type: 'command', label: 'New root folder', hint: 'Folders', action: 'new-folder' },
  { id: 'import-collection', type: 'command', label: 'Import collection', hint: 'Postman / OpenAPI / Insomnia', action: 'import-collection' },
  { id: 'import-curl', type: 'command', label: 'Import cURL', hint: 'Ad-hoc request', action: 'import-curl' },
  { id: 'new-environment', type: 'command', label: 'New environment', hint: 'Env', action: 'new-environment' },
  { id: 'open-settings', type: 'command', label: 'Open settings', hint: 'Proxy / CA / About', action: 'open-settings' }
]

function includes(text, q) {
  return String(text || '').toLowerCase().includes(q)
}

const normalizedQuery = computed(() => query.value.trim().toLowerCase())

const items = computed(() => {
  const q = normalizedQuery.value
  const out = []
  const commandHits = commands.filter((c) => !q || includes(c.label, q) || includes(c.hint, q))
  out.push(...commandHits)

  if (q && searchResult.value) {
    const folders = Array.isArray(searchResult.value.folders) ? searchResult.value.folders : []
    const requests = Array.isArray(searchResult.value.requests) ? searchResult.value.requests : []
    out.push(...folders.map((f) => ({ ...f, source_id: f.id, id: `folder:${f.id}`, type: 'folder-hit', label: f.name, hint: (f.path || []).join(' / ') })))
    out.push(...requests.map((r) => ({ ...r, source_id: r.id, id: `request:${r.id}`, type: 'request-hit', label: r.name, hint: r.url })))
  }

  const envList = Array.isArray(environments.value) ? environments.value : []
  out.push(
    ...envList
      .filter((e) => !q || includes(e.name, q))
      .map((e) => ({ ...e, id: `env:${e.id}`, type: 'environment', label: e.name, hint: e.is_active ? 'Active environment' : 'Environment' }))
  )

  const recent = Array.isArray(histories.value) ? histories.value.slice(0, 8) : []
  out.push(
    ...recent
      .filter((h) => !q || includes(h.url, q) || includes(h.method, q))
      .map((h) => ({ ...h, id: `history:${h.id}`, type: 'history', label: h.url, hint: `${h.method} ${h.status_code}` }))
  )
  return out
})

async function refreshStaticData() {
  try {
    const [hist, envs] = await Promise.all([ListHistory(''), ListEnvironments()])
    histories.value = Array.isArray(hist) ? hist : []
    environments.value = Array.isArray(envs) ? envs : []
  } catch (e) {
    emit('console', `[Palette] ${e?.message || e}`)
  }
}

let searchTimer = null
watch(query, (value) => {
  selectedIndex.value = 0
  if (searchTimer) clearTimeout(searchTimer)
  const q = value.trim()
  if (!q) {
    searchResult.value = null
    loading.value = false
    return
  }
  loading.value = true
  searchTimer = setTimeout(async () => {
    try {
      const res = await SearchTree(q, 80)
      searchResult.value = res && typeof res === 'object' ? res : null
    } catch (e) {
      emit('console', `[Palette] ${e?.message || e}`)
      searchResult.value = null
    } finally {
      loading.value = false
    }
  }, 150)
})

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    query.value = ''
    selectedIndex.value = 0
    await refreshStaticData()
    await nextTick()
    inputRef.value?.focus?.()
  }
)

function move(delta) {
  const n = items.value.length
  if (n === 0) return
  selectedIndex.value = (selectedIndex.value + delta + n) % n
}

function close() {
  emit('close')
}

function execute(item = items.value[selectedIndex.value]) {
  if (!item) return
  if (item.type === 'command') emit(item.action)
  else if (item.type === 'request-hit') emit('open-saved-request', item.source_id)
  else if (item.type === 'folder-hit') emit('open-folder-hit', { ...item, id: item.source_id })
  else if (item.type === 'environment') emit('open-environment', item.id.replace(/^env:/, ''))
  else if (item.type === 'history') {
    if (item.request_id) emit('open-saved-request', String(item.request_id))
    else emit('console', '[Palette] This history row has no saved request link.')
  }
  close()
}
</script>

<template>
  <Teleport to="#app">
    <div
      v-if="open"
      class="fixed inset-0 z-[80] flex items-start justify-center bg-black/55 px-4 pt-[10vh]"
      @mousedown.self="close"
    >
      <div class="w-full max-w-2xl overflow-hidden rounded-xl border border-gray-700 bg-[#1f1f1f] shadow-2xl" role="dialog" aria-modal="true">
        <div class="border-b border-gray-700 p-3">
          <input
            ref="inputRef"
            v-model="query"
            type="text"
            class="w-full rounded border border-gray-700 bg-[#111] px-3 py-2 text-sm text-gray-100 outline-none focus:border-orange-500"
            placeholder="Search requests, folders, environments, commands..."
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="execute()"
            @keydown.escape.prevent="close"
          />
        </div>
        <div class="app-scrollbar max-h-[60vh] overflow-y-auto p-2">
          <div v-if="loading" class="px-2 py-1 text-[11px] text-gray-500">Searching...</div>
          <div v-if="items.length === 0" class="p-3 text-sm text-gray-500">No matches.</div>
          <button
            v-for="(item, idx) in items"
            :key="item.id"
            type="button"
            class="flex w-full items-center gap-3 rounded px-3 py-2 text-left text-sm"
            :class="idx === selectedIndex ? 'bg-orange-500/15 text-orange-100' : 'text-gray-200 hover:bg-gray-800'"
            @mouseenter="selectedIndex = idx"
            @click="execute(item)"
          >
            <span class="w-20 shrink-0 text-[10px] font-semibold uppercase tracking-wide text-gray-500">{{ item.type.replace('-hit', '') }}</span>
            <span class="min-w-0 flex-1">
              <span class="block truncate">{{ item.label }}</span>
              <span v-if="item.hint" class="mt-0.5 block truncate text-[11px] text-gray-500">{{ item.hint }}</span>
            </span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
