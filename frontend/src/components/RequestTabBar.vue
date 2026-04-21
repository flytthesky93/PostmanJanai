<script setup>
const props = defineProps({
  tabs: { type: Array, required: true },
  activeTabId: { type: String, default: null }
})

const emit = defineEmits(['activate', 'close', 'new'])

function methodColor(m) {
  const x = String(m || 'GET').toUpperCase()
  switch (x) {
    case 'GET': return 'text-green-400'
    case 'POST': return 'text-orange-400'
    case 'PUT': return 'text-sky-400'
    case 'PATCH': return 'text-amber-300'
    case 'DELETE': return 'text-red-400'
    case 'HEAD': return 'text-purple-300'
    case 'OPTIONS': return 'text-gray-300'
    default: return 'text-gray-300'
  }
}

function onClose(e, id) {
  e.stopPropagation()
  e.preventDefault()
  emit('close', id)
}

function onMiddleClose(e, id) {
  // middle-click closes the tab (common IDE behaviour)
  if (e.button === 1) {
    e.preventDefault()
    emit('close', id)
  }
}
</script>

<template>
  <div
    class="flex shrink-0 items-stretch border-b border-[#2a2a2a] bg-[#1f1f1f]"
    role="tablist"
    aria-label="Open request tabs"
  >
    <div class="app-scrollbar flex min-w-0 flex-1 items-stretch overflow-x-auto">
      <button
        v-for="t in props.tabs"
        :key="t.id"
        type="button"
        role="tab"
        :aria-selected="t.id === props.activeTabId"
        :title="t.title + (t.dirty ? ' • unsaved changes' : '')"
        class="group relative flex min-w-[140px] max-w-[220px] shrink-0 items-center gap-2 border-r border-[#2a2a2a] px-3 py-2 text-xs transition-colors"
        :class="t.id === props.activeTabId
          ? 'bg-[#252525] text-gray-100'
          : 'bg-[#1a1a1a] text-gray-400 hover:bg-[#232323] hover:text-gray-200'"
        @click="emit('activate', t.id)"
        @mousedown="onMiddleClose($event, t.id)"
      >
        <span
          class="shrink-0 font-bold tabular-nums"
          :class="methodColor(t.method)"
          style="font-size: 10px; letter-spacing: 0.02em;"
        >{{ t.method }}</span>
        <span
          v-if="t.insecureTLS"
          class="shrink-0 rounded bg-red-500/20 px-1 py-0.5 text-[9px] font-bold uppercase tracking-wide text-red-200"
          title="TLS verification disabled for this request"
        >insec</span>
        <span class="min-w-0 flex-1 truncate text-left">
          <span
            v-if="t.dirty"
            class="mr-1 inline-block h-1.5 w-1.5 rounded-full bg-orange-400 align-middle"
            aria-label="Unsaved changes"
          />
          {{ t.title }}
        </span>
        <span
          class="ml-1 inline-flex h-4 w-4 shrink-0 items-center justify-center rounded text-gray-500 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-red-500/20 hover:text-red-300"
          :class="t.id === props.activeTabId ? 'opacity-70' : ''"
          role="button"
          aria-label="Close tab"
          @click="onClose($event, t.id)"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </span>
        <span
          v-if="t.id === props.activeTabId"
          class="absolute inset-x-0 bottom-0 h-[2px] bg-orange-500"
          aria-hidden="true"
        />
      </button>
    </div>
    <button
      type="button"
      class="flex shrink-0 items-center justify-center px-3 text-gray-500 hover:bg-[#252525] hover:text-orange-300"
      title="New request tab"
      aria-label="New request tab"
      @click="emit('new')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
    </button>
  </div>
</template>
