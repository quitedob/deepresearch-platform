/**
 * 模型管理 API
 * 所有模型配置从后端 API 动态获取，不硬编码
 */

import { apiClient } from './index'

// 缓存 providers 数据
let providersCache = null
let providersCacheTime = 0
const CACHE_TTL = 60000 // 1分钟缓存

/**
 * 获取所有可用模型
 * @param {string} [provider] - 过滤提供商
 */
export async function getModels(provider = null) {
  const params = provider ? { provider } : {}
  return await apiClient.get('/chat/models', { params })
}

/**
 * 获取提供商列表（包含完整模型配置，只返回启用的模型）
 * @param {boolean} [forceRefresh=false] - 强制刷新缓存
 */
export async function getProviders(forceRefresh = false) {
  const now = Date.now()
  
  // 使用缓存（缓存时间较短，确保及时更新）
  if (!forceRefresh && providersCache && (now - providersCacheTime) < CACHE_TTL) {
    return { data: providersCache }
  }
  
  const response = await apiClient.get('/llm/providers')
  
  // 更新缓存 - response 已经是解包后的数据
  // apiClient 的响应拦截器会返回 data.data（后端返回 {success: true, data: {providers, count}}）
  // 所以 response 直接就是 { count, providers } 对象
  if (response) {
    providersCache = response
    providersCacheTime = now
  }
  
  // 返回格式与缓存一致，包装在 { data: ... } 中
  return { data: response }
}

/**
 * 清除 providers 缓存
 */
export function clearProvidersCache() {
  providersCache = null
  providersCacheTime = 0
}

/**
 * 根据提供商ID获取深度思考模型
 * @param {string} providerId
 * @returns {Promise<string>}
 */
export async function getDeepThinkingModelByProvider(providerId) {
  try {
    const response = await getProviders()
    const providers = response.data?.providers || []
    const provider = providers.find(p => p.name === providerId)
    return provider?.deep_think_model || ''
  } catch (error) {
    console.error('获取深度思考模型失败:', error)
    return ''
  }
}

/**
 * 根据提供商ID获取默认模型
 * @param {string} providerId
 * @returns {Promise<string>}
 */
export async function getDefaultModelByProvider(providerId) {
  try {
    const response = await getProviders()
    const providers = response.data?.providers || []
    const provider = providers.find(p => p.name === providerId)
    return provider?.default_model || ''
  } catch (error) {
    console.error('获取默认模型失败:', error)
    return ''
  }
}

/**
 * 检查模型是否为深度思考模型
 * @param {string} modelName
 * @returns {Promise<boolean>}
 */
export async function isDeepThinkingModel(modelName) {
  try {
    const response = await getProviders()
    const providers = response.data?.providers || []
    for (const provider of providers) {
      const models = provider.models || []
      const model = models.find(m => m.name === modelName)
      if (model) {
        return model.is_deep_thinking === true
      }
    }
    return false
  } catch (error) {
    console.error('检查深度思考模型失败:', error)
    return false
  }
}

export default {
  getModels,
  getProviders,
  clearProvidersCache,
  getDeepThinkingModelByProvider,
  getDefaultModelByProvider,
  isDeepThinkingModel,
}
