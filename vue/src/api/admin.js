/**
 * 管理员API接口
 */

import { apiClient } from './index'

// ==================== 统计信息 ====================

/**
 * 获取管理员统计信息
 */
export async function getAdminStats() {
  return await apiClient.get('/admin/stats')
}

// ==================== 用户管理 ====================

/**
 * 获取用户列表
 * @param {number} limit - 分页大小
 * @param {number} offset - 分页偏移
 */
export async function listUsers(limit = 20, offset = 0) {
  return await apiClient.get('/admin/users', { params: { limit, offset } })
}

/**
 * 更新用户状态
 * @param {string} userId - 用户ID
 * @param {string} status - 状态 (active, banned)
 */
export async function updateUserStatus(userId, status) {
  return await apiClient.put(`/admin/users/${userId}/status`, { status })
}

/**
 * 更新用户会员状态
 * @param {string} userId - 用户ID
 * @param {string} membershipType - 会员类型 (free, premium)
 * @param {number} validDays - 有效天数
 */
export async function updateUserMembership(userId, membershipType, validDays = 30) {
  return await apiClient.put(`/admin/users/${userId}/membership`, { 
    membership_type: membershipType,
    valid_days: validDays
  })
}

/**
 * 重置用户配额
 * @param {string} userId - 用户ID
 */
export async function resetUserQuota(userId) {
  return await apiClient.post(`/admin/users/${userId}/reset-quota`)
}

/**
 * 设置用户配额
 * @param {string} userId - 用户ID
 * @param {number} chatLimit - 聊天限制
 * @param {number} researchLimit - 研究限制
 */
export async function setUserQuota(userId, chatLimit, researchLimit) {
  return await apiClient.put(`/admin/users/${userId}/quota`, {
    chat_limit: chatLimit,
    research_limit: researchLimit
  })
}

// ==================== 聊天记录管理 ====================

/**
 * 获取用户聊天记录
 * @param {string} userId - 用户ID
 * @param {number} limit - 分页大小
 * @param {number} offset - 分页偏移
 */
export async function getUserChatHistory(userId, limit = 50, offset = 0) {
  return await apiClient.get(`/admin/users/${userId}/chat-history`, { params: { limit, offset } })
}

/**
 * 导出用户聊天记录
 * @param {string} userId - 用户ID
 */
export async function exportUserChatHistory(userId) {
  const response = await apiClient.get(`/admin/users/${userId}/chat-history/export`, {
    responseType: 'blob'
  })
  return response
}

// ==================== 激活码管理 ====================

/**
 * 获取激活码列表
 * @param {number} limit - 分页大小
 * @param {number} offset - 分页偏移
 */
export async function listActivationCodes(limit = 20, offset = 0) {
  return await apiClient.get('/admin/activation-codes', { params: { limit, offset } })
}

/**
 * 创建激活码
 * @param {Object} data - 激活码数据
 * @param {number} data.max_activations - 最大激活次数
 * @param {number} data.valid_days - 会员有效天数
 * @param {number} data.expires_in_days - 激活码过期天数
 * @param {string} data.code - 自定义激活码（可选）
 */
export async function createActivationCode(data) {
  return await apiClient.post('/admin/activation-codes', data)
}

/**
 * 获取激活码详情
 * @param {string} codeId - 激活码ID
 */
export async function getActivationCodeDetails(codeId) {
  return await apiClient.get(`/admin/activation-codes/${codeId}`)
}

/**
 * 更新激活码
 * @param {string} codeId - 激活码ID
 * @param {Object} data - 更新数据
 */
export async function updateActivationCode(codeId, data) {
  return await apiClient.put(`/admin/activation-codes/${codeId}`, data)
}

/**
 * 删除激活码
 * @param {string} codeId - 激活码ID
 */
export async function deleteActivationCode(codeId) {
  return await apiClient.delete(`/admin/activation-codes/${codeId}`)
}

// ==================== 通知管理 ====================

/**
 * 获取通知列表
 * @param {number} limit - 分页大小
 * @param {number} offset - 分页偏移
 */
export async function listNotifications(limit = 20, offset = 0) {
  return await apiClient.get('/admin/notifications', { params: { limit, offset } })
}

/**
 * 创建通知
 * @param {Object} data - 通知数据
 * @param {string} data.title - 标题
 * @param {string} data.content - 内容
 * @param {string} data.type - 类型 (system, announce, alert)
 * @param {boolean} data.is_global - 是否全局通知
 */
export async function createNotification(data) {
  return await apiClient.post('/admin/notifications', data)
}

/**
 * 删除通知
 * @param {string} notificationId - 通知ID
 */
export async function deleteNotification(notificationId) {
  return await apiClient.delete(`/admin/notifications/${notificationId}`)
}

// ==================== 模型配置管理 ====================

/**
 * 获取提供商配置
 */
export async function getProviderConfigs() {
  return await apiClient.get('/admin/providers')
}

/**
 * 更新提供商配置
 * @param {string} provider - 提供商名称
 * @param {boolean} isEnabled - 是否启用
 */
export async function updateProviderConfig(provider, isEnabled) {
  return await apiClient.put('/admin/providers', { provider, is_enabled: isEnabled })
}

/**
 * 获取模型配置
 */
export async function getModelConfigs() {
  return await apiClient.get('/admin/models')
}

/**
 * 更新模型配置
 * @param {string} provider - 提供商名称
 * @param {string} modelName - 模型名称
 * @param {boolean} isEnabled - 是否启用
 */
export async function updateModelConfig(provider, modelName, isEnabled) {
  return await apiClient.put('/admin/models', { provider, model_name: modelName, is_enabled: isEnabled })
}

/**
 * 批量更新模型配置
 * @param {Array} configs - 配置数组
 */
export async function batchUpdateModelConfigs(configs) {
  return await apiClient.put('/admin/models/batch', { configs })
}

/**
 * 测试模型连接
 * @param {string} provider - 提供商名称
 * @param {string} model - 模型名称
 */
export async function testModel(provider, model) {
  return await apiClient.post('/admin/models/test', { provider, model })
}

/**
 * 获取所有已注册的模型（管理端专用）
 */
export async function getAllRegisteredModels() {
  return await apiClient.get('/admin/models/registered')
}

/**
 * 同步模型到数据库
 */
export async function syncModelsToDatabase() {
  return await apiClient.post('/admin/models/sync')
}

// ==================== 配额配置管理 ====================

/**
 * 获取所有配额配置
 */
export async function getQuotaConfigs() {
  return await apiClient.get('/admin/quota-configs')
}

/**
 * 更新配额配置（按会员层级）
 * @param {Object} data - 配额配置
 * @param {string} data.membership_type - 会员类型 (free, premium)
 * @param {number} data.chat_limit - 聊天配额
 * @param {number} data.research_limit - 研究配额
 * @param {number} data.reset_period_hours - 重置周期（小时）
 * @param {boolean} data.apply_to_all - 是否应用到所有该类型用户
 */
export async function updateQuotaConfig(data) {
  return await apiClient.put('/admin/quota-configs', data)
}

/**
 * 设置用户自定义配额
 * @param {string} userId - 用户ID
 * @param {number} chatLimit - 聊天配额
 * @param {number} researchLimit - 研究配额
 * @param {boolean} resetUsage - 是否重置使用量
 */
export async function setUserCustomQuota(userId, chatLimit, researchLimit, resetUsage = false) {
  return await apiClient.put(`/admin/users/${userId}/custom-quota`, {
    chat_limit: chatLimit,
    research_limit: researchLimit,
    reset_usage: resetUsage
  })
}

/**
 * 批量设置用户配额
 * @param {Array} userIds - 用户ID数组
 * @param {number} chatLimit - 聊天配额
 * @param {number} researchLimit - 研究配额
 * @param {boolean} resetUsage - 是否重置使用量
 */
export async function batchSetUserQuota(userIds, chatLimit, researchLimit, resetUsage = false) {
  return await apiClient.put('/admin/users/batch-quota', {
    user_ids: userIds,
    chat_limit: chatLimit,
    research_limit: researchLimit,
    reset_usage: resetUsage
  })
}

/**
 * 批量更新用户状态
 * @param {Array} userIds - 用户ID数组
 * @param {string} status - 状态 (active, banned)
 */
export async function batchUpdateUserStatus(userIds, status) {
  return await apiClient.put('/admin/users/batch-status', {
    user_ids: userIds,
    status: status
  })
}

/**
 * 批量重置用户配额
 * @param {Array} userIds - 用户ID数组
 */
export async function batchResetUserQuotas(userIds) {
  return await apiClient.post('/admin/users/batch-reset-quota', {
    user_ids: userIds
  })
}

/**
 * 批量删除用户
 * @param {Array} userIds - 用户ID数组
 */
export async function batchDeleteUsers(userIds) {
  return await apiClient.delete('/admin/users/batch', {
    data: { user_ids: userIds }
  })
}

// ==================== AI出题配置管理 ====================

/**
 * 更新AI出题配置
 * @param {string} defaultProvider - 默认提供商
 * @param {string} defaultModel - 默认模型
 */
export async function updateAIQuestionConfigAPI(defaultProvider, defaultModel) {
  return await apiClient.put('/admin/ai/question-config', {
    default_provider: defaultProvider,
    default_model: defaultModel
  })
}

export default {
  getAdminStats,
  listUsers,
  updateUserStatus,
  updateUserMembership,
  resetUserQuota,
  setUserQuota,
  getUserChatHistory,
  exportUserChatHistory,
  listActivationCodes,
  createActivationCode,
  getActivationCodeDetails,
  updateActivationCode,
  deleteActivationCode,
  listNotifications,
  createNotification,
  deleteNotification,
  getProviderConfigs,
  updateProviderConfig,
  getModelConfigs,
  updateModelConfig,
  batchUpdateModelConfigs,
  testModel,
  getAllRegisteredModels,
  syncModelsToDatabase,
  getQuotaConfigs,
  updateQuotaConfig,
  setUserCustomQuota,
  batchSetUserQuota,
  batchUpdateUserStatus,
  batchResetUserQuotas,
  batchDeleteUsers,
  updateAIQuestionConfigAPI
}
