<script setup>
defineProps({
  open: { type: Boolean, default: false },
  varKey: { type: String, default: '' },
  modelValue: { type: String, default: '' },
  position: { type: Object, default: () => ({ left: 0, top: 0 }) },
  readOnly: { type: Boolean, default: false },
  /** Label like {{name}} */
  chipLabel: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue', 'apply', 'close', 'enter', 'leave', 'keydown'])
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open && varKey"
      class="fixed z-[10000] w-[min(92vw,280px)] rounded-lg border border-gray-600 bg-[#252525] p-2.5 shadow-xl"
      :style="{ left: position.left + 'px', top: position.top + 'px' }"
      role="dialog"
      aria-label="Environment variable"
      @mouseenter="emit('enter')"
      @mouseleave="emit('leave')"
      @keydown="emit('keydown', $event)"
    >
      <div
        class="mb-1.5 truncate text-[10px] font-bold uppercase tracking-wide text-orange-400/90"
        :title="varKey"
      >
        {{ chipLabel }}
      </div>
      <label class="sr-only" :for="'env-pop-' + varKey">Value in active environment</label>
      <input
        :id="'env-pop-' + varKey"
        :value="modelValue"
        type="text"
        class="mb-2 w-full rounded border border-gray-600 bg-gray-900 px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
        :class="readOnly ? 'cursor-not-allowed opacity-70' : ''"
        placeholder="(empty)"
        autocomplete="off"
        :readonly="readOnly"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown.enter.prevent="!readOnly && $emit('apply')"
      />
      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="rounded border border-gray-600 px-2 py-1 text-[11px] text-gray-400 hover:bg-gray-800"
          @click="$emit('close')"
        >
          Close
        </button>
        <button
          v-if="!readOnly"
          type="button"
          class="rounded bg-orange-600 px-2.5 py-1 text-[11px] font-semibold text-white hover:bg-orange-700"
          @click="$emit('apply')"
        >
          Apply
        </button>
      </div>
    </div>
  </Teleport>
</template>
