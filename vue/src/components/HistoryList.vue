<template>
  <div class="history-section">
    <div class="history-header">
      <div class="history-title">对话记忆</div>
      <div class="history-actions">
        <button class="action-button new-chat-btn" @click="startNewChat" title="新建对话">
          <svg viewBox="0 0 24 24" fill="currentColor" class="icon">
            <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
          </svg>
        </button>
        <button class="action-button" @click="clearHistory" title="清空历史">
          <svg viewBox="0 0 24 24" fill="currentColor" class="icon">
            <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- 对话列表 -->
    <div class="history-items" v-if="filteredHistory.length > 0">
      <div
          v-for="item in filteredHistory"
          :key="item.id"
          class="history-item"
          :class="{ 
            active: item.id === chatStore.activeSessionId, 
            pinned: item.pinned,
            loading: switchingToId === item.id
          }"
          @click="handleSelectHistory(item.id)"
      >
        <div class="item-content">
          <div class="item-title">{{ item.title }}</div>
          <div class="item-preview">{{ item.last_message }}</div>
          <div class="item-meta">
            <span class="message-count">{{ item.message_count }} 条消息</span>
            <span class="timestamp">{{ formatTime(item.updated_at) }}</span>
            <span v-if="item.pinned" class="pinned-indicator">📌</span>
          </div>
        </div>
        <div class="item-actions">
          <button class="item-action" @click.stop="togglePin(item)" :title="item.pinned ? '取消置顶' : '置顶'">
            <svg viewBox="0 0 24 24" fill="currentColor" class="icon-small">
              <path d="M17 3H7c-1.1 0-2 .9-2 2v16l7-3 7 3V5c0-1.1-.9-2-2-2z"/>
            </svg>
          </button>
          <button class="item-action" @click.stop="deleteHistory(item.id)" title="删除">
            <svg viewBox="0 0 24 24" fill="currentColor" class="icon-small">
              <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div class="empty-state" v-else>
      <div class="empty-icon">💬</div>
      <h3>开始您的第一次对话</h3>
      <p>您的对话历史将在这里显示，AI会记住您的偏好和重要信息</p>
      <button class="start-chat-button" @click="startNewChat">
        开始对话
      </button>
    </div>

    <!-- 加载状态 -->
    <div class="loading-state" v-if="loading">
      <div class="loading-spinner"></div>
      <p>加载对话历史...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useChatStore } from '@/store';
import { chatAPI } from '@/api/index';

const chatStore = useChatStore();

// 响应式数据
const loading = ref(false);
const switchingToId = ref(null); // 正在切换到的会话ID

// 使用 store 中的历史列表数据，而不是本地状态
const historyList = computed(() => chatStore.historyList);

// 计算属性
const filteredHistory = computed(() => {
  // 确保 historyList.value 是数组，防止 null/undefined 导致迭代错误
  const list = historyList.value || [];
  if (!Array.isArray(list)) return [];
  
  let filtered = [...list]; // 创建副本以避免修改原数组

  // 置顶的会话排在前面，按更新时间倒序
  return filtered.sort((a, b) => {
    if (a.pinned && !b.pinned) return -1;
    if (!a.pinned && b.pinned) return 1;
    const dateA = a.updated_at ? new Date(a.updated_at) : new Date(0);
    const dateB = b.updated_at ? new Date(b.updated_at) : new Date(0);
    return dateB - dateA;
  });
});

// 方法
async function handleSelectHistory(id) {
  // 防止重复点击
  if (switchingToId.value === id || chatStore.isLoading) {
    return;
  }
  
  switchingToId.value = id;
  try {
    await chatStore.loadHistory(id);
  } catch (error) {
    console.error('加载历史记录失败:', error);
    const errorMessage = chatStore.lastError?.userMessage || '加载失败，请重试';
    alert(errorMessage);
    // 刷新会话列表，以防会话已被删除
    chatStore.fetchSessions();
  } finally {
    switchingToId.value = null;
  }
}

async function clearHistory() {
  if (confirm('确定要清空所有对话历史吗？此操作不可恢复。')) {
    try {
      // 删除所有会话
      await chatStore.deleteAllSessions();
      console.log('清空对话历史');
    } catch (error) {
      console.error('清空对话历史失败:', error);
      const errorMessage = chatStore.lastError?.userMessage || '清空失败，请重试';
      alert(errorMessage);
    }
  }
}

async function togglePin(item) {
  try {
    // 这里应该调用API切换置顶状态
    // 暂时本地实现
    item.pinned = !item.pinned;
    console.log(`${item.pinned ? '置顶' : '取消置顶'}对话:`, item.id);
  } catch (error) {
    console.error('切换置顶状态失败:', error);
    alert('操作失败，请重试');
  }
}

async function deleteHistory(id) {
  if (confirm('确定要删除这个对话吗？')) {
    try {
      // 使用store的deleteSession方法，它会自动处理状态同步
      await chatStore.deleteSession(id);
      console.log('删除对话:', id);
    } catch (error) {
      console.error('删除对话失败:', error);
      // 显示用户友好的错误信息
      const errorMessage = chatStore.lastError?.userMessage || '删除失败，请重试';
      alert(errorMessage);
    }
  }
}

function startNewChat() {
  // 开始新对话
  chatStore.startNewChat();
}

function formatTime(date) {
  if (!date) return '';
  
  // 确保 date 是 Date 对象
  const dateObj = typeof date === 'string' ? new Date(date) : date;
  
  // 检查是否是有效的日期
  if (isNaN(dateObj.getTime())) return '';
  
  const now = new Date();
  const diff = now - dateObj;
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);
  
  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  if (hours < 24) return `${hours}小时前`;
  if (days < 7) return `${days}天前`;
  
  return dateObj.toLocaleDateString('zh-CN');
}

// 生命周期 - 组件挂载时加载历史列表
onMounted(async () => {
  // 等待一个微任务周期，确保 token 已保存到 storage
  await new Promise(resolve => setTimeout(resolve, 0));
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
  if (token) {
    // 使用 store 的方法加载历史列表
    chatStore.fetchHistoryList();
  }
});
</script>

<style scoped>
.history-section {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.history-title {
  font-size: 16px;
  color: var(--text-primary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.history-actions {
  display: flex;
  gap: 8px;
}

.action-button {
  background: none;
  border: none;
  color: var(--text-secondary);
  padding: 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-button:hover {
  background-color: var(--hover-bg);
  color: var(--text-primary);
}

.new-chat-btn {
  background-color: var(--accent-blue);
  color: white;
  border-radius: 6px;
}

.new-chat-btn:hover {
  background-color: var(--accent-blue);
  opacity: 0.9;
  color: white;
}

.icon {
  width: 18px;
  height: 18px;
}

.history-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.history-item {
  padding: 15px;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.history-item:hover {
  background: var(--hover-bg);
  border-color: var(--border-color);
}

.history-item.active {
  background: rgba(59, 130, 246, 0.1);
  border-color: var(--button-bg);
}

.history-item.pinned {
  border-left: 3px solid #ffd700;
  background: rgba(255, 215, 0, 0.05);
}

.history-item.loading {
  opacity: 0.7;
  pointer-events: none;
  position: relative;
}

.history-item.loading::after {
  content: '';
  position: absolute;
  top: 50%;
  right: 15px;
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-color);
  border-top: 2px solid var(--button-bg);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  transform: translateY(-50%);
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-preview {
  color: var(--text-secondary);
  font-size: 13px;
  margin-bottom: 8px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.item-meta {
  display: flex;
  gap: 15px;
  font-size: 11px;
  color: var(--text-secondary);
  align-items: center;
}

.pinned-indicator {
  font-size: 12px;
  color: #ffd700;
}

.item-actions {
  display: flex;
  gap: 5px;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.history-item:hover .item-actions {
  opacity: 1;
}

.item-action {
  background: none;
  border: none;
  color: var(--text-secondary);
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.item-action:hover {
  background-color: var(--hover-bg);
  color: var(--text-primary);
}

.icon-small {
  width: 14px;
  height: 14px;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 15px;
}

.empty-state h3 {
  margin: 0 0 10px 0;
  color: var(--text-primary);
  font-size: 18px;
}

.empty-state p {
  margin: 0 0 20px 0;
  font-size: 14px;
  line-height: 1.5;
}

.start-chat-button {
  background-color: var(--button-bg);
  color: var(--button-text);
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.start-chat-button:hover {
  opacity: 0.9;
}

.loading-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border-color);
  border-top: 2px solid var(--button-bg);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 15px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 自定义滚动条样式 */
.history-section::-webkit-scrollbar {
  width: 6px;
}

.history-section::-webkit-scrollbar-track {
  background: transparent;
}

.history-section::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.history-section::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.4);
}

/* 移动端响应式 */
@media (max-width: 768px) {
  .history-section {
    padding: 16px 12px;
  }

  .history-header {
    margin-bottom: 16px;
  }

  .history-item {
    padding: 12px;
    min-height: 44px;
  }

  .item-actions {
    opacity: 1; /* 移动端始终显示操作按钮 */
  }

  .item-action {
    min-width: 36px;
    min-height: 36px;
    padding: 8px;
  }

  .action-button {
    min-width: 36px;
    min-height: 36px;
    padding: 8px;
  }

  .item-title {
    font-size: 14px;
  }

  .item-preview {
    font-size: 12px;
    -webkit-line-clamp: 1;
  }

  .item-meta {
    gap: 8px;
    font-size: 10px;
  }

  .empty-state {
    padding: 30px 16px;
  }

  .start-chat-button {
    min-height: 44px;
    padding: 12px 24px;
  }
}
</style>