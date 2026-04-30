<script setup>
import { ref, computed, onMounted, nextTick, watch, defineAsyncComponent } from 'vue'
import Sidebar from './components/Sidebar.vue'
import RequestTabBar from './components/RequestTabBar.vue'
import ResponsePanel from './components/ResponsePanel.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import { Execute } from '../wailsjs/wailsjs/go/delivery/HTTPHandler'
import { Get as GetSavedRequest } from '../wailsjs/wailsjs/go/delivery/SavedRequestHandler'
import * as EnvAPI from '../wailsjs/wailsjs/go/delivery/EnvironmentHandler'
import { useTabsStore } from './stores/tabsStore'
import { useKeyboardShortcuts } from './composables/useKeyboardShortcuts'

const RequestPanel = defineAsyncComponent(() => import('./components/RequestPanel.vue'))
const EnvironmentDetailPanel = defineAsyncComponent(() => import('./components/EnvironmentDetailPanel.vue'))
const SettingsPanel = defineAsyncComponent(() => import('./components/SettingsPanel.vue'))
const DashboardHome = defineAsyncComponent(() => import('./components/DashboardHome.vue'))
const CommandPalette = defineAsyncComponent(() => import('./components/CommandPalette.vue'))
const HelpModal = defineAsyncComponent(() => import('./components/HelpModal.vue'))
const RunnerModal = defineAsyncComponent(() => import('./components/RunnerModal.vue'))

const tabsStore = useTabsStore()
const { tabsMeta, activeTab, state: tabsState } = tabsStore

/** Response + loading are derived per-active-tab. */
const responseResult = computed(() => activeTab.value?.response || null)
const loading = computed(() => !!activeTab.value?.loading)

/** 'request' | 'environment' | 'settings' */
const mainWorkspaceMode = ref('request')
const selectedEnvironmentId = ref(null)
/** When opening Settings from request/env, restore this mode when closing. */
const workspaceBeforeSettings = ref(null)
const commandPaletteOpen = ref(false)
const helpOpen = ref(false)
const runnerOpen = ref(false)
const runnerSeedFolderId = ref('')

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
        kind: v.kind === 'secret' ? 'secret' : 'plain',
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
      kind: 'plain',
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
      kind: v.kind === 'secret' ? 'secret' : 'plain',
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
  if (mainWorkspaceMode.value === 'request') {
    captureCurrentPanelSnapshot()
  }
  mainWorkspaceMode.value = 'environment'
  selectedEnvironmentId.value = String(id)
}

function onToggleSettings() {
  if (mainWorkspaceMode.value === 'settings') {
    mainWorkspaceMode.value = workspaceBeforeSettings.value || 'request'
    workspaceBeforeSettings.value = null
    return
  }
  if (mainWorkspaceMode.value === 'request') {
    captureCurrentPanelSnapshot()
  }
  workspaceBeforeSettings.value = mainWorkspaceMode.value
  mainWorkspaceMode.value = 'settings'
}

function onMainWorkspaceRequest() {
  mainWorkspaceMode.value = 'request'
  selectedEnvironmentId.value = null
  workspaceBeforeSettings.value = null
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
  captureCurrentPanelSnapshot()
  tabsStore.closeTab(id)
}

function onTabNew() {
  captureCurrentPanelSnapshot()
  tabsStore.openBlank()
}

async function openNewTabFromDashboard() {
  tabsStore.openBlank()
  mainWorkspaceMode.value = 'request'
  await nextTick()
  await hydrateActiveTabIntoPanel()
}

function openCreateFolderFromDashboard() {
  sidebarRef.value?.openCreateRootFolder?.()
}

function openImportCollectionFromDashboard() {
  sidebarRef.value?.openImportCollectionModal?.()
}

function openCurlImportFromDashboard() {
  sidebarRef.value?.openCurlImportModal?.()
}

function openCreateEnvironmentFromDashboard() {
  mainWorkspaceMode.value = 'request'
  sidebarRef.value?.openCreateEnvironment?.()
}

function openCommandPalette() {
  commandPaletteOpen.value = true
}

function closeCommandPalette() {
  commandPaletteOpen.value = false
}

function openHelp() {
  helpOpen.value = true
}

function closeHelp() {
  helpOpen.value = false
}

function openRunner(folderId) {
  runnerSeedFolderId.value = folderId ? String(folderId) : ''
  runnerOpen.value = true
}

function closeRunner() {
  runnerOpen.value = false
}

function onPaletteFolderHit(hit) {
  mainWorkspaceMode.value = 'request'
  sidebarRef.value?.revealFolderHit?.(hit)
}

function onPaletteOpenSettings() {
  if (mainWorkspaceMode.value !== 'settings') {
    if (mainWorkspaceMode.value === 'request') {
      captureCurrentPanelSnapshot()
    }
    workspaceBeforeSettings.value = mainWorkspaceMode.value
    mainWorkspaceMode.value = 'settings'
  }
}

async function shortcutSendActive() {
  if (mainWorkspaceMode.value !== 'request' || !activeTab.value) return
  requestPanelRef.value?.sendCurrentRequest?.()
}

async function shortcutSaveActive() {
  if (mainWorkspaceMode.value !== 'request' || !activeTab.value) return
  requestPanelRef.value?.saveCurrentRequest?.()
}

async function shortcutNewTab() {
  captureCurrentPanelSnapshot()
  tabsStore.openBlank()
  mainWorkspaceMode.value = 'request'
  await nextTick()
  await hydrateActiveTabIntoPanel()
}

function shortcutCloseTab() {
  if (!tabsState.activeTabId) return
  onTabClose(tabsState.activeTabId)
}

function shortcutToggleEnvironment() {
  const active = envList.value.find((e) => e.is_active) || envList.value[0]
  if (active?.id) {
    onOpenEnvironment(active.id)
  }
}

function shortcutEscape() {
  if (helpOpen.value) {
    closeHelp()
    return
  }
  if (commandPaletteOpen.value) {
    closeCommandPalette()
    return
  }
  if (mainWorkspaceMode.value === 'settings') {
    onToggleSettings()
  }
}

useKeyboardShortcuts({
  send: shortcutSendActive,
  save: shortcutSaveActive,
  palette: openCommandPalette,
  newTab: shortcutNewTab,
  closeTab: shortcutCloseTab,
  toggleEnvironment: shortcutToggleEnvironment,
  escape: shortcutEscape
})

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

// Restore tabs on mount. Hydration runs from the watcher below once RequestPanel
// (defineAsyncComponent) has loaded and exposes hydrate() — not on the first nextTick.
onMounted(() => {
  tabsStore.restore()
})

// Keep RequestPanel in sync with the tab store whenever we can actually call hydrate():
// - after the async RequestPanel chunk loads (fixes cold start: tab titles from store but empty panel)
// - when switching tabs, or returning from Settings/Environment (panel remounts async again)
watch(
  () =>
    [
      mainWorkspaceMode.value,
      tabsState.activeTabId ?? '',
      requestPanelRef.value?.hydrate ? '1' : '0'
    ].join('|'),
  async () => {
    if (mainWorkspaceMode.value !== 'request' || !tabsState.activeTabId) return
    if (!requestPanelRef.value?.hydrate || !activeTab.value) return
    await nextTick()
    await hydrateActiveTabIntoPanel()
  },
  { flush: 'post' }
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
        @saved-request="onSavedRequestUpdated"
        @run-folder="(id) => openRunner(id)"
      />
    </div>

    <main
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      style="min-width: 0; min-height: 0; height: 100%"
    >
      <template v-if="mainWorkspaceMode === 'request' || mainWorkspaceMode === 'settings' || mainWorkspaceMode === 'environment'">
        <div
          class="flex shrink-0 flex-wrap items-center gap-3 border-b border-[#2a2a2a] bg-[#252525] px-4 py-2"
        >
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <span class="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-gray-500">Active environment</span>
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
          <button
            type="button"
            class="flex h-9 shrink-0 items-center justify-center gap-1.5 rounded border border-gray-600 bg-[#1a1a1a] px-3 text-xs font-semibold text-gray-200 hover:border-orange-500 hover:text-orange-200"
            title="Open Collection Runner"
            aria-label="Open Collection Runner"
            @click="() => openRunner(activeRootFolderId)"
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" class="h-4 w-4" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 0 1 0 1.971l-11.54 6.347a1.125 1.125 0 0 1-1.667-.985V5.653Z" />
            </svg>
            Runner
          </button>
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded border border-gray-600 bg-[#1a1a1a] text-sm font-bold text-gray-300 hover:border-orange-500 hover:text-orange-200"
            title="Help and keyboard shortcuts"
            aria-label="Help and keyboard shortcuts"
            @click="openHelp"
          >
            ?
          </button>
          <button
            type="button"
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded border text-gray-300 hover:border-orange-500 hover:text-orange-200"
            :class="
              mainWorkspaceMode === 'settings'
                ? 'border-orange-500/60 bg-orange-500/10 text-orange-200'
                : 'border-gray-600 bg-[#1a1a1a]'
            "
            :title="mainWorkspaceMode === 'settings' ? 'Close settings' : 'App settings'"
            :aria-label="mainWorkspaceMode === 'settings' ? 'Close settings' : 'App settings'"
            @click="onToggleSettings"
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" class="h-5 w-5" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.632 6.632 0 0 1 0 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.37.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.544-.56.942-1.11.942h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.37-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.632 6.632 0 0 1 0-.255c.007-.378-.138-.75-.43-.99l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.49l1.217.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.213-1.281Z"
              />
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
            </svg>
          </button>
        </div>
      </template>

      <template v-if="mainWorkspaceMode === 'request'">
        <RequestTabBar
          :tabs="tabsMeta"
          :active-tab-id="tabsState.activeTabId"
          @activate="onTabActivate"
          @close="onTabClose"
          @new="onTabNew"
        />
        <DashboardHome
          v-if="!activeTab"
          @new-tab="openNewTabFromDashboard"
          @new-folder="openCreateFolderFromDashboard"
          @import-collection="openImportCollectionFromDashboard"
          @import-curl="openCurlImportFromDashboard"
          @new-environment="openCreateEnvironmentFromDashboard"
          @open-help="openHelp"
          @open-saved-request="onOpenSavedRequest"
          @console="onRequestConsole"
        />
        <div v-else class="flex min-h-0 min-w-0 flex-1 flex-row overflow-hidden">
          <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border-r border-[#2a2a2a]">
            <RequestPanel
              ref="requestPanelRef"
              :active-root-folder-id="activeRootFolderId"
              :declared-env-keys="activeEnvDeclaredKeys"
              :active-env-values="activeEnvValues"
              :active-env-variables="activeEnvForPatch?.variables || []"
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
      <SettingsPanel
        v-else-if="mainWorkspaceMode === 'settings'"
        class="min-h-0 flex-1"
        :active="true"
        @console="onRequestConsole"
      />
      <EnvironmentDetailPanel
        v-else-if="mainWorkspaceMode === 'environment'"
        :environment-id="selectedEnvironmentId"
        @console="onRequestConsole"
        @saved="onEnvironmentChanged"
        @deleted="() => onEnvironmentDeleted(null)"
      />
    </main>

    <CommandPalette
      :open="commandPaletteOpen"
      @close="closeCommandPalette"
      @open-saved-request="onOpenSavedRequest"
      @open-folder-hit="onPaletteFolderHit"
      @open-environment="onOpenEnvironment"
      @new-tab="shortcutNewTab"
      @new-folder="openCreateFolderFromDashboard"
      @import-collection="openImportCollectionFromDashboard"
      @import-curl="openCurlImportFromDashboard"
      @new-environment="openCreateEnvironmentFromDashboard"
      @open-settings="onPaletteOpenSettings"
      @console="onRequestConsole"
    />
    <HelpModal :open="helpOpen" @close="closeHelp" />
    <RunnerModal
      :open="runnerOpen"
      :folder-id="runnerSeedFolderId"
      @close="closeRunner"
      @console="onRequestConsole"
    />
  </div>
</template>
