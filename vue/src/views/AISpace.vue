<template>
  <div class="ai-space-layout">
    <Sidebar />

    <main class="main-content">
      <div class="chat-section">
        <!-- Header with user profile -->
        <div class="chat-header">
          <div class="header-left">
            <button class="collapse-btn" @click="toggleSidebar" title="折叠侧边栏">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="3" y1="12" x2="21" y2="12"></line>
                <line x1="3" y1="6" x2="21" y2="6"></line>
                <line x1="3" y1="18" x2="21" y2="18"></line>
              </svg>
            </button>
            <div class="header-title">
              <h2>🎯 AI 题目生成助手</h2>
              <p class="subtitle">智能生成各类题目，支持单选、多选、判断、简答题</p>
            </div>
          </div>
          <div class="header-right">
            <!-- AI出题模型（由管理员在后台配置，不可用户选择） -->
            <div class="admin-model-display" :title="'模型由管理员配置'">
              <span class="admin-model-icon">🤖</span>
              <span class="admin-model-name">{{ aiQuestionModel || '加载中...' }}</span>
              <span class="admin-model-badge">管理员配置</span>
            </div>
            <UserProfileMenu :current-theme="currentTheme" @toggle-theme="toggleTheme" />
          </div>
        </div>

        <!-- Chat messages area - questions panel will be inside this -->
        <div class="chat-messages-wrapper">
          <div class="chat-messages" ref="messagesContainer">
            <div v-if="messages.length === 0" class="welcome-message">
              <div class="welcome-icon-wrapper">
                <div class="welcome-icon-inner">
                  <img src="@/assets/images/hero-brain-3d.png" alt="AI" class="welcome-img-3d" />
                </div>
              </div>
              <h3>欢迎使用AI题目生成器</h3>
              <p>告诉我你想生成什么类型的题目，例如：</p>
              <div class="example-prompts">
                <button @click="useExample('生成5道关于Python基础的单选题，难度中等')">
                  📝 Python基础单选题
                </button>
                <button @click="useExample('生成3道关于数据结构的多选题')">
                  📋 数据结构多选题
                </button>
                <button @click="useExample('生成5道关于计算机网络的判断题')">
                  ✅ 计算机网络判断题
                </button>
                <button @click="useExample('生成2道关于算法设计的简答题')">
                  📄 算法设计简答题
                </button>
              </div>
            </div>

            <div v-else class="messages-list">
              <div v-for="(msg, index) in messages" :key="msg.id || index" :class="['message', msg.role]">
                <div class="message-avatar">
                  {{ msg.role === 'user' ? '👤' : '🤖' }}
                </div>
                <div class="message-content">
                  <div class="message-bubble">
                    <!-- Editing Mode for User Messages -->
                    <div v-if="msg.role === 'user' && editingMessageId === msg.id" class="edit-area">
                      <textarea
                        v-model="editedContent"
                        ref="editTextareaRef"
                        class="edit-textarea"
                        rows="3"
                        @keydown.enter.exact.prevent="handleSendEdit"
                        @keydown.esc.prevent="cancelEdit"
                      ></textarea>
                    </div>
                    <div v-else-if="msg.role === 'assistant' && msg.isLoading" class="loading-indicator">
                      <span class="dot"></span>
                      <span class="dot"></span>
                      <span class="dot"></span>
                    </div>
                    <div v-else class="message-text">{{ msg.content }}</div>
                  </div>
                  <!-- Message Actions -->
                  <div class="message-actions" v-if="!msg.isLoading">
                    <button @click="copyContent(msg.content)" title="复制">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    </button>
                    <!-- User message: Edit button -->
                    <template v-if="msg.role === 'user'">
                      <button v-if="editingMessageId !== msg.id" @click="startEdit(msg)" title="编辑">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path></svg>
                      </button>
                      <button v-else @click="handleSendEdit" title="发送修改 (Enter)">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>
                      </button>
                    </template>
                    <!-- Assistant message: Regenerate button -->
                    <button v-if="msg.role === 'assistant'" @click="handleRegenerate(msg, index)" title="重新生成">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"></polyline><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path></svg>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Questions panel - inside chat area -->
          <transition name="slide">
            <div v-if="hasAnyQuestions && showQuestionsPanel" class="questions-panel">
              <div class="panel-header">
                <h3>📚 生成的题目 ({{ generatedQuestions.length }})</h3>
                <div class="panel-actions">
                  <button @click="toggleQuestionsPanel" class="action-btn toggle" title="隐藏面板">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="9 18 15 12 9 6"></polyline>
                    </svg>
                  </button>
                  <button @click="exportQuestions" class="action-btn export">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                      <polyline points="7 10 12 15 17 10"></polyline>
                      <line x1="12" y1="15" x2="12" y2="3"></line>
                    </svg>
                    导出
                  </button>
                  <button @click="clearQuestions" class="action-btn clear">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"></polyline>
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                    清空
                  </button>
                </div>
              </div>

              <!-- 轮次选择器 -->
              <div v-if="questionRounds.length > 1" class="round-tabs">
                <button
                  v-for="round in questionRounds"
                  :key="round.id"
                  :class="['round-tab', { active: activeRound === round.id }]"
                  @click="switchRound(round.id)"
                  :title="round.title"
                >
                  第{{ round.id + 1 }}轮 ({{ round.questions.length }})
                </button>
              </div>

              <div class="questions-list">
                <QuestionCard
                  v-for="(question, idx) in paginatedQuestions"
                  :key="question.id || idx"
                  :question="question"
                  :index="(currentPage - 1) * QUESTIONS_PER_PAGE + idx"
                  @answer="handleAnswer"
                  @check="checkAnswer"
                />
              </div>

              <!-- 分页控件 -->
              <div v-if="totalPages > 1" class="pagination">
                <button 
                  class="page-btn" 
                  :disabled="currentPage === 1"
                  @click="currentPage = 1"
                  title="首页"
                >
                  «
                </button>
                <button 
                  class="page-btn" 
                  :disabled="currentPage === 1"
                  @click="currentPage--"
                  title="上一页"
                >
                  ‹
                </button>
                <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
                <button 
                  class="page-btn" 
                  :disabled="currentPage === totalPages"
                  @click="currentPage++"
                  title="下一页"
                >
                  ›
                </button>
                <button 
                  class="page-btn" 
                  :disabled="currentPage === totalPages"
                  @click="currentPage = totalPages"
                  title="末页"
                >
                  »
                </button>
              </div>
            </div>
          </transition>

          <!-- Collapsed panel toggle button -->
          <div v-if="hasAnyQuestions && !showQuestionsPanel" class="panel-toggle-btn" @click="toggleQuestionsPanel">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
            <span class="toggle-badge">{{ totalQuestionsCount }}</span>
          </div>
        </div>

        <!-- Input area -->
        <div class="chat-input-area">
          <!-- 搜索开关 -->
          <div class="input-options">
            <label class="option-toggle" :class="{ active: useWebSearch }">
              <input type="checkbox" v-model="useWebSearch" />
              <span class="toggle-icon">🔍</span>
              <span class="toggle-label">联网搜索</span>
            </label>
          </div>
          <div class="input-wrapper">
            <textarea
              v-model="inputText"
              @keydown.enter.exact.prevent="sendMessage"
              @keydown.enter.shift.exact.prevent="insertNewline"
              placeholder="描述你想生成的题目类型（单选，多选，判断,简答）、数量、难度等..."
              rows="1"
              ref="inputRef"
              @input="autoGrow"
            ></textarea>
            <button @click="sendMessage" :disabled="!inputText.trim() || isGenerating" class="send-btn">
              <svg v-if="!isGenerating" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <line x1="22" y1="2" x2="11" y2="13"></line>
                <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
              </svg>
              <span v-else class="spinner"></span>
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, nextTick, watch, onMounted, onUnmounted, computed } from 'vue';
import Sidebar from '@/components/Sidebar.vue';
import UserProfileMenu from '@/components/UserProfileMenu.vue';
import QuestionCard from '@/components/QuestionCard.vue';
import { aiQuestionAPI } from '@/api/index';
import { useChatStore } from '@/store';

const chatStore = useChatStore();
const messages = ref([]);
const inputText = ref('');
const inputRef = ref(null);
const messagesContainer = ref(null);
const isGenerating = ref(false);

// AI出题模型配置（从 admin 后台获取）
const aiQuestionProvider = ref('');
const aiQuestionModelName = ref('');
const aiQuestionModel = computed(() => {
  if (aiQuestionModelName.value) {
    return aiQuestionModelName.value;
  }
  return '';
});

// 加载 admin 配置的 AI 出题模型
const loadAIQuestionConfig = async () => {
  try {
    const result = await aiQuestionAPI.getConfig();
    const config = result.config || result;
    if (config.default_provider) {
      aiQuestionProvider.value = config.default_provider;
    }
    if (config.default_model) {
      aiQuestionModelName.value = config.default_model;
    }
    console.log('[AISpace] 加载AI出题配置:', aiQuestionProvider.value, aiQuestionModelName.value);
  } catch (error) {
    console.warn('[AISpace] 获取AI出题配置失败，使用默认值:', error.message);
    // 回退到 store 中的默认值
    aiQuestionProvider.value = chatStore.currentProvider || 'deepseek';
    aiQuestionModelName.value = chatStore.currentModel || 'deepseek-chat';
  }
};

// 题目轮次管理
const questionRounds = ref([]); // [{id, title, questions, timestamp}]
const activeRound = ref(0);

// 网络搜索开关
const useWebSearch = ref(false);

// 会话管理
const currentSessionId = ref(null);
const currentSessionTitle = ref('');

// Edit functionality
const editingMessageId = ref(null);
const editedContent = ref('');
const editTextareaRef = ref(null);

// Questions panel toggle
const showQuestionsPanel = ref(true);

// 分页配置 - 每页显示3道题目，平衡了阅读效率和滚动便利性
const QUESTIONS_PER_PAGE = 3;
const currentPage = ref(1);

// 当前轮次的题目
const generatedQuestions = computed(() => {
  if (questionRounds.value.length === 0) return [];
  const round = questionRounds.value.find(r => r.id === activeRound.value);
  return round ? round.questions : [];
});

// 是否有任何题目（用于显示面板）
const hasAnyQuestions = computed(() => {
  return questionRounds.value.some(r => r.questions.length > 0);
});

// 所有轮次的题目总数
const totalQuestionsCount = computed(() => {
  return questionRounds.value.reduce((sum, r) => sum + r.questions.length, 0);
});

// 计算分页后的题目
const paginatedQuestions = computed(() => {
  const start = (currentPage.value - 1) * QUESTIONS_PER_PAGE;
  const end = start + QUESTIONS_PER_PAGE;
  return generatedQuestions.value.slice(start, end);
});

// 总页数
const totalPages = computed(() => {
  return Math.ceil(generatedQuestions.value.length / QUESTIONS_PER_PAGE);
});

// 当题目数量变化时，确保当前页有效
watch(() => generatedQuestions.value.length, () => {
  if (currentPage.value > totalPages.value && totalPages.value > 0) {
    currentPage.value = totalPages.value;
  }
});

// 主题相关
const currentTheme = ref(localStorage.getItem('app-theme') || 'dark');

const toggleTheme = () => {
  const newTheme = currentTheme.value === 'dark' ? 'light' : 'dark';
  currentTheme.value = newTheme;
  localStorage.setItem('app-theme', newTheme);
  
  if (newTheme === 'dark') {
    document.body.classList.add('dark');
    document.body.classList.remove('light');
  } else {
    document.body.classList.add('light');
    document.body.classList.remove('dark');
  }
};

// 折叠侧边栏
const toggleSidebar = () => {
  document.querySelector('.sidebar')?.classList.toggle('collapsed');
};

// Toggle questions panel
const toggleQuestionsPanel = () => {
  showQuestionsPanel.value = !showQuestionsPanel.value;
};

// 切换轮次
const switchRound = (roundId) => {
  activeRound.value = roundId;
  currentPage.value = 1;
};

// 使用示例提示
const useExample = (text) => {
  inputText.value = text;
  nextTick(() => {
    inputRef.value?.focus();
    autoGrow();
  });
};

// 自动增高输入框
const autoGrow = () => {
  const textarea = inputRef.value;
  if (textarea) {
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 150) + 'px';
  }
};

// 插入换行
const insertNewline = () => {
  const textarea = inputRef.value;
  if (textarea) {
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    inputText.value = inputText.value.substring(0, start) + '\n' + inputText.value.substring(end);
    nextTick(() => {
      textarea.selectionStart = textarea.selectionEnd = start + 1;
      autoGrow();
    });
  }
};

// 滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    }
  });
};

// Copy content
const copyContent = async (content) => {
  try {
    await navigator.clipboard.writeText(content);
  } catch (err) {
    console.error('Failed to copy:', err);
  }
};

// Start editing a message
const startEdit = (msg) => {
  editingMessageId.value = msg.id;
  editedContent.value = msg.content;
  nextTick(() => {
    const textarea = document.querySelector('.edit-textarea');
    if (textarea) {
      textarea.focus();
      textarea.select();
    }
  });
};

// Cancel editing
const cancelEdit = () => {
  editingMessageId.value = null;
  editedContent.value = '';
};

// Handle send edit - truncate history and resend
const handleSendEdit = () => {
  if (!editedContent.value.trim()) return;
  
  const messageIndex = messages.value.findIndex(m => m.id === editingMessageId.value);
  if (messageIndex === -1) return;
  
  // Truncate messages from this point
  messages.value = messages.value.slice(0, messageIndex);
  
  // Reset editing state
  const newContent = editedContent.value;
  editingMessageId.value = null;
  editedContent.value = '';
  
  // Send the edited content
  inputText.value = newContent;
  sendMessage();
};

// Handle regenerate - resend the previous user message
const handleRegenerate = (assistantMsg, index) => {
  if (index < 1) return;
  
  const userMessage = messages.value[index - 1];
  if (userMessage.role !== 'user') return;
  
  // Truncate from the user message
  messages.value = messages.value.slice(0, index - 1);
  
  // Resend
  inputText.value = userMessage.content;
  sendMessage();
};

// Generate unique message ID
const generateMessageId = () => {
  return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

// 发送消息
const sendMessage = async () => {
  const text = inputText.value.trim();
  if (!text || isGenerating.value) return;

  // 如果没有当前会话，先创建一个
  if (!currentSessionId.value) {
    try {
      const sessionResult = await aiQuestionAPI.createSession({
        title: text.substring(0, 50) + (text.length > 50 ? '...' : ''),
        provider: aiQuestionProvider.value || chatStore.currentProvider || 'deepseek',
        model: aiQuestionModelName.value || chatStore.currentModel || 'deepseek-chat'
      });
      currentSessionId.value = sessionResult.session?.id || sessionResult.id;
      currentSessionTitle.value = text.substring(0, 50);
      
      // 通知侧边栏刷新历史列表
      window.dispatchEvent(new CustomEvent('ai-question-session-created'));
    } catch (error) {
      console.error('创建会话失败:', error);
    }
  }

  // 添加用户消息
  const userMsgId = generateMessageId();
  messages.value.push({
    id: userMsgId,
    role: 'user',
    content: text
  });

  // 添加AI占位消息
  const aiMessageId = generateMessageId();
  const aiMessageIndex = messages.value.length;
  messages.value.push({
    id: aiMessageId,
    role: 'assistant',
    content: '',
    isLoading: true
  });

  // 记录当前轮次
  const currentRound = questionRounds.value.length;

  inputText.value = '';
  nextTick(() => {
    if (inputRef.value) {
      inputRef.value.style.height = 'auto';
    }
  });
  scrollToBottom();
  isGenerating.value = true;

  try {
    const modelName = aiQuestionModelName.value || chatStore.currentModel || 'deepseek-chat';
    const provider = aiQuestionProvider.value || chatStore.currentProvider || 'deepseek';

    // 构建历史消息（排除当前正在发送的消息和loading消息）
    const history = messages.value
      .slice(0, -2) // 排除刚添加的用户消息和AI占位消息
      .filter(m => !m.isLoading && m.content)
      .map(m => ({
        role: m.role,
        content: m.content
      }));

    const result = await aiQuestionAPI.generateQuestions({
      prompt: text,
      provider: provider,
      model: modelName,
      history: history,
      session_id: currentSessionId.value,
      use_web_search: useWebSearch.value
    });
    
    // 更新AI消息
    messages.value[aiMessageIndex] = {
      id: aiMessageId,
      role: 'assistant',
      content: result.message || '题目生成完成！请查看右侧面板。',
      isLoading: false
    };

    // 添加生成的题目到新轮次
    if (result.questions && result.questions.length > 0) {
      const newQuestions = result.questions.map((q, i) => ({
        ...q,
        id: q.id || `${Date.now()}_${i}`,
        round: currentRound,
        userAnswer: null,
        isChecked: false,
        isCorrect: null
      }));
      
      // 添加新轮次
      questionRounds.value.push({
        id: currentRound,
        title: text.substring(0, 30) + (text.length > 30 ? '...' : ''),
        questions: newQuestions,
        timestamp: new Date()
      });
      
      // 切换到新轮次
      activeRound.value = currentRound;
      
      // Auto show panel when questions are generated
      showQuestionsPanel.value = true;
      // 重置分页到第一页
      currentPage.value = 1;
    }

  } catch (error) {
    console.error('生成题目失败:', error);
    messages.value[aiMessageIndex] = {
      id: aiMessageId,
      role: 'assistant',
      content: `生成失败: ${error.message}。请稍后重试。`,
      isLoading: false
    };
  } finally {
    isGenerating.value = false;
    scrollToBottom();
  }
};

// 处理用户答题
const handleAnswer = ({ questionId, answer }) => {
  // 在当前轮次中查找题目
  const round = questionRounds.value.find(r => r.id === activeRound.value);
  if (round) {
    const question = round.questions.find(q => q.id === questionId);
    if (question) {
      question.userAnswer = answer;
    }
  }
};

// 检查答案
const checkAnswer = (questionId) => {
  const round = questionRounds.value.find(r => r.id === activeRound.value);
  if (!round) return;
  
  const question = round.questions.find(q => q.id === questionId);
  if (!question || question.userAnswer === null) return;

  question.isChecked = true;
  
  switch (question.type) {
    case 'single':
      question.isCorrect = question.userAnswer === question.correctAnswer;
      break;
    case 'multiple':
      const userAnswers = Array.isArray(question.userAnswer) ? question.userAnswer.sort() : [];
      const correctAnswers = Array.isArray(question.correctAnswer) ? question.correctAnswer.sort() : [];
      question.isCorrect = JSON.stringify(userAnswers) === JSON.stringify(correctAnswers);
      break;
    case 'judge':
      question.isCorrect = question.userAnswer === question.correctAnswer;
      break;
    case 'essay':
      question.isCorrect = null;
      break;
  }
};

// 导出题目
const exportQuestions = () => {
  // 导出所有轮次的题目
  const allQuestions = questionRounds.value.flatMap(r => r.questions);
  const data = JSON.stringify(allQuestions, null, 2);
  const blob = new Blob([data], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `questions_${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
};

// 清空题目
const clearQuestions = () => {
  if (confirm('确定要清空所有生成的题目吗？')) {
    questionRounds.value = [];
    activeRound.value = 0;
    currentPage.value = 1;
  }
};

// 开始新会话
const startNewSession = () => {
  currentSessionId.value = null;
  currentSessionTitle.value = '';
  messages.value = [];
  questionRounds.value = [];
  activeRound.value = 0;
  currentPage.value = 1;
};

// 加载会话
const loadSession = async (session, detail = null) => {
  try {
    currentSessionId.value = session.id;
    currentSessionTitle.value = session.title;
    
    // 如果没有传入detail，则获取会话详情
    if (!detail) {
      detail = await aiQuestionAPI.getSession(session.id);
    }
    
    // 加载消息
    if (detail.messages && detail.messages.length > 0) {
      messages.value = detail.messages.map(m => ({
        id: m.id,
        role: m.role,
        content: m.content,
        isLoading: false
      }));
    } else {
      messages.value = [];
    }
    
    // 加载题目 - 将所有题目放入一个轮次
    if (detail.questions && detail.questions.length > 0) {
      const loadedQuestions = detail.questions.map(q => ({
        ...q,
        userAnswer: null,
        isChecked: false,
        isCorrect: null
      }));
      questionRounds.value = [{
        id: 0,
        title: '历史题目',
        questions: loadedQuestions,
        timestamp: new Date()
      }];
      activeRound.value = 0;
      showQuestionsPanel.value = true;
    } else {
      questionRounds.value = [];
      activeRound.value = 0;
    }
    
    currentPage.value = 1;
    scrollToBottom();
  } catch (error) {
    console.error('加载会话失败:', error);
  }
};

// 监听侧边栏事件
const handleLoadSessionEvent = (event) => {
  const { session, detail } = event.detail;
  loadSession(session, detail);
};

const handleNewSessionEvent = () => {
  startNewSession();
};

// 监听消息变化，自动滚动
watch(messages, () => {
  scrollToBottom();
}, { deep: true });

onMounted(() => {
  // 加载 admin 配置的 AI 出题模型
  loadAIQuestionConfig();
  // 监听侧边栏事件
  window.addEventListener('ai-question-load-session', handleLoadSessionEvent);
  window.addEventListener('ai-question-new-session', handleNewSessionEvent);
});

onUnmounted(() => {
  window.removeEventListener('ai-question-load-session', handleLoadSessionEvent);
  window.removeEventListener('ai-question-new-session', handleNewSessionEvent);
});
</script>


<style scoped>
.ai-space-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  background: var(--primary-bg);
}

.main-content {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.chat-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Header */
.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-md) var(--spacing-lg);
  border-bottom: 1px solid var(--border-color);
  background: var(--secondary-bg);
  flex-shrink: 0;
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.collapse-btn {
  background: none;
  border: none;
  padding: 8px;
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.collapse-btn:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.header-title h2 {
  margin: 0;
  font-size: 18px;
  color: var(--text-primary);
}

.header-title .subtitle {
  margin: 2px 0 0 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Admin 配置的模型显示 */
.admin-model-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 13px;
  user-select: none;
}

.admin-model-icon {
  font-size: 16px;
}

.admin-model-name {
  font-weight: 500;
  white-space: nowrap;
}

.admin-model-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--accent-blue, #3b82f6);
  color: white;
  font-weight: 500;
  white-space: nowrap;
}

/* Chat messages wrapper - contains both messages and questions panel */
.chat-messages-wrapper {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}

/* Chat messages */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  min-width: 0;
}

/* ... existing CSS ... */

.welcome-message {
  text-align: center;
  padding: 60px 20px;
  max-width: 900px;
  margin: 0 auto;
}

.welcome-icon-wrapper {
  width: 120px;
  height: 120px;
  margin: 0 auto 24px;
  background: black; /* Force dark background for blend mode */
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 30px rgba(59, 130, 246, 0.3);
  position: relative;
  overflow: hidden;
}

.welcome-icon-wrapper::before {
  content: '';
  position: absolute;
  inset: -2px;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  z-index: 0;
  border-radius: 50%;
  opacity: 0.5;
}

.welcome-icon-inner {
  position: absolute;
  inset: 2px;
  background: #0f172a; /* Dark inner bg */
  border-radius: 50%;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.welcome-img-3d {
  width: 130%; /* Slightly larger to fill */
  height: 130%;
  object-fit: contain;
  mix-blend-mode: screen; /* Removes black background */
  filter: drop-shadow(0 0 10px rgba(59, 130, 246, 0.5));
  animation: pulseLogo 4s ease-in-out infinite;
}

@keyframes pulseLogo {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

.welcome-message h3 {
  margin: 0 0 12px 0;
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--text-primary) 0%, var(--accent-blue) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.welcome-message p {
  color: var(--text-secondary);
  margin-bottom: 32px;
  font-size: 16px;
}

.example-prompts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  width: 100%;
}

.example-prompts button {
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 16px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 12px;
  box-shadow: var(--shadow-elev);
}

.example-prompts button:hover {
  transform: translateY(-2px);
  border-color: var(--accent-blue);
  box-shadow: var(--shadow-elev-high);
  background: var(--hover-bg);
}

/* ... existing CSS ... */

/* Messages list */
.messages-list {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.message {
  display: flex;
  gap: 12px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.message.user {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--secondary-bg);
  border: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.message.user .message-avatar {
  background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
  border: none;
  color: white;
}

.message-bubble {
  max-width: 70%;
  padding: 14px 18px;
  border-radius: 18px;
  background: var(--secondary-bg);
  color: var(--text-primary);
  line-height: 1.5;
  font-size: 15px;
}

.message.user .message-bubble {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: white;
  border-radius: 18px 18px 4px 18px;
}

.message.assistant .message-bubble {
  border-radius: 18px 18px 18px 4px;
  border: 1px solid var(--border-color);
}

.message-content {
  display: flex;
  flex-direction: column;
  max-width: 70%;
}

.message.user .message-content {
  align-items: flex-end;
}

.message-text {
  white-space: pre-wrap;
  word-break: break-word;
}

/* Message Actions */
.message-actions {
  display: flex;
  gap: 6px;
  margin-top: 6px;
  opacity: 0;
  transition: opacity 0.2s;
}

.message:hover .message-actions {
  opacity: 1;
}

.message-actions button {
  background: none;
  border: none;
  padding: 4px;
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.message-actions button:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.message.user .message-actions button {
  color: rgba(255, 255, 255, 0.7);
}

.message.user .message-actions button:hover {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

/* Edit textarea */
.edit-area {
  width: 100%;
}

.edit-textarea {
  width: 100%;
  min-width: 300px;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--primary-bg);
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  outline: none;
}

.edit-textarea:focus {
  border-color: var(--accent-blue);
}

/* Loading indicator */
.loading-indicator {
  display: flex;
  gap: 4px;
  padding: 4px 0;
}

.loading-indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-secondary);
  animation: bounce 1.4s infinite ease-in-out both;
}

.loading-indicator .dot:nth-child(1) { animation-delay: -0.32s; }
.loading-indicator .dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

/* Questions panel - 50% width */
.questions-panel {
  width: 50%;
  max-width: 600px;
  min-width: 400px;
  display: flex;
  flex-direction: column;
  background: var(--primary-bg);
  border-left: 1px solid var(--border-color);
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--secondary-bg);
  flex-shrink: 0;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-primary);
}

.panel-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--primary-bg);
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.action-btn.export:hover {
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.action-btn.clear:hover {
  border-color: #ef4444;
  color: #ef4444;
}

.action-btn.toggle {
  padding: 4px 8px;
}

.action-btn.toggle:hover {
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

/* Round tabs */
.round-tabs {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  overflow-x: auto;
  flex-shrink: 0;
}

.round-tab {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 16px;
  background: var(--primary-bg);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.round-tab:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.round-tab.active {
  background: var(--accent-blue);
  border-color: var(--accent-blue);
  color: white;
}

/* Panel toggle button (when collapsed) */
.panel-toggle-btn {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px 8px;
  background: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-right: none;
  border-radius: 8px 0 0 8px;
  cursor: pointer;
  transition: all 0.2s;
  z-index: 5;
}

.panel-toggle-btn:hover {
  background: var(--hover-bg);
  padding-right: 12px;
}

.panel-toggle-btn svg {
  color: var(--text-secondary);
}

.toggle-badge {
  background: var(--accent-blue);
  color: white;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 20px;
  text-align: center;
}

.questions-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

/* 分页样式 */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 16px;
  border-top: 1px solid var(--border-color);
  background: var(--secondary-bg);
  flex-shrink: 0;
}

.page-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--primary-bg);
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  background: var(--hover-bg);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  padding: 0 12px;
  font-size: 14px;
  color: var(--text-secondary);
  min-width: 60px;
  text-align: center;
}

/* Slide animation */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

/* Input area */
.chat-input-area {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--secondary-bg);
  flex-shrink: 0;
}

/* Input options (search toggle) */
.input-options {
  display: flex;
  gap: 12px;
  max-width: 800px;
  margin: 0 auto 12px;
}

.option-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  background: var(--primary-bg);
  border: 1px solid var(--border-color);
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  transition: all 0.2s;
  user-select: none;
}

.option-toggle:hover {
  border-color: var(--accent-blue);
  color: var(--text-primary);
}

.option-toggle.active {
  background: rgba(0, 122, 255, 0.1);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.option-toggle input[type="checkbox"] {
  display: none;
}

.toggle-icon {
  font-size: 14px;
}

.toggle-label {
  font-weight: 500;
}

.input-wrapper {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  max-width: 800px;
  margin: 0 auto;
  background: var(--primary-bg);
  border: 1px solid var(--border-color);
  border-radius: 24px;
  padding: 8px 8px 8px 16px;
  transition: border-color 0.2s;
}

.input-wrapper:focus-within {
  border-color: var(--accent-blue);
}

.input-wrapper textarea {
  flex: 1;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 15px;
  resize: none;
  outline: none;
  min-height: 24px;
  max-height: 150px;
  line-height: 1.5;
  font-family: inherit;
}

.input-wrapper textarea::placeholder {
  color: var(--text-tertiary);
}

.send-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: var(--accent-blue);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;
}

.send-btn:hover:not(:disabled) {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(0, 122, 255, 0.3);
}

.send-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
  opacity: 0.6;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 1200px) {
  .questions-panel {
    width: 45%;
    min-width: 350px;
  }
}

@media (max-width: 900px) {
  .chat-messages-wrapper {
    flex-direction: column;
  }
  
  .questions-panel {
    width: 100%;
    max-width: none;
    min-width: unset;
    max-height: 50%;
    border-left: none;
    border-top: 1px solid var(--border-color);
  }
  
  .slide-enter-from,
  .slide-leave-to {
    transform: translateY(100%);
  }
}

@media (max-width: 768px) {
  .header-title h2 {
    font-size: 16px;
  }
  
  .header-title .subtitle {
    display: none;
  }
  
  .example-prompts {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .message-bubble {
    max-width: 85%;
  }
}
</style>
