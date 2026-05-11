<template>
  <teleport to="body">
    <div class="global-toast-container" role="status" aria-live="polite" aria-atomic="false">
      <transition-group name="gtoast" tag="div">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="gtoast"
          :class="`gtoast--${t.type}`"
          role="alert"
        >
          <span class="gtoast__icon">{{ icons[t.type] || 'ℹ️' }}</span>
          <span class="gtoast__msg">{{ t.message }}</span>
          <button
            class="gtoast__close"
            aria-label="关闭通知"
            @click="toast.removeToast(t.id)"
          >&times;</button>
        </div>
      </transition-group>
    </div>
  </teleport>
</template>

<script setup>
import { computed } from 'vue'
import toast from '@/utils/toast'

const toasts = computed(() => toast.state.toasts)
const icons = { success: '✅', error: '❌', warning: '⚠️', info: 'ℹ️' }
</script>

<style scoped>
.global-toast-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 99999;
  pointer-events: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 420px;
}

.gtoast {
  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: 10px;
  font-size: 14px;
  line-height: 1.5;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  color: #f0f0f0;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.gtoast--success { background: rgba(34, 120, 60, 0.92); }
.gtoast--error   { background: rgba(180, 40, 40, 0.92); }
.gtoast--warning { background: rgba(180, 130, 20, 0.92); color: #fff; }
.gtoast--info    { background: rgba(30, 80, 160, 0.92); }

.gtoast__icon { flex-shrink: 0; font-size: 16px; }
.gtoast__msg  { flex: 1; word-break: break-word; }

.gtoast__close {
  flex-shrink: 0;
  background: none;
  border: none;
  color: inherit;
  opacity: 0.7;
  font-size: 18px;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
}
.gtoast__close:hover { opacity: 1; }

/* transitions */
.gtoast-enter-active, .gtoast-leave-active {
  transition: all 0.3s ease;
}
.gtoast-enter-from {
  opacity: 0;
  transform: translateX(80px);
}
.gtoast-leave-to {
  opacity: 0;
  transform: translateX(80px);
}

@media (max-width: 480px) {
  .global-toast-container {
    left: 10px;
    right: 10px;
    max-width: none;
  }
}
</style>
