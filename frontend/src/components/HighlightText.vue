<script setup>
import { computed } from 'vue'

const props = defineProps({
  text: { type: String, default: '' },
  query: { type: String, default: '' }
})

/**
 * Break `text` into alternating plain/matched segments based on a
 * case-insensitive substring match of `query`. No regex escaping concerns:
 * we scan indexOf on a lowered copy and slice the original to preserve case.
 */
const segments = computed(() => {
  const text = String(props.text ?? '')
  const q = String(props.query ?? '').trim()
  if (!q || !text) {
    return [{ type: 'plain', value: text }]
  }
  const haystack = text.toLowerCase()
  const needle = q.toLowerCase()
  const out = []
  let i = 0
  while (i < text.length) {
    const hit = haystack.indexOf(needle, i)
    if (hit < 0) {
      out.push({ type: 'plain', value: text.slice(i) })
      break
    }
    if (hit > i) {
      out.push({ type: 'plain', value: text.slice(i, hit) })
    }
    out.push({ type: 'match', value: text.slice(hit, hit + needle.length) })
    i = hit + needle.length
    // Zero-length safety.
    if (needle.length === 0) break
  }
  return out
})
</script>

<template>
  <span>
    <template v-for="(seg, idx) in segments" :key="idx">
      <mark
        v-if="seg.type === 'match'"
        class="rounded bg-orange-500/30 px-[1px] text-orange-200"
      >{{ seg.value }}</mark>
      <template v-else>{{ seg.value }}</template>
    </template>
  </span>
</template>
