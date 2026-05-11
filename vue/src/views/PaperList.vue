<template>
  <div class="paper-list-page">
    <div class="page-header">
      <div class="header-left">
        <button class="back-btn" @click="$router.push('/home')">← 返回</button>
        <h1 class="page-title">我的论文</h1>
      </div>
      <button class="new-btn" @click="$router.push('/paper/generate')">+ 新建论文</button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="papers.length === 0" class="empty-state">
      <div class="empty-icon">📄</div>
      <p class="empty-title">还没有论文</p>
      <p class="empty-desc">点击「新建论文」开始 AI 辅助写作</p>
      <button class="new-btn" @click="$router.push('/paper/generate')">开始创作</button>
    </div>

    <!-- 论文列表 -->
    <div v-else class="paper-grid">
      <div
        v-for="paper in papers"
        :key="paper.id"
        class="paper-card"
        @click="openPaper(paper)"
      >
        <div class="card-header">
          <span class="paper-type-badge" :class="paper.paper_type">
            {{ paper.paper_type === 'liberal_arts' ? '文科' : '理科' }}
          </span>
          <span class="status-badge" :class="paper.status">{{ statusLabel(paper.status) }}</span>
        </div>
        <h3 class="paper-title">{{ paper.title }}</h3>
        <div class="paper-meta">
          <span>{{ paper.current_words.toLocaleString() }} / {{ paper.target_words.toLocaleString() }} 字</span>
          <span>{{ formatDate(paper.updated_at) }}</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: (paper.progress * 100) + '%' }"></div>
        </div>
        <div class="card-actions" @click.stop>
          <button class="action-btn" @click="openPaper(paper)">查看</button>
          <button class="action-btn danger" @click="confirmDelete(paper)">删除</button>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="total > limit" class="pagination">
      <button :disabled="offset === 0" @click="prevPage">上一页</button>
      <span>{{ Math.floor(offset / limit) + 1 }} / {{ Math.ceil(total / limit) }}</span>
      <button :disabled="offset + limit >= total" @click="nextPage">下一页</button>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
      <div class="modal">
        <h3>确认删除</h3>
        <p>删除「{{ deleteTarget.title }}」？此操作不可撤销。</p>
        <div class="modal-actions">
          <button @click="deleteTarget = null">取消</button>
          <button class="danger" @click="doDelete">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { paperAPI } from '@/api/paper'
import toast from '@/utils/toast'

const router = useRouter()
const papers = ref([])
const loading = ref(true)
const total = ref(0)
const limit = ref(20)
const offset = ref(0)
const deleteTarget = ref(null)

const statusLabel = (s) => ({
  drafting:    '生成中',
  planning:    '规划中',
  searching:   '搜索中',
  generating:  '写作中',
  reviewing:   '审查中',
  revising:    '修订中',
  synthesis:   '合并中',
  completed:   '已完成',
  failed:      '失败'
}[s] || s)

// 中间状态都视为"生成中"，跳转进度页
const IN_PROGRESS = new Set(['drafting','planning','searching','generating','reviewing','revising','synthesis'])

const formatDate = (d) => new Date(d).toLocaleDateString('zh-CN')

const loadPapers = async () => {
  loading.value = true
  try {
    const data = await paperAPI.list({ limit: limit.value, offset: offset.value })
    papers.value = data.items || []
    total.value = data.total || 0
  } catch (e) {
    toast.error('加载论文列表失败')
  } finally {
    loading.value = false
  }
}

const openPaper = (paper) => {
  if (paper.status === 'completed') {
    router.push(`/paper/${paper.id}`)
  } else if (IN_PROGRESS.has(paper.status)) {
    router.push(`/paper/${paper.id}/progress`)
  } else {
    router.push(`/paper/${paper.id}`)
  }
}

const confirmDelete = (paper) => { deleteTarget.value = paper }

const doDelete = async () => {
  try {
    await paperAPI.delete(deleteTarget.value.id)
    toast.success('删除成功')
    deleteTarget.value = null
    loadPapers()
  } catch (e) {
    toast.error('删除失败')
  }
}

const prevPage = () => { offset.value = Math.max(0, offset.value - limit.value); loadPapers() }
const nextPage = () => { offset.value += limit.value; loadPapers() }

onMounted(loadPapers)
</script>

<style scoped>
.paper-list-page {
  max-width: 1100px;
  margin: 0 auto;
  padding: 32px 24px;
  min-height: 100vh;
  background: var(--primary-bg, #0a0a0a);
  color: var(--text-primary, #f0f0f0);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.header-left { display: flex; align-items: center; gap: 16px; }

.back-btn {
  background: none;
  border: 1px solid rgba(255,255,255,0.15);
  color: var(--text-secondary, #aaa);
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}
.back-btn:hover { border-color: rgba(255,255,255,0.3); color: #fff; }

.page-title { font-size: 24px; font-weight: 700; margin: 0; }

.new-btn {
  background: linear-gradient(135deg, #3b82f6, #7c3aed);
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: opacity 0.2s;
}
.new-btn:hover { opacity: 0.85; }

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 0;
  color: var(--text-secondary, #aaa);
}

.spinner {
  width: 24px; height: 24px;
  border: 2px solid rgba(255,255,255,0.1);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.empty-state {
  text-align: center;
  padding: 80px 0;
}
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-title { font-size: 20px; font-weight: 600; margin: 0 0 8px; }
.empty-desc { color: var(--text-secondary, #aaa); margin: 0 0 24px; }

.paper-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.paper-card {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 14px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
}
.paper-card:hover {
  border-color: rgba(59,130,246,0.4);
  background: rgba(59,130,246,0.05);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
}

.paper-type-badge, .status-badge {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 20px;
  font-weight: 600;
}
.paper-type-badge { background: rgba(59,130,246,0.15); color: #60a5fa; }
.status-badge.completed { background: rgba(34,197,94,0.15); color: #4ade80; }
.status-badge.drafting,
.status-badge.planning,
.status-badge.searching,
.status-badge.generating,
.status-badge.reviewing,
.status-badge.revising,
.status-badge.synthesis { background: rgba(251,191,36,0.15); color: #fbbf24; }
.status-badge.failed { background: rgba(239,68,68,0.15); color: #f87171; }

.paper-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 12px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.paper-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--text-secondary, #aaa);
  margin-bottom: 10px;
}

.progress-bar {
  height: 3px;
  background: rgba(255,255,255,0.08);
  border-radius: 2px;
  margin-bottom: 16px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #7c3aed);
  border-radius: 2px;
  transition: width 0.3s;
}

.card-actions {
  display: flex;
  gap: 8px;
}
.action-btn {
  flex: 1;
  padding: 7px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-secondary, #aaa);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.action-btn:hover { border-color: rgba(255,255,255,0.25); color: #fff; }
.action-btn.danger:hover { border-color: rgba(239,68,68,0.5); color: #f87171; }

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 32px;
}
.pagination button {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-primary, #f0f0f0);
  cursor: pointer;
}
.pagination button:disabled { opacity: 0.3; cursor: not-allowed; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal {
  background: #1a1a2e;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 16px;
  padding: 28px;
  width: 360px;
}
.modal h3 { margin: 0 0 12px; font-size: 18px; }
.modal p { color: var(--text-secondary, #aaa); margin: 0 0 24px; }
.modal-actions { display: flex; gap: 12px; justify-content: flex-end; }
.modal-actions button {
  padding: 9px 20px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-primary, #f0f0f0);
  cursor: pointer;
  font-size: 14px;
}
.modal-actions button.danger {
  background: rgba(239,68,68,0.15);
  border-color: rgba(239,68,68,0.3);
  color: #f87171;
}
</style>
