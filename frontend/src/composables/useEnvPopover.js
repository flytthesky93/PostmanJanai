import { ref, computed, watch, onUnmounted } from 'vue'

const HIDE_MS = 380

/**
 * Shared hover popover for env {{var}} chips (mirror field, CodeMirror, …).
 * @param {{
 *   envValues: import('vue').MaybeRefOrGetter<Record<string, string>>,
 *   isReadOnly: import('vue').MaybeRefOrGetter<boolean>,
 *   onPatch: (payload: { key: string, value: string }) => void
 * }} options
 */
export function useEnvPopover(options) {
  const { envValues, isReadOnly, onPatch } = options

  const popoverOpen = ref(false)
  const popoverKey = ref('')
  const popoverDraft = ref('')
  const popoverPos = ref({ left: 0, top: 0 })
  let hideTimer = /** @type {ReturnType<typeof setTimeout> | null} */ (null)
  let lastHoverKey = ''
  let moveRaf = /** @type {number | null} */ (null)

  function readEnvMap() {
    const v = typeof envValues === 'function' ? envValues() : envValues
    return v && typeof v === 'object' ? v : {}
  }

  function readOnly() {
    const v = typeof isReadOnly === 'function' ? isReadOnly() : isReadOnly
    return !!v
  }

  function clearHide() {
    if (hideTimer) {
      clearTimeout(hideTimer)
      hideTimer = null
    }
  }

  function close() {
    clearHide()
    popoverOpen.value = false
    popoverKey.value = ''
    lastHoverKey = ''
  }

  function scheduleHide() {
    clearHide()
    hideTimer = setTimeout(() => {
      hideTimer = null
      close()
    }, HIDE_MS)
  }

  function placeNear(clientX, clientY) {
    const estW = 280
    const estH = 132
    const pad = 8
    let left = clientX + 12
    let top = clientY + 14
    if (left + estW > window.innerWidth - pad) left = window.innerWidth - estW - pad
    if (top + estH > window.innerHeight - pad) top = window.innerHeight - estH - pad
    if (left < pad) left = pad
    if (top < pad) top = pad
    popoverPos.value = { left, top }
  }

  /** @param {string} key trimmed name */
  function openForKey(key, clientX, clientY) {
    const k = String(key || '').trim()
    if (!k) {
      scheduleHide()
      return
    }
    clearHide()
    if (k !== lastHoverKey) {
      lastHoverKey = k
      popoverKey.value = k
      const m = readEnvMap()
      const raw = Object.prototype.hasOwnProperty.call(m, k) ? m[k] : ''
      popoverDraft.value = raw != null ? String(raw) : ''
      placeNear(clientX, clientY)
    }
    popoverOpen.value = true
  }

  /** For CodeMirror facet: `{ onEnter, onLeave }` */
  function chipHandlers() {
    return {
      onEnter: openForKey,
      onLeave: scheduleHide
    }
  }

  function onPopoverEnter() {
    clearHide()
  }

  function onPopoverLeave() {
    scheduleHide()
  }

  function applyPatch() {
    if (readOnly()) return
    const k = popoverKey.value.trim()
    if (!k) return
    onPatch?.({ key: k, value: popoverDraft.value })
    close()
  }

  function onPopoverKeydown(e) {
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
    }
  }

  const chipLabel = computed(() => {
    const k = popoverKey.value
    return k ? `{{${k}}}` : ''
  })

  function onDocKeydown(e) {
    if (e.key === 'Escape' && popoverOpen.value) close()
  }

  watch(popoverOpen, (open) => {
    if (open) document.addEventListener('keydown', onDocKeydown, true)
    else document.removeEventListener('keydown', onDocKeydown, true)
  })

  watch(
    () => readEnvMap(),
    () => {
      if (!popoverOpen.value || !popoverKey.value) return
      const k = popoverKey.value
      const m = readEnvMap()
      const raw = Object.prototype.hasOwnProperty.call(m, k) ? m[k] : ''
      popoverDraft.value = raw != null ? String(raw) : ''
    },
    { deep: true }
  )

  /** Throttle pointer move (mirror field). */
  function schedulePointerUpdate(fn) {
    if (moveRaf != null) return
    moveRaf = requestAnimationFrame(() => {
      moveRaf = null
      fn()
    })
  }

  onUnmounted(() => {
    document.removeEventListener('keydown', onDocKeydown, true)
    clearHide()
    if (moveRaf != null) cancelAnimationFrame(moveRaf)
  })

  return {
    popoverOpen,
    popoverKey,
    popoverDraft,
    popoverPos,
    chipLabel,
    chipHandlers,
    openForKey,
    close,
    scheduleHide,
    clearHide,
    onPopoverEnter,
    onPopoverLeave,
    applyPatch,
    onPopoverKeydown,
    schedulePointerUpdate
  }
}
