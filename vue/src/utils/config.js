/**
 * 统一的配置管理
 * 解决API基础URL硬编码问题
 */

// API基础URL - 统一管理
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

// WebSocket基础URL
export const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080/ws'

// 请求超时配置（毫秒）
export const REQUEST_TIMEOUT = {
  DEFAULT: 60000,      // 默认60秒
  CHAT: 120000,        // 聊天120秒
  RESEARCH: 300000,    // 研究5分钟
  UPLOAD: 180000,      // 上传3分钟
}

// 分页配置
export const PAGINATION = {
  DEFAULT_LIMIT: 20,
  MAX_SESSION_LIMIT: 100,
  MAX_MESSAGE_LIMIT: 200,
}

// 上下文配置
export const CONTEXT_CONFIG = {
  WARNING_THRESHOLD: 0.8,  // 80%时警告
  MAX_TOKENS_DEFAULT: 128000,
}

// 研究配置
export const RESEARCH_CONFIG = {
  TIMEOUT_MS: 30 * 60 * 1000,  // 30分钟
  POLL_INTERVAL_MS: 2000,       // 轮询间隔2秒
}

// 默认模型/提供商
export const DEFAULT_MODEL = 'deepseek-chat'
export const DEFAULT_PROVIDER = 'deepseek'

// 上下文刷新间隔
export const CONTEXT_REFRESH_INTERVAL = 30000 // 30秒

// 本地存储键名
export const STORAGE_KEYS = {
  AUTH_TOKEN: 'auth_token',
  REFRESH_TOKEN: 'refresh_token',
  THEME: 'theme',
  LANGUAGE: 'language',
  AUTO_SAVE: 'autoSave',
  SEND_KEY: 'sendKey',
  AUTO_CLEAN_DAYS: 'autoCleanDays',
  USER_PREFERENCES: 'user_preferences',
  MEMORY_SETTINGS: 'memory_settings',
  CHAT_HISTORY: 'chat_history',
  CHAT_SESSIONS: 'chat_sessions',
}

// 错误码
export const ERROR_CODES = {
  // 认证错误
  UNAUTHORIZED: 'ERR_UNAUTHORIZED',
  FORBIDDEN: 'ERR_FORBIDDEN',
  TOKEN_EXPIRED: 'ERR_TOKEN_EXPIRED',
  TOKEN_INVALID: 'ERR_TOKEN_INVALID',
  ADMIN_REQUIRED: 'ERR_ADMIN_REQUIRED',
  
  // 会话错误
  SESSION_NOT_FOUND: 'ERR_SESSION_NOT_FOUND',
  SESSION_FORBIDDEN: 'ERR_SESSION_FORBIDDEN',
  
  // 配额错误
  CHAT_QUOTA_EXCEEDED: 'ERR_CHAT_QUOTA_EXCEEDED',
  RESEARCH_QUOTA_EXCEEDED: 'ERR_RESEARCH_QUOTA_EXCEEDED',
  RATE_LIMIT_EXCEEDED: 'ERR_RATE_LIMIT_EXCEEDED',
  
  // 上下文错误
  CONTEXT_OVERFLOW: 'ERR_CONTEXT_OVERFLOW',
  MESSAGE_TOO_LONG: 'ERR_MESSAGE_TOO_LONG',
  
  // LLM错误
  LLM_UNAVAILABLE: 'ERR_LLM_UNAVAILABLE',
  LLM_TIMEOUT: 'ERR_LLM_TIMEOUT',
  LLM_ERROR: 'ERR_LLM_ERROR',
  MODEL_NOT_SUPPORTED: 'ERR_MODEL_NOT_SUPPORTED',
  
  // 网络错误
  NETWORK: 'ERR_NETWORK',
  TIMEOUT: 'ERR_TIMEOUT',
  UNKNOWN: 'ERR_UNKNOWN',
}

export default {
  API_BASE_URL,
  WS_BASE_URL,
  REQUEST_TIMEOUT,
  PAGINATION,
  CONTEXT_CONFIG,
  RESEARCH_CONFIG,
  STORAGE_KEYS,
  ERROR_CODES,
}
