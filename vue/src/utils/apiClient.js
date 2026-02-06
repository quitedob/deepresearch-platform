/**
 * API客户端工具 - 提供统一的HTTP请求封装
 */

import axios from 'axios'

// 创建axios实例
export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // 添加认证token
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 添加请求ID
    config.headers['X-Request-ID'] = generateRequestId()

    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response) {
      const { status, data } = error.response

      // 处理401未授权
      if (status === 401) {
        localStorage.removeItem('auth_token')
        sessionStorage.removeItem('auth_token')

        // 如果不在登录页，跳转到登录页
        if (!window.location.pathname.includes('/login')) {
          window.location.href = '/login'
        }
      }

      // 返回错误信息
      const errorMessage = data?.error || data?.message || `请求失败 (${status})`
      return Promise.reject(new Error(errorMessage))
    } else if (error.request) {
      return Promise.reject(new Error('网络连接失败，请检查网络设置'))
    }

    return Promise.reject(error)
  }
)

// 生成请求ID
function generateRequestId() {
  return 'req_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
}

// 设置认证token
export function setAuthToken(token) {
  if (token) {
    localStorage.setItem('auth_token', token)
    apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`
  } else {
    localStorage.removeItem('auth_token')
    delete apiClient.defaults.headers.common['Authorization']
  }
}

// 获取认证token
export function getAuthToken() {
  return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
}

// 检查是否已认证
export function isAuthenticated() {
  return !!getAuthToken()
}

// 清除认证信息
export function clearAuth() {
  localStorage.removeItem('auth_token')
  sessionStorage.removeItem('auth_token')
  delete apiClient.defaults.headers.common['Authorization']
}

/**
 * 创建安全的SSE连接 (使用fetch + ReadableStream)
 * 
 * 注意：此方法使用Authorization header而非URL查询参数传递token，
 * 避免token被记录在日志、浏览器历史、代理服务器中
 * 
 * @param {string} url - SSE端点URL
 * @param {Object} options - 配置选项
 * @param {Function} options.onMessage - 消息回调
 * @param {Function} options.onError - 错误回调
 * @param {Function} options.onComplete - 完成回调
 * @returns {Object} 包含abort方法的控制器对象
 */
export function createSSEConnection(url, options = {}) {
  const token = getAuthToken()
  const baseURL = apiClient.defaults.baseURL
  const fullURL = url.startsWith('http') ? url : `${baseURL}${url}`

  const abortController = new AbortController()
  const { onMessage, onError, onComplete } = options

  const fetchStream = async () => {
    try {
      const response = await fetch(fullURL, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Accept': 'text/event-stream',
          'Cache-Control': 'no-cache'
        },
        signal: abortController.signal
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))
              onMessage?.(data)
            } catch (e) {
              // 忽略解析错误
            }
          }
        }
      }

      onComplete?.()
    } catch (error) {
      if (error.name !== 'AbortError') {
        onError?.(error)
      }
    }
  }

  fetchStream()

  return {
    abort: () => abortController.abort(),
    close: () => abortController.abort()
  }
}

// 发送流式请求
export async function streamRequest(url, data, onMessage, onError, onComplete, signal) {
  const baseURL = apiClient.defaults.baseURL
  const token = getAuthToken()

  try {
    const response = await fetch(`${baseURL}${url}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'Accept': 'text/event-stream'
      },
      body: JSON.stringify(data),
      signal
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
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
            onMessage(data)
          } catch (e) {
            // 忽略解析错误
          }
        }
      }
    }

    onComplete()
  } catch (error) {
    if (error.name !== 'AbortError') {
      onError(error)
    }
  }
}

export default apiClient
