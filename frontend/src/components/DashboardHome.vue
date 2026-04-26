<script setup>
import { computed, onMounted, ref } from 'vue'
import { ListRootFolders } from '../../wailsjs/wailsjs/go/delivery/FolderHandler'
import { List as ListHistory } from '../../wailsjs/wailsjs/go/delivery/HistoryHandler'
import { List as ListEnvironments } from '../../wailsjs/wailsjs/go/delivery/EnvironmentHandler'

const emit = defineEmits([
  'new-tab',
  'new-folder',
  'import-collection',
  'import-curl',
  'new-environment',
  'open-help',
  'open-saved-request',
  'console'
])

const loading = ref(false)
const rootFolders = ref([])
const histories = ref([])
const environments = ref([])

const recent = computed(() => (Array.isArray(histories.value) ? histories.value.slice(0, 20) : []))
const stats = computed(() => ({
  roots: Array.isArray(rootFolders.value) ? rootFolders.value.length : 0,
  environments: Array.isArray(environments.value) ? environments.value.length : 0,
  recent: Array.isArray(histories.value) ? histories.value.length : 0
}))

function formatTime(value) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString()
}

function statusClass(code) {
  const n = Number(code || 0)
  if (n >= 200 && n < 300) return 'bg-emerald-500/15 text-emerald-300'
  if (n >= 300 && n < 400) return 'bg-sky-500/15 text-sky-300'
  if (n >= 400 && n < 500) return 'bg-yellow-500/15 text-yellow-300'
  if (n >= 500) return 'bg-red-500/15 text-red-300'
  return 'bg-gray-700 text-gray-300'
}

async function load() {
  loading.value = true
  try {
    const [folders, hist, envs] = await Promise.all([
      ListRootFolders(),
      ListHistory(''),
      ListEnvironments()
    ])
    rootFolders.value = Array.isArray(folders) ? folders : []
    histories.value = Array.isArray(hist) ? hist : []
    environments.value = Array.isArray(envs) ? envs : []
  } catch (e) {
    emit('console', `[Dashboard] ${e?.message || e}`)
  } finally {
    loading.value = false
  }
}

function openRecent(item) {
  const requestId = item?.request_id
  if (requestId) {
    emit('open-saved-request', String(requestId))
    return
  }
  emit('console', '[Dashboard] This history row is ad-hoc or its saved request no longer exists.')
}

onMounted(load)
</script>

<template>
  <div class="app-scrollbar min-h-0 flex-1 overflow-auto bg-[#181818] p-6">
    <div class="mx-auto flex max-w-5xl flex-col gap-6">
      <section class="rounded-xl border border-gray-700/80 bg-[#212121] p-5 shadow-lg">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-widest text-orange-400">PostmanJanai</p>
            <h1 class="mt-2 text-2xl font-semibold text-white">Dashboard</h1>
            <p class="mt-1 max-w-2xl text-sm text-gray-400">
              No request tab is open. Start a new request, import a collection, or jump back into recent work.
            </p>
          </div>
          <button
            type="button"
            class="rounded bg-orange-600 px-4 py-2 text-sm font-semibold text-white hover:bg-orange-700"
            @click="emit('new-tab')"
          >
            New Request
          </button>
        </div>
      </section>

      <section class="grid gap-3 md:grid-cols-3">
        <div class="rounded-lg border border-gray-700 bg-[#212121] p-4">
          <div class="text-[10px] font-semibold uppercase tracking-wide text-gray-500">Root folders</div>
          <div class="mt-2 text-2xl font-semibold text-gray-100">{{ stats.roots }}</div>
        </div>
        <div class="rounded-lg border border-gray-700 bg-[#212121] p-4">
          <div class="text-[10px] font-semibold uppercase tracking-wide text-gray-500">Environments</div>
          <div class="mt-2 text-2xl font-semibold text-gray-100">{{ stats.environments }}</div>
        </div>
        <div class="rounded-lg border border-gray-700 bg-[#212121] p-4">
          <div class="text-[10px] font-semibold uppercase tracking-wide text-gray-500">Recent history rows</div>
          <div class="mt-2 text-2xl font-semibold text-gray-100">{{ stats.recent }}</div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-700 bg-[#212121] p-4">
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 class="text-sm font-semibold text-white">Quick Actions</h2>
          <span v-if="loading" class="text-[11px] text-gray-500">Loading…</span>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-6">
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="emit('new-folder')">
            New folder
          </button>
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="emit('import-collection')">
            Import collection
          </button>
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="emit('import-curl')">
            Import cURL
          </button>
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="emit('new-environment')">
            New environment
          </button>
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="emit('open-help')">
            Help
          </button>
          <button type="button" class="rounded border border-gray-700 bg-[#1a1a1a] px-3 py-2 text-left text-xs font-semibold text-gray-200 hover:border-orange-500/60" @click="load">
            Refresh
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-gray-700 bg-[#212121]">
        <div class="border-b border-gray-700 px-4 py-3">
          <h2 class="text-sm font-semibold text-white">Recent</h2>
        </div>
        <div v-if="recent.length === 0" class="p-4 text-sm text-gray-500">
          No history yet. Send a request to populate this list.
        </div>
        <button
          v-for="item in recent"
          v-else
          :key="item.id"
          type="button"
          class="flex w-full items-start gap-3 border-b border-gray-800 px-4 py-3 text-left last:border-b-0 hover:bg-[#252525]"
          @click="openRecent(item)"
        >
          <span class="mt-0.5 rounded bg-gray-700 px-1.5 py-0.5 font-mono text-[10px] font-bold text-gray-200">{{ item.method }}</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm text-gray-200">{{ item.url }}</span>
            <span class="mt-1 block text-[11px] text-gray-500">{{ formatTime(item.created_at) }}</span>
          </span>
          <span class="rounded px-1.5 py-0.5 font-mono text-[10px] font-semibold" :class="statusClass(item.status_code)">
            {{ item.status_code }}
          </span>
        </button>
      </section>
    </div>
  </div>
</template>
