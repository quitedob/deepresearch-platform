/**
 * 会员API接口（用户端）
 */

import { membershipAPI } from './index'

/**
 * 获取会员信息
 */
export async function getMembership() {
  return await membershipAPI.getMembership()
}

/**
 * 获取配额信息
 */
export async function getQuota() {
  return await membershipAPI.getQuota()
}

/**
 * 使用激活码
 * @param {string} code - 激活码
 */
export async function activateCode(code) {
  return await membershipAPI.activateCode(code)
}

/**
 * 检查聊天配额
 */
export async function checkChatQuota() {
  return await membershipAPI.checkChatQuota()
}

/**
 * 检查研究配额
 */
export async function checkResearchQuota() {
  return await membershipAPI.checkResearchQuota()
}

export default {
  getMembership,
  getQuota,
  activateCode,
  checkChatQuota,
  checkResearchQuota
}
