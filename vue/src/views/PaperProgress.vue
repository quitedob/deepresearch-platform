<template>
  <div class="paper-progress-page">
    <div class="page-header">
      <h1 class="page-title">论文生成中</h1>
      <span class="paper-title-label">{{ paperTitle }}</span>
    </div>

    <!-- 进度概览 -->
    <div class="progress-overview">
      <div class="progress-ring-wrap">
        <svg class="progress-ring" viewBox="0 0 120 120">
          <circle cx="60" cy="60" r="50" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="8"/>
          <circle
            cx="60" cy="60" r="50"
            fill="none"
            stroke="url(#grad)"
            stroke-width="8"
            stroke-linecap="round"
            :stroke-dasharray="circumference"
            :stroke-dashoffset="dashOffset"
            transform="rotate(-90 60 60)"
            style="transition: stroke-dashoffset 0.5s ease"
          />
          <defs>
            <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#3b82f6"/>
              <stop offset="100%" stop-color="#7c3aed"/>
            </linearGradient>
          </defs>
        </svg>
        <div class="progress-center">
          <span class="progress-pct">{{ Math.round(progress * 100) }}%</span>
          <span class="progress-stage">{{ stageLabel }}</span>
        </div>
      </div>

      <div class="progress-stats">
        <div class="stat">
          <span class="stat-value">{{ currentWords.toLocaleString() }}</span>
          <span class="stat-label">当前字数</span>
        </div>
        <div class="stat">
          <span class="stat-value">{{ targetWords.toLocaleString() }}</span>
          <span class="stat-label">目标字数</span>
        </div>
        <div class="stat">
          <span class="stat-value">{{ elapsed }}</span>
          <span class="stat-label">已用时间</span>
        </div>
      </div>
    </div>

    <!-- 事件日志 -->
    <div class="event-log" ref="logRef">
      <div
        v-for="(evt, i) in events"
        :key="i"
        class="event-item"
        :class="evt.type"
      >
        <span class="event-time">{{ formatTime(evt.timestamp) }}</span>
        <span class="event-icon">{{ eventIcon(evt.type) }}</span>
        <span class="event-msg">{{ evt.message }}</span>
      </div>
    </div>

    <!-- 完成状态 -->
    <div v-if="status === 'completed'" class="done-banner">
      <span class="done-icon">🎉</span>
      <div>
        <p class="done-title">论文生成完成！</p>
        <p class="done-desc">共 {{ currentWords.toLocaleString() }} 字</p>
      </div>
      <button class="view-btn" @click="$router.push(`/paper/${paperId}`)">查看论文</button>
    </div>

    <!-- 失败状态 -->
    <div v-if="status === 'failed'" class="error-banner">
      <span>❌ 生成失败</span>
      <button @click="$router.push('/paper/generate')">重新生成</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { paperAPI, createPaperSSE } from '@/api/paper'
import toast from '@/utils/toast'

const route = useRoute()
const router = useRouter()
const paperId = route.params.id

const paperTitle = ref('')
const progress = ref(0)
const currentWords = ref(0)
const targetWords = ref(0)
const status = ref('drafting')
const events = ref([])
const logRef = ref(null)
const startTime = ref(Date.now())
const elapsed = ref('0:00')
const stageLabel = ref('准备中')

const circumference = 2 * Math.PI * 50
const dashOffset = computed(() => circumference * (1 - progress.value))

const stageMap = {
  planning: '规划大纲',
  searching: '搜索资料',
  generating: '生成章节',
  reviewing: '审查质量',
  revising: '修订内容',
  synthesis: '合并论文',
  completed: '已完成'
}

const eventIcon = (type) => ({
  status_update: '📝',
  chapter_started: '✏️',
  chapter_completed: '✅',
  chapter_regenerated: '🔄',
  review: '🔍',
  completed: '🎉',
  error: '❌',
  connected: '🔗'
}[type] || '•')

const formatTime = (ts) => {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

let closeSSE = null
let elapsedTimer = null

const scrollLog = () => {
  nextTick(() => {
    if (logRef.value) logRef.value.scrollTop = logRef.value.scrollHeight
  })
}

const handleEvent = (evt) => {
  if (evt.progress) progress.value = evt.progress
  if (evt.current_words) currentWords.value = evt.current_words
  if (evt.stage && stageMap[evt.stage]) stageLabel.value = stageMap[evt.stage]

  events.value.push({ ...evt, timestamp: evt.timestamp || new Date().toISOString() })
  scrollLog()

  if (evt.type === 'completed') {
    status.value = 'completed'
    progress.value = 1
  } else if (evt.type === 'error') {
    status.value = 'failed'
    toast.error(evt.message || '生成失败')
  }
}

onMounted(async () => {
  // 先获取当前状态
  try {
    const data = await paperAPI.getStatus(paperId)
    paperTitle.value = data.title
    progress.value = data.progress
    currentWords.value = data.current_words
    targetWords.value = data.target_words
    status.value = data.status

    if (data.status === 'completed') {
      router.replace(`/paper/${paperId}`)
      return
    }
  } catch (e) {
    toast.error('获取论文状态失败')
  }

  // 启动 SSE
  closeSSE = createPaperSSE(paperId, handleEvent, (err) => {
    toast.error('进度连接断开，请刷新页面')
  })

  // 计时器
  elapsedTimer = setInterval(() => {
    const s = Math.floor((Date.now() - startTime.value) / 1000)
    elapsed.value = `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
  }, 1000)
})

onUnmounted(() => {
  closeSSE?.()
  clearInterval(elapsedTimer)
})
</script>

<style scoped>
.paper-progress-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 32px 24px;
  min-height: 100vh;
  background: var(--primary-bg, #0a0a0a);
  color: var(--text-primary, #f0f0f0);
}

.page-header {
  margin-bottom: 32px;
}
.page-title { font-size: 24px; font-weight: 700; margin: 0 0 6px; }
.paper-title-label { font-size: 14px; color: var(--text-secondary, #aaa); }

.progress-overview {
  display: flex;
  align-items: center;
  gap: 40px;
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 16px;
  padding: 28px;
  margin-bottom: 24px;
}

.progress-ring-wrap {
  position: relative;
  width: 120px;
  height: 120px;
  flex-shrink: 0;
}
.progress-ring { width: 100%; height: 100%; }
.progress-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.progress-pct { font-size: 22px; font-weight: 700; }
.progress-stage { font-size: 11px; color: var(--text-secondary, #aaa); margin-top: 2px; }

.progress-stats {
  display: flex;
  gap: 32px;
}
.stat { display: flex; flex-direction: column; gap: 4px; }
.stat-value { font-size: 22px; font-weight: 700; }
.stat-label { font-size: 12px; color: var(--text-secondary, #aaa); }

.event-log {
  background: rgba(0,0,0,0.3);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 12px;
  padding: 16px;
  height: 300px;
  overflow-y: auto;
  font-size: 13px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.event-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  background: rgba(255,255,255,0.02);
}
.event-item.error { background: rgba(239,68,68,0.08); }
.event-item.completed { background: rgba(34,197,94,0.08); }
.event-time { color: rgba(255,255,255,0.3); font-size: 11px; white-space: nowrap; }
.event-icon { flex-shrink: 0; }
.event-msg { color: var(--text-secondary, #ccc); line-height: 1.4; }

.done-banner, .error-banner {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  border-radius: 14px;
  margin-top: 20px;
}
.done-banner {
  background: rgba(34,197,94,0.08);
  border: 1px solid rgba(34,197,94,0.2);
}
.error-banner {
  background: rgba(239,68,68,0.08);
  border: 1px solid rgba(239,68,68,0.2);
}
.done-icon { font-size: 32px; }
.done-title { font-size: 16px; font-weight: 600; margin: 0 0 4px; }
.done-desc { font-size: 13px; color: var(--text-secondary, #aaa); margin: 0; }
.view-btn {
  margin-left: auto;
  background: linear-gradient(135deg, #3b82f6, #7c3aed);
  color: #fff;
  border: none;
  padding: 10px 24px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

@media (max-width: 600px) {
  .progress-overview { flex-direction: column; align-items: flex-start; }
  .progress-stats { gap: 20px; }
}
</style>
