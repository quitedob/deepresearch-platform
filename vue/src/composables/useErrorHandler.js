/**
 * 统一错误处理 Composable
 * 提供错误解析、用户友好提示、建议操作等功能
 * 
 * 注意：此文件是错误处理的唯一来源，store/index.js 中的错误处理已移除重复定义
 */
import { ref, readonly } from 'vue'
import { useRouter } from 'vue-router'
import { clearTokens } from '@/utils/token'

// 错误码到用户友好消息的映射（统一定义，避免重复）
const ERROR_MESSAGES = {
  // 认证错误
  'ERR_UNAUTHORIZED': '请先登录后再操作',
  'ERR_FORBIDDEN': '您没有权限执行此操作',
  'ERR_TOKEN_EXPIRED': '登录已过期，请重新登录',
  'ERR_TOKEN_INVALID': '登录凭证无效，请重新登录',
  'ERR_ADMIN_REQUIRED': '需要管理员权限',
  
  // 会话错误
  'ERR_SESSION_NOT_FOUND': '会话不存在或已被删除',
  'ERR_SESSION_FORBIDDEN': '您无权访问此会话',
  'ERR_SESSION_EXPIRED': '会话已过期',
  
  // 配额错误
  'ERR_CHAT_QUOTA_EXCEEDED': '聊天次数已用完，请升级会员或等待配额重置',
  'ERR_RESEARCH_QUOTA_EXCEEDED': '深度研究次数已用完，请升级会员或等待配额重置',
  'ERR_QUOTA_EXCEEDED': '配额已用完',
  'ERR_RATE_LIMIT_EXCEEDED': '请求过于频繁，请稍后重试',
  
  // 上下文错误
  'ERR_CONTEXT_OVERFLOW': '对话上下文过长，建议创建新会话继续对话',
  'ERR_MESSAGE_TOO_LONG': '消息内容过长，请精简后重试',
  'ERR_QUERY_TOO_LONG': '查询内容过长，请精简后重试',
  
  // 模型错误
  'ERR_MODEL_NOT_SUPPORTED': '当前模型暂不可用，请切换其他模型',
  'ERR_MODEL_NOT_FOUND': '模型不存在',
  'ERR_LLM_UNAVAILABLE': 'AI服务暂时不可用，请稍后重试',
  'ERR_LLM_TIMEOUT': 'AI响应超时，请稍后重试',
  'ERR_LLM_ERROR': 'AI服务调用失败，请稍后重试',
  
  // 研究错误
  'ERR_RESEARCH_NOT_FOUND': '研究会话不存在或已被删除',
  'ERR_RESEARCH_FAILED': '研究任务执行失败',
  'ERR_RESEARCH_TIMEOUT': '研究任务超时',
  
  // 用户错误
  'ERR_USER_NOT_FOUND': '用户不存在',
  'ERR_USER_BANNED': '账户已被禁用，请联系管理员',
  'ERR_INVALID_CREDENTIALS': '用户名或密码错误',
  'ERR_EMAIL_EXISTS': '邮箱已被注册',
  'ERR_USERNAME_EXISTS': '用户名已被使用',
  
  // 激活码错误
  'ERR_ACTIVATION_CODE_INVALID': '激活码无效',
  'ERR_ACTIVATION_CODE_EXPIRED': '激活码已过期',
  'ERR_ACTIVATION_CODE_USED': '激活码已被使用',
  
  // 通用错误
  'ERR_INVALID_REQUEST': '请求参数无效，请检查输入',
  'ERR_INVALID_PARAMETER': '参数格式不正确',
  'ERR_MISSING_PARAMETER': '缺少必要参数',
  'ERR_INTERNAL_ERROR': '服务器内部错误，请稍后重试',
  'ERR_SERVICE_UNAVAILABLE': '服务暂时不可用，请稍后重试',
  'ERR_NOT_FOUND': '请求的资源不存在',
  'ERR_ALREADY_EXISTS': '资源已存在',
  'ERR_CONFLICT': '操作冲突，请刷新后重试',
  
  // 网络错误（扩展分类）
  'ERR_NETWORK': '网络连接失败，请检查网络设置',
  'ERR_NETWORK_DNS': 'DNS解析失败，请检查网络设置或域名配置',
  'ERR_NETWORK_CONNECTION': '无法连接到服务器，请检查网络连接',
  'ERR_NETWORK_REFUSED': '服务器拒绝连接，服务可能未启动',
  'ERR_NETWORK_RESET': '连接被重置，请稍后重试',
  'ERR_NETWORK_SSL': 'SSL/TLS证书错误，请检查安全设置',
  'ERR_TIMEOUT': '请求超时，请稍后重试',
  'ERR_TIMEOUT_CONNECT': '连接超时，服务器响应过慢',
  'ERR_TIMEOUT_READ': '读取超时，数据传输过慢',
  'ERR_UNKNOWN': '未知错误，请稍后重试',
  
  // 乐观锁错误
  'ERR_OPTIMISTIC_LOCK': '数据已被其他操作修改，请刷新后重试',
  'ERR_CONCURRENT_MODIFICATION': '并发修改冲突，请刷新后重试',
}

// 错误码到建议操作的映射
const ERROR_SUGGESTIONS = {
  'ERR_UNAUTHORIZED': 'relogin',
  'ERR_TOKEN_EXPIRED': 'relogin',
  'ERR_TOKEN_INVALID': 'relogin',
  'ERR_SESSION_NOT_FOUND': 'refresh_sessions',
  'ERR_CHAT_QUOTA_EXCEEDED': 'upgrade_membership',
  'ERR_RESEARCH_QUOTA_EXCEEDED': 'upgrade_membership',
  'ERR_CONTEXT_OVERFLOW': 'create_new_session',
  'ERR_MODEL_NOT_SUPPORTED': 'switch_model',
  'ERR_LLM_UNAVAILABLE': 'retry_later',
  'ERR_RATE_LIMIT_EXCEEDED': 'wait_and_retry',
  'ERR_USER_BANNED': 'contact_support',
}

/**
 * 解析网络错误，返回更详细的错误分类
 */
function parseNetworkError(error) {
  const errorMessage = error.message?.toLowerCase() || ''
  
  // DNS错误
  if (errorMessage.includes('dns') || errorMessage.includes('getaddrinfo')) {
    return {
      code: 'ERR_NETWORK_DNS',
      message: 'DNS解析失败',
      suggestion: 'check_network'
    }
  }
  
  // 连接被拒绝
  if (errorMessage.includes('econnrefused') || errorMessage.includes('connection refused')) {
    return {
      code: 'ERR_NETWORK_REFUSED',
      message: '服务器拒绝连接',
      suggestion: 'retry_later'
    }
  }
  
  // 连接重置
  if (errorMessage.includes('econnreset') || errorMessage.includes('connection reset')) {
    return {
      code: 'ERR_NETWORK_RESET',
      message: '连接被重置',
      suggestion: 'retry'
    }
  }
  
  // SSL/TLS错误
  if (errorMessage.includes('ssl') || errorMessage.includes('certificate') || errorMessage.includes('tls')) {
    return {
      code: 'ERR_NETWORK_SSL',
      message: 'SSL证书错误',
      suggestion: 'check_network'
    }
  }
  
  // 默认网络错误
  return {
    code: 'ERR_NETWORK_CONNECTION',
    message: '网络连接失败',
    suggestion: 'check_network'
  }
}

/**
 * 解析超时错误，返回更详细的错误分类
 */
function parseTimeoutError(error) {
  const errorMessage = error.message?.toLowerCase() || ''
  
  if (errorMessage.includes('connect')) {
    return {
      code: 'ERR_TIMEOUT_CONNECT',
      message: '连接超时',
      suggestion: 'retry_later'
    }
  }
  
  if (errorMessage.includes('read') || errorMessage.includes('response')) {
    return {
      code: 'ERR_TIMEOUT_READ',
      message: '读取超时',
      suggestion: 'retry'
    }
  }
  
  return {
    code: 'ERR_TIMEOUT',
    message: '请求超时',
    suggestion: 'retry'
  }
}

/**
 * 解析API错误
 */
function parseAPIError(error) {
  // 后端统一错误格式
  if (error.response?.data?.error) {
    const apiError = error.response.data.error
    return {
      code: apiError.code || 'ERR_UNKNOWN',
      message: apiError.message || '未知错误',
      details: apiError.details || '',
      field: apiError.field || '',
      extra: apiError.extra || {},
      httpStatus: error.response.status,
      suggestion: ERROR_SUGGESTIONS[apiError.code] || null,
      userMessage: ERROR_MESSAGES[apiError.code] || apiError.message || '操作失败',
      // 添加资源类型信息（如果有）
      resourceType: apiError.extra?.resource_type || null,
      requiredPermission: apiError.extra?.required_permission || null
    }
  }
  
  // 网络错误 - 使用详细分类
  if (error.code === 'ERR_NETWORK') {
    const networkError = parseNetworkError(error)
    return {
      ...networkError,
      httpStatus: 0,
      userMessage: ERROR_MESSAGES[networkError.code] || ERROR_MESSAGES['ERR_NETWORK']
    }
  }
  
  // 超时错误 - 使用详细分类
  if (error.code === 'ECONNABORTED') {
    const timeoutError = parseTimeoutError(error)
    return {
      ...timeoutError,
      httpStatus: 0,
      userMessage: ERROR_MESSAGES[timeoutError.code] || ERROR_MESSAGES['ERR_TIMEOUT']
    }
  }
  
  // 其他错误
  return {
    code: 'ERR_UNKNOWN',
    message: error.message || '未知错误',
    httpStatus: error.response?.status || 0,
    suggestion: null,
    userMessage: error.message || ERROR_MESSAGES['ERR_UNKNOWN']
  }
}

/**
 * 错误处理 Composable
 */
export function useErrorHandler() {
  const router = useRouter()
  const lastError = ref(null)
  const isError = ref(false)
  
  /**
   * 处理错误
   * @param {Error} error - 错误对象
   * @param {string} context - 错误上下文描述
   * @param {Object} options - 选项
   * @returns {Object} 解析后的错误信息
   */
  function handleError(error, context = '', options = {}) {
    const { showAlert = false, autoRedirect = true } = options
    
    const parsedError = parseAPIError(error)
    parsedError.context = context
    
    lastError.value = parsedError
    isError.value = true
    
    console.error(`[${context}] 错误:`, parsedError)
    
    // 自动处理某些错误
    if (autoRedirect) {
      if (parsedError.suggestion === 'relogin') {
        // 清除token并跳转登录
        clearTokens()
        router.push('/login')
        return parsedError
      }
    }
    
    // 显示提示
    if (showAlert) {
      alert(parsedError.userMessage)
    }
    
    return parsedError
  }
  
  /**
   * 清除错误状态
   */
  function clearError() {
    lastError.value = null
    isError.value = false
  }
  
  /**
   * 执行建议操作
   */
  /**
   * 执行建议操作
   * @param {string} suggestion - 建议操作类型
   * @returns {Object} 返回结构化的操作结果，便于组件显式处理
   */
  function executeSuggestion(suggestion) {
    const result = { action: suggestion, executed: true, params: {} }
    
    switch (suggestion) {
      case 'relogin':
        clearTokens()
        router.push('/login')
        break
      case 'refresh_sessions':
        // 触发会话列表刷新
        window.dispatchEvent(new CustomEvent('refresh-sessions'))
        break
      case 'create_new_session':
        window.dispatchEvent(new CustomEvent('create-new-session'))
        break
      case 'switch_model':
        window.dispatchEvent(new CustomEvent('open-model-selector'))
        break
      case 'upgrade_membership':
        router.push('/membership')
        break
      case 'contact_support':
        router.push('/help')
        break
      case 'retry':
      case 'retry_later':
        result.params = { shouldRetry: true }
        break
      case 'wait_and_retry':
        result.params = { shouldRetry: true, delay: 5000 }
        break
      case 'check_network':
        result.params = { checkNetwork: true }
        break
      default:
        console.log('未知建议操作:', suggestion)
        result.executed = false
    }
    
    return result
  }
  
  return {
    lastError: readonly(lastError),
    isError: readonly(isError),
    handleError,
    clearError,
    executeSuggestion,
    parseAPIError,
    ERROR_MESSAGES,
    ERROR_SUGGESTIONS
  }
}

export default useErrorHandler
