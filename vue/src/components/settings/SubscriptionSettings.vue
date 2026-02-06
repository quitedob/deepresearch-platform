<template>
  <div class="settings-section">
    <h3>订阅管理</h3>
    
    <!-- 当前计划卡片 -->
    <div class="current-plan-card" :class="{ 'premium': isPremium }">
      <div v-if="loading" class="loading-state">
        <span class="loading-spinner"></span>
        <span>加载中...</span>
      </div>
      <template v-else>
        <div class="plan-header">
          <div class="plan-info">
            <span class="plan-badge" :class="{ 'premium': isPremium }">
              {{ isPremium ? '🌟 高级版' : '📦 基础版' }}
            </span>
            <h4>{{ isPremium ? '高级版会员' : '基础版（免费）' }}</h4>
          </div>
          <div class="plan-status" v-if="isPremium && membershipExpiry">
            <span class="expiry-label">有效期至</span>
            <span class="expiry-date">{{ formatDate(membershipExpiry) }}</span>
          </div>
        </div>
        
        <div class="quota-section">
          <h5>本月配额使用情况</h5>
          <div class="quota-items">
            <div class="quota-item">
              <div class="quota-label">
                <span class="quota-icon">💬</span>
                <span>普通对话</span>
              </div>
              <div class="quota-bar">
                <div class="quota-fill" :style="{ width: chatUsagePercent + '%' }" :class="{ 'warning': chatUsagePercent > 80 }"></div>
              </div>
              <div class="quota-text">{{ chatUsed }} / {{ chatLimit }} 次</div>
            </div>
            <div class="quota-item">
              <div class="quota-label">
                <span class="quota-icon">🔬</span>
                <span>深度研究</span>
              </div>
              <div class="quota-bar">
                <div class="quota-fill" :style="{ width: researchUsagePercent + '%' }" :class="{ 'warning': researchUsagePercent > 80 }"></div>
              </div>
              <div class="quota-text">{{ researchUsed }} / {{ researchLimit }} 次</div>
            </div>
          </div>
          <p class="quota-reset-hint">配额将于每月1日重置</p>
        </div>
      </template>
    </div>

    <!-- 版本对比 -->
    <div class="plans-comparison">
      <h4>版本对比</h4>
      <div class="plans-grid">
        <div class="plan-card" :class="{ 'current': !isPremium }">
          <div class="plan-card-header">
            <h5>📦 基础版</h5>
            <span class="price">免费</span>
          </div>
          <ul class="plan-features">
            <li><span class="check">✓</span> 普通对话 10次/月</li>
            <li><span class="check">✓</span> 深度研究 1次/月</li>
            <li><span class="check">✓</span> 基础模型访问</li>
            <li><span class="check">✓</span> 标准响应速度</li>
          </ul>
          <div class="plan-card-footer" v-if="!isPremium">
            <span class="current-badge">当前版本</span>
          </div>
        </div>

        <div class="plan-card premium" :class="{ 'current': isPremium }">
          <div class="recommended-badge">推荐</div>
          <div class="plan-card-header">
            <h5>🌟 高级版</h5>
            <span class="price">激活码兑换</span>
          </div>
          <ul class="plan-features">
            <li><span class="check premium">✓</span> 普通对话 50次/月</li>
            <li><span class="check premium">✓</span> 深度研究 10次/月</li>
            <li><span class="check premium">✓</span> 高级模型访问</li>
            <li><span class="check premium">✓</span> 优先响应速度</li>
          </ul>
          <div class="plan-card-footer" v-if="isPremium">
            <span class="current-badge premium">当前版本</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 激活码兑换 -->
    <div class="activation-section">
      <h4>激活码兑换</h4>
      <p class="activation-hint">输入激活码升级到高级版，享受更多配额</p>
      <div class="activation-form">
        <input 
          v-model="activationCode" 
          type="text" 
          class="activation-input"
          placeholder="请输入激活码"
          :disabled="activating"
        />
        <button 
          @click="activateCode" 
          class="activate-btn"
          :disabled="!activationCode.trim() || activating"
        >
          {{ activating ? '激活中...' : '激活' }}
        </button>
      </div>
      <p v-if="activationError" class="activation-error">{{ activationError }}</p>
      <p v-if="activationSuccess" class="activation-success">{{ activationSuccess }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { membershipAPI } from '@/api/index';

const isPremium = ref(false);
const membershipExpiry = ref(null);
const chatUsed = ref(0);
const chatLimit = ref(10);
const researchUsed = ref(0);
const researchLimit = ref(1);
const activationCode = ref('');
const activating = ref(false);
const activationError = ref('');
const activationSuccess = ref('');
const loading = ref(true);

const chatUsagePercent = computed(() => Math.min((chatUsed.value / chatLimit.value) * 100, 100));
const researchUsagePercent = computed(() => Math.min((researchUsed.value / researchLimit.value) * 100, 100));

const formatDate = (dateStr) => {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' });
};

const loadMembershipInfo = async () => {
  loading.value = true;
  try {
    // 获取会员信息
    const membershipData = await membershipAPI.getMembership();
    isPremium.value = membershipData.is_premium || false;
    membershipExpiry.value = membershipData.expires_at;
    
    // 获取配额信息
    const quotaData = await membershipAPI.getQuota();
    chatUsed.value = quotaData.chat_used || 0;
    chatLimit.value = quotaData.chat_limit || (isPremium.value ? 50 : 10);
    researchUsed.value = quotaData.research_used || 0;
    researchLimit.value = quotaData.research_limit || (isPremium.value ? 10 : 1);
  } catch (error) {
    console.error('加载会员信息失败:', error);
    // 使用默认值
    if (isPremium.value) {
      chatLimit.value = 50;
      researchLimit.value = 10;
    } else {
      chatLimit.value = 10;
      researchLimit.value = 1;
    }
  } finally {
    loading.value = false;
  }
};

const activateCode = async () => {
  if (!activationCode.value.trim() || activating.value) return;
  
  activating.value = true;
  activationError.value = '';
  activationSuccess.value = '';
  
  try {
    await membershipAPI.activateCode(activationCode.value.trim());
    activationSuccess.value = '激活成功！您已升级到高级版';
    activationCode.value = '';
    await loadMembershipInfo();
  } catch (error) {
    console.error('激活失败:', error);
    activationError.value = error.message || '激活失败，请检查激活码是否正确';
  } finally {
    activating.value = false;
  }
};

// 组件挂载时加载数据
onMounted(() => {
  loadMembershipInfo();
});

// 每30秒刷新一次配额信息
let refreshInterval = null;
onMounted(() => {
  refreshInterval = setInterval(() => {
    loadMembershipInfo();
  }, 30000);
});

// 组件卸载时清除定时器
import { onUnmounted } from 'vue';
onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});
</script>

<style scoped>
.settings-section {
  max-width: 800px;
}

.settings-section h3 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: var(--spacing-xl);
  color: var(--text-primary);
}

.settings-section h4 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-md) 0;
}

/* 当前计划卡片 */
.current-plan-card {
  padding: var(--spacing-xl);
  background: var(--card-bg);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-large);
  margin-bottom: var(--spacing-xl);
}

.current-plan-card.premium {
  border-color: #f59e0b;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.05) 0%, rgba(245, 158, 11, 0.02) 100%);
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.plan-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-lg);
}

.plan-info h4 {
  margin: var(--spacing-sm) 0 0 0;
  font-size: 20px;
}

.plan-badge {
  display: inline-block;
  padding: 4px 12px;
  background: var(--secondary-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-medium);
  font-size: 13px;
  font-weight: 500;
}

.plan-badge.premium {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
}

.plan-status {
  text-align: right;
}

.expiry-label {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
}

.expiry-date {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

/* 配额部分 */
.quota-section h5 {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-md) 0;
}

.quota-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.quota-item {
  display: grid;
  grid-template-columns: 120px 1fr 80px;
  align-items: center;
  gap: var(--spacing-md);
}

.quota-label {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  font-size: 14px;
  color: var(--text-primary);
}

.quota-icon {
  font-size: 16px;
}

.quota-bar {
  height: 8px;
  background: var(--secondary-bg);
  border-radius: 4px;
  overflow: hidden;
}

.quota-fill {
  height: 100%;
  background: var(--accent-blue);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.quota-fill.warning {
  background: #f59e0b;
}

.quota-text {
  font-size: 13px;
  color: var(--text-secondary);
  text-align: right;
}

.quota-reset-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: var(--spacing-md) 0 0 0;
}

/* 版本对比 */
.plans-comparison {
  margin-bottom: var(--spacing-xl);
}

.plans-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-lg);
}

.plan-card {
  padding: var(--spacing-xl);
  background: var(--card-bg);
  border: 2px solid var(--border-color);
  border-radius: var(--radius-large);
  position: relative;
  transition: all 0.2s ease;
}

.plan-card.current {
  border-color: var(--accent-blue);
}

.plan-card.premium {
  border-color: #f59e0b;
}

.plan-card.premium.current {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.05) 0%, rgba(245, 158, 11, 0.02) 100%);
}

.recommended-badge {
  position: absolute;
  top: -10px;
  right: 16px;
  padding: 4px 12px;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
  border-radius: var(--radius-medium);
  font-size: 12px;
  font-weight: 600;
}

.plan-card-header {
  margin-bottom: var(--spacing-lg);
}

.plan-card-header h5 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--spacing-xs) 0;
}

.price {
  font-size: 14px;
  color: var(--text-secondary);
}

.plan-features {
  list-style: none;
  padding: 0;
  margin: 0 0 var(--spacing-lg) 0;
}

.plan-features li {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.check {
  color: var(--accent-green);
  font-weight: 600;
}

.check.premium {
  color: #f59e0b;
}

.plan-card-footer {
  padding-top: var(--spacing-md);
  border-top: 1px solid var(--border-color);
}

.current-badge {
  display: inline-block;
  padding: 4px 12px;
  background: var(--accent-blue);
  color: white;
  border-radius: var(--radius-medium);
  font-size: 12px;
  font-weight: 500;
}

.current-badge.premium {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

/* 激活码部分 */
.activation-section {
  padding: var(--spacing-xl);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-large);
}

.activation-hint {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-md) 0;
}

.activation-form {
  display: flex;
  gap: var(--spacing-md);
}

.activation-input {
  flex: 1;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--input-bg);
  color: var(--text-primary);
  border: 1px solid var(--input-border);
  border-radius: var(--radius-medium);
  font-size: 14px;
}

.activation-input:focus {
  outline: none;
  border-color: var(--input-focus-border);
}

.activate-btn {
  padding: var(--spacing-sm) var(--spacing-xl);
  background: var(--accent-blue);
  color: white;
  border: none;
  border-radius: var(--radius-medium);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.activate-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.activate-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.activation-error {
  color: #ef4444;
  font-size: 13px;
  margin: var(--spacing-sm) 0 0 0;
}

.activation-success {
  color: #10b981;
  font-size: 13px;
  margin: var(--spacing-sm) 0 0 0;
}

@media (max-width: 640px) {
  .plans-grid {
    grid-template-columns: 1fr;
  }
  
  .quota-item {
    grid-template-columns: 1fr;
    gap: var(--spacing-xs);
  }
  
  .quota-text {
    text-align: left;
  }
  
  .activation-form {
    flex-direction: column;
  }
}
</style>
