/**
 * 统一的Token管理工具
 * 解决Token管理分散的问题
 */

const TOKEN_KEY = 'auth_token'
const REFRESH_TOKEN_KEY = 'refresh_token'

/**
 * 获取认证Token
 * @returns {string|null}
 */
export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || sessionStorage.getItem(TOKEN_KEY)
}

/**
 * 设置认证Token
 * @param {string} token 
 * @param {boolean} remember - 是否记住登录（使用localStorage）
 */
export function setAuthToken(token, remember = true) {
  if (remember) {
    localStorage.setItem(TOKEN_KEY, token)
    sessionStorage.removeItem(TOKEN_KEY)
  } else {
    sessionStorage.setItem(TOKEN_KEY, token)
    localStorage.removeItem(TOKEN_KEY)
  }
}

/**
 * 移除认证Token
 */
export function removeAuthToken() {
  localStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(TOKEN_KEY)
}

/**
 * 获取刷新Token
 * @returns {string|null}
 */
export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY) || sessionStorage.getItem(REFRESH_TOKEN_KEY)
}

/**
 * 设置刷新Token
 * @param {string} token 
 * @param {boolean} remember
 */
export function setRefreshToken(token, remember = true) {
  if (remember) {
    localStorage.setItem(REFRESH_TOKEN_KEY, token)
    sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  } else {
    sessionStorage.setItem(REFRESH_TOKEN_KEY, token)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }
}

/**
 * 移除刷新Token
 */
export function removeRefreshToken() {
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
}

/**
 * 清除所有Token
 */
export function clearTokens() {
  removeAuthToken()
  removeRefreshToken()
}

/**
 * 检查是否已登录
 * @returns {boolean}
 */
export function isAuthenticated() {
  return !!getAuthToken()
}

/**
 * 解析JWT Token获取用户信息
 * @param {string} token 
 * @returns {object|null}
 */
export function parseJwtPayload(token) {
  if (!token) return null
  
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    )
    return JSON.parse(jsonPayload)
  } catch (e) {
    console.error('解析JWT失败:', e)
    return null
  }
}

/**
 * 检查Token是否过期
 * @param {string} token 
 * @returns {boolean}
 */
export function isTokenExpired(token) {
  const payload = parseJwtPayload(token)
  if (!payload || !payload.exp) return true
  
  // 提前5分钟认为过期
  const expirationTime = payload.exp * 1000 - 5 * 60 * 1000
  return Date.now() > expirationTime
}

// 兼容旧API名称
export const clearAuth = clearTokens

export default {
  getAuthToken,
  setAuthToken,
  removeAuthToken,
  getRefreshToken,
  setRefreshToken,
  removeRefreshToken,
  clearTokens,
  clearAuth, // 兼容
  isAuthenticated,
  parseJwtPayload,
  isTokenExpired
}
