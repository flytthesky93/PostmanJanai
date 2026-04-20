<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import Sidebar from './components/Sidebar.vue'
import RequestPanel from './components/RequestPanel.vue'
import RequestTabBar from './components/RequestTabBar.vue'
import ResponsePanel from './components/ResponsePanel.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import EnvironmentDetailPanel from './components/EnvironmentDetailPanel.vue'
import { Execute } from '../wailsjs/wailsjs/go/delivery/HTTPHandler'
import { Get as GetSavedRequest } from '../wailsjs/wailsjs/go/delivery/SavedRequestHandler'
import * as EnvAPI from '../wailsjs/wailsjs/go/delivery/EnvironmentHandler'
import { useTabsStore } from './stores/tabsStore'

const tabsStore = useTabsStore()
const { tabsMeta, activeTab, state: tabsState } = tabsStore

/** Response + loading are derived per-active-tab. */
const responseResult = computed(() => activeTab.value?.response || null)
const loading = computed(() => !!activeTab.value?.loading)

/** 'request' | 'environment' — environment replaces request + response columns */
const mainWorkspaceMode = ref('request')
const selectedEnvironmentId = ref(null)

/** @type {import('vue').Ref<Array<Record<string, unknown>>>} */
const environmentSummaries = ref([])
const activeEnvDropdown = ref('')

const envList = computed(() =>
  Array.isArray(environmentSummaries.value) ? environmentSummaries.value : []
)

/** Keys (enabled) in the active environment — for {{var}} chips / CodeMirror. */
const activeEnvDeclaredKeys = ref([])

/**
 * Active env id + variable rows for SaveVariables / hover values.
 * @type {import('vue').Ref<{ id: string, variables: Array<{ key: string, value: string, enabled: boolean, sort_order: number }> } | null>}
 */
const activeEnvForPatch = ref(null)

const activeEnvValues = computed(() => {
  const m = /** @type {Record<string, string>} */ ({})
  const st = activeEnvForPatch.value
  if (!st?.variables) return m
  for (const v of st.variables) {
    if (v.enabled !== false && v.key) m[v.key] = v.value ?? ''
  }
  return m
})

async function refreshActiveEnvContext() {
  activeEnvForPatch.value = null
  activeEnvDeclaredKeys.value = []
  try {
    const active = await EnvAPI.GetActive()
    if (!active?.id) return
    const full = await EnvAPI.Get(String(active.id))
    const vars = Array.isArray(full.variables) ? full.variables : []
    activeEnvForPatch.value = {
      id: String(full.id),
      variables: vars.map((v, i) => ({
        key: String(v.key || '').trim(),
        value: v.value ?? '',
        enabled: v.enabled !== false,
        sort_order: typeof v.sort_order === 'number' ? v.sort_order : i
      }))
    }
    activeEnvDeclaredKeys.value = activeEnvForPatch.value.variables
      .filter((v) => v.enabled && v.key)
      .map((v) => v.key)
  } catch {
    /* ignore */
  }
}

async function onPatchActiveEnvValue(payload) {
  const key = String(payload?.key || '').trim()
  const value = payload?.value != null ? String(payload.value) : ''
  const st = activeEnvForPatch.value
  if (!st?.id) {
    pushConsole('[Env] No active environment. Select one in the bar above.')
    return
  }
  if (!key) return

  const row = st.variables.find((v) => v.key === key)
  let added = false
  if (row) {
    row.value = value
    if (row.enabled === false) row.enabled = true
  } else {
    let maxSo = -1
    for (const v of st.variables) {
      const so = typeof v.sort_order === 'number' ? v.sort_order : 0
      if (so > maxSo) maxSo = so
    }
    st.variables.push({
      key,
      value,
      enabled: true,
      sort_order: maxSo + 1
    })
    added = true
  }

  const savePayload = st.variables
    .filter((v) => v.key)
    .map((v, i) => ({
      key: v.key,
      value: v.value ?? '',
      enabled: v.enabled !== false,
      sort_order: v.sort_order ?? i
    }))
  try {
    await EnvAPI.SaveVariables(st.id, savePayload)
    pushConsole(
      added
        ? `[Env] Added "${key}" to active environment.`
        : `[Env] Updated "${key}" in active environment.`
    )
    await refreshActiveEnvContext()
    try {
      await sidebarRef.value?.refreshEnvironments?.()
    } catch {
      /* ignore */
    }
  } catch (e) {
    pushConsole(`[Env] Could not save: ${e?.message || e}`)
    await refreshActiveEnvContext()
  }
}

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
  await refreshActiveEnvContext()
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

async function onEnvironmentChanged() {
  await loadEnvironmentSummaries()
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
  void loadEnvironmentSummaries()
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

/** True while we are programmatically hydrating the panel — ignore snapshot-change echoes. */
let suppressSnapshotUpdate = false

async function hydrateActiveTabIntoPanel() {
  const t = activeTab.value
  if (!t || !requestPanelRef.value?.hydrate) return
  suppressSnapshotUpdate = true
  try {
    requestPanelRef.value.hydrate(t.snapshot)
  } finally {
    // release one microtask later so any trailing watcher from hydrate does not overwrite baseline
    await nextTick()
    setTimeout(() => {
      suppressSnapshotUpdate = false
    }, 120)
  }
}

function onPanelSnapshotChange(snapshot) {
  if (suppressSnapshotUpdate) return
  tabsStore.updateActiveSnapshot(snapshot)
}

function onPanelBaselineCommitted() {
  tabsStore.markActiveBaseline()
}

function onPanelPromoteToSaved(dto) {
  tabsStore.promoteActiveToSaved(dto)
}

function captureCurrentPanelSnapshot() {
  if (!requestPanelRef.value?.snapshot) return
  tabsStore.updateActiveSnapshot(requestPanelRef.value.snapshot())
}

function onTabActivate(id) {
  if (id === tabsState.activeTabId) return
  captureCurrentPanelSnapshot()
  tabsStore.activateTab(id)
}

function onTabClose(id) {
  tabsStore.closeTab(id)
}

function onTabNew() {
  captureCurrentPanelSnapshot()
  tabsStore.openBlank()
}

async function onOpenSavedRequest(id) {
  if (!id) return
  try {
    captureCurrentPanelSnapshot()
    const dto = await GetSavedRequest(id)
    const prevActiveId = tabsState.activeTabId
    tabsStore.openSavedRequest(dto)
    // If target tab was already the active one, watch won't fire — hydrate manually.
    if (tabsState.activeTabId === prevActiveId) {
      await nextTick()
      await hydrateActiveTabIntoPanel()
    }
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

/** Sidebar Import cURL → open or reuse an ad-hoc tab. */
async function onApplyCurlImport(payload) {
  captureCurrentPanelSnapshot()
  const prevActiveId = tabsState.activeTabId
  tabsStore.openAdhocFromPayload(payload)
  if (tabsState.activeTabId === prevActiveId) {
    await nextTick()
    await hydrateActiveTabIntoPanel()
  }
}

const onExecuteRequest = async (payload) => {
  const execTabId = tabsState.activeTabId
  tabsStore.setTabLoading(execTabId, true)
  tabsStore.setTabResponse(execTabId, null)
  try {
    const res = await Execute(payload)
    if (res?.error_message) {
      pushConsole(`[HTTP] ${res.error_message}`)
    }
    tabsStore.setTabResponse(execTabId, res ? { ...res, error_message: '' } : null)
  } catch (e) {
    const msg = e?.message || String(e)
    pushConsole(`[HTTP] ${msg}`)
    tabsStore.setTabResponse(execTabId, null)
  } finally {
    tabsStore.setTabLoading(execTabId, false)
    try {
      await sidebarRef.value?.refreshHistory?.()
    } catch {
      /* ignore refresh errors */
    }
  }
}

// Restore tabs on mount, then hydrate the active one into the panel.
onMounted(async () => {
  tabsStore.restore()
  await nextTick()
  await hydrateActiveTabIntoPanel()
})

// If active tab changes for any reason (programmatic), keep the panel in sync.
watch(
  () => tabsState.activeTabId,
  async (newId, oldId) => {
    if (newId === oldId) return
    await nextTick()
    await hydrateActiveTabIntoPanel()
  }
)
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
        <RequestTabBar
          :tabs="tabsMeta"
          :active-tab-id="tabsState.activeTabId"
          @activate="onTabActivate"
          @close="onTabClose"
          @new="onTabNew"
        />
        <div class="flex min-h-0 min-w-0 flex-1 flex-row overflow-hidden">
          <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border-r border-[#2a2a2a]">
            <RequestPanel
              ref="requestPanelRef"
              :active-root-folder-id="activeRootFolderId"
              :declared-env-keys="activeEnvDeclaredKeys"
              :active-env-values="activeEnvValues"
              @send="onExecuteRequest"
              @console="onRequestConsole"
              @saved-request="onSavedRequestUpdated"
              @patch-active-env-value="onPatchActiveEnvValue"
              @snapshot-change="onPanelSnapshotChange"
              @baseline-committed="onPanelBaselineCommitted"
              @promote-to-saved="onPanelPromoteToSaved"
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
