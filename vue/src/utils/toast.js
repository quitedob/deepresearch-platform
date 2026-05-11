/**
 * 全局 Toast 通知工具
 * 基于事件总线模式，任何组件/模块都可以直接调用，无需注入
 */
import { reactive } from 'vue'

const state = reactive({
  toasts: [],
  _nextId: 1
})

const DEFAULT_DURATION = 4000
const ERROR_DURATION = 8000
const MAX_TOASTS = 5

function addToast(options) {
  const id = state._nextId++
  const toast = {
    id,
    type: 'info',
    message: '',
    duration: DEFAULT_DURATION,
    ...options
  }

  if (state.toasts.length >= MAX_TOASTS) {
    state.toasts.shift()
  }

  state.toasts.push(toast)

  if (toast.duration > 0) {
    setTimeout(() => removeToast(id), toast.duration)
  }

  return id
}

function removeToast(id) {
  const idx = state.toasts.findIndex(t => t.id === id)
  if (idx > -1) state.toasts.splice(idx, 1)
}

function clearAll() {
  state.toasts.splice(0)
}

const toast = {
  state,
  removeToast,
  clearAll,
  success(message) {
    return addToast({ type: 'success', message, duration: DEFAULT_DURATION })
  },
  error(message) {
    return addToast({ type: 'error', message, duration: ERROR_DURATION })
  },
  warning(message) {
    return addToast({ type: 'warning', message, duration: DEFAULT_DURATION + 2000 })
  },
  info(message) {
    return addToast({ type: 'info', message, duration: DEFAULT_DURATION })
  }
}

export default toast
