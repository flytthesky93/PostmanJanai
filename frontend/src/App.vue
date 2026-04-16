<script setup>
import { ref } from 'vue'
import Sidebar from './components/Sidebar.vue'
import RequestPanel from './components/RequestPanel.vue'
import ResponsePanel from './components/ResponsePanel.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import { Execute } from '../wailsjs/wailsjs/go/delivery/HTTPHandler'

const responseResult = ref(null)
const loading = ref(false)

/** Console log lines (below Response); used for Format JSON errors from Request. */
const consoleLines = ref([])
let consoleLineId = 0

function pushConsole(text) {
  consoleLineId += 1
  consoleLines.value.push({ id: consoleLineId, text })
  if (consoleLines.value.length > 200) {
    consoleLines.value.splice(0, consoleLines.value.length - 200)
  }
  consoleExpanded.value = true
}

function clearConsole() {
  consoleLines.value = []
}

function onRequestConsole(msg) {
  if (typeof msg === 'string' && msg) {
    pushConsole(msg)
  }
}

/** Console expanded/collapsed (default collapsed to give space to Response). */
const consoleExpanded = ref(false)

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
      <Sidebar />
    </div>

    <main
      class="flex min-h-0 min-w-0 flex-1 flex-row overflow-hidden"
      style="min-width: 0; min-height: 0; height: 100%"
    >
      <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border-r border-[#2a2a2a]">
        <RequestPanel @send="onExecuteRequest" @console="onRequestConsole" />
      </div>
      <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <ResponsePanel :result="responseResult" :loading="loading" />
        <ConsolePanel
          v-model:expanded="consoleExpanded"
          :lines="consoleLines"
          @clear="clearConsole"
        />
      </div>
    </main>
  </div>
</template>
