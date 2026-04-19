<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import EnvVarPopover from './EnvVarPopover.vue'
import { useEnvPopover } from '../composables/useEnvPopover.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
  /** Keys defined (and enabled) in the active environment — used for ! warning. */
  declaredKeys: { type: Array, default: () => [] },
  /** key → current value in active env (hover popover). */
  envValues: { type: Object, default: () => ({}) },
  placeholder: { type: String, default: '' },
  /** Use textarea + wrapped lines; otherwise single-line input (URL, table cells). */
  multiline: { type: Boolean, default: false },
  rows: { type: Number, default: 6 },
  /** Extra classes on the focus ring wrapper */
  wrapperClass: { type: String, default: '' },
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'keydown', 'patch-env-value'])

const local = ref(props.modelValue ?? '')
const mirrorRef = ref(/** @type {HTMLElement | null} */ (null))
const inputRef = ref(/** @type {HTMLInputElement | HTMLTextAreaElement | null} */ (null))
const rawRef = ref(/** @type {HTMLInputElement | HTMLTextAreaElement | null} */ (null))
/** True: visible text editor for full raw string (incl. `{{var}}`). False: chip mirror view. */
const rawMode = ref(false)

/** @type {CanvasRenderingContext2D | null} */
let measureCtx = /** @type {CanvasRenderingContext2D | null} */ (null)
function getMeasureCtx() {
  if (!measureCtx) {
    const c = document.createElement('canvas')
    measureCtx = c.getContext('2d')
  }
  return measureCtx
}

function setFontFromEl(el) {
  const s = window.getComputedStyle(el)
  const ctx = getMeasureCtx()
  ctx.font = `${s.fontWeight} ${s.fontSize} ${s.fontFamily}`
}

watch(
  () => props.modelValue,
  (v) => {
    const n = v ?? ''
    if (n !== local.value) local.value = n
  }
)

watch(local, (v) => {
  emit('update:modelValue', v)
}, { flush: 'sync' })

const declaredSet = computed(() => {
  const s = new Set()
  for (const k of props.declaredKeys || []) {
    if (typeof k === 'string' && k.trim()) s.add(k.trim())
  }
  return s
})

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function highlightHtml(text) {
  const t = text ?? ''
  const re = /\{\{\s*([^{}]*?)\s*\}\}/g
  let last = 0
  let out = ''
  let m
  while ((m = re.exec(t)) !== null) {
    out += escapeHtml(t.slice(last, m.index))
    const name = (m[1] || '').trim()
    const show = name || 'empty'
    const ok = name && declaredSet.value.has(name)
    const warn = name && !ok
    const chipClass = ok
      ? 'env-var-chip env-var-chip--ok'
      : 'env-var-chip env-var-chip--warn'
    out += `<span class="${chipClass}" title="Hover: value in env · Double-click: raw edit"><span class="env-var-chip__text">${escapeHtml(show)}</span>${warn ? '<span class="env-var-chip__bang" title="Not in active environment">!</span>' : ''}</span>`
    last = m.index + m[0].length
  }
  out += escapeHtml(t.slice(last))
  if (!out) out = '<span class="env-var-mirror__empty"> </span>'
  return out
}

const hl = computed(() => highlightHtml(local.value))

function syncScroll(e) {
  const el = e.target
  const mir = mirrorRef.value
  if (!mir) return
  mir.scrollTop = el.scrollTop
  mir.scrollLeft = el.scrollLeft
}

function onKeydownMultiline(e) {
  if (e.key === 'Escape') e.stopPropagation()
}

/** Caret index `pos` falls inside a closed `{{ … }}` span (half-open [start, end)). */
function isPosInsidePlaceholder(text, pos) {
  const t = text ?? ''
  const re = /\{\{\s*([^{}]*?)\s*\}\}/g
  let m
  while ((m = re.exec(t)) !== null) {
    const end = m.index + m[0].length
    if (pos >= m.index && pos < end) return true
  }
  return false
}

function onChipViewDblClick(e) {
  if (props.disabled) return
  const el = e.target
  if (!(el instanceof HTMLInputElement) && !(el instanceof HTMLTextAreaElement)) return
  const pos = typeof el.selectionStart === 'number' ? el.selectionStart : 0
  if (!isPosInsidePlaceholder(local.value, pos)) return
  e.preventDefault()
  closeEnvPopover()
  rawMode.value = true
  nextTick(() => {
    const r = rawRef.value
    if (!r) return
    r.focus()
    try {
      const len = local.value.length
      const p = Math.min(pos, len)
      r.setSelectionRange(p, p)
    } catch {
      /* ignore */
    }
  })
}

function exitRawMode() {
  rawMode.value = false
  nextTick(() => {
    const el = inputRef.value
    el?.focus()
  })
}

function onRawKeydown(e) {
  emit('keydown', e)
  if (e.key === 'Escape') {
    e.preventDefault()
    exitRawMode()
  }
}

let rawBlurTimer = /** @type {ReturnType<typeof setTimeout> | null} */ (null)
function onRawBlur() {
  rawBlurTimer = setTimeout(() => {
    rawBlurTimer = null
    rawMode.value = false
  }, 150)
}
function onRawFocus() {
  if (rawBlurTimer) {
    clearTimeout(rawBlurTimer)
    rawBlurTimer = null
  }
}

const {
  popoverOpen,
  popoverKey,
  popoverDraft,
  popoverPos,
  chipLabel: popoverChipLabel,
  openForKey,
  close: closeEnvPopover,
  scheduleHide: scheduleHidePopover,
  schedulePointerUpdate,
  onPopoverEnter,
  onPopoverLeave,
  applyPatch: applyEnvPatch,
  onPopoverKeydown
} = useEnvPopover({
  envValues: () => props.envValues,
  isReadOnly: () => props.disabled,
  onPatch: (p) => emit('patch-env-value', p)
})

function indexFromPointerInput(el, text, clientX) {
  const s = window.getComputedStyle(el)
  const padL = parseFloat(s.paddingLeft) + parseFloat(s.borderLeftWidth)
  const rect = el.getBoundingClientRect()
  const relX = clientX - rect.left - padL + el.scrollLeft
  if (relX <= 0) return 0
  setFontFromEl(el)
  const ctx = getMeasureCtx()
  let lo = 0
  let hi = text.length
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (ctx.measureText(text.slice(0, mid)).width <= relX) lo = mid
    else hi = mid - 1
  }
  return lo
}

function lineHeightPx(el) {
  const s = window.getComputedStyle(el)
  const lh = String(s.lineHeight || '')
  if (lh.endsWith('px')) return parseFloat(lh)
  const fs = parseFloat(s.fontSize) || 14
  if (lh === 'normal' || !lh) return fs * 1.25
  const n = parseFloat(lh)
  return Number.isFinite(n) ? n * fs : fs * 1.25
}

/** Visual lines as [start,end) in `text`, using char-wrap to match monospace-ish textarea. */
function buildWrappedSegments(el, text) {
  const s = window.getComputedStyle(el)
  const padL = parseFloat(s.paddingLeft) + parseFloat(s.borderLeftWidth)
  const padR = parseFloat(s.paddingRight) + parseFloat(s.borderRightWidth)
  const maxW = el.clientWidth - padL - padR
  if (!text.length) return [{ start: 0, end: 0 }]
  if (maxW < 4) return [{ start: 0, end: text.length }]
  setFontFromEl(el)
  const ctx = getMeasureCtx()
  const segs = []
  let i = 0
  while (i < text.length) {
    if (text[i] === '\n') {
      segs.push({ start: i, end: i + 1 })
      i++
      continue
    }
    const lineStart = i
    let j = i
    let w = 0
    while (j < text.length && text[j] !== '\n') {
      const cw = ctx.measureText(text[j]).width
      if (w + cw > maxW && j > lineStart) break
      w += cw
      j++
    }
    if (j === lineStart) j = Math.min(i + 1, text.length)
    segs.push({ start: lineStart, end: j })
    i = j
  }
  return segs.length ? segs : [{ start: 0, end: 0 }]
}

function indexInSegment(el, text, seg, relX) {
  const slice = text.slice(seg.start, seg.end)
  if (relX <= 0) return seg.start
  setFontFromEl(el)
  const ctx = getMeasureCtx()
  let lo = 0
  let hi = slice.length
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (ctx.measureText(slice.slice(0, mid)).width <= relX) lo = mid
    else hi = mid - 1
  }
  return Math.min(seg.start + lo, text.length)
}

function indexFromPointerTextarea(el, text, clientX, clientY) {
  const s = window.getComputedStyle(el)
  const padL = parseFloat(s.paddingLeft) + parseFloat(s.borderLeftWidth)
  const padT = parseFloat(s.paddingTop) + parseFloat(s.borderTopWidth)
  const rect = el.getBoundingClientRect()
  const relY = clientY - rect.top - padT + el.scrollTop
  const relX = clientX - rect.left - padL + el.scrollLeft
  const lh = lineHeightPx(el)
  const segs = buildWrappedSegments(el, text)
  let lineIdx = Math.floor(relY / lh)
  if (lineIdx < 0) lineIdx = 0
  if (lineIdx >= segs.length) lineIdx = segs.length - 1
  const seg = segs[lineIdx]
  return indexInSegment(el, text, seg, relX)
}

function placeholderKeyAtIndex(text, pos) {
  const re = /\{\{\s*([^{}]*?)\s*\}\}/g
  let m
  while ((m = re.exec(text)) !== null) {
    const name = (m[1] || '').trim()
    const start = m.index
    const end = start + m[0].length
    if (pos >= start && pos < end) return name || null
  }
  return null
}

function updatePopoverFromPointer(e) {
  if (props.disabled || rawMode.value) return
  const el = inputRef.value
  if (!el) return
  const text = local.value ?? ''
  const idx = props.multiline
    ? indexFromPointerTextarea(el, text, e.clientX, e.clientY)
    : indexFromPointerInput(el, text, e.clientX)
  const key = placeholderKeyAtIndex(text, idx)
  if (!key) {
    scheduleHidePopover()
    return
  }
  openForKey(key, e.clientX, e.clientY)
}

function onChipPointerMove(e) {
  if (props.disabled || rawMode.value) return
  schedulePointerUpdate(() => updatePopoverFromPointer(e))
}

function onChipPointerLeave() {
  scheduleHidePopover()
}

watch(rawMode, (raw) => {
  if (raw) closeEnvPopover()
})
</script>

<template>
  <div
    class="env-var-mirror-outer rounded border border-gray-700 bg-gray-900 transition-colors focus-within:border-orange-500"
    :class="[wrapperClass, disabled ? 'cursor-not-allowed opacity-60' : '']"
    :title="
      rawMode
        ? 'Escape: return to tag view'
        : 'Hover {{var}} for env value · Double-click inside {{}} for raw text'
    "
  >
    <!-- Raw editing: full visible string -->
    <div v-if="rawMode" class="w-full" :class="multiline ? 'min-h-[6rem]' : 'min-h-[2.5rem]'">
      <input
        v-if="!multiline"
        ref="rawRef"
        v-model="local"
        type="text"
        class="w-full rounded border-0 bg-transparent px-3 py-2 font-sans text-sm leading-normal text-gray-200 outline-none"
        :placeholder="placeholder"
        :disabled="disabled"
        autocomplete="off"
        spellcheck="false"
        @keydown="onRawKeydown"
        @focus="onRawFocus"
        @blur="onRawBlur"
      />
      <textarea
        v-else
        ref="rawRef"
        v-model="local"
        class="w-full resize-y border-0 bg-transparent px-2 py-1.5 font-mono text-xs leading-normal text-gray-200 outline-none"
        :placeholder="placeholder"
        :disabled="disabled"
        :rows="rows"
        spellcheck="false"
        @keydown="onRawKeydown"
        @focus="onRawFocus"
        @blur="onRawBlur"
      />
    </div>

    <!-- Chip mirror view -->
    <div
      v-else
      class="relative w-full"
      :class="multiline ? 'min-h-[6rem]' : 'min-h-[2.5rem]'"
    >
      <pre
        ref="mirrorRef"
        class="env-var-mirror__pre pointer-events-none absolute inset-0 m-0 overflow-hidden text-gray-200"
        :class="
          multiline
            ? 'min-h-full whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-xs leading-normal'
            : 'whitespace-pre px-3 py-2 font-sans text-sm leading-normal'
        "
        aria-hidden="true"
        v-html="hl"
      />
      <input
        v-if="!multiline"
        ref="inputRef"
        v-model="local"
        type="text"
        class="relative z-10 min-h-[2.5rem] w-full border-0 bg-transparent px-3 py-2 font-sans text-sm leading-normal text-transparent caret-orange-400 outline-none"
        :placeholder="placeholder"
        :disabled="disabled"
        autocomplete="off"
        spellcheck="false"
        @keydown="(e) => emit('keydown', e)"
        @dblclick="onChipViewDblClick"
        @mousemove="onChipPointerMove"
        @mouseleave="onChipPointerLeave"
      />
      <textarea
        v-else
        ref="inputRef"
        v-model="local"
        class="relative z-10 min-h-full w-full resize-y border-0 bg-transparent px-2 py-1.5 font-mono text-xs leading-normal text-transparent caret-orange-400 outline-none"
        :placeholder="placeholder"
        :disabled="disabled"
        :rows="rows"
        spellcheck="false"
        @scroll="syncScroll"
        @keydown="
          (e) => {
            onKeydownMultiline(e)
            emit('keydown', e)
          }
        "
        @dblclick="onChipViewDblClick"
        @mousemove="onChipPointerMove"
        @mouseleave="onChipPointerLeave"
      />
    </div>

    <EnvVarPopover
      :open="popoverOpen"
      :var-key="popoverKey"
      v-model="popoverDraft"
      :position="popoverPos"
      :read-only="disabled"
      :chip-label="popoverChipLabel"
      @enter="onPopoverEnter"
      @leave="onPopoverLeave"
      @keydown="onPopoverKeydown"
      @close="closeEnvPopover"
      @apply="applyEnvPatch"
    />
  </div>
</template>

<style scoped>
.env-var-mirror__pre {
  scrollbar-width: none;
}
.env-var-mirror__pre::-webkit-scrollbar {
  display: none;
}

.env-var-mirror-outer :deep(.env-var-chip) {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin: 0 1px;
  padding: 0 6px;
  border-radius: 6px;
  vertical-align: baseline;
  font-size: 0.85em;
  font-weight: 600;
  line-height: 1.35;
  animation: envVarChipIn 0.22s ease-out;
  transition:
    transform 0.15s ease,
    box-shadow 0.15s ease;
}

.env-var-mirror-outer :deep(.env-var-chip--ok) {
  border: 1px solid rgba(217, 119, 6, 0.45);
  background: rgba(120, 53, 15, 0.45);
  color: #fde68a;
  box-shadow: 0 0 0 1px rgba(251, 191, 36, 0.12);
}

.env-var-mirror-outer :deep(.env-var-chip--warn) {
  border: 1px solid rgba(239, 68, 68, 0.45);
  background: rgba(127, 29, 29, 0.4);
  color: #fecaca;
  box-shadow: 0 0 0 1px rgba(248, 113, 113, 0.2);
}

.env-var-mirror-outer :deep(.env-var-chip__bang) {
  font-weight: 800;
  color: #fcd34d;
  font-size: 0.95em;
  line-height: 1;
}

@keyframes envVarChipIn {
  from {
    opacity: 0;
    transform: scale(0.94);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
