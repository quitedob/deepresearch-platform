// src/store/index.js
import { defineStore } from 'pinia'
import { chatAPI, healthAPI } from '@/api/index'
import logger from '@/utils/logger'

/**
 * 错误处理工具函数
 * 解析后端返回的统一错误格式
 */
function parseAPIError(error) {
  // 如果是后端返回的统一错误格式
  if (error.response?.data?.error) {
    const apiError = error.response.data.error
    return {
      code: apiError.code || 'ERR_UNKNOWN',
      message: apiError.message || '未知错误',
      details: apiError.details || '',
      field: apiError.field || '',
      extra: apiError.extra || {},
      httpStatus: error.response.status,
      // 添加建议操作
      suggestion: getSuggestionForError(apiError.code)
    }
  }

  // 网络错误
  if (error.code === 'ERR_NETWORK') {
    return {
      code: 'ERR_NETWORK',
      message: '网络连接失败，请检查网络设�?,
      httpStatus: 0,
      suggestion: 'check_network'
    }
  }

  // 超时错误
  if (error.code === 'ECONNABORTED') {
    return {
      code: 'ERR_TIMEOUT',
      message: '请求超时，请稍后重试',
      httpStatus: 0,
      suggestion: 'retry'
    }
  }

  // 其他错误
  return {
    code: 'ERR_UNKNOWN',
    message: error.message || '未知错误',
    httpStatus: error.response?.status || 0,
    suggestion: null
  }
}

/**
 * 根据错误码获取建议操�? */
function getSuggestionForError(errorCode) {
  const suggestions = {
    'ERR_UNAUTHORIZED': 'relogin',
    'ERR_TOKEN_EXPIRED': 'relogin',
    'ERR_SESSION_NOT_FOUND': 'refresh_sessions',
    'ERR_CHAT_QUOTA_EXCEEDED': 'upgrade_membership',
    'ERR_RESEARCH_QUOTA_EXCEEDED': 'upgrade_membership',
    'ERR_CONTEXT_OVERFLOW': 'create_new_session',
    'ERR_MODEL_NOT_SUPPORTED': 'switch_model',
    'ERR_LLM_UNAVAILABLE': 'retry_later',
    'ERR_RATE_LIMIT_EXCEEDED': 'wait_and_retry',
  }
  return suggestions[errorCode] || null
}

/**
 * 根据错误码获取用户友好的提示信息
 */
function getErrorUserMessage(errorCode, defaultMessage) {
  const errorMessages = {
    'ERR_UNAUTHORIZED': '请先登录后再操作',
    'ERR_FORBIDDEN': '您没有权限执行此操作',
    'ERR_TOKEN_EXPIRED': '登录已过期，请重新登�?,
    'ERR_SESSION_NOT_FOUND': '会话不存在或已被删除',
    'ERR_SESSION_FORBIDDEN': '您无权访问此会话',
    'ERR_CHAT_QUOTA_EXCEEDED': '聊天次数已用完，请升级会员或等待配额重置',
    'ERR_RESEARCH_QUOTA_EXCEEDED': '深度研究次数已用完，请升级会员或等待配额重置',
    'ERR_CONTEXT_OVERFLOW': '对话上下文过长，建议创建新会话继续对�?,
    'ERR_MESSAGE_TOO_LONG': '消息内容过长，请精简后重�?,
    'ERR_MODEL_NOT_SUPPORTED': '当前模型暂不可用，请切换其他模型',
    'ERR_LLM_UNAVAILABLE': 'AI服务暂时不可用，请稍后重�?,
    'ERR_LLM_TIMEOUT': 'AI响应超时，请稍后重试',
    'ERR_LLM_ERROR': 'AI服务调用失败，请稍后重试',
    'ERR_RATE_LIMIT_EXCEEDED': '请求过于频繁，请稍后重试',
    'ERR_NETWORK': '网络连接失败，请检查网络设�?,
    'ERR_TIMEOUT': '请求超时，请稍后重试',
    'ERR_INTERNAL_ERROR': '服务器内部错误，请稍后重�?,
    'ERR_INVALID_REQUEST': '请求参数无效，请检查输�?,
    'ERR_MISSING_PARAMETER': '缺少必要参数',
  }
  return errorMessages[errorCode] || defaultMessage
}

export const useChatStore = defineStore('chat', {
  state: () => ({
    // ========== 模型配置 ==========
    // 初始化时�?localStorage 加载，确保刷新后保持选择
    currentModel: localStorage.getItem('chat-model') || 'deepseek-chat',
    currentProvider: localStorage.getItem('chat-provider') || 'deepseek',
    availableModels: [],
    availableProviders: {},

    // ========== 会话状�?==========
    activeSessionId: null,        // 当前活动会话ID
    currentSession: null,         // 当前会话详情（包含provider、model等）
    sessions: [],                 // 会话列表（原historyList�?
    // ========== 消息状�?==========
    messages: [],                 // 当前会话的消息列�?
    // ========== UI状�?==========
    isTyping: false,              // AI是否正在输入
    isLoading: false,             // 是否正在加载
    isSettingsModalOpen: false,   // 设置模态框是否打开

    // ========== 请求控制 ==========
    currentRequestController: null,

    // ========== 研究模式 ==========
    isResearchMode: false,
    researchSessionId: null,

    // ========== 系统状�?==========
    systemStatus: null,
    connectionStatus: 'disconnected', // disconnected, connecting, connected, error

    // ========== 用户设置 ==========
    personalizationSettings: {
      userNickname: '',
      userProfession: '',
      chatGptCharacteristics: '',
      additionalInfo: '',
      enableForNewChats: true,
    },
    memorySettings: {
      memoryEnabled: true,
      customSystemPrompt: '',
      maxContextTokens: 128000,
    },

    // ========== 上下文状�?==========
    contextStatus: null,

    // ========== 错误状�?==========
    lastError: null,              // 最后一次错�?
    // ========== 输入状态（集中管理�?==========
    inputText: '',                // 当前输入框内�?    inputDrafts: {},              // 各会话的草稿 { sessionId: content }

    // ========== 主题设置 ==========
    theme: 'dark',                // 'dark' | 'light'
  }),

  getters: {
    /**
     * 是否有活动会�?     */
    hasActiveSession: (state) => !!state.activeSessionId,

    /**
     * 获取消息数量
     */
    messageCount: (state) => state.messages.length,

    /**
     * 获取会话列表（按更新时间排序�?     */
    sortedSessions: (state) => {
      return [...(state.sessions || [])].sort((a, b) => {
        const dateA = new Date(a.updated_at || a.created_at)
        const dateB = new Date(b.updated_at || b.created_at)
        return dateB - dateA
      })
    },

    /**
     * 历史列表（兼容旧API�?     * 确保返回数组，防�?null/undefined
     */
    historyList: (state) => state.sessions || [],

    /**
     * 上下文是否接近限�?     */
    isContextNearLimit: (state) => state.contextStatus?.is_near_limit || false,

    /**
     * 上下文是否超过限�?     */
    isContextOverLimit: (state) => state.contextStatus?.is_over_limit || false,

    /**
     * 上下文使用百分比
     */
    contextUsagePercent: (state) => state.contextStatus?.usage_percent || 0,

    /**
     * 是否已连�?     */
    isConnected: (state) => state.connectionStatus === 'connected',
  },

  actions: {
    // ==================== 错误处理 ====================

    /**
     * 处理API错误
     * @param {Error} error - 错误对象
     * @param {string} context - 错误上下文描�?     * @returns {Object} 解析后的错误信息
     */
    handleError(error, context = '') {
      const parsedError = parseAPIError(error)
      parsedError.userMessage = getErrorUserMessage(parsedError.code, parsedError.message)
      parsedError.context = context

      this.lastError = parsedError
      logger.error(`[${context}] 错误:`, parsedError)

      return parsedError
    },

    /**
     * 清除错误状�?     */
    clearError() {
      this.lastError = null
    },

    // ==================== 模型管理 ====================

    /**
     * 设置当前模型
     * @param {string} modelName - 模型名称
     */
    setModel(modelName) {
      this.currentModel = modelName

      // 从可用模型列表中查找提供�?      const model = this.availableModels.find(m => m.name === modelName || m.id === modelName)
      if (model) {
        this.currentProvider = model.provider
      } else {
        // 根据模型名称推断提供�?        if (modelName.startsWith('deepseek')) {
          this.currentProvider = 'deepseek'
        } else if (modelName.startsWith('glm')) {
          this.currentProvider = 'zhipu'
        } else if (modelName.includes(':')) {
          this.currentProvider = 'ollama'
        }
      }

      // 持久化到 localStorage
      localStorage.setItem('chat-model', this.currentModel)
      localStorage.setItem('chat-provider', this.currentProvider)

      logger.log('[Store] 设置模型:', modelName, '提供�?', this.currentProvider, '(已持久化)')
    },

    /**
     * 设置当前提供�?     * @param {string} providerName - 提供商名�?     */
    setProvider(providerName) {
      this.currentProvider = providerName
      localStorage.setItem('chat-provider', this.currentProvider)
      logger.log('[Store] 设置提供�?', providerName, '(已持久化)')
    },

    /**
     * 初始化模型配置（�?localStorage 加载�?     * 应在应用启动时调�?     */
    initModel() {
      const savedModel = localStorage.getItem('chat-model')
      const savedProvider = localStorage.getItem('chat-provider')

      if (savedModel) {
        this.currentModel = savedModel
      }
      if (savedProvider) {
        this.currentProvider = savedProvider
      }

      logger.log('[Store] 初始化模型配�?', this.currentModel, '提供�?', this.currentProvider)
    },

    /**
     * 获取可用模型列表
     */
    async fetchModels() {
      try {
        const response = await chatAPI.getModels()
        const models = response.models || response || []
        this.availableModels = models
        return models
      } catch (error) {
        this.handleError(error, '获取模型列表')
        return []
      }
    },

    // ==================== 会话管理 ====================

    /**
     * 获取会话列表
     */
    async fetchSessions() {
      this.isLoading = true
      try {
        const response = await chatAPI.getSessions()
        const sessions = response.sessions || response || []

        this.sessions = sessions.map(session => ({
          id: session.id,
          title: session.title || '新对�?,
          provider: session.provider,
          model: session.model,
          messageCount: session.message_count || 0,
          lastMessage: session.last_message || '',
          updatedAt: session.updated_at ? new Date(session.updated_at) : new Date(),
          createdAt: session.created_at ? new Date(session.created_at) : new Date(),
          pinned: session.is_pinned || false
        }))

        logger.log('[Store] 会话列表加载成功:', this.sessions.length, '个会�?)
        return this.sessions
      } catch (error) {
        this.handleError(error, '获取会话列表')
        this.sessions = []
        return []
      } finally {
        this.isLoading = false
      }
    },

    // 兼容旧API名称
    async fetchHistoryList() {
      return this.fetchSessions()
    },

    /**
     * 切换到指定会�?     * @param {string} sessionId - 会话ID
     */
    async switchSession(sessionId) {
      if (this.activeSessionId === sessionId) {
        return // 已经是当前会�?      }

      // 中止当前请求
      this.abortCurrentRequest()
      this.isLoading = true
      this.clearError()

      // 保存旧消息以便恢�?      const previousMessages = [...this.messages]
      const previousSessionId = this.activeSessionId
      const previousSession = this.currentSession

      // 先清空当前状态，防止老数据残�?      this.messages = []
      this.contextStatus = null

      try {
        // 加载会话消息
        const response = await chatAPI.getMessages(sessionId)
        const messages = response.messages || response || []

        // 更新消息列表
        this.messages = messages.map(m => ({
          ...m,
          id: m.id || (Math.random() + Date.now()),
          role: m.role.toLowerCase(),
        }))

        // 更新当前会话ID
        this.activeSessionId = sessionId

        // 消息加载完成后刷新上下文状�?        this.fetchContextStatus(sessionId)

        // 同步会话的模型和提供�?        const session = this.sessions.find(s => s.id === sessionId)
        if (session) {
          this.currentSession = session
          this.currentModel = session.model
          this.currentProvider = session.provider
        } else {
          // 如果会话不在列表中，尝试从API获取详情并添加到列表
          try {
            const sessionDetail = await chatAPI.getSession(sessionId)
            const sessionData = sessionDetail.data || sessionDetail
            this.currentSession = sessionData
            this.currentModel = sessionData.model
            this.currentProvider = sessionData.provider

            // 添加到会话列表以便下次使�?            this.sessions.unshift({
              id: sessionData.id,
              title: sessionData.title || '新对�?,
              provider: sessionData.provider,
              model: sessionData.model,
              messageCount: sessionData.message_count || 0,
              lastMessage: sessionData.last_message || '',
              updatedAt: sessionData.updated_at ? new Date(sessionData.updated_at) : new Date(),
              createdAt: sessionData.created_at ? new Date(sessionData.created_at) : new Date(),
              pinned: false
            })
          } catch (e) {
            logger.error('[Store] 获取会话详情失败:', e)
            // 重新抛出错误让调用者处�?            throw e
          }
        }

        logger.log('[Store] 切换到会�?', sessionId, '消息�?', this.messages.length)
      } catch (error) {
        const parsedError = this.handleError(error, '加载会话')

        // 根据错误类型处理
        if (parsedError.code === 'ERR_SESSION_NOT_FOUND') {
          // 会话不存在，从列表中移除并清空当前状�?          this.sessions = this.sessions.filter(s => s.id !== sessionId)
          this.activeSessionId = null
          this.currentSession = null
        } else {
          // 其他错误，尝试恢复之前的状�?          this.messages = previousMessages
          this.activeSessionId = previousSessionId
          this.currentSession = previousSession
        }

        throw error
      } finally {
        this.isLoading = false
      }
    },

    // 兼容旧API名称
    async loadHistory(sessionId) {
      return this.switchSession(sessionId)
    },

    /**
     * 创建新会�?     * @param {Object} options - 会话选项
     */
    async createSession(options = {}) {
      // 先清空当前状�?      this.abortCurrentRequest()
      this.messages = []
      this.contextStatus = null
      this.isTyping = false
      this.clearError()

      const sessionData = {
        title: options.title || '新对�?,
        llm_provider: options.provider || this.currentProvider,
        model_name: options.model || this.currentModel,
        system_prompt: options.systemPrompt || '',
      }

      try {
        const response = await chatAPI.createSession(sessionData)
        // 处理新的响应格式
        const data = response.data || response
        const newSession = {
          id: data.id,
          title: data.title || sessionData.title,
          provider: data.provider || sessionData.llm_provider,
          model: data.model || sessionData.model_name,
          messageCount: 0,
          updatedAt: new Date(),
          createdAt: new Date(),
        }

        // 添加到会话列�?        this.sessions.unshift(newSession)

        // 同步设置所有相关状�?        this.activeSessionId = newSession.id
        this.currentSession = newSession
        this.currentModel = newSession.model
        this.currentProvider = newSession.provider

        logger.log('[Store] 创建新会�?', newSession.id)
        return newSession
      } catch (error) {
        this.handleError(error, '创建会话')
        // 创建失败时恢复之前的会话状�?        if (this.sessions.length > 0 && !this.activeSessionId) {
          // 可以选择切换到第一个会话或保持空状�?        }
        throw error
      }
    },

    /**
     * 删除会话
     * @param {string} sessionId - 会话ID
     */
    async deleteSession(sessionId) {
      try {
        await chatAPI.deleteSession(sessionId)

        // 从列表中移除
        this.sessions = this.sessions.filter(s => s.id !== sessionId)

        // 如果删除的是当前会话，清空所有相关状�?        if (this.activeSessionId === sessionId) {
          this.activeSessionId = null
          this.currentSession = null
          this.messages = []
          this.contextStatus = null
          this.isTyping = false
          this.clearError()
        }

        logger.log('[Store] 删除会话:', sessionId)
        return { success: true, sessionId }
      } catch (error) {
        const parsedError = this.handleError(error, '删除会话')

        // 如果会话已不存在，也从本地列表移�?        if (parsedError.code === 'ERR_SESSION_NOT_FOUND') {
          this.sessions = this.sessions.filter(s => s.id !== sessionId)
          if (this.activeSessionId === sessionId) {
            this.activeSessionId = null
            this.currentSession = null
            this.messages = []
          }
        }

        throw error
      }
    },

    /**
     * 删除所有会�?     */
    async deleteAllSessions() {
      const sessionIds = this.sessions.map(s => s.id)

      if (sessionIds.length === 0) {
        this.sessions = []
        this.activeSessionId = null
        this.currentSession = null
        this.messages = []
        this.contextStatus = null
        this.isTyping = false
        this.clearError()
        return { success: true, failedCount: 0 }
      }

      try {
        // 使用批量删除API，一次性删除所有会�?        await chatAPI.batchDeleteSessions(sessionIds)
      } catch (error) {
        logger.warn('[Store] 批量删除失败，回退到逐个删除:', error)
        // 回退到逐个删除
        const errors = []
        for (const sessionId of sessionIds) {
          try {
            await chatAPI.deleteSession(sessionId)
          } catch (err) {
            errors.push({ sessionId, error: err })
          }
        }
        if (errors.length > 0) {
          logger.warn('[Store] 部分会话删除失败:', errors)
        }
      }

      // 清空本地状�?      this.sessions = []
      this.activeSessionId = null
      this.currentSession = null
      this.messages = []
      this.contextStatus = null
      this.isTyping = false
      this.clearError()

      logger.log('[Store] 删除所有会话完�?)
      return { success: true, failedCount: 0 }
    },

    // 兼容旧API名称
    async deleteAllHistories() {
      return this.deleteAllSessions()
    },

    /**
     * 清空所有历史记录（包括本地和服务器�?     * 修复：DataManagementSettings.vue 调用此方�?     */
    async clearAllHistory() {
      try {
        // 删除服务器上的所有会�?        await this.deleteAllSessions()

        // 清除本地存储
        localStorage.removeItem('chat_history')
        localStorage.removeItem('chat_sessions')

        logger.log('[Store] 所有历史记录已清空')
        return { success: true }
      } catch (error) {
        this.handleError(error, '清空历史记录')
        throw error
      }
    },

    // ==================== 消息管理 ====================

    /**
     * 添加消息
     * @param {Object} message - 消息对象
     */
    addMessage(message) {
      const messageWithId = {
        ...message,
        id: message.id || Date.now() + Math.random(),
        duration: null,
        timestamp: message.timestamp || new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      }
      this.messages.push(messageWithId)
      return messageWithId.id
    },

    /**
     * 更新消息内容
     * 修复：使�?Vue 响应式方式更新，确保 UI 实时刷新
     */
    updateMessageContent({ messageId, contentChunk, keepThinking = false, metadata = null }) {
      const messageIndex = this.messages.findIndex(m => m.id === messageId)
      if (messageIndex !== -1) {
        const message = this.messages[messageIndex]
        const updates = {}

        if (!keepThinking) {
          updates.content = contentChunk
        }

        if (metadata) {
          updates.metadata = { ...message.metadata, ...metadata }
        }

        // 使用数组替换来触发响应式更新
        this.messages[messageIndex] = { ...message, ...updates }
      }
    },

    /**
     * 追加消息内容（用于流式响应）
     * 修复：使�?Vue 响应式方式更新，确保 UI 实时刷新
     */
    appendMessageContent({ messageId, contentChunk }) {
      const messageIndex = this.messages.findIndex(m => m.id === messageId)
      if (messageIndex !== -1) {
        const message = this.messages[messageIndex]
        if (message.content === null) {
          // 使用数组替换来触发响应式更新
          this.messages[messageIndex] = { ...message, content: contentChunk }
        } else {
          // 使用数组替换来触发响应式更新
          this.messages[messageIndex] = { ...message, content: message.content + contentChunk }
        }
      }
    },

    /**
     * 设置消息持续时间
     */
    setMessageDuration(messageId, duration) {
      const message = this.messages.find(m => m.id === messageId)
      if (message) {
        message.duration = duration
      }
    },

    /**
     * 从指定索引替换消�?     */
    replaceMessagesFromIndex(startIndex, newMessages = []) {
      this.messages.splice(startIndex)
      if (Array.isArray(newMessages) && newMessages.length > 0) {
        this.messages.push(...newMessages)
      }
    },

    // ==================== 请求控制 ====================

    /**
     * 设置请求控制�?     */
    setCurrentRequestController(controller) {
      this.currentRequestController = controller
    },

    /**
     * 中止当前请求
     */
    abortCurrentRequest() {
      if (this.currentRequestController) {
        this.currentRequestController.abort()
        this.currentRequestController = null
      }
      this.setTypingStatus(false)
    },

    // ==================== UI状�?====================

    /**
     * 设置输入状�?     */
    setTypingStatus(status) {
      this.isTyping = status
    },

    /**
     * 清空当前聊天
     */
    clearChat() {
      this.abortCurrentRequest()
      this.messages = []
      this.isTyping = false
      this.activeSessionId = null
      this.currentSession = null
      this.contextStatus = null
      this.clearError()
    },

    /**
     * 开始新对话
     */
    startNewChat() {
      this.clearChat()
      logger.log('[Store] 开始新对话')
    },

    /**
     * 打开设置模态框
     */
    openSettingsModal() {
      this.isSettingsModalOpen = true
    },

    /**
     * 关闭设置模态框
     */
    closeSettingsModal() {
      this.isSettingsModalOpen = false
    },

    // ==================== 研究模式 ====================

    /**
     * 设置研究模式
     */
    setResearchMode(isResearch, sessionId = null) {
      this.isResearchMode = isResearch
      this.researchSessionId = sessionId
      logger.log('[Store] 研究模式:', isResearch, '会话ID:', sessionId)
    },

    // ==================== 系统状�?====================

    /**
     * 设置连接状�?     */
    setConnectionStatus(status) {
      this.connectionStatus = status
    },

    /**
     * 设置系统状�?     */
    setSystemStatus(status) {
      this.systemStatus = status
    },

    /**
     * 检查连�?     */
    async checkConnection() {
      this.setConnectionStatus('connecting')

      try {
        const healthResult = await healthAPI.check()
        this.setSystemStatus(healthResult)
        this.setConnectionStatus('connected')
        return true
      } catch (error) {
        logger.error('[Store] 连接检查失�?', error)
        this.setConnectionStatus('error')
        return false
      }
    },

    // ==================== 用户设置 ====================

    /**
     * 保存个性化设置
     */
    savePersonalizationSettings(settings) {
      this.personalizationSettings = { ...this.personalizationSettings, ...settings }
      logger.log('[Store] 个性化设置已保�?)
    },

    /**
     * 设置记忆设置
     */
    setMemorySettings(settings) {
      this.memorySettings = { ...this.memorySettings, ...settings }
    },

    /**
     * 设置上下文状�?     */
    setContextStatus(status) {
      this.contextStatus = status
    },

    // ==================== 输入状态管�?====================

    /**
     * 设置输入文本
     * 修复：集中管理输入状态，避免多处修改导致状态不一�?     */
    setInputText(text) {
      this.inputText = text
    },

    /**
     * 保存当前会话的草�?     */
    saveDraft(sessionId, content) {
      if (!sessionId) sessionId = 'new'
      if (content?.trim()) {
        this.inputDrafts[sessionId] = {
          content,
          timestamp: Date.now()
        }
      } else {
        delete this.inputDrafts[sessionId]
      }
    },

    /**
     * 加载会话草稿
     */
    loadDraft(sessionId) {
      if (!sessionId) sessionId = 'new'
      const draft = this.inputDrafts[sessionId]
      if (draft) {
        // 草稿24小时内有�?        if (Date.now() - draft.timestamp < 24 * 60 * 60 * 1000) {
          return draft.content
        }
        delete this.inputDrafts[sessionId]
      }
      return ''
    },

    /**
     * 清除会话草稿
     */
    clearDraft(sessionId) {
      if (!sessionId) sessionId = 'new'
      delete this.inputDrafts[sessionId]
    },

    /**
     * 切换会话时保存和恢复草稿
     */
    handleSessionSwitch(oldSessionId, newSessionId) {
      // 保存旧会话的输入内容
      if (this.inputText?.trim()) {
        this.saveDraft(oldSessionId, this.inputText)
      }

      // 加载新会话的草稿
      this.inputText = this.loadDraft(newSessionId)
    },

    /**
     * 检查上下文限制并返回建�?     * @returns {Object|null} 如果接近或超过限制，返回建议信息
     */
    checkContextLimit() {
      if (!this.contextStatus) return null

      if (this.contextStatus.is_over_limit) {
        return {
          type: 'error',
          code: 'ERR_CONTEXT_OVERFLOW',
          message: '上下文已超出限制',
          suggestion: 'create_new_session',
          userMessage: '对话上下文过长，建议创建新会话继续对�?,
          currentTokens: this.contextStatus.current_tokens,
          maxTokens: this.contextStatus.max_tokens,
        }
      }

      if (this.contextStatus.is_near_limit) {
        return {
          type: 'warning',
          code: 'CONTEXT_NEAR_LIMIT',
          message: '上下文接近限�?,
          suggestion: 'consider_new_session',
          userMessage: '对话上下文即将达到限制，建议适时创建新会�?,
          currentTokens: this.contextStatus.current_tokens,
          maxTokens: this.contextStatus.max_tokens,
          usagePercent: this.contextStatus.usage_percent,
        }
      }

      return null
    },

    /**
     * 获取并更新上下文状�?     * @param {string} sessionId - 会话ID
     */
    async fetchContextStatus(sessionId) {
      if (!sessionId) return null

      try {
        const status = await chatAPI.getContextStatus(sessionId)
        this.contextStatus = status
        return status
      } catch (error) {
        logger.warn('[Store] 获取上下文状态失�?', error)
        return null
      }
    },

    /**
     * 总结当前会话并创建新会话
     */
    async summarizeAndCreateNew() {
      if (!this.activeSessionId) {
        throw new Error('没有活动会话')
      }

      try {
        const result = await chatAPI.summarizeAndNewSession(this.activeSessionId)

        // 切换到新会话
        if (result.new_session_id) {
          await this.fetchSessions()
          await this.switchSession(result.new_session_id)
        }

        return result
      } catch (error) {
        this.handleError(error, '总结并创建新会话')
        throw error
      }
    },

    // ==================== 主题管理 ====================

    /**
     * 初始化主�?     */
    initTheme() {
      const savedTheme = localStorage.getItem('app-theme')
      if (savedTheme) {
        this.theme = savedTheme
      }
      this.applyTheme()
    },

    /**
     * 设置主题
     */
    setTheme(theme) {
      this.theme = theme
      localStorage.setItem('app-theme', theme)
      this.applyTheme()
    },

    /**
     * 切换主题
     */
    toggleTheme() {
      this.setTheme(this.theme === 'dark' ? 'light' : 'dark')
    },

    /**
     * 应用主题�?DOM
     */
    applyTheme() {
      if (this.theme === 'dark') {
        document.body.classList.add('dark')
        document.body.classList.remove('light')
      } else {
        document.body.classList.add('light')
        document.body.classList.remove('dark')
      }
    },

  }
})
