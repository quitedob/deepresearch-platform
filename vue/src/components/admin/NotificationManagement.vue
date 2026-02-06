<template>
  <div class="notification-management">
    <div class="section-header">
      <h2>通知管理</h2>
      <button class="btn-primary" @click="showCreateModal = true">+ 发送通知</button>
    </div>

    <div class="notifications-list">
      <div v-for="notification in notifications" :key="notification.id" class="notification-item">
        <div class="notification-header">
          <span :class="['type-badge', notification.type]">
            {{ getTypeLabel(notification.type) }}
          </span>
          <span class="notification-title">{{ notification.title }}</span>
          <span class="notification-date">{{ formatDate(notification.created_at) }}</span>
        </div>
        <div class="notification-content">{{ notification.content }}</div>
        <div class="notification-actions">
          <span v-if="notification.is_global" class="global-badge">全局通知</span>
          <button class="btn-icon" @click="deleteNotification(notification)" title="删除">🗑️</button>
        </div>
      </div>
      <div v-if="notifications.length === 0" class="empty-state">暂无通知</div>
    </div>

    <div class="pagination">
      <button :disabled="offset === 0" @click="prevPage">上一页</button>
      <span>第 {{ currentPage }} 页 / 共 {{ totalPages }} 页</span>
      <button :disabled="offset + limit >= total" @click="nextPage">下一页</button>
    </div>

    <!-- 创建通知弹窗 -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal">
        <h3>发送通知</h3>
        <div class="form-group">
          <label>通知类型</label>
          <select v-model="createForm.type">
            <option value="system">系统通知</option>
            <option value="announce">公告</option>
            <option value="alert">警告</option>
          </select>
        </div>
        <div class="form-group">
          <label>标题 *</label>
          <input type="text" v-model="createForm.title" placeholder="通知标题" />
        </div>
        <div class="form-group">
          <label>内容 *</label>
          <textarea v-model="createForm.content" rows="4" placeholder="通知内容"></textarea>
        </div>
        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" v-model="createForm.isGlobal" />
            发送给所有用户
          </label>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showCreateModal = false">取消</button>
          <button class="btn-primary" @click="createNotification">发送</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { listNotifications, createNotification as createNotificationAPI, deleteNotification as deleteNotificationAPI } from '@/api/admin'

const notifications = ref([])
const total = ref(0)
const limit = ref(20)
const offset = ref(0)
const showCreateModal = ref(false)

const createForm = ref({
  type: 'system',
  title: '',
  content: '',
  isGlobal: true
})

const currentPage = computed(() => Math.floor(offset.value / limit.value) + 1)
const totalPages = computed(() => Math.ceil(total.value / limit.value))

const loadNotifications = async () => {
  try {
    const response = await listNotifications(limit.value, offset.value)
    const data = response.data || response
    notifications.value = data.notifications || []
    total.value = data.total || 0
  } catch (error) {
    console.error('加载通知列表失败:', error)
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

const getTypeLabel = (type) => {
  const labels = { system: '系统', announce: '公告', alert: '警告' }
  return labels[type] || type
}

const prevPage = () => {
  if (offset.value > 0) {
    offset.value -= limit.value
    loadNotifications()
  }
}

const nextPage = () => {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    loadNotifications()
  }
}

const createNotification = async () => {
  if (!createForm.value.title || !createForm.value.content) {
    alert('请填写标题和内容')
    return
  }
  try {
    await createNotificationAPI({
      type: createForm.value.type,
      title: createForm.value.title,
      content: createForm.value.content,
      is_global: createForm.value.isGlobal
    })
    showCreateModal.value = false
    createForm.value = { type: 'system', title: '', content: '', isGlobal: true }
    loadNotifications()
    alert('通知已发送')
  } catch (error) {
    console.error('发送通知失败:', error)
    alert('发送失败')
  }
}

const deleteNotification = async (notification) => {
  if (!confirm('确定要删除这条通知吗？')) return
  try {
    await deleteNotificationAPI(notification.id)
    loadNotifications()
  } catch (error) {
    console.error('删除通知失败:', error)
  }
}

onMounted(() => {
  loadNotifications()
})
</script>

<style scoped>
.notification-management { width: 100%; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.section-header h2 { font-size: 18px; font-weight: 600; }
.notifications-list { display: flex; flex-direction: column; gap: 12px; }
.notification-item { border: 1px solid var(--border-color); border-radius: 8px; padding: 16px; background: var(--card-bg); }
.notification-header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
.type-badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
.type-badge.system { background: #dbeafe; color: #1e40af; }
.type-badge.announce { background: #fef3c7; color: #92400e; }
.type-badge.alert { background: #fee2e2; color: #991b1b; }
.notification-title { font-weight: 600; flex: 1; }
.notification-date { color: var(--text-tertiary); font-size: 13px; }
.notification-content { color: var(--text-primary); font-size: 14px; line-height: 1.5; }
.notification-actions { display: flex; justify-content: space-between; align-items: center; margin-top: 12px; }
.global-badge { font-size: 12px; color: var(--text-secondary); background: var(--secondary-bg); padding: 2px 8px; border-radius: 4px; }
.btn-icon { background: none; border: none; cursor: pointer; font-size: 16px; padding: 4px; }
.empty-state { text-align: center; color: var(--text-tertiary); padding: 40px; }
.pagination { display: flex; justify-content: center; align-items: center; gap: 16px; margin-top: 20px; }
.pagination button { padding: 8px 16px; border: 1px solid var(--border-color); background: var(--card-bg); border-radius: 6px; cursor: pointer; color: var(--text-primary); }
.pagination button:disabled { opacity: 0.5; cursor: not-allowed; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--card-bg); border-radius: 12px; padding: 24px; min-width: 500px; max-width: 90vw; color: var(--text-primary); }
.modal h3 { margin-bottom: 20px; font-size: 18px; color: var(--text-primary); }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; color: var(--text-secondary); }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--input-bg); color: var(--text-primary); }
.form-group textarea { resize: vertical; }
.checkbox-group label { display: flex; align-items: center; gap: 8px; cursor: pointer; }
.checkbox-group input[type="checkbox"] { width: auto; }
.modal-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 20px; }
.btn-primary { padding: 8px 16px; background: var(--button-bg); color: var(--button-text); border: none; border-radius: 6px; cursor: pointer; }
.btn-secondary { padding: 8px 16px; background: var(--secondary-bg); color: var(--text-primary); border: none; border-radius: 6px; cursor: pointer; }
</style>
