<template>
  <div class="quota-config">
    <div class="section-header">
      <h2>配额配置</h2>
      <p class="description">设置各会员层级的默认配额</p>
    </div>

    <div class="config-cards">
      <div v-for="config in configs" :key="config.membership_type" class="config-card">
        <div class="card-header">
          <span class="membership-badge" :class="config.membership_type">
            {{ config.membership_type === 'premium' ? '⭐ 高级会员' : '普通用户' }}
          </span>
        </div>
        
        <div class="config-form">
          <div class="form-group">
            <label>聊天配额（普通+深度思考）</label>
            <input type="number" v-model.number="config.chat_limit" min="0" />
          </div>
          
          <div class="form-group">
            <label>深度研究配额</label>
            <input type="number" v-model.number="config.research_limit" min="0" />
          </div>
          
          <div v-if="config.membership_type === 'premium'" class="form-group">
            <label>配额重置周期（小时）</label>
            <input type="number" v-model.number="config.reset_period_hours" min="0" />
          </div>

          <div class="form-group checkbox-group">
            <label>
              <input type="checkbox" v-model="config.apply_to_all" />
              同时应用到所有{{ config.membership_type === 'premium' ? '高级会员' : '普通用户' }}
            </label>
          </div>
          
          <button class="btn-primary" @click="saveConfig(config)">
            保存配置
          </button>
        </div>
      </div>
    </div>

    <div class="batch-section">
      <h3>批量设置用户配额</h3>
      <div class="batch-form">
        <div class="form-row">
          <div class="form-group">
            <label>聊天配额</label>
            <input type="number" v-model.number="batchForm.chatLimit" min="0" />
          </div>
          <div class="form-group">
            <label>研究配额</label>
            <input type="number" v-model.number="batchForm.researchLimit" min="0" />
          </div>
        </div>
        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" v-model="batchForm.resetUsage" />
            同时重置使用量
          </label>
        </div>
        <div class="form-group">
          <label>用户ID列表（每行一个）</label>
          <textarea v-model="batchForm.userIds" rows="4" placeholder="输入用户ID，每行一个"></textarea>
        </div>
        <button class="btn-primary" @click="batchSetQuota">批量设置</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getQuotaConfigs, updateQuotaConfig, batchSetUserQuota } from '@/api/admin'

const configs = ref([])
const batchForm = ref({
  chatLimit: 10,
  researchLimit: 1,
  resetUsage: false,
  userIds: ''
})

const loadConfigs = async () => {
  try {
    const response = await getQuotaConfigs()
    const data = response.data || response
    configs.value = (data.configs || []).map(c => ({
      ...c,
      apply_to_all: false
    }))
    
    // 如果没有配置，创建默认配置
    if (configs.value.length === 0) {
      configs.value = [
        { membership_type: 'free', chat_limit: 10, research_limit: 1, reset_period_hours: 0, apply_to_all: false },
        { membership_type: 'premium', chat_limit: 50, research_limit: 10, reset_period_hours: 5, apply_to_all: false }
      ]
    }
  } catch (error) {
    console.error('加载配额配置失败:', error)
    // 使用默认配置
    configs.value = [
      { membership_type: 'free', chat_limit: 10, research_limit: 1, reset_period_hours: 0, apply_to_all: false },
      { membership_type: 'premium', chat_limit: 50, research_limit: 10, reset_period_hours: 5, apply_to_all: false }
    ]
  }
}

const saveConfig = async (config) => {
  try {
    await updateQuotaConfig({
      membership_type: config.membership_type,
      chat_limit: config.chat_limit,
      research_limit: config.research_limit,
      reset_period_hours: config.reset_period_hours || 0,
      apply_to_all: config.apply_to_all
    })
    alert(config.apply_to_all ? '配置已保存并应用到所有用户' : '配置已保存')
  } catch (error) {
    console.error('保存配额配置失败:', error)
    alert('保存失败')
  }
}

const batchSetQuota = async () => {
  const userIds = batchForm.value.userIds.split('\n').map(id => id.trim()).filter(id => id)
  if (userIds.length === 0) {
    alert('请输入用户ID')
    return
  }
  
  try {
    const response = await batchSetUserQuota(
      userIds,
      batchForm.value.chatLimit,
      batchForm.value.researchLimit,
      batchForm.value.resetUsage
    )
    const data = response.data || response
    alert(`批量设置完成：成功 ${data.success_count}/${data.total_count}`)
  } catch (error) {
    console.error('批量设置失败:', error)
    alert('批量设置失败')
  }
}

onMounted(() => {
  loadConfigs()
})
</script>

<style scoped>
.quota-config { width: 100%; }
.section-header { margin-bottom: 24px; }
.section-header h2 { font-size: 18px; font-weight: 600; margin-bottom: 4px; }
.description { color: var(--text-secondary); font-size: 14px; }
.config-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 24px; margin-bottom: 32px; }
.config-card { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 12px; padding: 20px; }
.card-header { margin-bottom: 16px; }
.membership-badge { padding: 6px 12px; border-radius: 6px; font-size: 14px; font-weight: 600; }
.membership-badge.free { background: var(--secondary-bg); color: var(--text-primary); }
.membership-badge.premium { background: #fef3c7; color: #92400e; }
.config-form { display: flex; flex-direction: column; gap: 16px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 14px; font-weight: 500; color: var(--text-secondary); }
.form-group input[type="number"], .form-group textarea { padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--input-bg); color: var(--text-primary); }
.form-group textarea { resize: vertical; }
.checkbox-group label { display: flex; align-items: center; gap: 8px; cursor: pointer; }
.checkbox-group input[type="checkbox"] { width: auto; }
.btn-primary { padding: 10px 16px; background: var(--button-bg); color: var(--button-text); border: none; border-radius: 6px; cursor: pointer; font-weight: 500; }
.btn-primary:hover { background: #5a67d8; }
.batch-section { background: var(--secondary-bg); border-radius: 12px; padding: 20px; }
.batch-section h3 { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.batch-form { display: flex; flex-direction: column; gap: 16px; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
</style>
