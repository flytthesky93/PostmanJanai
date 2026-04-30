<script setup>
import { ref, watch } from 'vue'
import { PMJ_SCRIPT_HELP_ROWS } from '../data/pmjScriptReference.js'

const props = defineProps({
  open: { type: Boolean, default: false }
})

const emit = defineEmits(['close'])

const helpSection = ref('overview')
const scriptingRows = PMJ_SCRIPT_HELP_ROWS

watch(
  () => props.open,
  (v) => {
    if (v) helpSection.value = 'overview'
  }
)

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
    title: 'Captures & Assertions',
    text: 'Saved requests gain tabs: Captures (extract JSON / headers / … into env or memory), Assertions (phase-8 rule checks), plus Pre-request and Post-response script (phase 9, pmj API). Response → Results aggregates script console, script tests, captures, and assertions.'
  },
  {
    title: 'Scripting hints',
    text: 'In Pre-request / Post-response editors use Ctrl + Space for IDE-style suggestions (pmj.*, console, JSON). JavaScript runs in the goja sandbox; see Help → Scripting (pmj). Post-response exposes response + test; Pre-request exposes request.*'
  },
  {
    title: 'Collection Runner',
    text: 'Click Runner in the header (or right-click a folder → Run folder…) to run all requests under a folder sequentially. Captures chain into the next request, totals stream live, and reports can be exported as JSON or Markdown.'
  },
  {
    title: 'Runner options',
    text: 'In the Runner modal you can repeat the plan up to 50 times (Iterations), pause between requests up to 60s (Delay), and shorten the wait for a hung server (Timeout per request). Stop on first failure halts the run early; Cancel works during delays too.'
  },
  {
    title: 'Runner request detail',
    text: 'Click any row in a run to see the resolved request and response that was actually sent and received — variable substitution is already applied, so you no longer need to re-run a request to debug it.'
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
      class="fixed inset-0 z-[90] flex items-start justify-center bg-black/60 px-4 pt-[6vh]"
      role="presentation"
      @mousedown.self="close"
    >
      <div
        class="app-scrollbar flex max-h-[88vh] w-full max-w-4xl flex-col overflow-auto rounded-xl border border-gray-700 bg-[#1f1f1f] shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-title"
        @mousedown.stop
        @keydown.escape.prevent="close"
      >
        <div class="sticky top-0 z-10 shrink-0 border-b border-gray-700 bg-[#1f1f1f]/95 backdrop-blur-sm px-5 py-4">
          <div class="flex items-start justify-between gap-4">
            <div>
              <p class="text-[10px] font-bold uppercase tracking-widest text-orange-400">Help</p>
              <h2 id="help-title" class="mt-1 text-lg font-semibold text-white">Using PostmanJanai Faster</h2>
              <p class="mt-1 text-sm text-gray-400">
                Shortcuts · tips · scripting reference (<span class="font-mono text-gray-300">pmj</span>
                <span class="text-gray-500">/</span>
                <span class="font-mono text-gray-300">pm</span>).
              </p>
            </div>
            <button
              type="button"
              class="rounded border border-gray-600 bg-[#2a2a2a] px-2 py-1 text-xs font-semibold text-gray-300 hover:border-orange-500/60 hover:text-orange-200"
              @click="close"
            >
              Close
            </button>
          </div>

          <div class="mt-4 flex flex-wrap gap-2 border-t border-gray-800 pt-4">
            <button
              type="button"
              class="rounded-t px-3 py-2 text-xs font-semibold transition-colors"
              :class="
                helpSection === 'overview'
                  ? 'border-b-2 border-orange-500 text-white'
                  : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
              "
              @click="helpSection = 'overview'"
            >
              Shortcuts · Tips
            </button>
            <button
              type="button"
              class="rounded-t px-3 py-2 text-xs font-semibold transition-colors"
              :class="
                helpSection === 'scripting'
                  ? 'border-b-2 border-orange-500 text-white'
                  : 'border-b-2 border-transparent text-gray-500 hover:text-gray-300'
              "
              @click="helpSection = 'scripting'"
            >
              Scripting (pmj)
            </button>
          </div>
        </div>

        <div v-if="helpSection === 'overview'" class="grid gap-5 p-5 lg:grid-cols-[1fr_1fr]">
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

        <div v-else class="p-5">
          <p class="text-sm leading-relaxed text-gray-400">
            JavaScript được chạy trong sandbox (engine goja): không có Node/builtin <span class="font-mono text-gray-400">require</span> hay I/O filesystem.
            <span class="font-mono text-orange-400/90">pmj</span> là API trong app —
            <span class="font-mono text-orange-400/90">pm</span> chỉ là alias trong script. Trình soạn script (tab Pre-request / Post-response của request đã
            lưu): <kbd class="rounded border border-gray-600 bg-[#151515] px-1 py-px font-mono text-[11px] text-orange-200">Ctrl</kbd>
            <kbd class="ml-1 rounded border border-gray-600 bg-[#151515] px-1 py-px font-mono text-[11px] text-orange-200">Space</kbd>
            để mở gợi ý gõ nhanh (pmj, console.log, sau dấu
            <span class="font-mono text-gray-400">.</span>
            các thành viên request/response/environment… tuỳ giai đoạn).
          </p>

          <div class="mt-4 overflow-x-auto rounded-lg border border-gray-700 bg-[#181818]">
            <table class="pj-help-table min-w-[640px] w-full border-collapse text-left text-xs">
              <thead>
                <tr class="sticky top-0 border-b border-gray-700 bg-[#121212] text-[10px] font-bold uppercase tracking-wider text-gray-400">
                  <th class="px-3 py-2.5">Thành phần</th>
                  <th class="px-3 py-2.5">Ví dụ / cú pháp</th>
                  <th class="w-24 px-2 py-2.5 text-center">Trước gửi</th>
                  <th class="w-24 px-2 py-2.5 text-center">Sau phản hồi</th>
                  <th class="px-3 py-2.5 min-w-[200px]">Mô tả</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-800 text-gray-300">
                <tr v-for="(row, i) in scriptingRows" :key="i">
                  <td class="px-3 py-2.5 align-top font-medium text-gray-200">{{ row.component }}</td>
                  <td class="px-3 py-2.5 align-top font-mono text-[11px] text-orange-200/95">{{ row.example }}</td>
                  <td class="px-2 py-2.5 align-top text-center text-gray-400">{{ row.pre ? '✓' : '—' }}</td>
                  <td class="px-2 py-2.5 align-top text-center text-gray-400">{{ row.post ? '✓' : '—' }}</td>
                  <td class="px-3 py-2.5 align-top leading-relaxed text-gray-400">{{ row.description }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>
