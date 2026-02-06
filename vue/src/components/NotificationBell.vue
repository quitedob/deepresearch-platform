<template>
  <div class="notification-bell">
    <button class="bell-btn" @click="toggleDropdown">
      🔔
      <span v-if="unreadCount > 0" class="badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
    </button>
    
    <div v-if="showDropdown" class="dropdown" @click.stop>
      <div class="dropdown-header">
        <span>通知</span>
        <button v-if="notifications.length > 0" class="mark-all-btn" @click="markAllRead">全部已读</button>
      </div>
      <div class="notifications-list">
        <div v-if="notifications.length === 0" class="empty-state">暂无通知</div>
        <div 
          v-for="notification in notifications" 
          :key="notification.id" 
          class="notification-item"
          @click="markRead(notification)"
        >
          <span :class="['type-dot', notification.type]"></span>
          <div class="notification-content">
            <div class="notification-title">{{ notification.title }}</div>
            <div class="notification-text">{{ notification.content }}</div>
            <div class="notification-time">{{ formatTime(notification.created_at) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { getNotifications, getUnreadCount, markAsRead, markAllAsRead } from '@/api/notification'

const showDropdown = ref(false)
const notifications = ref([])
const unreadCount = ref(0)

const loadNotifications = async () => {
  try {
    const response = await getNotifications(10, 0)
    notifications.value = (response.data || response).notifications || []
  } catch (error) {
    console.error('加载通知失败:', error)
  }
}

const loadUnreadCount = async () => {
  try {
    const response = await getUnreadCount()
    unreadCount.value = (response.data || response).unread_count || 0
  } catch (error) {
    console.error('获取未读数量失败:', error)
  }
}

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
  if (showDropdown.value) {
    loadNotifications()
  }
}

const formatTime = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  return date.toLocaleDateString('zh-CN')
}

const markRead = async (notification) => {
  try {
    await markAsRead(notification.id)
    loadUnreadCount()
  } catch (error) {
    console.error('标记已读失败:', error)
  }
}

const markAllRead = async () => {
  try {
    await markAllAsRead()
    unreadCount.value = 0
  } catch (error) {
    console.error('标记全部已读失败:', error)
  }
}

const handleClickOutside = (e) => {
  if (!e.target.closest('.notification-bell')) {
    showDropdown.value = false
  }
}

onMounted(async () => {
  // 等待一个微任务周期，确保 token 已保存到 storage
  await new Promise(resolve => setTimeout(resolve, 0));
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
  if (token) {
    loadUnreadCount()
  }
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.notification-bell { position: relative; }
.bell-btn { background: none; border: none; font-size: 20px; cursor: pointer; position: relative; padding: 8px; }
.badge { position: absolute; top: 0; right: 0; background: #ef4444; color: white; font-size: 10px; padding: 2px 5px; border-radius: 10px; min-width: 16px; text-align: center; }
.dropdown { position: absolute; top: 100%; right: 0; width: 320px; background: white; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.15); z-index: 1000; overflow: hidden; }
.dropdown-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid #e5e7eb; font-weight: 600; }
.mark-all-btn { background: none; border: none; color: #667eea; font-size: 13px; cursor: pointer; }
.notifications-list { max-height: 400px; overflow-y: auto; }
.empty-state { padding: 40px; text-align: center; color: #9ca3af; }
.notification-item { display: flex; gap: 12px; padding: 12px 16px; cursor: pointer; transition: background 0.2s; }
.notification-item:hover { background: #f9fafb; }
.type-dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 6px; flex-shrink: 0; }
.type-dot.system { background: #3b82f6; }
.type-dot.announce { background: #f59e0b; }
.type-dot.alert { background: #ef4444; }
.notification-content { flex: 1; min-width: 0; }
.notification-title { font-weight: 500; font-size: 14px; margin-bottom: 4px; }
.notification-text { font-size: 13px; color: #6b7280; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notification-time { font-size: 12px; color: #9ca3af; margin-top: 4px; }
</style>
