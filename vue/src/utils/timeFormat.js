/**
 * 时间格式化工具函数
 * 统一管理时间相关的计算和显示
 */

// 时间常量（毫秒）
const MS_PER_SECOND = 1000
const MS_PER_MINUTE = MS_PER_SECOND * 60
const MS_PER_HOUR = MS_PER_MINUTE * 60
const MS_PER_DAY = MS_PER_HOUR * 24

/**
 * 毫秒转天数（向下取整）
 */
export const millisecondsToDays = (ms) => Math.floor(ms / MS_PER_DAY)

/**
 * 毫秒转小时（向下取整）
 */
export const millisecondsToHours = (ms) => Math.floor(ms / MS_PER_HOUR)

/**
 * 毫秒转分钟（向下取整）
 */
export const millisecondsToMinutes = (ms) => Math.floor(ms / MS_PER_MINUTE)

/**
 * 格式化相对时间（如 "3分钟前"、"2小时前"、"5天前"）
 * @param {Date|number} timestamp - Date对象或时间戳
 * @returns {string} 中文相对时间描述
 */
export const formatRelativeTime = (timestamp) => {
  const now = new Date()
  const diff = now - timestamp
  const minutes = millisecondsToMinutes(diff)
  const hours = millisecondsToHours(diff)
  const days = millisecondsToDays(diff)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  return `${days}天前`
}

/**
 * 计算到期剩余天数（向上取整）
 * @param {string|Date} dateStr - 到期日期
 * @returns {number} 剩余天数（负数表示已过期）
 */
export const daysUntilExpiry = (dateStr) => {
  const date = new Date(dateStr)
  const now = new Date()
  return Math.ceil((date - now) / MS_PER_DAY)
}

/**
 * 检查时间戳是否在指定天数内
 * @param {number} timestamp - 时间戳（毫秒）
 * @param {number} days - 天数
 * @returns {boolean}
 */
export const isWithinDays = (timestamp, days) => {
  return (Date.now() - timestamp) < days * MS_PER_DAY
}

export default {
  MS_PER_SECOND,
  MS_PER_MINUTE,
  MS_PER_HOUR,
  MS_PER_DAY,
  millisecondsToDays,
  millisecondsToHours,
  millisecondsToMinutes,
  formatRelativeTime,
  daysUntilExpiry,
  isWithinDays,
}
