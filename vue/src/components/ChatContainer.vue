<template>
  <div class="chat-container">
    <div class="chat-header">
      <ModelSelector />
      <ContextStatusIndicator 
        v-if="chatStore.activeSessionId" 
        :session-id="chatStore.activeSessionId"
        @session-changed="handleSessionChanged"
        @limit-reached="handleLimitReached"
      />
      <UserProfileMenu :current-theme="currentTheme" @toggle-theme="$emit('toggle-theme')" />
    </div>

    <div class="scroll-area" ref="chatEl">
      <div v-if="chatStore.messages.length === 0" class="welcome-message">
        <div class="home-welcome-icon">
          <div class="home-welcome-inner">
            <img src="@/assets/images/hero-brain-3d.png" alt="AI" class="home-welcome-img" />
          </div>
        </div>
        <div class="welcome-title">AI 智能助手</div>
        <div class="welcome-subtitle">
          基于本地大语言模型，为您提供多种场景的专业支持
        </div>
        <div class="scenario-cards">
          <ScenarioCard
              v-for="card in scenarioCards"
              :key="card.value"
              :title="card.label"
              :description="card.description"
              @click="selectSceneFromCard(card.value)"
          />
        </div>
      </div>

      <div v-else class="messages-list">
        <div v-for="msg in chatStore.messages" :key="msg.id">
          <MessageItem
              v-if="!msg.type"
              :message="msg"
              :conversation-id="chatStore.activeSessionId"
              :is-research-mode="chatStore.isResearchMode"
              :research-session-id="chatStore.researchSessionId"
              @edit-and-send="emit('edit-and-send', $event)"
              @regenerate="emit('regenerate', $event)"
              @evidence-updated="emit('evidence-updated', $event)"
          />
          <ResearchActivities v-if="msg.type === 'activities'" :activities="msg.payload" />
          <ResearchReport v-if="msg.type === 'report'" :report="msg.payload" />
        </div>
        <!-- The TypingIndicator component is no longer needed here as it's handled inside MessageItem -->
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue';
import { useChatStore } from '@/store';
import ScenarioCard from '@/components/ScenarioCard.vue';
import MessageItem from '@/components/MessageItem.vue';
import UserProfileMenu from './UserProfileMenu.vue';
import ModelSelector from './ModelSelector.vue';
import ContextStatusIndicator from './ContextStatusIndicator.vue';
import ResearchActivities from './ResearchActivities.vue';
import ResearchReport from './ResearchReport.vue';

defineProps({ currentTheme: String });

// Declare all events the component can emit
const emit = defineEmits([
    'toggle-theme',
    'send-message-from-container',
    'edit-and-send',
    'regenerate',
    'evidence-updated',
    'open-ppt-generator',
    'session-changed'
]);

// 处理会话变更（总结后新建）
const handleSessionChanged = (data) => {
  emit('session-changed', data);
};

// 处理上下文达到上限
const handleLimitReached = (status) => {
  console.log('上下文达到上限:', status);
};

const chatStore = useChatStore();
const chatEl = ref(null);

const scenarioCards = [
  { value: 'research', label: '研究报告', description: '生成专业的研究报告...' },
  { value: 'ppt', label: 'PPT生成', description: '根据主题创建完整的PPT内容...' },
  { value: 'blog', label: '博客撰写', description: '创作技术博客、行业分析...' },
  { value: 'counseling', label: '心理辅导', description: '提供情感支持、压力疏导...' }
];

const selectSceneFromCard = (scenario) => {
  if (scenario === 'ppt') {
    // 打开PPT生成器
    emit('open-ppt-generator');
  } else {
    const messageText = `你好，我想生成一份关于"${scenario}"的研究报告`;
    emit('send-message-from-container', messageText);
  }
};

const scrollToBottom = () => {
  nextTick(() => {
    if (chatEl.value) {
      chatEl.value.scrollTop = chatEl.value.scrollHeight;
    }
  });
};

// Watch for new messages to scroll down
watch(
    () => chatStore.messages.length,
    scrollToBottom
);

// Watch the last message's content length to scroll during streaming
watch(
    () => {
        const lastMessage = chatStore.messages[chatStore.messages.length - 1];
        return lastMessage ? lastMessage.content?.length : 0;
    },
    scrollToBottom
);
</script>

<style scoped>
.chat-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: var(--primary-bg);
  overflow: hidden;
  height: 100%;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-md) var(--spacing-lg);
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--secondary-bg);
  z-index: 10;
  backdrop-filter: var(--blur);
  -webkit-backdrop-filter: var(--blur);
}

.scroll-area {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-lg);
  scroll-behavior: smooth;
}

.welcome-message {
  max-width: 800px;
  margin: 0 auto;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 40px 20px;
}

.home-welcome-icon {
  width: 100px;
  height: 100px;
  margin: 0 auto 24px;

  background: black;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 40px rgba(59, 130, 246, 0.4);
  position: relative;
  overflow: hidden;
  flex-shrink: 0;
  border: 1px solid rgba(59, 130, 246, 0.2);
}

.home-welcome-icon::before {
  content: '';
  position: absolute;
  inset: -2px;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  z-index: 0;
  border-radius: 50%;
  opacity: 0.6;
}

.home-welcome-inner {
  position: absolute;
  inset: 3px;
  background: #020617; /* Even darker bg */
  border-radius: 50%;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.home-welcome-img {
  width: 140%;
  height: 140%;
  object-fit: contain;
  mix-blend-mode: screen;
  filter: drop-shadow(0 0 15px rgba(59, 130, 246, 0.6));
  animation: floatHome 6s ease-in-out infinite;
}

@keyframes floatHome {
  0%, 100% { transform: translateY(0) scale(1); }
  50% { transform: translateY(-5px) scale(1.02); }
}

.welcome-title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 8px;
  color: var(--text-primary);
  letter-spacing: -0.032em;
  background: var(--gradient-blue);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.welcome-subtitle {
  font-size: 16px;
  color: var(--text-secondary);
  margin-bottom: 32px;
  max-width: 600px;
  line-height: 1.5;
  font-weight: 400;
}

.scenario-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  width: 100%;
  max-width: 900px;
}

.messages-list {
  max-width: 900px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

/* Responsive Design */
@media (max-width: 1024px) {
  .messages-list {
    max-width: 100%;
    padding: 0 var(--spacing-sm);
  }
}

@media (max-width: 768px) {
  .chat-header {
    padding: var(--spacing-sm) var(--spacing-md);
    flex-wrap: wrap;
    gap: var(--spacing-sm);
  }

  .scroll-area {
    padding: var(--spacing-md);
  }

  .welcome-title {
    font-size: 28px;
  }

  .welcome-subtitle {
    font-size: 16px;
    padding: 0 var(--spacing-sm);
  }

  .scenario-cards {
    grid-template-columns: 1fr;
    gap: var(--spacing-sm);
    padding: 0 var(--spacing-sm);
  }
  
  .messages-list {
    gap: var(--spacing-md);
  }
}

@media (max-width: 480px) {
  .chat-container {
    min-height: 100vh;
    min-height: 100dvh; /* 动态视口高度，适配移动端浏览器 */
  }

  .chat-header {
    padding: var(--spacing-xs) var(--spacing-sm);
  }

  .scroll-area {
    padding: var(--spacing-sm);
  }

  .welcome-message {
    padding: var(--spacing-md) var(--spacing-sm);
  }

  .welcome-title {
    font-size: 22px;
  }

  .welcome-subtitle {
    font-size: 13px;
    line-height: 1.4;
  }
  
  .scenario-cards {
    max-width: 100%;
  }
}
</style>