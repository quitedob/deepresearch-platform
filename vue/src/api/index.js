import axios from 'axios'
import { API_BASE_URL, REQUEST_TIMEOUT } from '@/utils/config'
import { getAuthToken, clearTokens } from '@/utils/token'

// 创建axios实例
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: REQUEST_TIMEOUT.DEFAULT,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    const token = getAuthToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    // 处理新的统一响应格式
    const data = response.data
    // 如果响应包含success字段且为false，抛出错误
    if (data && data.success === false && data.error) {
      const error = new Error(data.error.message || '请求失败')
      error.response = { data, status: response.status }
      return Promise.reject(error)
    }
    // 返回data字段或整个响应
    return data.data !== undefined ? data.data : data
  },
  (error) => {
    // 提取错误消息
    let errorMessage = '请求失败'
    
    if (error.response?.data) {
      const data = error.response.data
      // 尝试从不同格式中提取错误消息
      if (data.message) {
        // 后端统一响应格式: {"code": 401, "message": "账户已被禁用"}
        errorMessage = data.message
      } else if (data.error?.message) {
        errorMessage = data.error.message
      } else if (data.error && typeof data.error === 'string') {
        errorMessage = data.error
      } else if (typeof data === 'string') {
        errorMessage = data
      }
    }
    
    // 创建带有友好消息的错误
    const friendlyError = new Error(errorMessage)
    friendlyError.response = error.response
    friendlyError.originalError = error
    
    // 处理401错误
    if (error.response?.status === 401) {
      const errorCode = error.response?.data?.error?.code || error.response?.data?.code
      // 只有在token相关错误时才跳转登录（排除登录失败的情况）
      // 登录失败时 message 会是 "用户名或密码错误" 或 "账户已被禁用"
      const isLoginError = errorMessage.includes('密码') || errorMessage.includes('禁用') || errorMessage.includes('账户')
      // 检查是否是 token 过期/无效的错误
      const isTokenError = errorCode === 'ERR_TOKEN_EXPIRED' || 
                          errorCode === 'ERR_TOKEN_INVALID' ||
                          errorCode === 'ERR_UNAUTHORIZED' ||
                          errorMessage.includes('expired') ||
                          errorMessage.includes('invalid') ||
                          errorMessage.includes('过期') ||
                          errorMessage.includes('无效')
      
      if (!isLoginError && isTokenError) {
        clearTokens()
        // 避免在登录页面重复跳转
        if (!window.location.pathname.includes('/login')) {
          window.location.href = '/login'
        }
      }
    }
    
    return Promise.reject(friendlyError)
  }
)

// 聊天API
export const chatAPI = {
  // 创建聊天会话
  createSession(data) {
    return apiClient.post('/chat/sessions', data)
  },

  // 获取聊天会话列表
  getSessions(limit = 20, offset = 0) {
    return apiClient.get('/chat/sessions', { params: { limit, offset } })
  },

  // 获取聊天会话详情
  getSession(sessionId) {
    return apiClient.get(`/chat/sessions/${sessionId}`)
  },

  // 更新聊天会话
  updateSession(sessionId, data) {
    return apiClient.put(`/chat/sessions/${sessionId}`, data)
  },

  // 删除聊天会话
  deleteSession(sessionId) {
    return apiClient.delete(`/chat/sessions/${sessionId}`)
  },

  // 获取会话消息列表
  getMessages(sessionId, limit = 50, offset = 0) {
    return apiClient.get(`/chat/sessions/${sessionId}/messages`, { params: { limit, offset } })
  },

  // 清空会话消息
  clearMessages(sessionId) {
    return apiClient.delete(`/chat/sessions/${sessionId}/messages`)
  },

  // 发送聊天消息（非流式）
  chat(data) {
    return apiClient.post('/chat/chat', data)
  },

  // 获取可用模型列表
  getModels(provider = null) {
    const params = provider ? { provider } : {}
    return apiClient.get('/chat/models', { params })
  },

  // 获取会话上下文状态
  getContextStatus(sessionId) {
    return apiClient.get(`/chat/sessions/${sessionId}/context-status`)
  },

  // 总结并创建新会话
  summarizeAndNewSession(sessionId) {
    return apiClient.post(`/chat/sessions/${sessionId}/summarize-new`)
  }
}

// LLM API
export const llmAPI = {
  // 获取LLM提供商列表
  getProviders() {
    return apiClient.get('/llm/providers')
  },

  // 获取所有可用模型
  getModels() {
    return apiClient.get('/llm/models')
  },

  // 测试LLM提供商
  testProvider(data) {
    return apiClient.post('/llm/test', data)
  },

  // 获取LLM指标
  getMetrics() {
    return apiClient.get('/llm/metrics')
  }
}

// 研究API
export const researchAPI = {
  // 开始研究
  startResearch(data) {
    return apiClient.post('/research/start', data)
  },

  // 获取研究状态
  getResearchStatus(sessionId) {
    return apiClient.get(`/research/status/${sessionId}`)
  },

  // 获取研究会话列表
  getResearchSessions(params = {}) {
    return apiClient.get('/research/sessions', { params })
  },

  // 导出研究结果
  exportResearch(sessionId, format = 'json') {
    return apiClient.get(`/research/export/${sessionId}`, {
      params: { format },
      responseType: 'blob'
    })
  },

  // 搜索研究结果
  searchResearch(query, params = {}) {
    return apiClient.get('/research/search', {
      params: { query, ...params }
    })
  },

  // 获取研究统计
  getResearchStats() {
    return apiClient.get('/research/statistics')
  }
}

// 认证API
export const authAPI = {
  // 用户登录
  login(credentials) {
    return apiClient.post('/auth/login', credentials)
  },

  // 用户注册
  register(userData) {
    return apiClient.post('/auth/register', userData)
  },

  // 刷新令牌
  refreshToken(refreshToken) {
    return apiClient.post('/auth/refresh', { refresh_token: refreshToken })
  },

  // 用户登出
  logout() {
    return apiClient.post('/auth/logout')
  },

  // 验证令牌
  verifyToken() {
    return apiClient.get('/auth/verify')
  }
}

// 用户API
export const userAPI = {
  // 获取用户信息
  getProfile() {
    return apiClient.get('/user/profile')
  },

  // 更新用户信息
  updateProfile(data) {
    return apiClient.put('/user/profile', data)
  },

  // 获取用户偏好设置
  getPreferences() {
    return apiClient.get('/user/preferences')
  },

  // 更新用户偏好设置
  updatePreferences(data) {
    return apiClient.put('/user/preferences', data)
  },

  // 获取记忆设置
  getMemorySettings() {
    return apiClient.get('/user/memory-settings')
  },

  // 更新记忆设置
  updateMemorySettings(data) {
    return apiClient.put('/user/memory-settings', data)
  }
}

// 健康检查API
export const healthAPI = {
  check() {
    return apiClient.get('/health')
  },

  detailed() {
    return apiClient.get('/health/detailed')
  }
}

// MCP工具API
export const mcpAPI = {
  // 获取可用工具列表
  getTools() {
    return apiClient.get('/mcp/tools')
  },

  // 调用工具
  callTool(toolName, params) {
    return apiClient.post('/mcp/tools/call', {
      tool_name: toolName,
      parameters: params
    })
  }
}

// 支持的LLM提供商（仅作为类型参考，实际值从 API 获取）
export const PROVIDERS = {
  DEEPSEEK: 'deepseek',
  ZHIPU: 'zhipu',
  OLLAMA: 'ollama',
  OPENROUTER: 'openrouter'
}

// 注意：模型列表应从 API 动态获取，不要硬编码
// 使用 import { getProviders } from '@/api/model' 获取完整模型配置

// 消息角色
export const MESSAGE_ROLES = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system'
}

// 研究类型
export const RESEARCH_TYPES = {
  QUICK: 'quick',
  DEEP: 'deep',
  COMPREHENSIVE: 'comprehensive'
}

// 研究状态
export const RESEARCH_STATUS = {
  PLANNING: 'planning',
  EXECUTING: 'executing',
  SYNTHESIS: 'synthesis',
  COMPLETED: 'completed',
  FAILED: 'failed'
}

// 会员API
export const membershipAPI = {
  // 获取会员信息
  getMembership() {
    return apiClient.get('/membership')
  },

  // 获取配额信息
  getQuota() {
    return apiClient.get('/membership/quota')
  },

  // 使用激活码
  activateCode(code) {
    return apiClient.post('/membership/activate', { code })
  },

  // 检查聊天配额
  checkChatQuota() {
    return apiClient.get('/membership/check-chat-quota')
  },

  // 检查研究配额
  checkResearchQuota() {
    return apiClient.get('/membership/check-research-quota')
  }
}

// 通知API
export const notificationAPI = {
  // 获取通知列表
  getNotifications(limit = 20, offset = 0) {
    return apiClient.get('/notifications', { params: { limit, offset } })
  },

  // 获取未读通知数量
  getUnreadCount() {
    return apiClient.get('/notifications/unread-count')
  },

  // 标记通知为已读
  markAsRead(notificationId) {
    return apiClient.post(`/notifications/${notificationId}/read`)
  },

  // 标记所有通知为已读
  markAllAsRead() {
    return apiClient.post('/notifications/read-all')
  }
}

// AI题目生成API
export const aiQuestionAPI = {
  // 生成题目
  generateQuestions(data) {
    return apiClient.post('/ai/generate-questions', data)
  },

  // 创建会话
  createSession(data) {
    return apiClient.post('/ai/question-sessions', data)
  },

  // 获取会话列表
  getSessions(limit = 20, offset = 0) {
    return apiClient.get('/ai/question-sessions', { params: { limit, offset } })
  },

  // 获取会话详情
  getSession(sessionId) {
    return apiClient.get(`/ai/question-sessions/${sessionId}`)
  },

  // 更新会话标题
  updateSessionTitle(sessionId, title) {
    return apiClient.put(`/ai/question-sessions/${sessionId}`, { title })
  },

  // 删除会话
  deleteSession(sessionId) {
    return apiClient.delete(`/ai/question-sessions/${sessionId}`)
  },

  // 保存题目到会话
  saveQuestions(sessionId, questions) {
    return apiClient.post(`/ai/question-sessions/${sessionId}/questions`, { questions })
  },

  // 获取AI出题配置
  getConfig() {
    return apiClient.get('/ai/question-config')
  }
}

export { apiClient }
export default {
  auth: authAPI,
  chat: chatAPI,
  llm: llmAPI,
  research: researchAPI,
  user: userAPI,
  health: healthAPI,
  mcp: mcpAPI,
  membership: membershipAPI,
  notification: notificationAPI,
  aiQuestion: aiQuestionAPI
}
