/**
 * 聊天API接口 - 对齐后端 /api/v1/chat
 */

import { chatAPI as api } from './index'

/**
 * 创建会话
 * @param {Object} sessionData - 会话数据
 * @param {string} sessionData.title - 会话标题
 * @param {string} sessionData.llm_provider - LLM提供商 (deepseek, zhipu, ollama)
 * @param {string} sessionData.model_name - 模型名称
 * @param {string} [sessionData.system_prompt] - 系统提示词
 */
export async function createSession(sessionData) {
  return await api.createSession(sessionData)
}

/**
 * 获取会话列表
 * @param {number} [limit=50] - 分页大小
 * @param {number} [offset=0] - 分页偏移
 */
export async function getSessions(limit = 50, offset = 0) {
  return await api.getSessions(limit, offset)
}

/**
 * 获取会话详情
 * @param {string} sessionId - 会话ID
 */
export async function getSession(sessionId) {
  return await api.getSession(sessionId)
}

/**
 * 更新会话
 * @param {string} sessionId - 会话ID
 * @param {Object} updateData - 更新数据
 */
export async function updateSession(sessionId, updateData) {
  return await api.updateSession(sessionId, updateData)
}

/**
 * 删除会话
 * @param {string} sessionId - 会话ID
 */
export async function deleteSession(sessionId) {
  return await api.deleteSession(sessionId)
}

/**
 * 获取会话消息
 * @param {string} sessionId - 会话ID
 * @param {number} [limit=50] - 分页大小
 * @param {number} [offset=0] - 分页偏移
 */
export async function getMessages(sessionId, limit = 50, offset = 0) {
  return await api.getMessages(sessionId, limit, offset)
}

/**
 * 清空会话消息
 * @param {string} sessionId - 会话ID
 */
export async function clearMessages(sessionId) {
  return await api.clearMessages(sessionId)
}

/**
 * 发送消息（非流式）
 * @param {Object} chatRequest - 聊天请求
 * @param {string} chatRequest.session_id - 会话ID
 * @param {string} chatRequest.message - 消息内容
 * @param {boolean} [chatRequest.stream=false] - 是否流式
 * @param {boolean} [chatRequest.use_web_search=false] - 是否使用网络搜索
 * @param {boolean} [chatRequest.use_deep_think=false] - 是否使用深度思考
 */
export async function chat(chatRequest) {
  return await api.chat(chatRequest)
}

/**
 * 发送消息（流式）- 使用fetch API处理SSE
 * @param {Object} chatRequest - 聊天请求
 * @param {string} chatRequest.session_id - 会话ID
 * @param {string} chatRequest.message - 消息内容
 * @param {boolean} [chatRequest.use_web_search=false] - 是否使用网络搜索
 * @param {boolean} [chatRequest.use_deep_think=false] - 是否使用深度思考
 * @param {Function} onMessage - 消息回调
 * @param {Function} onError - 错误回调
 * @param {Function} onComplete - 完成回调
 * @param {AbortSignal} [signal] - 中止信号
 */
export async function chatStream(chatRequest, onMessage, onError, onComplete, signal) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')

  try {
    const response = await fetch(`${baseURL}/chat/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'Accept': 'text/event-stream'
      },
      body: JSON.stringify({ 
        ...chatRequest, 
        stream: true,
        use_web_search: chatRequest.use_web_search || false,
        use_deep_think: chatRequest.use_deep_think || false,
      }),
      signal
    })

    if (!response.ok) {
      // 尝试解析错误响应体
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`
      try {
        const errorBody = await response.json()
        if (errorBody.error?.message) {
          errorMessage = errorBody.error.message
        } else if (errorBody.error) {
          errorMessage = typeof errorBody.error === 'string' ? errorBody.error : JSON.stringify(errorBody.error)
        }
      } catch {
        // 忽略解析错误，使用默认错误消息
      }
      throw new Error(errorMessage)
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = '' // 用于处理跨chunk的不完整行
    let currentEventType = 'message' // 当前SSE事件类型

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      
      // 保留最后一个可能不完整的行
      buffer = lines.pop() || ''

      for (const line of lines) {
        const trimmedLine = line.trim()
        
        // 空行表示事件结束，重置事件类型
        if (trimmedLine === '') {
          currentEventType = 'message'
          continue
        }
        
        // 解析事件类型行
        if (trimmedLine.startsWith('event:')) {
          currentEventType = trimmedLine.slice(6).trim()
          continue
        }
        
        // 解析数据行
        if (trimmedLine.startsWith('data:')) {
          const dataStr = trimmedLine.slice(5).trim()
          if (!dataStr) continue
          
          try {
            const data = JSON.parse(dataStr)
            
            // 根据事件类型处理
            if (currentEventType === 'error') {
              // 后端发送的 event: error (兼容旧格式)
              const errorMsg = data.error || data.message || '未知错误'
              onError(new Error(errorMsg))
              return
            }
            
            // 处理 event: message 或默认事件
            if (data.type === 'content' && data.content) {
              onMessage(data.content)
            } else if (data.type === 'start') {
              // 流开始，可以用于UI状态更新
              console.debug('[SSE] Stream started')
            } else if (data.type === 'end') {
              onComplete()
              return
            } else if (data.type === 'error' || data.error) {
              // 统一的错误格式: type: error 或 error 字段
              const errorMsg = data.error || data.message || '未知错误'
              onError(new Error(errorMsg))
              return
            }
          } catch (e) {
            // 记录解析错误但不中断流
            console.warn('[SSE] Parse error:', e.message, 'Raw data:', dataStr)
          }
        }
      }
    }

    // 处理buffer中剩余的数据
    if (buffer.trim()) {
      console.debug('[SSE] Remaining buffer:', buffer)
    }

    onComplete()
  } catch (error) {
    if (error.name !== 'AbortError') {
      console.error('[SSE] Stream error:', error)
      onError(error)
    }
  }
}

/**
 * 获取可用模型列表
 * @param {string} [provider] - 提供商过滤
 */
export async function getModels(provider = null) {
  return await api.getModels(provider)
}

/**
 * 获取会话上下文状态
 * @param {string} sessionId - 会话ID
 */
export async function getContextStatus(sessionId) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
  
  const response = await fetch(`${baseURL}/chat/sessions/${sessionId}/context-status`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  })
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`)
  }
  
  return await response.json()
}

/**
 * 总结当前会话并创建新会话
 * @param {string} sessionId - 会话ID
 */
export async function summarizeAndNewSession(sessionId) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
  
  const response = await fetch(`${baseURL}/chat/sessions/${sessionId}/summarize-and-new`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  })
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`)
  }
  
  return await response.json()
}

// 导出chatAPI对象以便直接使用
export { api as chatAPI }
