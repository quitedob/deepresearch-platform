<template>
  <div class="home-layout">
    <Sidebar />

    <main class="main-content">
      <div class="chat-interface">
        <ChatContainer
            :current-theme="chatStore.theme"
            @toggle-theme="chatStore.toggleTheme"
            @send-message-from-container="handleSendMessage"
            @edit-and-send="handleEditAndSend"
            @regenerate="handleRegenerate"
        />

        <!-- Input Area with Apple-style Design -->
        <div class="input-area-wrapper">
          <!-- Stop Generation Button -->
          <div v-if="chatStore.isTyping" class="generation-controls">
            <button @click="stopGeneration" class="stop-btn" title="中止生成">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <rect x="6" y="6" width="12" height="12"></rect>
              </svg>
              <span>中止生成</span>
            </button>
          </div>

          <div class="input-container">
            <InputBox
              @send-message="handleSendMessage"
              @send-research="handleSendResearch"
              @send-web-search="handleSendWebSearch"
              @send-deep-think="handleSendDeepThink"
            />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onBeforeUnmount, watch } from 'vue';
import { useChatStore } from '@/store';
import Sidebar from '@/components/Sidebar.vue';
import ChatContainer from '@/components/ChatContainer.vue';
import InputBox from '@/components/InputBox.vue';
import { chatAPI } from '@/api/index';
import { getProviders } from '@/api/model';
import { API_BASE_URL, DEFAULT_MODEL, DEFAULT_PROVIDER, RESEARCH_CONFIG } from '@/utils/config';
import { isWithinDays } from '@/utils/timeFormat';
import { getAuthToken } from '@/utils/token';

/**
 * 处理API错误，返回用户友好的错误信息
 */
const handleAPIError = (error) => {
  if (error.response?.data?.error) {
    const apiError = error.response.data.error;
    return apiError.message || '请求失败';
  }
  if (error.message) {
    return error.message;
  }
  return '未知错误';
};

const chatStore = useChatStore();

// ==================== 请求防抖和草稿保存 ====================

// 请求防抖状态
const isRequestPending = ref(false);
const lastRequestTime = ref(0);
const REQUEST_DEBOUNCE_MS = 500; // 500ms防抖

// 草稿保存
const DRAFT_STORAGE_KEY = 'chat_draft';

/**
 * 保存草稿到本地存储
 */
const saveDraft = (sessionId, content) => {
  if (!content?.trim()) {
    localStorage.removeItem(`${DRAFT_STORAGE_KEY}_${sessionId || 'new'}`);
    return;
  }
  const draft = {
    sessionId: sessionId || 'new',
    content,
    timestamp: Date.now()
  };
  localStorage.setItem(`${DRAFT_STORAGE_KEY}_${sessionId || 'new'}`, JSON.stringify(draft));
};

/**
 * 加载草稿
 */
const loadDraft = (sessionId) => {
  const key = `${DRAFT_STORAGE_KEY}_${sessionId || 'new'}`;
  const saved = localStorage.getItem(key);
  if (saved) {
    try {
      const draft = JSON.parse(saved);
      // 草稿24小时内有效
      if (isWithinDays(draft.timestamp, 1)) {
        return draft.content;
      }
      localStorage.removeItem(key);
    } catch (e) {
      localStorage.removeItem(key);
    }
  }
  return null;
};

/**
 * 清除草稿
 */
const clearDraft = (sessionId) => {
  localStorage.removeItem(`${DRAFT_STORAGE_KEY}_${sessionId || 'new'}`);
};

/**
 * 检查是否可以发送请求（防抖）
 */
const canSendRequest = () => {
  const now = Date.now();
  if (isRequestPending.value) {
    console.warn('[Home] 请求正在进行中，请稍候');
    return false;
  }
  if (now - lastRequestTime.value < REQUEST_DEBOUNCE_MS) {
    console.warn('[Home] 请求过于频繁，请稍候');
    return false;
  }
  return true;
};

/**
 * 标记请求开始
 */
const markRequestStart = () => {
  isRequestPending.value = true;
  lastRequestTime.value = Date.now();
};

/**
 * 标记请求结束
 */
const markRequestEnd = () => {
  isRequestPending.value = false;
};

/**
 * 验证消息是否属于当前会话
 */
const validateMessageSession = (messageId, expectedSessionId) => {
  const message = chatStore.messages.find(m => m.id === messageId);
  if (!message) return false;
  // 如果消息有sessionId属性，验证它
  if (message.sessionId && message.sessionId !== expectedSessionId) {
    console.warn('[Home] 消息不属于当前会话');
    return false;
  }
  return true;
};

// 监听会话切换，清理进行中的请求
watch(() => chatStore.activeSessionId, (newId, oldId) => {
  if (newId !== oldId && oldId) {
    // 会话切换时中止当前请求
    chatStore.abortCurrentRequest();
    markRequestEnd();
  }
});

/**
 * 检查上下文是否超限，如果超限则提示用户新建对话
 * @returns {boolean} true 表示可以继续发送，false 表示需要处理超限
 */
const checkContextLimit = async () => {
  if (!chatStore.activeSessionId) return true;
  
  try {
    const status = await chatAPI.getContextStatus(chatStore.activeSessionId);
    chatStore.setContextStatus(status);
    
    if (status.is_over_limit) {
      const confirmed = confirm(
        `⚠️ 上下文已达上限 (${Math.round(status.usage_percent)}%)\n\n` +
        `当前对话的上下文已达到 ${formatTokens(status.max_tokens)} token 上限。\n` +
        `继续对话可能导致AI响应质量下降。\n\n` +
        `点击"确定"将总结当前对话并开启新对话，\n` +
        `点击"取消"继续在当前对话中发送。`
      );
      
      if (confirmed) {
        await handleSummarizeAndNew();
        return false; // 已处理，不继续发送
      }
    } else if (status.is_near_limit) {
      console.log('⚠️ 上下文接近上限:', status.usage_percent.toFixed(1) + '%');
    }
    
    return true;
  } catch (error) {
    console.error('检查上下文状态失败:', error);
    return true; // 出错时允许继续
  }
};

/**
 * 格式化 token 数量
 */
const formatTokens = (tokens) => {
  if (tokens >= 1000) {
    return (tokens / 1000).toFixed(0) + 'K';
  }
  return tokens.toString();
};

/**
 * 总结当前对话并新建
 */
const handleSummarizeAndNew = async () => {
  if (!chatStore.activeSessionId) return;
  
  try {
    chatStore.setTypingStatus(true);
    chatStore.addMessage({
      role: 'assistant',
      content: '正在总结当前对话并创建新会话...',
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    });
    
    const result = await chatAPI.summarizeAndNewSession(chatStore.activeSessionId);
    
    if (result.success && result.new_session_id) {
      // 刷新历史列表
      await chatStore.fetchSessions();
      // 切换到新会话
      await chatStore.switchSession(result.new_session_id);
      
      alert(`✅ 已创建新对话！\n\n上一对话总结已保存到新对话的系统提示中。`);
    }
  } catch (error) {
    console.error('总结并新建会话失败:', error);
    const errorMsg = handleAPIError(error);
    alert(`操作失败: ${errorMsg}`);
  } finally {
    chatStore.setTypingStatus(false);
  }
};

/**
 * 根据模型名称获取提供商
 */
const getProviderFromModel = (modelName) => {
  if (!modelName) return DEFAULT_PROVIDER;
  
  if (modelName.startsWith('deepseek')) {
    return 'deepseek';
  } else if (modelName.startsWith('glm')) {
    return 'zhipu';
  }
  
  // 默认返回配置的 provider
  return DEFAULT_PROVIDER;
};

/**
 * 处理深度思考请求 - 从 API 获取当前 provider 对应的深度思考模型
 */
const handleSendDeepThink = async (text) => {
  if (!text.trim()) return;

  // 检查上下文限制
  const canContinue = await checkContextLimit();
  if (!canContinue) return;

  // 1) 添加用户消息
  chatStore.addMessage({
    role: 'user',
    content: text,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  });

  // 2) 添加助手占位
  const assistantMessageId = chatStore.addMessage({
    role: 'assistant',
    content: '正在进行深度思考分析...',
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  });

  chatStore.setTypingStatus(true);

  try {
    // 从 API 获取当前 provider 的深度思考模型
    const currentProvider = chatStore.currentProvider || DEFAULT_PROVIDER;
    let deepThinkModel = '';
    let provider = currentProvider;
    
    try {
      const response = await getProviders();
      const providers = response.data?.providers || [];
      const providerConfig = providers.find(p => p.name === currentProvider);
      if (providerConfig) {
        deepThinkModel = providerConfig.deep_think_model;
        provider = providerConfig.name;
      }
    } catch (apiError) {
      console.warn('获取 provider 配置失败，使用默认值:', apiError);
    }
    
    // 如果 API 获取失败，使用空值让后端处理
    if (!deepThinkModel) {
      console.warn('未找到深度思考模型配置');
    }

    // 如果没有活动会话，先创建一个
    let sessionId = chatStore.activeSessionId;
    if (!sessionId) {
      const newSession = await chatAPI.createSession({
        title: text.substring(0, 50) + (text.length > 50 ? '...' : ''),
        llm_provider: provider,
        model_name: deepThinkModel
      });
      sessionId = newSession.id;
      chatStore.activeSessionId = sessionId;
      await chatStore.fetchHistoryList();
    }

    // 使用深度思考模型进行对话
    const response = await chatAPI.chat({
      session_id: sessionId,
      message: text,
      stream: false
    });

    // 更新助手消息
    chatStore.updateMessageContent({ 
      messageId: assistantMessageId, 
      contentChunk: response.content || response.message 
    });

    chatStore.setTypingStatus(false);
    if (!chatStore.activeSessionId) {
      chatStore.fetchHistoryList();
    }
  } catch (error) {
    const msg = handleAPIError(error);
    chatStore.updateMessageContent({ 
      messageId: assistantMessageId, 
      contentChunk: `\n\n[错误] ${msg}` 
    });
    chatStore.setTypingStatus(false);
  }
};

/**
 * 处理联网搜索请求
 */
const handleSendWebSearch = async (text) => {
  if (!text.trim()) return;

  // 检查上下文限制
  const canContinue = await checkContextLimit();
  if (!canContinue) return;

  // 1) 添加用户消息
  chatStore.addMessage({
    role: 'user',
    content: text,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  });

  // 2) 添加助手占位（使用 null 内容触发动画）
  const assistantMessageId = chatStore.addMessage({
    role: 'assistant',
    content: null, // null 会触发 MessageItem 的 thinking 动画
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  });

  chatStore.setTypingStatus(true);

  try {
    // 如果没有活动会话，先创建一个
    let sessionId = chatStore.activeSessionId;
    if (!sessionId) {
      const modelName = chatStore.currentModel || DEFAULT_MODEL;
      const provider = chatStore.currentProvider || getProviderFromModel(modelName);
      const newSession = await chatAPI.createSession({
        title: text.substring(0, 50) + (text.length > 50 ? '...' : ''),
        llm_provider: provider,
        model_name: modelName
      });
      sessionId = newSession.id;
      chatStore.activeSessionId = sessionId;
      await chatStore.fetchHistoryList();
    }

    // 使用联网搜索API（后台执行所有步骤）
    const response = await fetch(`${API_BASE_URL}/chat/chat/web-search`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getAuthToken()}`
      },
      body: JSON.stringify({
        session_id: sessionId,
        message: text,
        stream: false
      })
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.detail || `HTTP ${response.status}`);
    }

    const result = await response.json();

    // 搜索完成，更新为最终答案
    chatStore.updateMessageContent({ 
      messageId: assistantMessageId, 
      contentChunk: result.message.content 
    });

    chatStore.setTypingStatus(false);
    if (!chatStore.activeSessionId) {
      chatStore.fetchHistoryList();
    }
  } catch (error) {
    const msg = handleAPIError(error);
    chatStore.updateMessageContent({ 
      messageId: assistantMessageId, 
      contentChunk: `联网搜索失败: ${msg}` 
    });
    chatStore.setTypingStatus(false);
  }
};

/**
 * ✅ 处理深度研究请求 - 使用 SSE 接收后端推送，不再轮询
 */
const handleSendResearch = async (text) => {
  if (!text.trim()) return;

  // 检查上下文限制
  const canContinue = await checkContextLimit();
  if (!canContinue) return;

  const { researchAPI } = await import('@/api/index');

  chatStore.addMessage({
    role: 'user',
    content: text,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  });

  const assistantMessageId = chatStore.addMessage({
    role: 'assistant',
    content: null,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    metadata: {
      type: 'research',
      evidence: [],
      tools_used: []
    }
  });

  chatStore.setTypingStatus(true);

  try {
    const modelName = chatStore.currentModel || DEFAULT_MODEL;
    const provider = chatStore.currentProvider || getProviderFromModel(modelName);

    const researchResponse = await researchAPI.startResearch({
      query: text,
      research_type: 'comprehensive',
      sources: ['web', 'academic'],
      include_images: false,
      llm_config: {
        provider: provider,
        model: modelName
      }
    });

    if (researchResponse.success) {
      chatStore.setResearchMode(true, researchResponse.session_id);

      // ✅ 使用 SSE 监听后端推送，传递token作为查询参数
      const token = getAuthToken();
      const eventSource = new EventSource(
        `${API_BASE_URL}/research/stream/${researchResponse.session_id}?token=${encodeURIComponent(token)}`
      );

      eventSource.onmessage = async (event) => {
        try {
          const data = JSON.parse(event.data);
          console.log('收到 SSE 事件:', data.type, '完整数据:', data);

          if (data.type === 'connected') {
            console.log('✓ SSE 连接成功，等待后端推送...');
          }
          else if (data.type === 'heartbeat') {
            // 心跳事件，保持连接活跃，不需要处理
            console.log('💓 收到心跳:', new Date(data.timestamp * 1000).toLocaleTimeString());
          }
          else if (data.type === 'status_update') {
            const status = data.status;
            console.log('状态更新:', status);
            
            if (status === 'in_progress') {
              const progress = data.data?.progress || {};
              
              let progressMsg = '🔍 正在进行深度研究...\n\n';
              
              if (progress.tools_used && progress.tools_used.length > 0) {
                progressMsg += `**使用的工具**: ${progress.tools_used.join(', ')}\n`;
              }
              
              if (progress.findings_count > 0) {
                progressMsg += `**发现数量**: ${progress.findings_count}\n`;
              }
              
              progressMsg += '\n*研究进行中，请稍候...*';
              
              chatStore.updateMessageContent({
                messageId: assistantMessageId,
                contentChunk: progressMsg,
                keepThinking: true
              });
            }
            // ✅ 处理 status_update 中的 completed 状态
            else if (status === 'completed') {
              console.log('✓ 通过 status_update 收到完成通知');
              // 不关闭连接，等待 completed 事件推送完整报告
            }
          }
          else if (data.type === 'completed') {
            console.log('✓ 研究完成，收到最终报告');
            eventSource.close();
            
            // ✅ 直接使用后端生成的完整报告文本
            const responseData = data.data;
            const reportText = responseData?.report_text || '研究完成，但报告为空。';
            const metadata = responseData?.metadata || { type: 'research', session_id: responseData?.session_id };
            
            console.log('报告长度:', reportText.length, '字符');
            console.log('证据数量:', metadata.evidence?.length || 0);
            
            chatStore.updateMessageContent({
              messageId: assistantMessageId,
              contentChunk: reportText,
              metadata: metadata  // ✅ 传递完整的 metadata（包含证据链）
            });
            
            chatStore.setTypingStatus(false);
            chatStore.setResearchMode(false, null);
          }
          else if (data.type === 'failed' || data.type === 'error') {
            console.error('✗ 研究失败:', data.error);
            eventSource.close();
            
            chatStore.updateMessageContent({
              messageId: assistantMessageId,
              contentChunk: `深度研究失败: ${data.error || '未知错误'}`
            });
            chatStore.setTypingStatus(false);
            chatStore.setResearchMode(false, null);
          }
        } catch (error) {
          console.error('处理 SSE 事件失败:', error);
        }
      };

      // ✅ 重连计数器
      let reconnectAttempts = 0;
      const MAX_RECONNECT_ATTEMPTS = 3;
      const RECONNECT_DELAY_MS = 2000;

      eventSource.onerror = (error) => {
        console.error('SSE 连接错误:', error);
        
        // 检查是否应该尝试重连
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS && chatStore.isTyping) {
          reconnectAttempts++;
          console.log(`⚠️ SSE 连接断开，尝试重连 (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`);
          
          chatStore.updateMessageContent({
            messageId: assistantMessageId,
            contentChunk: `🔄 连接中断，正在重连 (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`,
            keepThinking: true
          });
          
          // 延迟后尝试重连
          setTimeout(() => {
            if (chatStore.isTyping && eventSource.readyState === EventSource.CLOSED) {
              // 创建新的EventSource连接（带token）
              const reconnectToken = getAuthToken();
              const newEventSource = new EventSource(
                `${API_BASE_URL}/research/stream/${researchResponse.session_id}?token=${encodeURIComponent(reconnectToken)}`
              );
              // 复制事件处理器
              newEventSource.onmessage = eventSource.onmessage;
              newEventSource.onerror = eventSource.onerror;
              // 更新引用
              Object.assign(eventSource, newEventSource);
            }
          }, RECONNECT_DELAY_MS);
        } else {
          // 超过重连次数或已停止，关闭连接
          eventSource.close();
          
          chatStore.updateMessageContent({
            messageId: assistantMessageId,
            contentChunk: '深度研究连接中断，请重试。\n\n💡 提示：您可以稍后在研究历史中查看是否有已完成的结果。'
          });
          chatStore.setTypingStatus(false);
          chatStore.setResearchMode(false, null);
        }
      };

      // ✅ 增加超时时间到 30 分钟，研究可能需要较长时间
      const timeoutId = setTimeout(() => {
        console.warn('⚠️ 研究超时（30分钟）');
        cleanupEventSource();
        if (chatStore.isTyping) {
          chatStore.updateMessageContent({
            messageId: assistantMessageId,
            contentChunk: '深度研究超时（30分钟），请稍后重试'
          });
          chatStore.setTypingStatus(false);
          chatStore.setResearchMode(false, null);
        }
      }, RESEARCH_CONFIG.TIMEOUT_MS);

      // ✅ 统一的EventSource清理函数
      const cleanupEventSource = () => {
        clearTimeout(timeoutId);
        if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
          eventSource.close();
        }
      };

      // ✅ 在连接关闭时清除超时
      const originalClose = eventSource.close.bind(eventSource);
      eventSource.close = () => {
        clearTimeout(timeoutId);
        originalClose();
      };
      
      // ✅ 组件卸载时清理（通过存储引用）
      // 将清理函数存储到store，以便组件卸载时调用
      chatStore._researchCleanup = cleanupEventSource;
    } else {
      throw new Error(researchResponse.error || '启动研究失败');
    }
    
  } catch (error) {
    const msg = handleAPIError(error);
    chatStore.updateMessageContent({ 
      messageId: assistantMessageId, 
      contentChunk: `深度研究失败: ${msg}` 
    });
    chatStore.setTypingStatus(false);
    chatStore.setResearchMode(false, null);
  }
};

// ✅ 组件卸载时清理EventSource
onBeforeUnmount(() => {
  // 清理研究流EventSource
  if (chatStore._researchCleanup) {
    chatStore._researchCleanup();
    chatStore._researchCleanup = null;
  }
  // 中止当前请求
  chatStore.abortCurrentRequest();
});

/**
 * Main function to send a message and handle simple chat response (using Kimi model).
 * 修复：添加请求防抖、草稿保存、自动重试
 */
const handleSendMessage = async (text, retryCount = 0) => {
  if (!text.trim()) return;

  // 防抖检查
  if (!canSendRequest()) {
    // 保存草稿以防丢失
    saveDraft(chatStore.activeSessionId, text);
    return;
  }

  // 检查上下文限制
  const canContinue = await checkContextLimit();
  if (!canContinue) return;

  // 标记请求开始
  markRequestStart();

  const controller = new AbortController();
  chatStore.setCurrentRequestController(controller);

  const startTime = performance.now();
  
  // 记录当前会话ID，用于后续校验
  const currentSessionId = chatStore.activeSessionId;

  chatStore.addMessage({
    role: 'user',
    content: text,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    sessionId: currentSessionId // 标记消息所属会话
  });

  const assistantMessageId = chatStore.addMessage({
    role: 'assistant',
    content: null,
    timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    sessionId: currentSessionId
  });

  chatStore.setTypingStatus(true);

  try {
    // 获取认证token
    const token = getAuthToken();
    if (!token) {
      throw new Error('请先登录');
    }

    // 如果没有活动会话，先创建一个
    let sessionId = chatStore.activeSessionId;
    if (!sessionId) {
      const modelName = chatStore.currentModel || DEFAULT_MODEL;
      const provider = chatStore.currentProvider || getProviderFromModel(modelName);
      const newSession = await chatAPI.createSession({
        title: text.substring(0, 50) + (text.length > 50 ? '...' : ''),
        llm_provider: provider,
        model_name: modelName
      });
      sessionId = newSession.id;
      chatStore.activeSessionId = sessionId;
      // 刷新历史列表
      await chatStore.fetchHistoryList();
    }

    // 使用后端API进行对话
    const response = await fetch(`${API_BASE_URL}/chat/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        session_id: sessionId,
        message: text,
        stream: true
      }),
      signal: controller.signal
    });

    if (!response.ok) {
      // 尝试解析错误响应
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
      try {
        const errorData = await response.json();
        if (errorData.error?.message) {
          errorMessage = errorData.error.message;
        } else if (errorData.message) {
          errorMessage = errorData.message;
        }
        
        // 特殊处理配额超限错误
        if (errorData.error?.code === 'ERR_CHAT_QUOTA_EXCEEDED') {
          const extra = errorData.error.extra || {};
          errorMessage = `聊天配额已用完 (${extra.remaining || 0}/${extra.limit || 0})，请升级会员或等待配额重置`;
          
          // 移除刚添加的消息
          chatStore.replaceMessagesFromIndex(chatStore.messages.length - 2);
          chatStore.setTypingStatus(false);
          chatStore.setCurrentRequestController(null);
          markRequestEnd();
          
          // 显示友好提示
          alert(errorMessage);
          return;
        }
      } catch (e) {
        // 解析失败，使用默认错误信息
      }
      throw new Error(errorMessage);
    }

    // 校验会话是否已切换
    if (chatStore.activeSessionId !== sessionId) {
      console.warn('[Home] 会话已切换，忽略响应');
      return;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let streamEnded = false;
    let buffer = ''; // 用于处理跨chunk的不完整行
    let currentEventType = 'message'; // 当前SSE事件类型

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          streamEnded = true;
          break;
        }

        // 再次校验会话
        if (chatStore.activeSessionId !== sessionId) {
          console.warn('[Home] 会话已切换，停止处理流');
          break;
        }

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        
        // 保留最后一个可能不完整的行
        buffer = lines.pop() || '';

        for (const line of lines) {
          const trimmedLine = line.trim();
          
          // 空行表示事件结束，重置事件类型
          if (trimmedLine === '') {
            currentEventType = 'message';
            continue;
          }
          
          // 解析事件类型行
          if (trimmedLine.startsWith('event:')) {
            currentEventType = trimmedLine.slice(6).trim();
            continue;
          }
          
          // 解析数据行
          if (trimmedLine.startsWith('data:')) {
            const dataStr = trimmedLine.slice(5).trim();
            if (!dataStr) continue;
            
            // 检查是否是结束标记
            if (dataStr === '[DONE]') {
              streamEnded = true;
              continue;
            }
            
            try {
              const data = JSON.parse(dataStr);
              
              // 根据事件类型处理
              if (currentEventType === 'error') {
                // 后端发送的 event: error (兼容旧格式)
                const errorMsg = data.error || data.message || '未知错误';
                throw new Error(errorMsg);
              }
              
              // 处理 event: message 或默认事件
              if (data.type === 'content' && data.content) {
                // ✅ 实时追加内容到消息气泡
                chatStore.appendMessageContent({
                  messageId: assistantMessageId,
                  contentChunk: data.content
                });
              } else if (data.type === 'start') {
                // 流开始，可以用于UI状态更新
                console.debug('[SSE] Stream started');
              } else if (data.type === 'end') {
                streamEnded = true;
                continue;
              } else if (data.type === 'error' || data.error) {
                // 统一的错误格式: type: error 或 error 字段
                const errorMsg = data.error || data.message || '未知错误';
                throw new Error(errorMsg);
              }
              
              // 检查是否有结束标记
              if (data.done || data.finished) {
                streamEnded = true;
              }
            } catch (e) {
              // 如果是我们主动抛出的错误，继续抛出
              if (e.message && !e.message.includes('JSON')) {
                throw e;
              }
              // 记录解析错误但不中断流
              console.warn('[SSE] Parse error:', e.message, 'Raw data:', dataStr);
            }
          }
        }
      }
    } finally {
      // 确保流读取器被关闭
      try {
        reader.releaseLock();
      } catch (e) {
        // 忽略
      }
    }

    // 发送成功，清除草稿
    clearDraft(sessionId);

    // 完成处理 - 确保总是重置状态
    const endTime = performance.now();
    const duration = ((endTime - startTime) / 1000).toFixed(1);
    chatStore.setMessageDuration(assistantMessageId, duration);
    chatStore.setTypingStatus(false);
    chatStore.setCurrentRequestController(null);
    markRequestEnd();
    
    console.log('[Home] 流式响应完成, streamEnded:', streamEnded);

    // 刷新历史记录
    if (!chatStore.activeSessionId) {
      chatStore.fetchHistoryList();
    }

  } catch (error) {
    markRequestEnd();
    
    if (error.name === 'AbortError') {
      // 请求被中止
      chatStore.setTypingStatus(false);
      chatStore.setCurrentRequestController(null);
      return;
    }

    // 网络错误自动重试（最多3次）
    const MAX_RETRIES = 3;
    if (isNetworkError(error) && retryCount < MAX_RETRIES) {
      console.log(`[Home] 网络错误，${retryCount + 1}/${MAX_RETRIES} 次重试...`);
      chatStore.updateMessageContent({
        messageId: assistantMessageId,
        contentChunk: `网络错误，正在重试 (${retryCount + 1}/${MAX_RETRIES})...`
      });
      
      // 指数退避重试
      await new Promise(resolve => setTimeout(resolve, 1000 * (retryCount + 1)));
      
      // 移除失败的消息
      chatStore.replaceMessagesFromIndex(chatStore.messages.length - 2);
      chatStore.setTypingStatus(false);
      chatStore.setCurrentRequestController(null);
      
      // 重试
      return handleSendMessage(text, retryCount + 1);
    }

    // 保存草稿以防丢失
    saveDraft(chatStore.activeSessionId, text);

    const errorMessage = handleAPIError(error);
    chatStore.updateMessageContent({
      messageId: assistantMessageId,
      contentChunk: `**错误:** ${errorMessage}\n\n💡 您的消息已保存为草稿，刷新页面后可恢复。`
    });
    const endTime = performance.now();
    const duration = ((endTime - startTime) / 1000).toFixed(1);
    chatStore.setMessageDuration(assistantMessageId, duration);
    chatStore.setTypingStatus(false);
    chatStore.setCurrentRequestController(null);
  }
};

/**
 * 检查是否是网络错误
 */
const isNetworkError = (error) => {
  return error.message?.includes('network') ||
         error.message?.includes('Network') ||
         error.message?.includes('fetch') ||
         error.name === 'TypeError';
};

/**
 * Handles the 'edit-and-send' event from a MessageItem.
 * 修复：添加sessionId校验，防止会话切换导致消息混乱；添加防抖
 */
const handleEditAndSend = ({ messageId, newContent }) => {
  // 防抖检查
  if (!canSendRequest()) {
    console.warn('[Home] 请求过于频繁');
    return;
  }

  const messageIndex = chatStore.messages.findIndex(m => m.id === messageId);
  if (messageIndex === -1) {
    console.warn('[Home] 消息不存在');
    return;
  }

  // 记录当前会话ID，用于后续校验
  const currentSessionId = chatStore.activeSessionId;
  
  // 验证消息是否属于当前会话
  const message = chatStore.messages[messageIndex];
  if (message.sessionId && message.sessionId !== currentSessionId) {
    console.warn('[Home] 消息不属于当前会话，取消编辑操作');
    return;
  }
  
  // Abort any ongoing requests
  chatStore.abortCurrentRequest();
  markRequestEnd(); // 重置请求状态
  
  // 再次校验会话是否已切换
  if (chatStore.activeSessionId !== currentSessionId) {
    console.warn('[Home] 会话已切换，取消编辑操作');
    return;
  }
  
  // Truncate the history from the edited message onwards
  chatStore.replaceMessagesFromIndex(messageIndex);
  // Send the edited content as a new message
  handleSendMessage(newContent);
};

/**
 * Handles the 'regenerate' event from a MessageItem.
 * 修复：添加sessionId校验，防止会话切换导致消息混乱；添加防抖
 */
const handleRegenerate = (assistantMessage) => {
  // 防抖检查
  if (!canSendRequest()) {
    console.warn('[Home] 请求过于频繁');
    return;
  }

  const messageIndex = chatStore.messages.findIndex(m => m.id === assistantMessage.id);
  // Ensure there is a user message before the assistant message
  if (messageIndex < 1) {
    console.warn('[Home] 无法找到对应的用户消息');
    return;
  }

  const userMessage = chatStore.messages[messageIndex - 1];
  if (userMessage.role !== 'user') {
    console.warn('[Home] 前一条消息不是用户消息');
    return;
  }

  // 记录当前会话ID
  const currentSessionId = chatStore.activeSessionId;
  
  // 验证消息是否属于当前会话
  if (assistantMessage.sessionId && assistantMessage.sessionId !== currentSessionId) {
    console.warn('[Home] 消息不属于当前会话，取消重新生成操作');
    return;
  }
  
  // Abort any ongoing requests
  chatStore.abortCurrentRequest();
  markRequestEnd(); // 重置请求状态
  
  // 再次校验会话是否已切换
  if (chatStore.activeSessionId !== currentSessionId) {
    console.warn('[Home] 会话已切换，取消重新生成操作');
    return;
  }
  
  // Truncate the history, removing the previous user message and the assistant response
  chatStore.replaceMessagesFromIndex(messageIndex - 1);
  // Resend the content of that user message
  handleSendMessage(userMessage.content);
};

/**
 * Stops the current AI response generation.
 */
const stopGeneration = () => {
  chatStore.abortCurrentRequest();
};
</script>

<style scoped>
.home-layout {
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
  overflow-y: hidden;
  background: var(--primary-bg);
  position: relative;
}

.chat-interface {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  position: relative;
}

.input-area-wrapper {
  padding: var(--spacing-lg);
  box-sizing: border-box;
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.generation-controls {
  display: flex;
  justify-content: center;
  animation: slideUp 0.3s ease;
}

.stop-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border: none;
  background: var(--accent-red);
  color: white;
  border-radius: var(--radius-large);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(255, 59, 48, 0.3);
  transition: all 0.2s ease;
}

.stop-btn:hover {
  background: #ff2d55;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(255, 59, 48, 0.4);
}

.stop-btn:active {
  transform: translateY(0);
}

.input-container {
  width: 100%;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .home-layout {
    height: 100dvh;
  }

  .input-area-wrapper {
    padding: var(--spacing-md);
  }
}

@media (max-width: 480px) {
  .input-area-wrapper {
    padding: var(--spacing-sm);
  }
}
</style>
