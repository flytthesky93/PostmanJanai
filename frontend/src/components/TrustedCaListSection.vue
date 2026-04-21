<script setup>
/**
 * Danh sách chứng chỉ CA đang lưu trong app (DB) — dùng trong màn Settings.
 */
defineProps({
  items: { type: Array, default: () => [] },
  disabled: { type: Boolean, default: false }
})

defineEmits(['toggle', 'remove'])

function shortId(id) {
  const s = String(id || '')
  if (s.length <= 14) return s
  return `${s.slice(0, 8)}…${s.slice(-4)}`
}
</script>

<template>
  <section>
    <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Trusted CA certificates (in app)</div>
    <p class="mt-1 text-xs text-gray-500">
      Certificates stored in the local database and merged with the system trust store for HTTPS. Toggle to enable or disable
      without deleting.
    </p>

    <div class="mt-3 overflow-hidden rounded border border-gray-800 bg-[#141414]">
      <div v-if="!items.length" class="px-3 py-4 text-xs text-gray-600">No custom CA certificates stored yet.</div>
      <ul v-else class="divide-y divide-gray-800">
        <li
          v-for="c in items"
          :key="c.id"
          class="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5 hover:bg-[#1a1a1a]/80"
        >
          <div class="min-w-0 flex-1">
            <div class="truncate font-mono text-xs font-semibold text-gray-200" :title="c.label">{{ c.label }}</div>
            <div class="mt-0.5 flex flex-wrap items-center gap-2 text-[10px] text-gray-500">
              <span class="font-mono" :title="c.id">id: {{ shortId(c.id) }}</span>
              <span v-if="c.created_at">{{ c.created_at }}</span>
              <span
                class="rounded px-1.5 py-0.5 font-semibold uppercase"
                :class="c.enabled ? 'bg-emerald-500/15 text-emerald-300' : 'bg-gray-700/50 text-gray-500'"
              >
                {{ c.enabled ? 'enabled' : 'disabled' }}
              </span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <label class="flex cursor-pointer items-center gap-1.5 text-[11px] text-gray-400">
              <input
                type="checkbox"
                class="rounded border-gray-600 bg-[#1a1a1a]"
                :checked="c.enabled"
                :disabled="disabled"
                @change="$emit('toggle', c.id, $event.target.checked)"
              />
              <span>On</span>
            </label>
            <button
              type="button"
              class="rounded px-2 py-1 text-[11px] text-red-400 hover:bg-red-500/10 hover:text-red-300 disabled:opacity-40"
              :disabled="disabled"
              @click="$emit('remove', c.id)"
            >
              Remove
            </button>
          </div>
        </li>
      </ul>
    </div>
  </section>
</template>
