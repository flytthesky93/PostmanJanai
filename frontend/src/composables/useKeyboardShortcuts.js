import { onMounted, onUnmounted } from 'vue'

function isEditableTarget(target) {
  const el = target
  const tag = String(el?.tagName || '').toLowerCase()
  return tag === 'input' || tag === 'textarea' || tag === 'select' || !!el?.isContentEditable
}

export function useKeyboardShortcuts(actions = {}) {
  function onKeydown(e) {
    const key = String(e.key || '').toLowerCase()
    const mod = e.ctrlKey || e.metaKey
    if (!mod && key !== 'escape') return

    if (key === 'escape') {
      actions.escape?.(e)
      return
    }
    if (!mod) return

    if (key === 'enter') {
      e.preventDefault()
      actions.send?.(e)
      return
    }
    if (key === 's') {
      e.preventDefault()
      actions.save?.(e)
      return
    }
    if (key === 'k') {
      e.preventDefault()
      actions.palette?.(e)
      return
    }
    if (key === 't') {
      e.preventDefault()
      actions.newTab?.(e)
      return
    }
    if (key === 'w') {
      e.preventDefault()
      actions.closeTab?.(e)
      return
    }

    if (isEditableTarget(e.target)) return
    if (key === 'e' && e.shiftKey) {
      e.preventDefault()
      actions.toggleEnvironment?.(e)
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', onKeydown, true)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', onKeydown, true)
  })

  return { onKeydown }
}
