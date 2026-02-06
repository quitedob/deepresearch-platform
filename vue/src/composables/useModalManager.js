/**
 * Modal管理器 - 统一管理多个模态框的z-index和行为
 * 修复：多个模态框同时打开时z-index管理混乱，点击背景关闭的行为不一致
 */
import { ref, computed, onUnmounted } from 'vue'

// 全局状态
const modalStack = ref([])
const baseZIndex = 1000
const zIndexStep = 10

/**
 * 生成唯一ID
 */
const generateId = () => `modal_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

/**
 * Modal管理器组合式函数
 */
export function useModalManager() {
  const modalId = ref(null)
  const isOpen = ref(false)

  /**
   * 打开模态框
   * @param {Object} options - 配置选项
   * @param {boolean} options.closeOnBackdrop - 点击背景是否关闭，默认true
   * @param {boolean} options.closeOnEscape - 按ESC是否关闭，默认true
   * @param {Function} options.onClose - 关闭回调
   */
  const open = (options = {}) => {
    const id = generateId()
    modalId.value = id
    isOpen.value = true

    const modalInfo = {
      id,
      closeOnBackdrop: options.closeOnBackdrop !== false,
      closeOnEscape: options.closeOnEscape !== false,
      onClose: options.onClose || null,
      zIndex: baseZIndex + modalStack.value.length * zIndexStep
    }

    modalStack.value.push(modalInfo)

    // 添加ESC键监听
    if (modalInfo.closeOnEscape) {
      document.addEventListener('keydown', handleEscapeKey)
    }

    return modalInfo
  }

  /**
   * 关闭模态框
   */
  const close = () => {
    if (!modalId.value) return

    const index = modalStack.value.findIndex(m => m.id === modalId.value)
    if (index !== -1) {
      const modalInfo = modalStack.value[index]
      
      // 调用关闭回调
      if (modalInfo.onClose) {
        modalInfo.onClose()
      }

      modalStack.value.splice(index, 1)
    }

    modalId.value = null
    isOpen.value = false

    // 移除ESC键监听
    document.removeEventListener('keydown', handleEscapeKey)
  }

  /**
   * 处理ESC键
   */
  const handleEscapeKey = (event) => {
    if (event.key === 'Escape') {
      // 只关闭最顶层的模态框
      const topModal = modalStack.value[modalStack.value.length - 1]
      if (topModal && topModal.id === modalId.value && topModal.closeOnEscape) {
        close()
      }
    }
  }

  /**
   * 处理背景点击
   */
  const handleBackdropClick = (event) => {
    // 确保点击的是背景而不是内容
    if (event.target === event.currentTarget) {
      const modalInfo = modalStack.value.find(m => m.id === modalId.value)
      if (modalInfo && modalInfo.closeOnBackdrop) {
        close()
      }
    }
  }

  /**
   * 获取当前模态框的z-index
   */
  const zIndex = computed(() => {
    const modalInfo = modalStack.value.find(m => m.id === modalId.value)
    return modalInfo ? modalInfo.zIndex : baseZIndex
  })

  /**
   * 是否是最顶层模态框
   */
  const isTopModal = computed(() => {
    if (modalStack.value.length === 0) return false
    return modalStack.value[modalStack.value.length - 1].id === modalId.value
  })

  // 组件卸载时清理
  onUnmounted(() => {
    if (isOpen.value) {
      close()
    }
  })

  return {
    isOpen,
    open,
    close,
    zIndex,
    isTopModal,
    handleBackdropClick
  }
}

/**
 * 获取当前打开的模态框数量
 */
export function getOpenModalCount() {
  return modalStack.value.length
}

/**
 * 关闭所有模态框
 */
export function closeAllModals() {
  while (modalStack.value.length > 0) {
    const modal = modalStack.value.pop()
    if (modal.onClose) {
      modal.onClose()
    }
  }
}

export default useModalManager
