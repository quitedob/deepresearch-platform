<template>
  <div class="error-boundary">
    <slot v-if="!hasError" />

    <div v-else class="error-container" :class="errorType">
      <div class="error-content">
        <div class="error-icon">
          <span :class="iconClass">{{ getIcon() }}</span>
        </div>

        <div class="error-details">
          <h3 class="error-title">{{ errorTitle }}</h3>
          <p class="error-message">{{ errorMessage }}</p>

          <div v-if="showDetails && errorDetails" class="error-technical">
            <details>
              <summary>技术详情</summary>
              <pre>{{ errorDetails }}</pre>
            </details>
          </div>

          <div v-if="suggestions.length > 0" class="error-suggestions">
            <h4>建议解决方案：</h4>
            <ul>
              <li v-for="(suggestion, index) in suggestions" :key="index">
                {{ suggestion }}
              </li>
            </ul>
          </div>
        </div>

        <div class="error-actions">
          <button @click="retry" v-if="canRetry" class="btn btn-primary">
            🔄 重试
          </button>
          <button @click="refresh" v-if="canRefresh" class="btn btn-outline">
            🔃 刷新页面
          </button>
          <button @click="goHome" class="btn btn-outline">
            🏠 返回首页
          </button>
          <button @click="reportError" class="btn btn-outline">
            📧 报告问题
          </button>
        </div>
      </div>

      <!-- 错误报告模态框 -->
      <div v-if="showReportModal" class="modal-overlay" @click="closeReportModal">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>报告错误</h3>
            <button @click="closeReportModal" class="btn-close">×</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>错误描述（可选）</label>
              <textarea
                v-model="reportDescription"
                placeholder="请描述您遇到的问题..."
                class="form-textarea"
                rows="4"
              ></textarea>
            </div>

            <div class="form-group">
              <label>联系邮箱（可选）</label>
              <input
                v-model="reportEmail"
                type="email"
                placeholder="your@email.com"
                class="form-input"
              />
            </div>

            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="includeSystemInfo" />
                包含系统信息（推荐）
              </label>
            </div>
          </div>
          <div class="modal-footer">
            <button @click="closeReportModal" class="btn btn-outline">
              取消
            </button>
            <button @click="submitReport" class="btn btn-primary" :disabled="submitting">
              {{ submitting ? '提交中...' : '提交报告' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onErrorCaptured, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import toast from '@/utils/toast'

const router = useRouter()

// Props
const props = defineProps({
  fallbackComponent: {
    type: [String, Object],
    default: null
  },
  onError: {
    type: Function,
    default: null
  },
  retryCount: {
    type: Number,
    default: 3
  },
  showDetails: {
    type: Boolean,
    default: false
  }
})

// 错误状态
const hasError = ref(false)
const error = ref(null)
const errorInfo = ref(null)
const retryCount = ref(0)

// 报告模态框
const showReportModal = ref(false)
const reportDescription = ref('')
const reportEmail = ref('')
const includeSystemInfo = ref(true)
const submitting = ref(false)

// 计算属性
const errorType = computed(() => {
  if (!error.value) return 'default'

  if (error.value.name === 'ChunkLoadError' || error.value.message?.includes('Loading chunk')) {
    return 'network'
  }
  if (error.value.name === 'TypeError' || error.value.message?.includes('Cannot read')) {
    return 'javascript'
  }
  if (error.value.status >= 500) {
    return 'server'
  }
  if (error.value.status >= 400) {
    return 'client'
  }

  return 'default'
})

const errorTitle = computed(() => {
  const titles = {
    network: '网络连接错误',
    javascript: 'JavaScript错误',
    server: '服务器错误',
    client: '请求错误',
    default: '发生错误'
  }
  return titles[errorType.value] || titles.default
})

const errorMessage = computed(() => {
  if (!error.value) return '未知错误'

  const userFriendlyMessages = {
    network: '无法连接到服务器，请检查您的网络连接后重试。',
    javascript: '页面运行时出现错误，请刷新页面重试。',
    server: '服务器暂时无法响应，请稍后重试。',
    client: '请求参数有误，请检查后重试。'
  }

  return userFriendlyMessages[errorType.value] || error.value.message || '未知错误'
})

const errorDetails = computed(() => {
  if (!error.value) return null

  return `错误类型: ${error.value.name || 'Unknown'}\n` +
         `错误信息: ${error.value.message || 'No message'}\n` +
         `错误堆栈: ${error.value.stack || 'No stack trace'}\n` +
         `组件信息: ${errorInfo.value || 'No component info'}`
})

const suggestions = computed(() => {
  const suggestionsMap = {
    network: [
      '检查网络连接是否正常',
      '尝试刷新页面',
      '检查防火墙设置',
      '稍后重试'
    ],
    javascript: [
      '刷新页面',
      '清除浏览器缓存',
      '检查浏览器控制台',
      '尝试使用其他浏览器'
    ],
    server: [
      '稍后重试',
      '检查服务器状态',
      '联系技术支持',
      '查看系统公告'
    ],
    client: [
      '检查输入参数',
      '重新操作',
      '查看使用说明',
      '联系客服'
    ],
    default: [
      '刷新页面',
      '稍后重试',
      '检查网络连接',
      '联系技术支持'
    ]
  }

  return suggestionsMap[errorType.value] || suggestionsMap.default
})

const iconClass = computed(() => {
  return `icon-${errorType.value}`
})

const canRetry = computed(() => {
  return retryCount.value < props.retryCount
})

const canRefresh = computed(() => {
  return true
})

// 方法
const getIcon = () => {
  const icons = {
    network: '🌐',
    javascript: '⚠️',
    server: '🔧',
    client: '❌',
    default: '⚠️'
  }
  return icons[errorType.value] || icons.default
}

const handleError = (err, instance, info) => {
  console.error('ErrorBoundary caught an error:', err)
  console.error('Component instance:', instance)
  console.error('Error info:', info)

  hasError.value = true
  error.value = err
  errorInfo.value = info

  // 调用自定义错误处理函数
  if (props.onError) {
    props.onError(err, instance, info)
  }

  // 记录错误到监控系统
  logError(err, info)
}

const retry = () => {
  if (!canRetry.value) return

  retryCount.value++
  hasError.value = false
  error.value = null
  errorInfo.value = null

  // 重新渲染组件
  console.log(`Retrying... (${retryCount.value}/${props.retryCount})`)
}

const refresh = () => {
  window.location.reload()
}

const goHome = () => {
  router.push('/')
}

const reportError = () => {
  showReportModal.value = true
}

const closeReportModal = () => {
  showReportModal.value = false
  reportDescription.value = ''
  reportEmail.value = ''
  includeSystemInfo.value = true
}

const submitReport = async () => {
  submitting.value = true

  try {
    const reportData = {
      error: {
        name: error.value?.name,
        message: error.value?.message,
        stack: error.value?.stack
      },
      componentInfo: errorInfo.value,
      userAgent: navigator.userAgent,
      url: window.location.href,
      timestamp: new Date().toISOString(),
      description: reportDescription.value,
      email: reportEmail.value,
      systemInfo: includeSystemInfo.value ? getSystemInfo() : null
    }

    // 模拟提交错误报告
    await new Promise(resolve => setTimeout(resolve, 1500))

    console.log('Error report submitted:', reportData)
    toast.success('错误报告已提交，我们会尽快处理！')
    closeReportModal()
  } catch (err) {
    console.error('Failed to submit error report:', err)
    toast.error('提交失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}

const getSystemInfo = () => {
  return {
    browser: navigator.userAgent,
    platform: navigator.platform,
    language: navigator.language,
    cookieEnabled: navigator.cookieEnabled,
    onLine: navigator.onLine,
    screenResolution: `${screen.width}x${screen.height}`,
    viewportSize: `${window.innerWidth}x${window.innerHeight}`,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    timestamp: new Date().toISOString()
  }
}

const logError = (err, info) => {
  // 这里可以集成错误监控服务，如 Sentry
  console.group('🚨 Error Boundary Log')
  console.error('Error:', err)
  console.error('Component Info:', info)
  console.error('URL:', window.location.href)
  console.error('User Agent:', navigator.userAgent)
  console.groupEnd()
}

// 捕获错误
onErrorCaptured(handleError)

// 全局错误处理
onMounted(() => {
  // 捕获未处理的Promise拒绝
  window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason)
    handleError(event.reason, null, 'Unhandled Promise Rejection')
  })

  // 捕获全局错误
  window.addEventListener('error', (event) => {
    console.error('Global error:', event.error)
    handleError(event.error, null, 'Global Error')
  })
})
</script>

<style scoped>
.error-boundary {
  width: 100%;
  height: 100%;
}

.error-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  padding: 2rem;
}

.error-content {
  max-width: 600px;
  text-align: center;
}

.error-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.icon-network {
  color: #ff9800;
}

.icon-javascript {
  color: #f44336;
}

.icon-server {
  color: #9c27b0;
}

.icon-client {
  color: #2196f3;
}

.icon-default {
  color: #607d8b;
}

.error-details {
  margin-bottom: 2rem;
}

.error-title {
  margin: 0 0 1rem 0;
  color: #2c3e50;
  font-size: 1.5rem;
}

.error-message {
  margin: 0 0 1.5rem 0;
  color: #5a6c7d;
  line-height: 1.6;
}

.error-technical {
  margin-bottom: 1.5rem;
  text-align: left;
}

.error-technical details {
  background: #f8f9fa;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  padding: 1rem;
  text-align: left;
}

.error-technical summary {
  cursor: pointer;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 0.5rem;
}

.error-technical pre {
  margin: 0;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 0.8rem;
  color: #dc3545;
  white-space: pre-wrap;
  word-break: break-all;
  background: #fff;
  padding: 0.5rem;
  border-radius: 4px;
  overflow-x: auto;
}

.error-suggestions {
  text-align: left;
  margin-bottom: 2rem;
}

.error-suggestions h4 {
  margin: 0 0 0.75rem 0;
  color: #2c3e50;
  font-size: 1rem;
}

.error-suggestions ul {
  margin: 0;
  padding-left: 1.5rem;
}

.error-suggestions li {
  margin-bottom: 0.5rem;
  color: #5a6c7d;
  line-height: 1.5;
}

.error-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

/* 按钮样式 */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  border: none;
  font-size: 0.9rem;
  transition: all 0.3s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #667eea;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #5a6fd8;
}

.btn-outline {
  background: transparent;
  color: #667eea;
  border: 1px solid #667eea;
}

.btn-outline:hover:not(:disabled) {
  background: #667eea;
  color: white;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e1e8ed;
}

.modal-header h3 {
  margin: 0;
  color: #2c3e50;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #5a6c7d;
  padding: 0.25rem;
  border-radius: 4px;
}

.btn-close:hover {
  background: #f1f3f4;
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #e1e8ed;
}

/* 表单样式 */
.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: #2c3e50;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  font-size: 0.9rem;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.2);
}

.form-textarea {
  resize: vertical;
  font-family: inherit;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.9rem;
  color: #2c3e50;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .error-container {
    padding: 1rem;
    min-height: 300px;
  }

  .error-icon {
    font-size: 3rem;
  }

  .error-title {
    font-size: 1.2rem;
  }

  .error-actions {
    flex-direction: column;
    align-items: center;
  }

  .modal-content {
    width: 95%;
    margin: 1rem;
  }
}

@media (max-width: 480px) {
  .error-content {
    padding: 0.5rem;
  }

  .error-icon {
    font-size: 2.5rem;
  }

  .error-actions {
    width: 100%;
  }

  .btn {
    flex: 1;
    min-width: 120px;
  }
}
</style>