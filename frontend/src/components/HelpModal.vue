<script setup>
const props = defineProps({
  open: { type: Boolean, default: false }
})

const emit = defineEmits(['close'])

const shortcuts = [
  { keys: 'Ctrl/Cmd + K', action: 'Open command palette' },
  { keys: 'Ctrl/Cmd + Enter', action: 'Send the active request' },
  { keys: 'Ctrl/Cmd + S', action: 'Save or update the active request' },
  { keys: 'Ctrl/Cmd + T', action: 'Open a new request tab' },
  { keys: 'Ctrl/Cmd + W', action: 'Close the current request tab' },
  { keys: 'Ctrl/Cmd + Shift + E', action: 'Open the active environment editor' },
  { keys: 'Esc', action: 'Close palette, settings, or this help dialog' }
]

const tips = [
  {
    title: 'Dashboard',
    text: 'Close the last request tab to return to Dashboard. From there you can start a request, import, or jump to recent history.'
  },
  {
    title: 'Command Palette',
    text: 'Use Ctrl/Cmd + K to search folders, saved requests, environments, recent history, and common commands.'
  },
  {
    title: 'Variable Preview',
    text: 'When URL or raw body contains {{variables}}, the resolved preview appears below the editor. Secret values are masked as ***.'
  },
  {
    title: 'Duplicate',
    text: 'Use the three-dot menu on folders or requests to duplicate. Folder duplication copies nested folders and saved requests.'
  },
  {
    title: 'Copy as cURL',
    text: 'Use Copy cURL beside Send to copy a runnable cURL command using the same snippet pipeline as the Snippet panel.'
  },
  {
    title: 'Captures & Tests',
    text: 'When a request is saved, two extra tabs appear: Captures (extract JSON / header / status into env or memory variables after Send) and Tests (assertions like status eq 200). Results show under Response → Tests.'
  },
  {
    title: 'Collection Runner',
    text: 'Click Runner in the header (or right-click a folder → Run folder…) to run all requests under a folder sequentially. Captures chain into the next request, totals stream live, and reports can be exported as JSON or Markdown.'
  }
]

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="#app">
    <div
      v-if="props.open"
      class="fixed inset-0 z-[90] flex items-start justify-center bg-black/60 px-4 pt-[8vh]"
      role="presentation"
      @mousedown.self="close"
    >
      <div
        class="app-scrollbar max-h-[82vh] w-full max-w-3xl overflow-auto rounded-xl border border-gray-700 bg-[#1f1f1f] shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-title"
        @mousedown.stop
        @keydown.escape.prevent="close"
      >
        <div class="flex items-start justify-between gap-4 border-b border-gray-700 px-5 py-4">
          <div>
            <p class="text-[10px] font-bold uppercase tracking-widest text-orange-400">Help</p>
            <h2 id="help-title" class="mt-1 text-lg font-semibold text-white">Using PostmanJanai Faster</h2>
            <p class="mt-1 text-sm text-gray-400">Quick guide to productivity features and the collection runner.</p>
          </div>
          <button
            type="button"
            class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-1 text-xs font-semibold text-gray-300 hover:border-orange-500/60 hover:text-orange-200"
            @click="close"
          >
            Close
          </button>
        </div>

        <div class="grid gap-5 p-5 lg:grid-cols-[1fr_1fr]">
          <section class="rounded-lg border border-gray-700 bg-[#181818]">
            <div class="border-b border-gray-700 px-4 py-3">
              <h3 class="text-sm font-semibold text-white">Keyboard Shortcuts</h3>
            </div>
            <div class="divide-y divide-gray-800">
              <div v-for="item in shortcuts" :key="item.keys" class="flex items-start gap-3 px-4 py-3">
                <kbd class="shrink-0 rounded border border-gray-600 bg-[#111] px-2 py-1 font-mono text-[11px] text-orange-200">
                  {{ item.keys }}
                </kbd>
                <span class="text-sm text-gray-300">{{ item.action }}</span>
              </div>
            </div>
          </section>

          <section class="rounded-lg border border-gray-700 bg-[#181818]">
            <div class="border-b border-gray-700 px-4 py-3">
              <h3 class="text-sm font-semibold text-white">Productivity Tips</h3>
            </div>
            <div class="divide-y divide-gray-800">
              <div v-for="tip in tips" :key="tip.title" class="px-4 py-3">
                <div class="text-sm font-semibold text-gray-100">{{ tip.title }}</div>
                <p class="mt-1 text-xs leading-relaxed text-gray-400">{{ tip.text }}</p>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </Teleport>
</template>
