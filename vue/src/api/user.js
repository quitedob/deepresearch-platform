/**
 * 用户API接口 - 对齐后端 /api/v1/users
 */

import { authAPI, userAPI } from './index'

/**
 * 用户注册
 * @param {Object} userData - 注册数据
 * @param {string} userData.username - 用户名
 * @param {string} userData.email - 邮箱
 * @param {string} userData.password - 密码
 * @param {string} [userData.full_name] - 全名
 */
export async function register(userData) {
  return await authAPI.register(userData)
}

/**
 * 用户登录
 * @param {Object} credentials - 登录凭证
 * @param {string} credentials.username - 用户名或邮箱
 * @param {string} credentials.password - 密码
 */
export async function login(credentials) {
  const response = await authAPI.login(credentials)
  
  // 保存token
  if (response.access_token) {
    localStorage.setItem('auth_token', response.access_token)
    if (response.refresh_token) {
      localStorage.setItem('refresh_token', response.refresh_token)
    }
  }
  
  return response
}

/**
 * 获取当前用户信息
 */
export async function getCurrentUser() {
  return await userAPI.getProfile()
}

/**
 * 更新用户资料
 * @param {Object} profileData - 资料数据
 */
export async function updateProfile(profileData) {
  return await userAPI.updateProfile(profileData)
}

/**
 * 获取用户偏好设置
 */
export async function getPreferences() {
  return await userAPI.getPreferences()
}

/**
 * 更新用户偏好设置
 * @param {Object} preferences - 偏好设置
 */
export async function updatePreferences(preferences) {
  return await userAPI.updatePreferences(preferences)
}

/**
 * 刷新访问令牌
 * @param {string} refreshToken - 刷新令牌
 */
export async function refreshToken(refreshToken) {
  const response = await authAPI.refreshToken(refreshToken)
  
  // 更新token
  if (response.access_token) {
    localStorage.setItem('auth_token', response.access_token)
    if (response.refresh_token) {
      localStorage.setItem('refresh_token', response.refresh_token)
    }
  }
  
  return response
}

/**
 * 用户登出
 */
export async function logout() {
  try {
    await authAPI.logout()
  } finally {
    // 清除本地存储的token
    localStorage.removeItem('auth_token')
    localStorage.removeItem('refresh_token')
    sessionStorage.removeItem('auth_token')
  }
}

/**
 * 验证令牌
 */
export async function verifyToken() {
  return await authAPI.verifyToken()
}

/**
 * 检查是否已登录
 */
export function isLoggedIn() {
  return !!localStorage.getItem('auth_token') || !!sessionStorage.getItem('auth_token')
}

/**
 * 获取当前token
 */
export function getToken() {
  return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token')
}

// 导出userAPI对象以便直接使用
export { userAPI, authAPI }
