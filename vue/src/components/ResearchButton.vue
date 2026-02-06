<template>
  <div class="research-button-wrapper">
    <button 
      class="research-btn" 
      :class="{ 'researching': isResearching }"
      @click="startResearch"
      :disabled="isResearching || !message.trim()"
      :title="buttonTitle"
    >
      <span class="research-icon" :class="{ 'spinning': isResearching }">
        {{ isResearching ? '🔄' : '🔍' }}
      </span>
      <span class="research-text">
        {{ isResearching ? '研究中...' : '深度研究' }}
      </span>
    </button>
    
    <!-- 研究进度显示 -->
    <div v-if="isResearching && researchProgress" class="research-progress">
      <div class="progress-header">
        <span>研究进度</span>
        <button @click="cancelResearch" class="cancel-btn">取消</button>
      </div>
      <div class="progress-content">
        <pre>{{ researchProgress }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useChatStore } from '@/store';
import { startResearch as apiStartResearch, streamResearchProgress } from '@/api/research';

const props = defineProps({
  message: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['research-complete', 'research-error']);

const chatStore = useChatStore();
const isResearching = ref(false);
const researchProgress = ref('');
const researchStreamController = ref(null);
const currentSessionId = ref(null);
const agentTasks = ref([]); // 并行Agent任务状态

const buttonTitle = computed(() => {
  if (!props.message.trim()) {
    return '请先输入要研究的内容';
  }
  if (isResearching.value) {
    return '正在进行深度研究...';
  }
  return '启动Agentic RAG深度研究，获取更全面的信息';
});

// 移除 getStepIcon 函数，不再需要

// 启动研究
const startResearch = async () => {
  if (isResearching.value || !props.message.trim()) return;
  
  isResearching.value = true;
  researchProgress.value = '🚀 研究任务已启动，正在初始化...';
  agentTasks.value = [];
  
  try {
    const response = await apiStartResearch({
      query: props.message,
      research_type: 'deep'
    });
    currentSessionId.value = response.session_id;
    
    // 使用 streamResearchProgress（fetch + ReadableStream）
    researchStreamController.value = streamResearchProgress(
      response.session_id,
      handleResearchEvent,
      handleResearchError,
      handleResearchComplete
    );
    
  } catch (error) {
    console.error('启动研究失败:', error);
    const errorMessage = error.message || '启动失败';
    researchProgress.value = `❌ 启动失败: ${errorMessage}`;
    emit('research-error', errorMessage);
    isResearching.value = false;
  }
};

// 处理研究进度事件
const handleResearchEvent = (data) => {
  if (data.type === 'status_update' && data.data) {
    const progress = data.data.progress || 0;
    const step = data.data.current_step || '';
    
    let msg = `🔍 ${step}`;
    if (progress > 0) {
      msg += ` (${Math.round(progress * 100)}%)`;
    }
    
    // 处理并行Agent任务信息
    if (data.data.task_name) {
      const taskName = data.data.task_name;
      const taskStatus = data.data.task_status || 'running';
      const existing = agentTasks.value.find(t => t.name === taskName);
      if (existing) {
        existing.status = taskStatus;
      } else {
        agentTasks.value.push({ name: taskName, status: taskStatus });
      }
      
      const statusIcon = taskStatus === 'completed' ? '✓' : taskStatus === 'failed' ? '✗' : '⟳';
      msg += `\n${statusIcon} ${taskName}: ${taskStatus}`;
    }
    
    // 显示所有Agent状态
    if (agentTasks.value.length > 1) {
      msg += '\n---';
      for (const task of agentTasks.value) {
        const icon = task.status === 'completed' ? '✅' : task.status === 'failed' ? '❌' : '🔄';
        msg += `\n${icon} ${task.name}: ${task.status === 'running' ? '调研中...' : task.status === 'completed' ? '已完成' : '失败'}`;
      }
    }
    
    researchProgress.value = msg;
  }
};

// 处理研究错误
const handleResearchError = (error) => {
  console.error('研究失败:', error);
  const errorMessage = error.message || '未知错误';
  researchProgress.value = `❌ 研究失败: ${errorMessage}`;
  emit('research-error', errorMessage);
  isResearching.value = false;
};

// 处理研究完成
const handleResearchComplete = (data) => {
  isResearching.value = false;
  researchProgress.value = '✓ 研究完成！';
  
  if (researchStreamController.value) {
    researchStreamController.value.abort();
    researchStreamController.value = null;
  }
  
  const reportText = data?.report_text || '研究完成，但报告为空。';
  const metadata = data?.metadata || {};
  
  chatStore.addMessage({
    role: 'assistant',
    content: reportText,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    type: 'research_report',
    metadata: metadata
  });
  
  emit('research-complete', reportText);
  
  setTimeout(() => {
    researchProgress.value = '';
    agentTasks.value = [];
  }, 3000);
};

// 取消研究
const cancelResearch = () => {
  if (researchStreamController.value) {
    researchStreamController.value.abort();
    researchStreamController.value = null;
  }
  
  isResearching.value = false;
  researchProgress.value = '❌ 研究已取消';
  agentTasks.value = [];
  
  setTimeout(() => {
    researchProgress.value = '';
  }, 2000);
};
</script>

<style scoped>
.research-button-wrapper {
  position: relative;
}

.research-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.research-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.research-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.research-btn.researching {
  background: linear-gradient(135deg, #ff9a56 0%, #ff6b6b 100%);
}

.research-icon {
  display: inline-block;
  font-size: 16px;
  transition: transform 0.3s ease;
}

.research-icon.spinning {
  animation: spin 2s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.research-progress {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--primary-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  margin-top: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  z-index: 100;
  max-width: 400px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  font-weight: 500;
  color: var(--text-primary);
}

.cancel-btn {
  background: none;
  border: none;
  color: var(--error-color);
  cursor: pointer;
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.cancel-btn:hover {
  background-color: var(--error-bg);
}

.progress-content {
  padding: 12px 16px;
  max-height: 150px;
  overflow-y: auto;
}

.progress-content pre {
  margin: 0;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* 滚动条样式 */
.progress-content::-webkit-scrollbar {
  width: 4px;
}

.progress-content::-webkit-scrollbar-track {
  background: var(--secondary-bg);
  border-radius: 2px;
}

.progress-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 2px;
}

.progress-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}
</style> 