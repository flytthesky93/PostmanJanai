<script setup>
import { ref, shallowRef, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import EnvVarPopover from './EnvVarPopover.vue'
import { useEnvPopover } from '../composables/useEnvPopover.js'
import { EditorView, basicSetup } from 'codemirror'
import { autocompletion } from '@codemirror/autocomplete'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { xml } from '@codemirror/lang-xml'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorState, EditorSelection, Facet, Compartment, RangeSet, Transaction } from '@codemirror/state'
import { placeholder, ViewPlugin, Decoration, WidgetType } from '@codemirror/view'
import { createPmjCompletionSource } from '../data/pmjScriptReference.js'

const declaredEnvKeysFacet = Facet.define({
  combine: (values) => {
    const s = new Set()
    for (const v of values) {
      if (!v) continue
      for (const x of v) {
        if (typeof x === 'string' && x.trim()) s.add(x.trim())
      }
    }
    return s
  }
})

function normalizeEnvKeysToSet(arr) {
  const s = new Set()
  if (!Array.isArray(arr)) return s
  for (const k of arr) {
    if (typeof k === 'string' && k.trim()) s.add(k.trim())
  }
  return s
}

/** True if document index `pos` lies inside a `{{ … }}` span (half-open [start, end)). */
function posInsideEnvPlaceholder(text, pos) {
  const t = text ?? ''
  const re = /\{\{\s*([^{}]*?)\s*\}\}/g
  let m
  while ((m = re.exec(t)) !== null) {
    const end = m.index + m[0].length
    if (pos >= m.index && pos < end) return true
  }
  return false
}

/** Bridges CodeMirror widgets → Vue hover popover (same behaviour as URL env chips). */
const chipPopoverHandlersFacet = Facet.define({
  combine: (vals) => {
    for (let i = vals.length - 1; i >= 0; i--) {
      if (vals[i]) return vals[i]
    }
    return { onEnter() {}, onLeave() {} }
  }
})

/** Inline chip matching request URL env tags; document text stays `{{name}}` for substitution. */
class EnvVarChipWidget extends WidgetType {
  /**
   * @param {string} name trimmed var name (may be "")
   * @param {boolean} declared in active env
   * @param {{ onEnter?: (key: string, x: number, y: number) => void, onLeave?: () => void }} handlers
   */
  constructor(name, declared, handlers) {
    super()
    this.name = name
    this.declared = declared
    this.handlers = handlers || {}
  }

  eq(other) {
    return other instanceof EnvVarChipWidget && this.name === other.name && this.declared === other.declared
  }

  toDOM() {
    const wrap = document.createElement('span')
    wrap.className = this.declared ? 'cm-envVarChip cm-envVarChip--ok' : 'cm-envVarChip cm-envVarChip--warn'
    wrap.title = this.declared
      ? 'Hover: value in env · Double-click: raw {{…}}'
      : 'Not in active env · Hover: edit · Double-click: raw {{…}}'
    const label = document.createElement('span')
    label.className = 'cm-envVarChip__text'
    label.textContent = this.name ? this.name : 'empty'
    wrap.appendChild(label)
    if (this.name && !this.declared) {
      const bang = document.createElement('span')
      bang.className = 'cm-envVarChip__bang'
      bang.textContent = '!'
      bang.title = 'Not in active environment'
      wrap.appendChild(bang)
    }
    const h = this.handlers
    wrap.addEventListener('mouseenter', (e) => {
      h.onEnter?.(this.name, e.clientX, e.clientY)
    })
    wrap.addEventListener('mouseleave', () => {
      h.onLeave?.()
    })
    return wrap
  }

  ignoreEvent() {
    return false
  }
}

const envVarChipPlugin = ViewPlugin.fromClass(
  class {
    constructor(view) {
      this.decorations = this.build(view)
    }
    update(u) {
      this.decorations = this.build(u.view)
    }
    build(view) {
      const text = view.state.doc.toString()
      const declared = view.state.facet(declaredEnvKeysFacet)
      const handlers = view.state.facet(chipPopoverHandlersFacet)
      const re = /\{\{\s*([^{}]*?)\s*\}\}/g
      const deco = []
      let m
      while ((m = re.exec(text)) !== null) {
        const from = m.index
        const to = m.index + m[0].length
        const name = (m[1] || '').trim()
        const ok = name !== '' && declared.has(name)
        deco.push(
          Decoration.replace({
            widget: new EnvVarChipWidget(name, ok, handlers),
            inclusive: false,
            block: false
          }).range(from, to)
        )
      }
      if (deco.length === 0) return Decoration.none
      return Decoration.set(deco, true)
    }
  },
  { decorations: (v) => v.decorations }
)

/** RangeSet of `{{…}}` spans for atomic cursor motion (same spans as replace chips). */
function buildEnvPlaceholderAtomicSet(text) {
  const re = /\{\{\s*([^{}]*?)\s*\}\}/g
  const parts = []
  let m
  while ((m = re.exec(text)) !== null) {
    parts.push({ from: m.index, to: m.index + m[0].length, value: 1 })
  }
  if (parts.length === 0) return RangeSet.empty
  return RangeSet.of(parts)
}

function envPlaceholderAtomicRanges(view) {
  return buildEnvPlaceholderAtomicSet(view.state.doc.toString())
}

/** Mirrors `@codemirror/view` `skipAtomicRanges` (not exported). */
function skipEnvAtomicRanges(atoms, pos, bias) {
  for (;;) {
    let moved = 0
    for (const set of atoms) {
      set.between(pos - 1, pos + 1, (from, to) => {
        if (pos > from && pos < to) {
          const side = moved || bias || (pos - from < to - pos ? -1 : 1)
          pos = side < 0 ? from : to
          moved = side
        }
      })
    }
    if (!moved) return pos
  }
}

/** Mirrors `@codemirror/view` `skipAtomsForSelection` for env placeholders only. */
function snapSelectionPastEnvAtoms(sel, atoms) {
  let ranges = null
  for (let i = 0; i < sel.ranges.length; i++) {
    const range = sel.ranges[i]
    let updated = null
    if (range.empty) {
      const pos = skipEnvAtomicRanges(atoms, range.from, 0)
      if (pos !== range.from) updated = EditorSelection.cursor(pos, -1)
    } else {
      const from = skipEnvAtomicRanges(atoms, range.from, -1)
      const to = skipEnvAtomicRanges(atoms, range.to, 1)
      if (from !== range.from || to !== range.to) {
        updated = EditorSelection.range(
          range.from === range.anchor ? from : to,
          range.from === range.head ? from : to
        )
      }
    }
    if (updated) {
      if (!ranges) ranges = sel.ranges.slice()
      ranges[i] = updated
    }
  }
  return ranges ? EditorSelection.create(ranges, sel.mainIndex) : sel
}

/**
 * `EditorView.atomicRanges` only affects `moveByChar` / similar; pointer selection does not use it.
 * Snap caret / selection endpoints out of `{{…}}` so chips behave as one unit for clicks too.
 */
let envPlaceholderAtomicSnapReentrant = false
const envPlaceholderAtomicSnap = EditorView.updateListener.of((update) => {
  if (envPlaceholderAtomicSnapReentrant || !update.selectionSet) return
  const atoms = [buildEnvPlaceholderAtomicSet(update.state.doc.toString())]
  const next = snapSelectionPastEnvAtoms(update.state.selection, atoms)
  if (next.eq(update.state.selection, true)) return
  envPlaceholderAtomicSnapReentrant = true
  try {
    update.view.dispatch({
      selection: next,
      scrollIntoView: update.view.hasFocus,
      annotations: Transaction.addToHistory.of(false)
    })
  } finally {
    envPlaceholderAtomicSnapReentrant = false
  }
})

const props = defineProps({
  modelValue: { type: String, default: '' },
  readOnly: { type: Boolean, default: false },
  language: { type: String, default: 'json' },
  placeholder: { type: String, default: '' },
  /** Keys in active env (request editor only) — highlights {{name}}; missing → warn style + subtle animation. */
  declaredEnvKeys: { type: Array, default: () => [] },
  /** key → value in active env (hover popover). */
  envValues: { type: Object, default: () => ({}) },
  /** `pmj-pre` | `pmj-post` enables pmj API autocomplete in script editors */
  completionMode: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue', 'request-raw-edit', 'patch-env-value'])

const host = ref(null)
const view = shallowRef(null)
const envKeysCompartment = new Compartment()
let syncingFromParent = false

const {
  popoverOpen: chipPopoverOpen,
  popoverKey: chipPopoverKey,
  popoverDraft: chipPopoverDraft,
  popoverPos: chipPopoverPos,
  chipLabel: chipPopoverChipLabel,
  chipHandlers,
  close: closeChipEnvPopover,
  onPopoverEnter: onChipPopoverEnter,
  onPopoverLeave: onChipPopoverLeave,
  applyPatch: applyChipEnvPatch,
  onPopoverKeydown: onChipPopoverKeydown
} = useEnvPopover({
  envValues: () => props.envValues,
  isReadOnly: () => props.readOnly,
  onPatch: (p) => emit('patch-env-value', p)
})

function langExtension() {
  if (props.language === 'xml') return xml()
  if (props.language === 'javascript') return javascript({ jsx: false, typescript: false })
  return json()
}

function pmjAutocompleteExtensions() {
  const raw = String(props.completionMode || '').trim().toLowerCase()
  if (raw !== 'pmj-pre' && raw !== 'pmj-post') return []
  const phase = raw === 'pmj-post' ? 'post' : 'pre'
  return [
    autocompletion({
      override: [createPmjCompletionSource(phase)],
      maxRenderedOptions: 48,
      defaultKeymap: true,
      icons: false
    })
  ]
}

function buildExtensions() {
  const ph = String(props.placeholder || '').trim()
  const extensions = [
    basicSetup,
    langExtension(),
    oneDark,
    ...pmjAutocompleteExtensions()
  ]
  extensions.push(
    chipPopoverHandlersFacet.of(chipHandlers()),
    envKeysCompartment.of(declaredEnvKeysFacet.of(normalizeEnvKeysToSet(props.declaredEnvKeys))),
    envVarChipPlugin,
    EditorView.atomicRanges.of(envPlaceholderAtomicRanges),
    envPlaceholderAtomicSnap
  )
  if (!props.readOnly) {
    extensions.push(
      EditorView.domEventHandlers({
        dblclick: (event, view) => {
          const pos = view.posAtCoords({ x: event.clientX, y: event.clientY })
          if (pos == null) return false
          const text = view.state.doc.toString()
          if (!posInsideEnvPlaceholder(text, pos)) return false
          event.preventDefault()
          closeChipEnvPopover()
          emit('request-raw-edit')
          return true
        }
      })
    )
  }
  extensions.push(
    ...(ph ? [placeholder(ph)] : []),
    EditorState.readOnly.of(props.readOnly),
    EditorView.updateListener.of((u) => {
      if (u.docChanged && !syncingFromParent && !props.readOnly) {
        emit('update:modelValue', u.state.doc.toString())
      }
    }),
    EditorView.theme({
      '&': { height: '100%' },
      '.cm-scroller': {
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
        fontSize: '12px',
        lineHeight: '1.45'
      },
      '.cm-editor': {
        borderRadius: '0.375rem',
        backgroundColor: '#1e1e1e'
      },
      '.cm-editor.cm-focused': { outline: '2px solid rgba(249, 115, 22, 0.35)' },
      '.cm-gutters': { borderRight: '1px solid #2d2d2d' },
      '.cm-placeholder': { color: '#6b7280' },
      '.cm-envVarChip': {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '2px',
        margin: '0 1px',
        padding: '0 6px',
        borderRadius: '6px',
        verticalAlign: 'baseline',
        fontSize: '0.85em',
        fontWeight: '600',
        lineHeight: '1.35',
        boxDecorationBreak: 'clone'
      },
      '.cm-envVarChip--ok': {
        border: '1px solid rgba(217, 119, 6, 0.45)',
        backgroundColor: 'rgba(120, 53, 15, 0.45)',
        color: '#fde68a',
        boxShadow: '0 0 0 1px rgba(251, 191, 36, 0.12)'
      },
      '.cm-envVarChip--warn': {
        border: '1px solid rgba(239, 68, 68, 0.45)',
        backgroundColor: 'rgba(127, 29, 29, 0.4)',
        color: '#fecaca',
        boxShadow: '0 0 0 1px rgba(248, 113, 113, 0.2)'
      },
      '.cm-envVarChip__bang': {
        fontWeight: '800',
        color: '#fcd34d',
        fontSize: '0.95em',
        lineHeight: '1'
      }
    })
  )
  return extensions
}

function mountEditor() {
  if (!host.value) return
  const state = EditorState.create({
    doc: props.modelValue ?? '',
    extensions: buildExtensions()
  })
  view.value = new EditorView({ state, parent: host.value })
}

onMounted(() => {
  mountEditor()
})

watch(
  () => [props.language, props.placeholder, props.readOnly, props.completionMode],
  async () => {
    view.value?.destroy()
    view.value = null
    await nextTick()
    mountEditor()
  }
)

watch(
  () => props.declaredEnvKeys,
  () => {
    if (!view.value) return
    view.value.dispatch({
      effects: envKeysCompartment.reconfigure(
        declaredEnvKeysFacet.of(normalizeEnvKeysToSet(props.declaredEnvKeys))
      )
    })
  },
  { deep: true }
)

watch(
  () => props.modelValue,
  (v) => {
    if (!view.value) return
    const next = v ?? ''
    const cur = view.value.state.doc.toString()
    if (next === cur) return
    syncingFromParent = true
    view.value.dispatch({
      changes: { from: 0, to: view.value.state.doc.length, insert: next }
    })
    syncingFromParent = false
  }
)

onBeforeUnmount(() => {
  view.value?.destroy()
  view.value = null
})

/** Latest document text (for Send/Save — avoids v-model lag vs CodeMirror). */
function getDocText() {
  return view.value?.state.doc.toString() ?? ''
}

defineExpose({ getDocText })
</script>

<template>
  <div class="json-cm-root relative flex min-h-0 min-w-0 flex-1 flex-col">
    <div
      ref="host"
      class="json-cm-host min-h-0 min-w-0 flex-1 overflow-hidden rounded border border-gray-700"
    />
    <EnvVarPopover
      :open="chipPopoverOpen"
      :var-key="chipPopoverKey"
      v-model="chipPopoverDraft"
      :position="chipPopoverPos"
      :read-only="readOnly"
      :chip-label="chipPopoverChipLabel"
      @enter="onChipPopoverEnter"
      @leave="onChipPopoverLeave"
      @keydown="onChipPopoverKeydown"
      @close="closeChipEnvPopover"
      @apply="applyChipEnvPatch"
    />
  </div>
</template>

<style scoped>
.json-cm-host {
  min-height: 120px;
}
.json-cm-host :deep(.cm-editor) {
  height: 100%;
  min-height: 120px;
}
.json-cm-host :deep(.cm-scroller) {
  min-height: 120px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0) transparent;
}
.json-cm-host:hover :deep(.cm-scroller) {
  scrollbar-color: rgba(255, 255, 255, 0.2) transparent;
}

.json-cm-host :deep(.cm-scroller)::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
.json-cm-host :deep(.cm-scroller)::-webkit-scrollbar-track {
  background: transparent;
}
.json-cm-host :deep(.cm-scroller)::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 4px;
}
.json-cm-host:hover :deep(.cm-scroller)::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.12);
}
.json-cm-host:hover :deep(.cm-scroller)::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.22);
}

.json-cm-host :deep(.cm-envVarChip) {
  animation: cmEnvVarPop 0.22s ease-out;
}

@keyframes cmEnvVarPop {
  from {
    opacity: 0.65;
    filter: brightness(0.92);
  }
  to {
    opacity: 1;
    filter: brightness(1);
  }
}
</style>
