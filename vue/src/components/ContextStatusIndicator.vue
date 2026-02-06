<template>
  <div class="context-status-indicator" v-if="visible && contextStatus">
    <div class="status-content">
      <div class="status-bar-container">
        <div class="status-bar">
          <div 
            class="status-fill" 
            :style="{ width: contextStatus.usage_percent + '%' }"
            :class="statusClass"
          ></div>
        </div>
        <span class="status-label">{{ statusLabel }}</span>
      </div>
      
      <div class="status-details" v-if="expanded">
        <div class="detail-item">
          <span class="detail-label">已用:</span>
          <span class="detail-value">{{ formatTokens(contextStatus.current_tokens) }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">上限:</span>
          <span class="detail-value">{{ formatTokens(contextStatus.max_tokens) }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">消息:</span>
          <span class="detail-value">{{ contextStatus.message_count }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">记忆:</span>
          <span class="detail-value" :class="{ enabled: contextStatus.memory_enabled }">
            {{ contextStatus.memory_enabled ? '已启用' : '已禁用' }}
          </span>
        </div>
      </div>
    </div>
    
    <div class="status-actions">
      <button @click="expanded = !expanded" class="expand-btn" :title="expanded ? '收起' : '展开'">
        {{ expanded ? '▲' : '▼' }}
      </button>
      <button 
        v-if="contextStatus.is_over_limit || contextStatus.is_near_limit" 
        @click="handleSummarize" 
        class="summarize-btn"
        :disabled="summarizing"
        title="总结并新建对话"
      >
        {{ summarizing ? '处理中...' : '📝 新建' }}
      </button>
    </div>

    <!-- 超限提醒弹窗 -->
    <div class="limit-warning-modal" v-if="showLimitWarning">
      <div class="modal-overlay" @click="showLimitWarning = false"></div>
      <div class="modal-content">
        <h3>⚠️ 上下文已达上限</h3>
        <p>当前对话的上下文已达到 {{ formatTokens(contextStatus.max_tokens) }} token 上限。</p>
        <p>建议总结当前对话并开启新对话，以保持AI的响应质量。</p>
        <div class="modal-actions">
          <button @click="confirmSummarize" class="confirm-btn" :disabled="summarizing">
            {{ summarizing ? '处理中...' : '确认总结并新建' }}
          </button>
          <button @click="showLimitWarning = false" class="cancel-btn">
            稍后处理
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { useChatStore } from '@/store';
import { chatAPI } from '@/api/index';

const props = defineProps({
  sessionId: String,
  visible: {
    type: Boolean,
    default: true
  }
});

const emit = defineEmits(['session-changed', 'limit-reached']);

const chatStore = useChatStore();
const contextStatus = ref(null);
const expanded = ref(false);
const summarizing = ref(false);
const showLimitWarning = ref(false);
let refreshInterval = null;

const statusClass = computed(() => {
  if (!contextStatus.value) return '';
  if (contextStatus.value.is_over_limit) return 'danger';
  if (contextStatus.value.is_near_limit) return 'warning';
  return 'normal';
});

const statusLabel = computed(() => {
  if (!contextStatus.value) return '';
  const percent = contextStatus.value.usage_percent.toFixed(0);
  if (contextStatus.value.is_over_limit) return `${percent}% 已超限`;
  if (contextStatus.value.is_near_limit) return `${percent}% 接近上限`;
  return `${percent}%`;
});

const formatTokens = (tokens) => {
  if (tokens >= 1000) {
    return (tokens / 1000).toFixed(1) + 'K';
  }
  return tokens.toString();
};

const loadContextStatus = async () => {
  const sessionId = props.sessionId || chatStore.activeSessionId;
  if (!sessionId) {
    contextStatus.value = null;
    return;
  }
  
  try {
    const status = await chatAPI.getContextStatus(sessionId);
    contextStatus.value = status;
    
    // 同步到store
    chatStore.setContextStatus(status);
    
    // 检查是否超限，自动弹出提醒
    if (status.is_over_limit && !showLimitWarning.value) {
      showLimitWarning.value = true;
      emit('limit-reached', status);
    }
  } catch (error) {
    console.error('加载上下文状态失败:', error);
    // 解析错误信息
    if (error.response?.data?.error) {
      const apiError = error.response.data.error;
      console.warn('API错误:', apiError.code, apiError.message);
    }
  }
};

const handleSummarize = () => {
  showLimitWarning.value = true;
};

const confirmSummarize = async () => {
  const sessionId = props.sessionId || chatStore.activeSessionId;
  if (!sessionId) return;
  
  summarizing.value = true;
  try {
    const result = await chatAPI.summarizeAndNewSession(sessionId);
    
    if (result.success && result.new_session_id) {
      // 刷新历史列表
      await chatStore.fetchSessions();
      // 切换到新会话
      await chatStore.switchSession(result.new_session_id);
      
      emit('session-changed', {
        oldSessionId: sessionId,
        newSessionId: result.new_session_id,
        summary: result.summary
      });
      
      showLimitWarning.value = false;
      const summaryPreview = result.summary ? result.summary.substring(0, 200) + '...' : '无';
      alert(`已创建新对话！\n\n上一对话总结：\n${summaryPreview}`);
    }
  } catch (error) {
    console.error('总结并新建会话失败:', error);
    // 显示友好的错误信息
    let errorMessage = '操作失败，请重试';
    if (error.response?.data?.error) {
      errorMessage = error.response.data.error.message || errorMessage;
    }
    alert(errorMessage);
  } finally {
    summarizing.value = false;
  }
};

// 监听会话变化
watch(() => props.sessionId, loadContextStatus);
watch(() => chatStore.activeSessionId, loadContextStatus);
watch(() => chatStore.messages.length, loadContextStatus);

onMounted(() => {
  loadContextStatus();
  // 每30秒刷新一次状态
  refreshInterval = setInterval(loadContextStatus, 30000);
});

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});

// 暴露刷新方法
defineExpose({ refresh: loadContextStatus });
</script>

<style scoped>
.context-status-indicator {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xs) var(--spacing-sm);
  background: var(--secondary-bg);
  border-radius: var(--radius-medium);
  font-size: 12px;
}

.status-content {
  flex: 1;
}

.status-bar-container {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.status-bar {
  width: 80px;
  height: 6px;
  background: var(--border-color);
  border-radius: 3px;
  overflow: hidden;
}

.status-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.3s, background 0.3s;
}

.status-fill.normal {
  background: var(--accent-blue);
}

.status-fill.warning {
  background: #f59e0b;
}

.status-fill.danger {
  background: #ef4444;
}

.status-label {
  color: var(--text-secondary);
  white-space: nowrap;
}

.status-details {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
  margin-top: var(--spacing-xs);
  padding-top: var(--spacing-xs);
  border-top: 1px solid var(--border-color);
}

.detail-item {
  display: flex;
  gap: 4px;
}

.detail-label {
  color: var(--text-secondary);
}

.detail-value {
  color: var(--text-primary);
}

.detail-value.enabled {
  color: var(--accent-green, #10b981);
}

.status-actions {
  display: flex;
  gap: 4px;
}

.expand-btn,
.summarize-btn {
  padding: 2px 6px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-small);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 10px;
  transition: all 0.2s;
}

.expand-btn:hover,
.summarize-btn:hover {
  background: var(--hover-bg);
}

.summarize-btn {
  background: var(--accent-blue);
  color: white;
  border-color: var(--accent-blue);
}

.summarize-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Modal */
.limit-warning-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
}

.modal-content {
  position: relative;
  background: var(--secondary-bg);
  padding: var(--spacing-xl);
  border-radius: var(--radius-large);
  max-width: 400px;
  width: 90%;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
}

.modal-content h3 {
  margin: 0 0 var(--spacing-md);
  color: var(--text-primary);
}

.modal-content p {
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-sm);
  line-height: 1.5;
}

.modal-actions {
  display: flex;
  gap: var(--spacing-md);
  margin-top: var(--spacing-lg);
}

.confirm-btn {
  flex: 1;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--accent-blue);
  color: white;
  border: none;
  border-radius: var(--radius-medium);
  cursor: pointer;
  font-size: 14px;
}

.confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.cancel-btn {
  flex: 1;
  padding: var(--spacing-sm) var(--spacing-md);
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-medium);
  cursor: pointer;
  font-size: 14px;
}

.cancel-btn:hover {
  background: var(--hover-bg);
}
</style>
