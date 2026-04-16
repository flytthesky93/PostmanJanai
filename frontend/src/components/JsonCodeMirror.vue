<script setup>
import { ref, shallowRef, watch, onMounted, onBeforeUnmount } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { json } from '@codemirror/lang-json'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorState } from '@codemirror/state'

const props = defineProps({
  modelValue: { type: String, default: '' },
  /** true: read-only (response); false: editable (request body). */
  readOnly: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const host = ref(null)
const view = shallowRef(null)
let syncingFromParent = false

function buildExtensions() {
  return [
    basicSetup,
    json(),
    oneDark,
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
      '.cm-gutters': { borderRight: '1px solid #2d2d2d' }
    })
  ]
}

onMounted(() => {
  if (!host.value) return
  const state = EditorState.create({
    doc: props.modelValue ?? '',
    extensions: buildExtensions()
  })
  view.value = new EditorView({ state, parent: host.value })
})

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
}
</style>
