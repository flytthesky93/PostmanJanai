<script setup>
import { ref, shallowRef, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { json } from '@codemirror/lang-json'
import { xml } from '@codemirror/lang-xml'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorState } from '@codemirror/state'
import { placeholder } from '@codemirror/view'

const props = defineProps({
  modelValue: { type: String, default: '' },
  /** true: read-only (response); false: editable (request body). */
  readOnly: { type: Boolean, default: false },
  /** Syntax: JSON (default) or XML for request body. */
  language: { type: String, default: 'json' },
  /** Shown inside the editor when the document is empty (request body only). */
  placeholder: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue'])

const host = ref(null)
const view = shallowRef(null)
let syncingFromParent = false

function langExtension() {
  return props.language === 'xml' ? xml() : json()
}

function buildExtensions() {
  const ph = String(props.placeholder || '').trim()
  return [
    basicSetup,
    langExtension(),
    oneDark,
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
      '.cm-placeholder': { color: '#6b7280' }
    })
  ]
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
  () => [props.language, props.placeholder],
  async () => {
    view.value?.destroy()
    view.value = null
    await nextTick()
    mountEditor()
  }
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
</script>

<template>
  <div
    ref="host"
    class="json-cm-host min-h-0 flex-1 overflow-hidden rounded border border-gray-700"
  />
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

/* Match app-scrollbar: subtle thumb on hover (WebKit) */
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
</style>
