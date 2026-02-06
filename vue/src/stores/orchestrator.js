import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '@/api/index'

export const useOrchestratorStore = defineStore('orchestrator', () => {
  // State
  const agents = ref([])
  const tasks = ref([])
  const orchestrationStatus = ref('stopped')
  const configuration = ref({})
  const metrics = ref({
    totalTasksCompleted: 0,
    totalTasksFailed: 0,
    averageProcessingTime: 0,
    overallSuccessRate: 0
  })

  // Computed
  const activeAgents = computed(() =>
    agents.value.filter(agent => agent.status === 'working')
  )

  const availableAgents = computed(() =>
    agents.value.filter(agent => agent.status === 'idle')
  )

  const runningTasks = computed(() =>
    tasks.value.filter(task => task.status === 'running')
  )

  const pendingTasks = computed(() =>
    tasks.value.filter(task => task.status === 'pending')
  )

  // Actions
  const initialize = async (config) => {
    try {
      configuration.value = config

      const response = await apiClient.post('/orchestrator/initialize', config)

      if (response.success) {
        orchestrationStatus.value = 'initialized'
        await refreshAgents()
        await refreshTasks()
        return true
      } else {
        throw new Error(response.error || 'Initialization failed')
      }
    } catch (error) {
      console.error('Failed to initialize orchestrator:', error)
      throw error
    }
  }

  const start = async () => {
    try {
      const response = await apiClient.post('/orchestrator/start')

      if (response.success) {
        orchestrationStatus.value = 'running'
        await refreshStatus()
        return true
      } else {
        throw new Error(response.error || 'Failed to start orchestration')
      }
    } catch (error) {
      console.error('Failed to start orchestration:', error)
      throw error
    }
  }

  const pause = async () => {
    try {
      const response = await apiClient.post('/orchestrator/pause')

      if (response.success) {
        orchestrationStatus.value = 'paused'
        return true
      } else {
        throw new Error(response.error || 'Failed to pause orchestration')
      }
    } catch (error) {
      console.error('Failed to pause orchestration:', error)
      throw error
    }
  }

  const stop = async () => {
    try {
      const response = await apiClient.post('/orchestrator/stop')

      if (response.success) {
        orchestrationStatus.value = 'stopped'
        await refreshStatus()
        return true
      } else {
        throw new Error(response.error || 'Failed to stop orchestration')
      }
    } catch (error) {
      console.error('Failed to stop orchestration:', error)
      throw error
    }
  }

  const addAgent = async (agentConfig) => {
    try {
      const response = await apiClient.post('/orchestrator/agents', agentConfig)

      if (response.success) {
        await refreshAgents()
        return response.agent
      } else {
        throw new Error(response.error || 'Failed to add agent')
      }
    } catch (error) {
      console.error('Failed to add agent:', error)
      throw error
    }
  }

  const removeAgent = async (agentId) => {
    try {
      const response = await apiClient.delete(`/orchestrator/agents/${agentId}`)

      if (response.success) {
        await refreshAgents()
        return true
      } else {
        throw new Error(response.error || 'Failed to remove agent')
      }
    } catch (error) {
      console.error('Failed to remove agent:', error)
      throw error
    }
  }

  const updateAgentConfiguration = async (agentId, config) => {
    try {
      const response = await apiClient.put(`/orchestrator/agents/${agentId}`, config)

      if (response.success) {
        await refreshAgents()
        return true
      } else {
        throw new Error(response.error || 'Failed to update agent configuration')
      }
    } catch (error) {
      console.error('Failed to update agent configuration:', error)
      throw error
    }
  }

  const assignTask = async (taskConfig) => {
    try {
      const response = await apiClient.post('/orchestrator/tasks', taskConfig)

      if (response.success) {
        await refreshTasks()
        return response.task
      } else {
        throw new Error(response.error || 'Failed to assign task')
      }
    } catch (error) {
      console.error('Failed to assign task:', error)
      throw error
    }
  }

  const cancelTask = async (taskId) => {
    try {
      const response = await apiClient.delete(`/orchestrator/tasks/${taskId}`)

      if (response.success) {
        await refreshTasks()
        return true
      } else {
        throw new Error(response.error || 'Failed to cancel task')
      }
    } catch (error) {
      console.error('Failed to cancel task:', error)
      throw error
    }
  }

  const removeTask = async (taskId) => {
    try {
      const response = await apiClient.delete(`/orchestrator/tasks/${taskId}?force=true`)

      if (response.success) {
        await refreshTasks()
        return true
      } else {
        throw new Error(response.error || 'Failed to remove task')
      }
    } catch (error) {
      console.error('Failed to remove task:', error)
      throw error
    }
  }

  const refreshAgents = async () => {
    try {
      const response = await apiClient.get('/orchestrator/agents')

      if (response.success) {
        agents.value = response.agents.map(agent => ({
          ...agent,
          recent_tasks: agent.recent_tasks || [],
          capabilities: agent.capabilities || []
        }))
      }
    } catch (error) {
      console.error('Failed to refresh agents:', error)
    }
  }

  const refreshTasks = async () => {
    try {
      const response = await apiClient.get('/orchestrator/tasks')

      if (response.success) {
        tasks.value = response.tasks
      }
    } catch (error) {
      console.error('Failed to refresh tasks:', error)
    }
  }

  const refreshStatus = async () => {
    try {
      const response = await apiClient.get('/orchestrator/status')

      if (response.success) {
        orchestrationStatus.value = response.status
        metrics.value = response.metrics

        // Refresh agents and tasks
        await Promise.all([
          refreshAgents(),
          refreshTasks()
        ])
      }
    } catch (error) {
      console.error('Failed to refresh status:', error)
    }
  }

  const getAgentById = (agentId) => {
    return agents.value.find(agent => agent.id === agentId)
  }

  const getTaskById = (taskId) => {
    return tasks.value.find(task => task.id === taskId)
  }

  const getAgentMetrics = (agentId) => {
    const agent = getAgentById(agentId)
    if (!agent) return null

    return {
      tasksCompleted: agent.tasks_completed,
      tasksFailed: agent.tasks_failed,
      successRate: agent.success_rate,
      averageProcessingTime: agent.average_processing_time,
      totalProcessingTime: agent.total_processing_time,
      currentTasks: agent.current_tasks,
      queuedTasks: agent.queued_tasks
    }
  }

  const getTaskHistory = async (agentId, limit = 10) => {
    try {
      const response = await apiClient.get(`/orchestrator/agents/${agentId}/tasks?limit=${limit}`)

      if (response.success) {
        return response.tasks
      } else {
        throw new Error(response.error || 'Failed to get task history')
      }
    } catch (error) {
      console.error('Failed to get task history:', error)
      throw error
    }
  }

  const getOrchestrationLogs = async (limit = 50) => {
    try {
      const response = await apiClient.get(`/orchestrator/logs?limit=${limit}`)

      if (response.success) {
        return response.logs
      } else {
        throw new Error(response.error || 'Failed to get logs')
      }
    } catch (error) {
      console.error('Failed to get orchestration logs:', error)
      throw error
    }
  }

  const reset = () => {
    agents.value = []
    tasks.value = []
    orchestrationStatus.value = 'stopped'
    configuration.value = {}
    metrics.value = {
      totalTasksCompleted: 0,
      totalTasksFailed: 0,
      averageProcessingTime: 0,
      overallSuccessRate: 0
    }
  }

  return {
    // State
    agents,
    tasks,
    orchestrationStatus,
    configuration,
    metrics,

    // Computed
    activeAgents,
    availableAgents,
    runningTasks,
    pendingTasks,

    // Actions
    initialize,
    start,
    pause,
    stop,
    addAgent,
    removeAgent,
    updateAgentConfiguration,
    assignTask,
    cancelTask,
    removeTask,
    refreshAgents,
    refreshTasks,
    refreshStatus,
    getAgentById,
    getTaskById,
    getAgentMetrics,
    getTaskHistory,
    getOrchestrationLogs,
    reset
  }
})