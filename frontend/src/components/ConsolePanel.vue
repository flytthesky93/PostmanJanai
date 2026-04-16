<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  /** @type {import('vue').PropType<Array<{ id: number, text: string }>>} */
  lines: {
    type: Array,
    default: () => []
  },
  /** When true, log area is visible; when false, only the header bar (collapsed). */
  expanded: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['clear', 'update:expanded'])

const scrollEl = ref(null)

function scrollToBottom() {
  nextTick(() => {
    const el = scrollEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

function toggleExpanded() {
  emit('update:expanded', !props.expanded)
}

watch(
  () => [props.lines.length, props.expanded],
  () => {
    if (props.expanded) scrollToBottom()
  }
)
</script>

<template>
  <div class="flex shrink-0 flex-col border-t border-gray-800 bg-[#141414]">
    <!-- Header bar: always visible; click to expand/collapse -->
    <div
      class="flex min-h-[32px] shrink-0 items-center gap-2 border-b border-gray-800 px-2 py-1.5"
      :class="expanded ? 'bg-[#141414]' : 'bg-[#181818]'"
    >
      <button
        type="button"
        class="flex min-w-0 flex-1 items-center gap-2 rounded text-left outline-none hover:bg-gray-800/50"
        :title="expanded ? 'Collapse console' : 'Expand console'"
        @click="toggleExpanded"
      >
        <span class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Console</span>
        <span v-if="lines.length" class="text-[10px] text-gray-600">({{ lines.length }})</span>
        <span class="ml-auto shrink-0 text-[10px] text-gray-500">{{ expanded ? 'Collapse' : 'Expand' }}</span>
        <svg
          class="h-4 w-4 shrink-0 text-gray-500 transition-transform duration-200"
          :class="expanded ? 'rotate-180' : ''"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      <button
        type="button"
        class="shrink-0 rounded px-2 py-0.5 text-[10px] text-gray-500 hover:bg-gray-800 hover:text-gray-300"
        title="Clear log"
        @click.stop="$emit('clear')"
      >
        Clear
      </button>
    </div>

    <!-- Log content (expanded only) -->
    <div
      v-show="expanded"
      ref="scrollEl"
      class="min-h-0 overflow-auto p-2 font-mono text-[11px] leading-relaxed"
      style="height: 140px; min-height: 100px; max-height: 35vh"
    >
      <div v-if="!lines.length" class="select-none text-gray-600">No messages.</div>
      <div v-for="line in lines" :key="line.id" class="mb-1 break-words text-red-300">
        {{ line.text }}
      </div>
    </div>
  </div>
</template>
