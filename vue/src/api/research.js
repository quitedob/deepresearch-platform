/**
 * 研究API接口 - 对齐后端 /api/v1/research
 */

import { researchAPI as api } from './index'

/**
 * 开始深度研究
 * @param {Object} researchData - 研究数据
 * @param {string} researchData.query - 研究问题
 * @param {string} [researchData.research_type='deep'] - 研究类型 (quick, deep, comprehensive)
 * @param {Object} [researchData.llm_config] - LLM配置
 * @param {Object} [researchData.tools_config] - 工具配置
 * @param {Object} [researchData.options] - 研究选项
 */
export async function startResearch(researchData) {
  return await api.startResearch(researchData)
}

/**
 * 获取研究状态
 * @param {string} sessionId - 研究会话ID
 */
export async function getResearchStatus(sessionId) {
  return await api.getResearchStatus(sessionId)
}

/**
 * 获取研究会话列表
 * @param {Object} [params] - 查询参数
 * @param {number} [params.limit=20] - 分页大小
 * @param {number} [params.offset=0] - 分页偏移
 * @param {string} [params.status] - 状态过滤
 */
export async function getResearchSessions(params = {}) {
  return await api.getResearchSessions(params)
}

/**
 * 流式获取研究进度 - 使用fetch + ReadableStream (安全方式)
 * 
 * 支持并行多Agent任务进度展示
 * 
 * @param {string} sessionId - 研究会话ID
 * @param {Function} onUpdate - 进度更新回调
 * @param {Function} onError - 错误回调
 * @param {Function} onComplete - 完成回调
 * @returns {Object} 包含abort方法的控制器对象
 */
export function streamResearchProgress(sessionId, onUpdate, onError, onComplete) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'
  const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')

  const abortController = new AbortController()

  const fetchStream = async () => {
    try {
      const response = await fetch(`${baseURL}/research/stream/${sessionId}`, {
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

              if (data.type === 'connected') {
                // 连接成功
              } else if (data.type === 'heartbeat') {
                // 心跳，忽略
              } else if (data.type === 'status_update') {
                onUpdate(data)
              } else if (data.type === 'completed') {
                onComplete(data.data)
                return
              } else if (data.type === 'failed' || data.type === 'error') {
                onError(new Error(data.error || '研究失败'))
                return
              }
            } catch (e) {
              // 忽略解析错误
            }
          }
        }
      }

      onComplete(null)
    } catch (error) {
      if (error.name !== 'AbortError') {
        onError(error)
      }
    }
  }

  fetchStream()

  return {
    abort: () => abortController.abort(),
    close: () => abortController.abort()
  }
}

/**
 * 导出研究结果
 * @param {string} sessionId - 研究会话ID
 * @param {string} [format='json'] - 导出格式 (json, markdown)
 */
export async function exportResearch(sessionId, format = 'json') {
  const blob = await api.exportResearch(sessionId, format)

  // 创建下载链接
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `research_${sessionId}.${format === 'markdown' ? 'md' : format}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

/**
 * 搜索研究结果
 * @param {string} query - 搜索关键词
 * @param {Object} [params] - 查询参数
 */
export async function searchResearch(query, params = {}) {
  return await api.searchResearch(query, params)
}

/**
 * 获取研究统计
 */
export async function getResearchStats() {
  return await api.getResearchStats()
}

// 研究类型常量
export const RESEARCH_TYPES = {
  QUICK: 'quick',           // 快速研究 (5-10分钟)
  DEEP: 'deep',             // 深度研究 (15-30分钟)
  COMPREHENSIVE: 'comprehensive'  // 全面研究 (30-60分钟)
}

// 研究状态常量
export const RESEARCH_STATUS = {
  PLANNING: 'planning',     // 规划中
  EXECUTING: 'executing',   // 执行中
  SYNTHESIS: 'synthesis',   // 综合中
  COMPLETED: 'completed',   // 已完成
  FAILED: 'failed'          // 失败
}

// 研究工具常量
export const RESEARCH_TOOLS = {
  WEB_SEARCH: 'web_search',
  ARXIV: 'arxiv',
  WIKIPEDIA: 'wikipedia',
  MCP_TOOLS: 'mcp_tools'
}

// 导出researchAPI对象以便直接使用
export { api as researchAPI }
