/**
 * 论文生成 API 客户端
 */
import { apiClient } from './index'
import { getAuthToken } from '@/utils/token'

export const paperAPI = {
  // 开始论文生成
  start(data) {
    return apiClient.post('/paper/start', data)
  },

  // 获取论文状态
  getStatus(id) {
    return apiClient.get(`/paper/status/${id}`)
  },

  // 获取论文结果
  getResult(id) {
    return apiClient.get(`/paper/result/${id}`)
  },

  // 预览论文内容（返回文本，不下载）
  preview(id, format = 'markdown') {
    return apiClient.get(`/paper/preview/${id}`, { params: { format } })
  },

  // 导出论文
  export(id, format = 'markdown') {
    return apiClient.get(`/paper/export/${id}`, {
      params: { format },
      responseType: 'blob'
    })
  },

  // 获取论文列表
  list(params = {}) {
    return apiClient.get('/paper/list', { params })
  },

  // 删除论文
  delete(id) {
    return apiClient.delete(`/paper/${id}`)
  },

  // 重新生成章节
  regenerateChapter(data) {
    return apiClient.post('/paper/regenerate', data)
  },

  // 手动更新章节内容
  updateChapter(chapterId, paperId, content) {
    return apiClient.patch(`/paper/chapter/${chapterId}`, { paper_id: paperId, content })
  },

  // 获取模板列表
  getTemplates() {
    return apiClient.get('/paper/templates')
  },

  // 获取引用格式列表
  getCitationStyles() {
    return apiClient.get('/paper/citation-styles')
  }
}

/**
 * 创建论文进度 SSE 连接
 * 使用 fetch + ReadableStream（浏览器 EventSource 不支持自定义 header）
 * @param {string} paperId
 * @param {function} onEvent - 事件回调 (event) => void
 * @param {function} onError - 错误回调 (err) => void
 * @returns {function} 关闭连接的函数
 */
export function createPaperSSE(paperId, onEvent, onError) {
  const token = getAuthToken()
  let abortController = new AbortController()
  let closed = false

  const connect = async () => {
    try {
      // 问题1修复：复用 apiClient 的 baseURL，从同一配置源读取
      const { API_BASE_URL } = await import('@/utils/config')
      const response = await fetch(`${API_BASE_URL}/paper/stream/${paperId}`, {
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: 'text/event-stream'
        },
        signal: abortController.signal
      })

      if (!response.ok) {
        onError?.(new Error(`SSE 连接失败: ${response.status}`))
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (!closed) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() // 保留不完整的行

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const event = JSON.parse(line.slice(6))
              onEvent?.(event)
            } catch {
              // 忽略解析错误
            }
          }
        }
      }
    } catch (err) {
      if (err.name !== 'AbortError') {
        onError?.(err)
      }
    }
  }

  connect()

  return () => {
    closed = true
    abortController.abort()
  }
}

export default paperAPI
