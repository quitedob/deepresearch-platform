<template>
  <div class="membership-info">
    <div class="membership-card" :class="membership.membership_type">
      <div class="card-header">
        <span class="membership-type">
          {{ membership.membership_type === 'premium' ? '⭐ 高级会员' : '普通用户' }}
        </span>
        <span v-if="membership.expires_at" class="expires-info">
          {{ formatExpires(membership.expires_at) }}
        </span>
      </div>
      
      <div class="quota-section">
        <div class="quota-item">
          <div class="quota-label">聊天配额</div>
          <div class="quota-bar">
            <div class="quota-fill" :style="{ width: chatPercent + '%' }"></div>
          </div>
          <div class="quota-text">{{ quota.chat_remaining }} / {{ quota.chat_limit }}</div>
        </div>
        
        <div class="quota-item">
          <div class="quota-label">深度研究</div>
          <div class="quota-bar">
            <div class="quota-fill" :style="{ width: researchPercent + '%' }"></div>
          </div>
          <div class="quota-text">{{ quota.research_remaining }} / {{ quota.research_limit }}</div>
        </div>
      </div>

      <div v-if="quota.reset_at" class="reset-info">
        配额将于 {{ formatTime(quota.reset_at) }} 重置
      </div>
    </div>

    <div v-if="membership.membership_type !== 'premium'" class="upgrade-section">
      <button class="upgrade-btn" @click="showActivateModal = true">🎁 激活会员</button>
    </div>

    <!-- 激活码弹窗 -->
    <div v-if="showActivateModal" class="modal-overlay" @click.self="showActivateModal = false">
      <div class="modal">
        <h3>激活高级会员</h3>
        <div class="form-group">
          <label>请输入激活码</label>
          <input type="text" v-model="activationCode" placeholder="输入激活码" />
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showActivateModal = false">取消</button>
          <button class="btn-primary" @click="activate">激活</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getMembership, getQuota, activateCode } from '@/api/membership'

const membership = ref({ membership_type: 'free' })
const quota = ref({ chat_remaining: 0, chat_limit: 10, research_remaining: 0, research_limit: 1 })
const showActivateModal = ref(false)
const activationCode = ref('')

const chatPercent = computed(() => {
  if (quota.value.chat_limit === 0) return 0
  return (quota.value.chat_remaining / quota.value.chat_limit) * 100
})

const researchPercent = computed(() => {
  if (quota.value.research_limit === 0) return 0
  return (quota.value.research_remaining / quota.value.research_limit) * 100
})

const loadMembership = async () => {
  try {
    const response = await getMembership()
    membership.value = (response.data || response).membership || { membership_type: 'free' }
  } catch (error) {
    console.error('加载会员信息失败:', error)
  }
}

const loadQuota = async () => {
  try {
    const response = await getQuota()
    quota.value = response.data || response
  } catch (error) {
    console.error('加载配额信息失败:', error)
  }
}

const formatExpires = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const days = Math.ceil((date - now) / (1000 * 60 * 60 * 24))
  if (days <= 0) return '已过期'
  if (days <= 7) return `${days}天后到期`
  return `${date.toLocaleDateString('zh-CN')} 到期`
}

const formatTime = (dateStr) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN')
}

const activate = async () => {
  if (!activationCode.value) {
    alert('请输入激活码')
    return
  }
  try {
    await activateCode(activationCode.value)
    alert('激活成功！')
    showActivateModal.value = false
    activationCode.value = ''
    loadMembership()
    loadQuota()
  } catch (error) {
    console.error('激活失败:', error)
    alert('激活码无效或已过期')
  }
}

onMounted(() => {
  loadMembership()
  loadQuota()
})

defineExpose({ loadQuota })
</script>

<style scoped>
.membership-info { padding: 16px; }
.membership-card { background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%); border-radius: 12px; padding: 16px; }
.membership-card.premium { background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.membership-type { font-weight: 600; font-size: 16px; }
.expires-info { font-size: 12px; color: #6b7280; }
.quota-section { display: flex; flex-direction: column; gap: 12px; }
.quota-item { }
.quota-label { font-size: 13px; color: #6b7280; margin-bottom: 4px; }
.quota-bar { height: 8px; background: rgba(0,0,0,0.1); border-radius: 4px; overflow: hidden; }
.quota-fill { height: 100%; background: #667eea; border-radius: 4px; transition: width 0.3s; }
.quota-text { font-size: 12px; color: #374151; margin-top: 4px; text-align: right; }
.reset-info { margin-top: 12px; font-size: 12px; color: #6b7280; text-align: center; }
.upgrade-section { margin-top: 16px; }
.upgrade-btn { width: 100%; padding: 12px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 500; cursor: pointer; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; padding: 24px; min-width: 350px; }
.modal h3 { margin-bottom: 20px; font-size: 18px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 14px; }
.form-group input { width: 100%; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 6px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
.btn-primary { padding: 8px 16px; background: #667eea; color: white; border: none; border-radius: 6px; cursor: pointer; }
.btn-secondary { padding: 8px 16px; background: #e5e7eb; color: #374151; border: none; border-radius: 6px; cursor: pointer; }
</style>
