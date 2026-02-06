/**
 * 通知API接口（用户端）
 */

import { notificationAPI } from './index'

/**
 * 获取通知列表
 * @param {number} limit - 分页大小
 * @param {number} offset - 分页偏移
 */
export async function getNotifications(limit = 20, offset = 0) {
  return await notificationAPI.getNotifications(limit, offset)
}

/**
 * 获取未读通知数量
 */
export async function getUnreadCount() {
  return await notificationAPI.getUnreadCount()
}

/**
 * 标记通知为已读
 * @param {string} notificationId - 通知ID
 */
export async function markAsRead(notificationId) {
  return await notificationAPI.markAsRead(notificationId)
}

/**
 * 标记所有通知为已读
 */
export async function markAllAsRead() {
  return await notificationAPI.markAllAsRead()
}

export default {
  getNotifications,
  getUnreadCount,
  markAsRead,
  markAllAsRead
}
