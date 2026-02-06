<template>
  <div class="history-section">
    <div class="history-header">
      <div class="history-title">出题记录</div>
      <div class="history-actions">
        <button class="action-button new-chat-btn" @click="startNewSession" title="新建出题">
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

    <!-- 会话列表 -->
    <div class="history-items" v-if="sessions.length > 0">
      <div
        v-for="item in sessions"
        :key="item.id"
        class="history-item"
        :class="{ active: item.id === activeSessionId }"
        @click="selectSession(item)"
      >
        <div class="item-content">
          <div class="item-title">{{ item.title || '新出题会话' }}</div>
          <div class="item-preview">{{ item.question_count || 0 }} 道题目</div>
          <div class="item-meta">
            <span class="timestamp">{{ formatTime(item.updated_at) }}</span>
          </div>
        </div>
        <div class="item-actions">
          <button class="item-action" @click.stop="deleteSession(item.id)" title="删除">
            <svg viewBox="0 0 24 24" fill="currentColor" class="icon-small">
              <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div class="empty-state" v-else-if="!loading">
      <div class="empty-icon">🎯</div>
      <h3>开始AI出题</h3>
      <p>您的出题记录将在这里显示</p>
      <button class="start-chat-button" @click="startNewSession">
        开始出题
      </button>
    </div>

    <!-- 加载状态 -->
    <div class="loading-state" v-if="loading">
      <div class="loading-spinner"></div>
      <p>加载出题记录...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { aiQuestionAPI } from '@/api/index';

const emit = defineEmits(['select-session', 'new-session']);

const sessions = ref([]);
const loading = ref(false);
const activeSessionId = ref(null);

// 加载会话列表
const fetchSessions = async () => {
  loading.value = true;
  try {
    const result = await aiQuestionAPI.getSessions(20, 0);
    sessions.value = result.sessions || result || [];
  } catch (error) {
    console.error('加载出题记录失败:', error);
    sessions.value = [];
  } finally {
    loading.value = false;
  }
};

// 选择会话
const selectSession = async (session) => {
  activeSessionId.value = session.id;
  emit('select-session', session);
  
  // 直接调用API加载会话详情，然后通过事件通知AISpace
  try {
    const detail = await aiQuestionAPI.getSession(session.id);
    // 触发自定义事件，传递会话数据
    window.dispatchEvent(new CustomEvent('ai-question-load-session', { 
      detail: { session, detail } 
    }));
  } catch (error) {
    console.error('加载会话失败:', error);
  }
};

// 新建会话
const startNewSession = () => {
  activeSessionId.value = null;
  emit('new-session');
  
  // 触发自定义事件
  window.dispatchEvent(new CustomEvent('ai-question-new-session'));
};

// 删除会话
const deleteSession = async (id) => {
  if (!confirm('确定要删除这个出题记录吗？')) return;
  
  try {
    await aiQuestionAPI.deleteSession(id);
    sessions.value = sessions.value.filter(s => s.id !== id);
    if (activeSessionId.value === id) {
      activeSessionId.value = null;
      startNewSession();
    }
  } catch (error) {
    console.error('删除失败:', error);
    alert('删除失败，请重试');
  }
};

// 清空历史
const clearHistory = async () => {
  if (!confirm('确定要清空所有出题记录吗？此操作不可恢复。')) return;
  
  try {
    for (const session of sessions.value) {
      await aiQuestionAPI.deleteSession(session.id);
    }
    sessions.value = [];
    activeSessionId.value = null;
    startNewSession();
  } catch (error) {
    console.error('清空失败:', error);
    alert('清空失败，请重试');
  }
};

// 格式化时间
const formatTime = (date) => {
  if (!date) return '';
  const dateObj = typeof date === 'string' ? new Date(date) : date;
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
};

// 监听会话创建事件
const handleSessionCreated = () => {
  fetchSessions();
};

// 暴露方法供父组件调用
defineExpose({
  fetchSessions,
  setActiveSession: (id) => { activeSessionId.value = id; }
});

onMounted(async () => {
  // 等待一个微任务周期，确保 token 已保存到 storage
  await new Promise(resolve => setTimeout(resolve, 0));
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
  if (token) {
    fetchSessions();
  }
  window.addEventListener('ai-question-session-created', handleSessionCreated);
});

onUnmounted(() => {
  window.removeEventListener('ai-question-session-created', handleSessionCreated);
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
}

.item-meta {
  display: flex;
  gap: 15px;
  font-size: 11px;
  color: var(--text-secondary);
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
</style>
