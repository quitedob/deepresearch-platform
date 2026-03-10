<template>
  <div class="model-config">
    <div class="section-header">
      <h2>模型配置</h2>
      <p class="description">控制哪些AI模型对用户可见</p>
      <div class="header-actions">
        <button class="btn-outline" @click="refreshAll" :disabled="loading">
          🔄 刷新
        </button>
        <button class="btn-outline" @click="syncModels" :disabled="loading">
          🔄 同步模型
        </button>
        <button class="btn-primary" @click="enableAllModels" :disabled="loading">
          ✅ 全部启用
        </button>
        <button class="btn-secondary" @click="disableAllModels" :disabled="loading">
          🚫 全部禁用
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">加载中...</div>

    <div v-else>
      <!-- AI出题默认配置 -->
      <div class="ai-question-section">
        <h3>🎯 AI出题默认配置</h3>
        <p class="section-description">设置AI出题功能的默认LLM提供商和模型</p>
        <div class="ai-question-config">
          <div class="config-row">
            <label>默认提供商</label>
            <select v-model="aiQuestionConfig.default_provider" @change="updateAIQuestionConfig">
              <option value="deepseek">DeepSeek</option>
              <option value="zhipu">智谱AI</option>
              <option value="ollama">Ollama</option>
              <option value="openrouter">OpenRouter</option>
            </select>
          </div>
          <div class="config-row">
            <label>默认模型</label>
            <select v-model="aiQuestionConfig.default_model" @change="updateAIQuestionConfig">
              <option v-for="model in availableModelsForProvider" :key="model" :value="model">
                {{ model }}
              </option>
            </select>
          </div>
          <div class="config-status" v-if="aiQuestionConfigSaved">
            ✅ 配置已保存
          </div>
        </div>
      </div>

      <div class="providers-section">
        <h3>提供商配置</h3>
        <div class="provider-list">
          <div v-for="provider in providers" :key="provider.provider" class="provider-item">
            <div class="provider-info">
              <span class="provider-name">{{ provider.display_name }}</span>
              <span class="provider-id">({{ provider.provider }})</span>
            </div>
            <label class="switch">
              <input 
                type="checkbox" 
                :checked="provider.is_enabled"
                @change="toggleProvider(provider)"
              />
              <span class="slider"></span>
            </label>
          </div>
        </div>
      </div>

      <div class="models-section">
        <h3>模型配置</h3>
        <div class="model-groups">
          <div v-for="(models, providerName) in groupedModels" :key="providerName" class="model-group">
            <div class="group-header">
              <h4>{{ getProviderDisplayName(providerName) }}</h4>
              <div class="group-actions">
                <button class="btn-xs" @click="enableProviderModels(providerName)">全部启用</button>
                <button class="btn-xs btn-outline" @click="disableProviderModels(providerName)">全部禁用</button>
              </div>
            </div>
            <div class="model-list">
              <div v-for="model in models" :key="model.model_name" class="model-item">
                <div class="model-info">
                  <span class="model-name">{{ model.display_name }}</span>
                  <span class="model-id">({{ model.model_name }})</span>
                  <span v-if="testResults[model.model_name]" 
                        :class="['test-status', testResults[model.model_name].success ? 'success' : 'error']">
                    {{ testResults[model.model_name].success ? '✅' : '❌' }}
                    {{ testResults[model.model_name].duration }}ms
                  </span>
                </div>
                <div class="model-actions">
                  <button class="btn-xs btn-test" @click="testModelConnection(model)" :disabled="testing[model.model_name]">
                    {{ testing[model.model_name] ? '测试中...' : '🧪 测试' }}
                  </button>
                  <label class="switch">
                    <input 
                      type="checkbox" 
                      :checked="model.is_enabled"
                      @change="toggleModel(model)"
                    />
                    <span class="slider"></span>
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getProviderConfigs, updateProviderConfig, getModelConfigs, updateModelConfig, batchUpdateModelConfigs, updateAIQuestionConfigAPI, testModel, syncModelsToDatabase } from '@/api/admin'
import { aiQuestionAPI } from '@/api/index'
import { getProviders, clearProvidersCache } from '@/api/model'

const providers = ref([])
const models = ref([])
const loading = ref(false)
const testing = ref({})
const testResults = ref({})

// AI出题配置
const aiQuestionConfig = ref({
  default_provider: 'deepseek',
  default_model: 'deepseek-chat'
})
const aiQuestionConfigSaved = ref(false)

// 从 API 获取的 provider 模型映射
const providerModelsMap = ref({})

// 根据选择的provider获取可用模型（从 API 动态获取）
const availableModelsForProvider = computed(() => {
  return providerModelsMap.value[aiQuestionConfig.value.default_provider] || []
})

// 加载 provider 模型映射
const loadProviderModels = async () => {
  try {
    const response = await getProviders()
    const providersData = response.data?.providers || []
    const mapping = {}
    providersData.forEach(p => {
      const modelNames = (p.models || []).map(m => m.name || m)
      mapping[p.name] = modelNames
    })
    providerModelsMap.value = mapping
  } catch (error) {
    console.error('加载 provider 模型映射失败:', error)
  }
}

const groupedModels = computed(() => {
  const groups = {}
  models.value.forEach(model => {
    if (!groups[model.provider]) {
      groups[model.provider] = []
    }
    groups[model.provider].push(model)
  })
  return groups
})

const getProviderDisplayName = (providerName) => {
  const provider = providers.value.find(p => p.provider === providerName)
  return provider?.display_name || providerName
}

const loadProviders = async () => {
  try {
    const response = await getProviderConfigs()
    providers.value = (response.data || response).providers || []
  } catch (error) {
    console.error('加载提供商配置失败:', error)
    handleAPIError(error, '加载提供商配置')
  }
}

const loadModels = async () => {
  try {
    const response = await getModelConfigs()
    models.value = (response.data || response).models || []
  } catch (error) {
    console.error('加载模型配置失败:', error)
    handleAPIError(error, '加载模型配置')
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([loadProviders(), loadModels(), loadAIQuestionConfig(), loadProviderModels()])
  } finally {
    loading.value = false
  }
}

// 加载AI出题配置
const loadAIQuestionConfig = async () => {
  try {
    const response = await aiQuestionAPI.getConfig()
    if (response.config) {
      aiQuestionConfig.value = {
        default_provider: response.config.default_provider || 'deepseek',
        default_model: response.config.default_model || 'deepseek-chat'
      }
    }
  } catch (error) {
    console.error('加载AI出题配置失败:', error)
  }
}

// 更新AI出题配置
const updateAIQuestionConfig = async () => {
  try {
    await updateAIQuestionConfigAPI(
      aiQuestionConfig.value.default_provider,
      aiQuestionConfig.value.default_model
    )
    aiQuestionConfigSaved.value = true
    setTimeout(() => {
      aiQuestionConfigSaved.value = false
    }, 2000)
  } catch (error) {
    console.error('更新AI出题配置失败:', error)
    handleAPIError(error, '更新AI出题配置')
  }
}

const toggleProvider = async (provider) => {
  try {
    await updateProviderConfig(provider.provider, !provider.is_enabled)
    provider.is_enabled = !provider.is_enabled
    clearProvidersCache()
    localStorage.setItem('model_config_updated', Date.now().toString())
  } catch (error) {
    console.error('更新提供商配置失败:', error)
    handleAPIError(error, '更新提供商配置')
  }
}

const toggleModel = async (model) => {
  try {
    await updateModelConfig(model.provider, model.model_name, !model.is_enabled)
    model.is_enabled = !model.is_enabled
    // 清除前端缓存，确保其他页面获取最新配置
    clearProvidersCache()
    // 通知其他标签页模型配置已变更
    localStorage.setItem('model_config_updated', Date.now().toString())
  } catch (error) {
    console.error('更新模型配置失败:', error)
    handleAPIError(error, '更新模型配置')
  }
}

// 批量操作 - 使用批量API替代循环调用
const enableAllModels = async () => {
  if (!confirm('确定要启用所有模型吗？')) return
  loading.value = true
  try {
    // 筛选需要启用的模型
    const configs = models.value
      .filter(m => !m.is_enabled)
      .map(m => ({
        provider: m.provider,
        model_name: m.model_name,
        is_enabled: true
      }))
    
    if (configs.length === 0) {
      alert('所有模型已启用')
      return
    }
    
    // 使用单个批量API调用替代N个单独调用
    await batchUpdateModelConfigs(configs)
    
    // 更新本地状态
    models.value.forEach(m => m.is_enabled = true)
    clearProvidersCache()
    localStorage.setItem('model_config_updated', Date.now().toString())
    alert(`已启用 ${configs.length} 个模型`)
  } catch (error) {
    handleAPIError(error, '批量启用模型')
  } finally {
    loading.value = false
  }
}

const disableAllModels = async () => {
  if (!confirm('确定要禁用所有模型吗？')) return
  loading.value = true
  try {
    // 筛选需要禁用的模型
    const configs = models.value
      .filter(m => m.is_enabled)
      .map(m => ({
        provider: m.provider,
        model_name: m.model_name,
        is_enabled: false
      }))
    
    if (configs.length === 0) {
      alert('所有模型已禁用')
      return
    }
    
    // 使用单个批量API调用替代N个单独调用
    await batchUpdateModelConfigs(configs)
    
    // 更新本地状态
    models.value.forEach(m => m.is_enabled = false)
    clearProvidersCache()
    localStorage.setItem('model_config_updated', Date.now().toString())
    alert(`已禁用 ${configs.length} 个模型`)
  } catch (error) {
    handleAPIError(error, '批量禁用模型')
  } finally {
    loading.value = false
  }
}

const enableProviderModels = async (providerName) => {
  const configs = models.value
    .filter(m => m.provider === providerName && !m.is_enabled)
    .map(m => ({
      provider: m.provider,
      model_name: m.model_name,
      is_enabled: true
    }))
  
  if (configs.length === 0) return
  
  try {
    await batchUpdateModelConfigs(configs)
    models.value
      .filter(m => m.provider === providerName)
      .forEach(m => m.is_enabled = true)
    clearProvidersCache()
    localStorage.setItem('model_config_updated', Date.now().toString())
  } catch (error) {
    handleAPIError(error, '批量启用模型')
  }
}

const disableProviderModels = async (providerName) => {
  const configs = models.value
    .filter(m => m.provider === providerName && m.is_enabled)
    .map(m => ({
      provider: m.provider,
      model_name: m.model_name,
      is_enabled: false
    }))
  
  if (configs.length === 0) return
  
  try {
    await batchUpdateModelConfigs(configs)
    models.value
      .filter(m => m.provider === providerName)
      .forEach(m => m.is_enabled = false)
    clearProvidersCache()
    localStorage.setItem('model_config_updated', Date.now().toString())
  } catch (error) {
    handleAPIError(error, '批量禁用模型')
  }
}

// 测试模型连接
const testModelConnection = async (model) => {
  testing.value[model.model_name] = true
  try {
    const response = await testModel(model.provider, model.model_name)
    // 响应可能在 response.data 或直接在 response 中
    const result = response.data || response
    testResults.value[model.model_name] = {
      success: result.success,
      duration: result.duration,
      error: result.error
    }
  } catch (error) {
    // 处理 API 错误响应
    const errorData = error.response?.data
    testResults.value[model.model_name] = {
      success: false,
      duration: 0,
      error: errorData?.message || errorData?.error?.message || error.message
    }
  } finally {
    testing.value[model.model_name] = false
  }
}

// 同步模型到数据库
const syncModels = async () => {
  if (!confirm('确定要同步已注册的模型到数据库吗？')) return
  loading.value = true
  try {
    const response = await syncModelsToDatabase()
    alert(`同步完成！新增 ${response.added_count} 个模型`)
    await refreshAll()
  } catch (error) {
    handleAPIError(error, '同步模型')
  } finally {
    loading.value = false
  }
}

// 统一错误处理
const handleAPIError = (error, context) => {
  let message = `${context}失败`
  if (error.response?.data?.error) {
    const apiError = error.response.data.error
    message = apiError.message || message
  }
  alert(message)
}

onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.model-config { width: 100%; }
.section-header { margin-bottom: 24px; display: flex; flex-wrap: wrap; justify-content: space-between; align-items: flex-start; gap: 16px; }
.section-header h2 { font-size: 18px; font-weight: 600; margin-bottom: 4px; }
.description { color: #6b7280; font-size: 14px; }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; }

/* AI出题配置 */
.ai-question-section { margin-bottom: 32px; padding: 20px; background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%); border-radius: 12px; border: 1px solid #bae6fd; }
.ai-question-section h3 { font-size: 16px; font-weight: 600; margin-bottom: 8px; color: #0369a1; }
.section-description { color: #6b7280; font-size: 13px; margin-bottom: 16px; }
.ai-question-config { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; }
.config-row { display: flex; align-items: center; gap: 8px; }
.config-row label { font-size: 14px; font-weight: 500; color: var(--text-secondary); }
.config-row select { padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; font-size: 14px; background: var(--input-bg); min-width: 150px; color: var(--text-primary); }
.config-row select:focus { outline: none; border-color: var(--accent-blue); }
.config-status { color: #059669; font-size: 13px; font-weight: 500; }

.providers-section, .models-section { margin-bottom: 32px; }
h3 { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: var(--text-primary); }
h4 { font-size: 14px; font-weight: 600; margin: 0; color: var(--text-secondary); }
.provider-list, .model-list { display: flex; flex-direction: column; gap: 12px; }
.provider-item, .model-item { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: var(--secondary-bg); border-radius: 8px; }
.provider-info, .model-info { display: flex; align-items: center; gap: 8px; }
.provider-name, .model-name { font-weight: 500; }
.provider-id, .model-id { color: var(--text-tertiary); font-size: 13px; }
.model-groups { display: flex; flex-direction: column; gap: 24px; }
.model-group { padding: 16px; background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 12px; }
.group-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.group-actions { display: flex; gap: 8px; }
.switch { position: relative; display: inline-block; width: 48px; height: 24px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--tertiary-bg); transition: .3s; border-radius: 24px; }
.slider:before { position: absolute; content: ""; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: .3s; border-radius: 50%; }
input:checked + .slider { background-color: var(--accent-blue); }
input:checked + .slider:before { transform: translateX(24px); }
.loading-state { text-align: center; padding: 40px; color: #6b7280; }
.btn-primary { padding: 8px 16px; background: var(--button-bg); color: var(--button-text); border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-secondary { padding: 8px 16px; background: var(--secondary-bg); color: var(--text-primary); border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-outline { padding: 8px 16px; background: var(--card-bg); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-xs { padding: 4px 8px; font-size: 12px; background: var(--accent-blue); color: white; border: none; border-radius: 4px; cursor: pointer; }
.btn-xs.btn-outline { background: var(--card-bg); color: var(--text-primary); border: 1px solid var(--border-color); }
.btn-primary:disabled, .btn-secondary:disabled, .btn-outline:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-test { background: #f59e0b; color: white; }
.btn-test:hover { background: #d97706; }
.model-actions { display: flex; align-items: center; gap: 8px; }
.test-status { font-size: 12px; margin-left: 8px; }
.test-status.success { color: #059669; }
.test-status.error { color: #dc2626; }
</style>
