<script setup>
import { ref, computed, onMounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import RequestPanel from './components/RequestPanel.vue'
import ResponsePanel from './components/ResponsePanel.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import EnvironmentDetailPanel from './components/EnvironmentDetailPanel.vue'
import { Execute } from '../wailsjs/wailsjs/go/delivery/HTTPHandler'
import { Get as GetSavedRequest } from '../wailsjs/wailsjs/go/delivery/SavedRequestHandler'
import * as EnvAPI from '../wailsjs/wailsjs/go/delivery/EnvironmentHandler'

const responseResult = ref(null)
const loading = ref(false)

/** 'request' | 'environment' — environment replaces request + response columns */
const mainWorkspaceMode = ref('request')
const selectedEnvironmentId = ref(null)

/** @type {import('vue').Ref<Array<Record<string, unknown>>>} */
const environmentSummaries = ref([])
const activeEnvDropdown = ref('')

const envList = computed(() =>
  Array.isArray(environmentSummaries.value) ? environmentSummaries.value : []
)

async function loadEnvironmentSummaries() {
  try {
    const list = await EnvAPI.List()
    environmentSummaries.value = Array.isArray(list) ? list : []
    const active = environmentSummaries.value.find((e) => e.is_active === true)
    activeEnvDropdown.value = active?.id ? String(active.id) : ''
  } catch (e) {
    environmentSummaries.value = []
    const msg = e?.message || String(e)
    pushConsole(`[Env] Could not load environments: ${msg}`)
  }
}

async function onActiveEnvDropdownChange() {
  const v = activeEnvDropdown.value
  try {
    if (!v) {
      await EnvAPI.ClearActive()
    } else {
      await EnvAPI.SetActive(v)
    }
    await loadEnvironmentSummaries()
    try {
      await sidebarRef.value?.refreshEnvironments?.()
    } catch {
      /* ignore */
    }
  } catch (e) {
    pushConsole(`[Env] Active environment: ${e?.message || e}`)
    await loadEnvironmentSummaries()
  }
}

function onOpenEnvironment(id) {
  if (!id) return
  mainWorkspaceMode.value = 'environment'
  selectedEnvironmentId.value = String(id)
}

function onMainWorkspaceRequest() {
  mainWorkspaceMode.value = 'request'
  selectedEnvironmentId.value = null
}

function onEnvironmentChanged() {
  loadEnvironmentSummaries()
  try {
    sidebarRef.value?.refreshEnvironments?.()
  } catch {
    /* ignore */
  }
}

function onEnvironmentDeleted(id) {
  const sid = id != null ? String(id) : selectedEnvironmentId.value
  if (sid && selectedEnvironmentId.value === sid) {
    selectedEnvironmentId.value = null
    mainWorkspaceMode.value = 'request'
  }
  loadEnvironmentSummaries()
  try {
    sidebarRef.value?.refreshEnvironments?.()
  } catch {
    /* ignore */
  }
}

onMounted(() => {
  loadEnvironmentSummaries()
})

/** Selected root folder (sidebar); sent as root_folder_id for HTTP history. */
const activeRootFolderId = ref(null)

/** Console expanded/collapsed (default collapsed to give space to Response). */
const consoleExpanded = ref(false)

/** Console log lines (below Response); used for Format JSON errors from Request. */
const consoleLines = ref([])
let consoleLineId = 0

function pushConsole(text) {
  consoleLineId += 1
  consoleLines.value.push({ id: consoleLineId, text })
  if (consoleLines.value.length > 200) {
    consoleLines.value.splice(0, consoleLines.value.length - 200)
  }
}

function clearConsole() {
  consoleLines.value = []
}

function onRequestConsole(msg) {
  if (typeof msg === 'string' && msg) {
    pushConsole(msg)
  }
}

const sidebarRef = ref(null)
const requestPanelRef = ref(null)

async function onOpenSavedRequest(id) {
  if (!id) return
  try {
    const dto = await GetSavedRequest(id)
    requestPanelRef.value?.loadFromSavedRequest?.(dto)
  } catch (e) {
    const msg = e?.message || String(e)
    pushConsole(`[Library] Could not open saved request: ${msg}`)
  }
}

async function onSavedRequestUpdated() {
  try {
    await sidebarRef.value?.refreshCatalog?.()
  } catch {
    /* ignore */
  }
}

/** Sidebar Import cURL → ad-hoc editor state */
function onApplyCurlImport(payload) {
  requestPanelRef.value?.applyImportPayload?.(payload)
}

const onExecuteRequest = async (payload) => {
  loading.value = true
  responseResult.value = null
  try {
    const res = await Execute(payload)
    responseResult.value = res
  } catch (e) {
    const msg = e?.message || String(e)
    responseResult.value = {
      status_code: 0,
      duration_ms: 0,
      response_size_bytes: 0,
      response_body: '',
      body_truncated: false,
      error_message: msg
    }
  } finally {
    loading.value = false
    try {
      await sidebarRef.value?.refreshHistory?.()
    } catch {
      /* ignore refresh errors */
    }
  }
}
</script>

<template>
  <!-- sidebar | request | response (three columns) -->
  <div
    class="font-sans text-gray-300"
    style="
      width: 100%;
      height: 100%;
      min-width: 0;
      min-height: 0;
      overflow: hidden;
      display: flex;
      flex-direction: row;
      align-items: stretch;
      background: #1c1c1c;
    "
  >
    <div
      style="
        width: 256px;
        min-width: 256px;
        max-width: 256px;
        flex: 0 0 256px;
        height: 100%;
        min-height: 0;
        overflow: hidden;
        background: #212121;
        border-right: 1px solid #2a2a2a;
      "
    >
      <Sidebar
        ref="sidebarRef"
        :active-root-folder-id="activeRootFolderId"
        @update:activeRootFolderId="(v) => (activeRootFolderId = v)"
        @open-saved-request="onOpenSavedRequest"
        @open-environment="onOpenEnvironment"
        @main-workspace-request="onMainWorkspaceRequest"
        @environments-changed="onEnvironmentChanged"
        @environment-deleted="onEnvironmentDeleted"
        @console="onRequestConsole"
        @apply-curl-import="onApplyCurlImport"
      />
    </div>

    <main
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      style="min-width: 0; min-height: 0; height: 100%"
    >
      <template v-if="mainWorkspaceMode === 'request'">
        <div
          class="flex shrink-0 flex-wrap items-center gap-3 border-b border-[#2a2a2a] bg-[#252525] px-4 py-2"
        >
          <span class="text-[11px] font-semibold uppercase tracking-wide text-gray-500">Active environment</span>
          <select
            v-model="activeEnvDropdown"
            class="max-w-[220px] min-w-0 flex-1 rounded border border-gray-600 bg-[#1a1a1a] px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
            aria-label="Active environment"
            @change="onActiveEnvDropdownChange"
          >
            <option value="">No environment</option>
            <option v-for="e in envList" :key="e.id" :value="e.id">
              {{ e.name }}
            </option>
          </select>
        </div>
        <div class="flex min-h-0 min-w-0 flex-1 flex-row overflow-hidden">
          <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border-r border-[#2a2a2a]">
            <RequestPanel
              ref="requestPanelRef"
              :active-root-folder-id="activeRootFolderId"
              @send="onExecuteRequest"
              @console="onRequestConsole"
              @saved-request="onSavedRequestUpdated"
            />
          </div>
          <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <ResponsePanel :result="responseResult" :loading="loading" />
            <ConsolePanel
              v-model:expanded="consoleExpanded"
              :lines="consoleLines"
              @clear="clearConsole"
            />
          </div>
        </div>
      </template>
      <EnvironmentDetailPanel
        v-else
        :environment-id="selectedEnvironmentId"
        @console="onRequestConsole"
        @saved="onEnvironmentChanged"
        @deleted="() => onEnvironmentDeleted(null)"
      />
    </main>
  </div>
</template>
