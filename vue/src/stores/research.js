/**
 * 研究Store - 管理深度研究状态
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { researchAPI } from '@/api/index'

export const useResearchStore = defineStore('research', () => {
  // 状态
  const sessions = ref([])
  const currentSession = ref(null)
  const isLoading = ref(false)
  const isStreaming = ref(false)
  const researchProgress = ref(0)
  const currentStep = ref('')
  const researchResults = ref([])
  const researchMetadata = ref({})
  const errorMessage = ref('')
  
  // 研究配置
  const researchConfig = ref({
    type: 'deep',
    maxSources: 20,
    maxIterations: 10,
    enableCache: true,
    llmProvider: 'deepseek',
    llmModel: 'deepseek-chat',
    tools: ['web_search', 'arxiv', 'wikipedia'],
    language: 'zh-CN'
  })
  
  // 可用工具
  const availableTools = ref([
    {
      name: 'web_search_prime',
      displayName: '增强网络搜索',
      description: '智谱增强搜索，返回更丰富的结果',
      enabled: true
    },
    {
      name: 'web_search',
      displayName: '网络搜索',
      description: '搜索最新的网络信息',
      enabled: true
    },
    {
      name: 'arxiv',
      displayName: 'arXiv学术搜索',
      description: '搜索学术论文和研究',
      enabled: true
    },
    {
      name: 'wikipedia',
      displayName: 'Wikipedia',
      description: '搜索维基百科知识',
      enabled: true
    },
    {
      name: 'web_reader',
      displayName: '网页读取',
      description: '抓取网页内容进行分析',
      enabled: true
    },
    {
      name: 'zread_repo',
      displayName: '开源仓库读取',
      description: '读取GitHub开源仓库文档和代码',
      enabled: true
    }
  ])
  
  // 并行Agent任务状态
  const agentTasks = ref([])
  
  // 计算属性
  const activeSessions = computed(() => 
    sessions.value.filter(s => s.status === 'executing' || s.status === 'planning')
  )
  
  const completedSessions = computed(() => 
    sessions.value.filter(s => s.status === 'completed')
  )
  
  const failedSessions = computed(() => 
    sessions.value.filter(s => s.status === 'failed')
  )
  
  const researchStats = computed(() => ({
    total: sessions.value.length,
    active: activeSessions.value.length,
    completed: completedSessions.value.length,
    failed: failedSessions.value.length,
    successRate: sessions.value.length > 0 
      ? (completedSessions.value.length / sessions.value.length * 100).toFixed(1)
      : 0
  }))
  
  const enabledTools = computed(() => 
    availableTools.value.filter(tool => tool.enabled)
  )
  
  // 开始研究
  async function startResearch(query, config = {}) {
    if (isLoading.value || isStreaming.value) {
      console.warn('已有研究任务在进行中')
      return null
    }
    
    isLoading.value = true
    errorMessage.value = ''
    
    try {
      const researchData = {
        query,
        research_type: config.type || researchConfig.value.type,
        llm_config: {
          provider: config.llmProvider || researchConfig.value.llmProvider,
          model: config.llmModel || researchConfig.value.llmModel
        },
        tools_config: {
          enabled_tools: config.tools || researchConfig.value.tools,
          max_sources: config.maxSources || researchConfig.value.maxSources,
          max_iterations: config.maxIterations || researchConfig.value.maxIterations,
          enable_cache: config.enableCache ?? researchConfig.value.enableCache
        },
        options: {
          language: config.language || researchConfig.value.language,
          output_format: 'structured',
          include_sources: true,
          include_reasoning: true
        }
      }
      
      const response = await researchAPI.startResearch(researchData)
      
      if (response.success) {
        const session = {
          id: response.session_id,
          query,
          status: 'planning',
          progress: 0,
          startTime: new Date(),
          config: researchData,
          results: [],
          metadata: {}
        }
        
        sessions.value.unshift(session)
        currentSession.value = session
        
        console.log('研究任务已启动:', response.session_id)
        return session
      } else {
        throw new Error(response.message || '启动研究失败')
      }
    } catch (error) {
      console.error('启动研究失败:', error)
      errorMessage.value = error.message
      return null
    } finally {
      isLoading.value = false
    }
  }
  
  // 更新会话状态
  function updateSessionStatus(sessionId, statusData) {
    const session = sessions.value.find(s => s.id === sessionId)
    if (!session) return
    
    session.status = statusData.status || session.status
    session.progress = statusData.progress || session.progress
    session.metadata = { ...session.metadata, ...statusData.metadata }
    
    if (statusData.current_step) {
      currentStep.value = statusData.current_step
    }
    
    if (statusData.progress !== undefined) {
      researchProgress.value = statusData.progress
    }
    
    if (statusData.tasks) {
      session.tasks = statusData.tasks
    }
    
    // 跟踪并行Agent任务状态
    if (statusData.task_name) {
      const existing = agentTasks.value.find(t => t.name === statusData.task_name)
      if (existing) {
        existing.status = statusData.task_status || existing.status
        existing.message = statusData.message || existing.message
      } else {
        agentTasks.value.push({
          name: statusData.task_name,
          status: statusData.task_status || 'running',
          message: statusData.message || ''
        })
      }
    }
    
    if (statusData.results) {
      session.results = statusData.results
      researchResults.value = statusData.results
    }
    
    if (currentSession.value && currentSession.value.id === sessionId) {
      currentSession.value = { ...session }
    }
  }
  
  // 完成会话
  function completeSession(sessionId, data) {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.status = 'completed'
      session.progress = 100
      session.endTime = new Date()
      session.results = data.results || session.results
      session.metadata = { ...session.metadata, ...data.metadata }
      
      if (currentSession.value && currentSession.value.id === sessionId) {
        currentSession.value = { ...session }
        researchResults.value = session.results
        researchMetadata.value = session.metadata
      }
    }
    
    researchProgress.value = 100
    currentStep.value = '研究完成'
    isStreaming.value = false
    agentTasks.value = []
  }
  
  // 会话失败
  function failSession(sessionId, error) {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.status = 'failed'
      session.error = error
      session.endTime = new Date()
      
      if (currentSession.value && currentSession.value.id === sessionId) {
        currentSession.value = { ...session }
      }
    }
    errorMessage.value = error
    isStreaming.value = false
  }
  
  // 获取研究会话列表
  async function getResearchSessions(params = {}) {
    isLoading.value = true
    
    try {
      const response = await researchAPI.getResearchSessions(params)
      if (response.success) {
        sessions.value = response.sessions || []
      }
    } catch (error) {
      console.error('获取研究会话失败:', error)
    } finally {
      isLoading.value = false
    }
  }
  
  // 获取研究状态
  async function getResearchStatus(sessionId) {
    try {
      const response = await researchAPI.getResearchStatus(sessionId)
      if (response.success) {
        updateSessionStatus(sessionId, response.status_data)
        return response.status_data
      }
    } catch (error) {
      console.error('获取研究状态失败:', error)
      return null
    }
  }
  
  // 导出研究结果
  async function exportResearch(sessionId, format = 'json') {
    try {
      const blob = await researchAPI.exportResearch(sessionId, format)
      
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `research_${sessionId}.${format === 'markdown' ? 'md' : format}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      
      console.log('研究结果已导出')
    } catch (error) {
      console.error('导出研究结果失败:', error)
    }
  }
  
  // 搜索研究结果
  async function searchResearch(query, params = {}) {
    try {
      const response = await researchAPI.searchResearch(query, params)
      return response.results || []
    } catch (error) {
      console.error('搜索研究结果失败:', error)
      return []
    }
  }
  
  // 设置当前会话
  function setCurrentSession(session) {
    currentSession.value = session
    if (session) {
      researchResults.value = session.results || []
      researchMetadata.value = session.metadata || {}
    }
  }
  
  // 更新研究配置
  function updateResearchConfig(config) {
    researchConfig.value = { ...researchConfig.value, ...config }
  }
  
  // 切换工具
  function toggleTool(toolName, enabled) {
    const tool = availableTools.value.find(t => t.name === toolName)
    if (tool) {
      tool.enabled = enabled
    }
  }
  
  // 清空结果
  function clearResults() {
    researchResults.value = []
    researchMetadata.value = {}
    researchProgress.value = 0
    currentStep.value = ''
    errorMessage.value = ''
    agentTasks.value = []
  }
  
  // 停止当前研究
  function stopCurrentResearch() {
    isStreaming.value = false
    
    if (currentSession.value && currentSession.value.status === 'executing') {
      currentSession.value.status = 'cancelled'
      currentSession.value.endTime = new Date()
    }
    
    console.log('研究任务已停止')
  }
  
  // 重置状态
  function $reset() {
    sessions.value = []
    currentSession.value = null
    isLoading.value = false
    isStreaming.value = false
    researchProgress.value = 0
    currentStep.value = ''
    researchResults.value = []
    researchMetadata.value = {}
    errorMessage.value = ''
    agentTasks.value = []
  }
  
  return {
    // 状态
    sessions,
    currentSession,
    isLoading,
    isStreaming,
    researchProgress,
    currentStep,
    researchResults,
    researchMetadata,
    errorMessage,
    researchConfig,
    availableTools,
    agentTasks,
    
    // 计算属性
    activeSessions,
    completedSessions,
    failedSessions,
    researchStats,
    enabledTools,
    
    // 动作
    startResearch,
    updateSessionStatus,
    completeSession,
    failSession,
    getResearchSessions,
    getResearchStatus,
    exportResearch,
    searchResearch,
    setCurrentSession,
    updateResearchConfig,
    toggleTool,
    clearResults,
    stopCurrentResearch,
    $reset
  }
})
