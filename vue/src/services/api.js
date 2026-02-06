/**
 * API服务层 - 提供统一的API调用接口
 */

import axios from 'axios'

// 创建axios实例
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    config.headers['X-Request-ID'] = generateRequestId()
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response) {
      const { status, data } = error.response
      
      switch (status) {
        case 401:
          localStorage.removeItem('auth_token')
          sessionStorage.removeItem('auth_token')
          window.location.href = '/login'
          break
        case 403:
          console.error('权限不足')
          break
        case 404:
          console.error('资源不存在')
          break
        case 429:
          console.error('请求过于频繁')
          break
        case 500:
          console.error('服务器错误')
          break
      }
      
      return Promise.reject(new Error(data?.error || data?.message || `HTTP ${status}`))
    } else if (error.request) {
      return Promise.reject(new Error('网络连接失败'))
    }
    return Promise.reject(error)
  }
)

// 生成请求ID
function generateRequestId() {
  return 'req_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
}

// 处理API错误
export function handleAPIError(error) {
  if (error.response) {
    const { status, data } = error.response
    return data?.error || data?.message || `请求失败 (${status})`
  } else if (error.request) {
    return '网络连接失败，请检查网络设置'
  }
  return error.message || '未知错误'
}

// 健康检查API
export const healthAPI = {
  checkHealth() {
    return api.get('/health')
  },
  
  getDetailedHealth() {
    return api.get('/health/detailed')
  },
  
  getPerformanceStats() {
    return api.get('/monitoring/performance')
  }
}

// LLM服务
export class LLMService {
  async getProviders() {
    try {
      const response = await api.get('/llm/providers')
      return response.providers || []
    } catch (error) {
      console.error('获取LLM提供商失败:', error)
      return []
    }
  }

  async getModels(provider = null) {
    try {
      const params = provider ? { provider } : {}
      const response = await api.get('/chat/models', { params })
      return response.models || []
    } catch (error) {
      console.error('获取模型列表失败:', error)
      return []
    }
  }

  async testProvider(provider, model, testMessage = 'Hello') {
    return api.post('/llm/test', { provider, model, messages: [testMessage] })
  }

  async getMetrics() {
    try {
      const response = await api.get('/llm/metrics')
      return response.metrics || {}
    } catch (error) {
      console.error('获取LLM指标失败:', error)
      return {}
    }
  }
}

// 聊天服务
export class ChatService {
  async createSession(sessionData) {
    return api.post('/chat/sessions', sessionData)
  }

  async getSessions(limit = 20, offset = 0) {
    return api.get('/chat/sessions', { params: { limit, offset } })
  }

  async getSession(sessionId) {
    return api.get(`/chat/sessions/${sessionId}`)
  }

  async sendMessage(messageData) {
    return api.post('/chat/chat', messageData)
  }

  async sendMessageStream(messageData, onMessage, onError, onComplete) {
    const baseURL = api.defaults.baseURL
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')

    try {
      const response = await fetch(`${baseURL}/chat/chat/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
          'Accept': 'text/event-stream'
        },
        body: JSON.stringify(messageData)
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const chunk = decoder.decode(value)
        const lines = chunk.split('\n')

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              if (data.type === 'content') {
                onMessage(data.content)
              } else if (data.type === 'end') {
                onComplete()
                return
              } else if (data.error) {
                onError(new Error(data.error))
                return
              }
            } catch (e) {
              // 忽略解析错误
            }
          }
        }
      }
      
      onComplete()
    } catch (error) {
      onError(error)
    }
  }

  async getMessages(sessionId, limit = 50, offset = 0) {
    return api.get(`/chat/sessions/${sessionId}/messages`, { params: { limit, offset } })
  }

  async deleteSession(sessionId) {
    return api.delete(`/chat/sessions/${sessionId}`)
  }
}

// 研究服务
export class ResearchService {
  async startResearch(researchData) {
    return api.post('/research/start', researchData)
  }

  async getResearchStatus(sessionId) {
    return api.get(`/research/status/${sessionId}`)
  }

  async streamResearchStatus(sessionId, onUpdate, onError, onComplete) {
    const baseURL = api.defaults.baseURL
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')

    try {
      const response = await fetch(`${baseURL}/research/stream/${sessionId}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Accept': 'text/event-stream'
        }
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const chunk = decoder.decode(value)
        const lines = chunk.split('\n')

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              if (data.type === 'status_update') {
                onUpdate(data.data)
              } else if (data.type === 'completed') {
                onComplete(data.data)
                return
              } else if (data.type === 'failed' || data.error) {
                onError(new Error(data.error || '研究失败'))
                return
              }
            } catch (e) {
              console.warn('解析研究状态数据失败:', e)
            }
          }
        }
      }
    } catch (error) {
      onError(error)
    }
  }

  async getResearchSessions(params = {}) {
    return api.get('/research/sessions', { params })
  }

  async exportResearch(sessionId, format = 'json') {
    return api.get(`/research/export/${sessionId}`, {
      params: { format },
      responseType: 'blob'
    })
  }

  async searchResearch(query, params = {}) {
    return api.get('/research/search', { params: { query, ...params } })
  }
}

// 用户服务
export class UserService {
  async getProfile() {
    return api.get('/users/me')
  }

  async updateProfile(profileData) {
    return api.put('/users/me', profileData)
  }

  async getPreferences() {
    return api.get('/users/preferences')
  }

  async updatePreferences(preferences) {
    return api.put('/users/preferences', preferences)
  }

  async getStatistics() {
    return api.get('/users/stats')
  }
}

// 文件上传API
export async function uploadSingleFile(file, options = {}) {
  const formData = new FormData()
  formData.append('file', file)
  
  if (options.processContent) {
    formData.append('process_content', 'true')
  }
  
  try {
    const response = await api.post('/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: options.onProgress
    })
    return response
  } catch (error) {
    console.error('文件上传失败:', error)
    throw error
  }
}

export async function uploadMultipleFiles(files, options = {}) {
  const formData = new FormData()
  
  for (const file of files) {
    formData.append('files', file)
  }
  
  if (options.processContent) {
    formData.append('process_content', 'true')
  }
  
  try {
    const response = await api.post('/files/upload/batch', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: options.onProgress
    })
    return response
  } catch (error) {
    console.error('批量文件上传失败:', error)
    throw error
  }
}

// 证据链API
export const evidenceAPI = {
  // 获取对话证据
  async getConversationEvidence(conversationId) {
    try {
      const response = await api.get(`/research/evidence/conversation/${conversationId}`)
      return response
    } catch (error) {
      console.error('获取对话证据失败:', error)
      return { evidence: [] }
    }
  },

  // 获取研究证据
  async getResearchEvidence(researchSessionId) {
    try {
      const response = await api.get(`/research/evidence/${researchSessionId}`)
      return response
    } catch (error) {
      console.error('获取研究证据失败:', error)
      return { evidence: [] }
    }
  },

  // 标记证据使用状态
  async markEvidenceUsed(evidenceId, used) {
    return api.put(`/research/evidence/${evidenceId}/used`, { used })
  },

  // 获取证据统计
  async getEvidenceStats() {
    try {
      const response = await api.get('/research/evidence/stats')
      return response
    } catch (error) {
      console.error('获取证据统计失败:', error)
      return { avg_relevance_score: 0 }
    }
  }
}

// 导出服务实例
export const llmService = new LLMService()
export const chatService = new ChatService()
export const researchService = new ResearchService()
export const userService = new UserService()

// 导出获取提供商函数
export async function getProviders() {
  return llmService.getProviders()
}

// 导出API实例
export default api
