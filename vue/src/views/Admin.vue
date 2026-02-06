<template>
  <div class="admin-container">
    <div class="admin-layout">
      <div class="admin-header">
        <div class="header-left">
          <h1>管理员控制台</h1>
          <p class="header-subtitle">系统概览与管理</p>
        </div>
        <div class="admin-stats-grid">
          <div class="admin-stat-card">
            <div class="admin-stat-icon">
              <img src="@/assets/images/admin-users-3d.png" class="admin-stat-img" />
            </div>
            <div class="admin-stat-info">
              <span class="admin-stat-value">{{ stats.total_users || 0 }}</span>
              <span class="admin-stat-label">总用户数</span>
            </div>
          </div>
          <div class="admin-stat-card">
            <div class="admin-stat-icon">
              <img src="@/assets/images/admin-vip-3d.png" class="admin-stat-img" />
            </div>
            <div class="admin-stat-info">
              <span class="admin-stat-value">{{ stats.premium_users || 0 }}</span>
              <span class="admin-stat-label">高级会员</span>
            </div>
          </div>
          <div class="admin-stat-card">
            <div class="admin-stat-icon">
              <img src="@/assets/images/admin-key-3d.png" class="admin-stat-img" />
            </div>
            <div class="admin-stat-info">
              <span class="admin-stat-value">{{ stats.active_codes || 0 }}</span>
              <span class="admin-stat-label">有效激活码</span>
            </div>
          </div>
        </div>
      </div>

      <div class="admin-body">
        <div class="sidebar-nav">
          <button 
            v-for="tab in tabs" 
            :key="tab.id"
            :class="['nav-btn', { active: activeTab === tab.id }]"
            @click="activeTab = tab.id"
          >
            <span class="nav-icon">{{ tab.icon }}</span>
            {{ tab.name }}
          </button>
        </div>

        <div class="main-content">
          <!-- 用户管理 -->
          <div v-if="activeTab === 'users'" class="content-panel fade-in">
            <UserManagement :key="`users-${refreshKey}`" />
          </div>

          <!-- 模型配置 -->
          <div v-if="activeTab === 'models'" class="content-panel fade-in">
            <ModelConfig :key="`models-${refreshKey}`" />
          </div>

          <!-- 激活码管理 -->
          <div v-if="activeTab === 'codes'" class="content-panel fade-in">
            <ActivationCodeManagement :key="`codes-${refreshKey}`" />
          </div>

          <!-- 通知管理 -->
          <div v-if="activeTab === 'notifications'" class="content-panel fade-in">
            <NotificationManagement :key="`notifications-${refreshKey}`" />
          </div>

          <!-- 配额配置 -->
          <div v-if="activeTab === 'quota'" class="content-panel fade-in">
            <QuotaConfig :key="`quota-${refreshKey}`" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, provide } from 'vue'
import { getAdminStats } from '@/api/admin'
import UserManagement from '@/components/admin/UserManagement.vue'
import ModelConfig from '@/components/admin/ModelConfig.vue'
import ActivationCodeManagement from '@/components/admin/ActivationCodeManagement.vue'
import NotificationManagement from '@/components/admin/NotificationManagement.vue'
import QuotaConfig from '@/components/admin/QuotaConfig.vue'

const activeTab = ref('users')
const stats = ref({})
const refreshKey = ref(0)

const tabs = [
  { id: 'users', name: '用户管理', icon: '👤' },
  { id: 'models', name: '模型配置', icon: '🤖' },
  { id: 'codes', name: '激活码管理', icon: '🎫' },
  { id: 'notifications', name: '通知管理', icon: '📢' },
  { id: 'quota', name: '配额配置', icon: '⚙️' }
]

// 提供刷新方法给子组件
const refreshStats = async () => {
  await loadStats()
}
provide('refreshStats', refreshStats)

// 监听tab切换，触发数据刷新
watch(activeTab, (newTab, oldTab) => {
  if (newTab !== oldTab) {
    // 增加refreshKey触发子组件重新加载
    refreshKey.value++
  }
})

const loadStats = async () => {
  try {
    const response = await getAdminStats()
    stats.value = response.data || response
  } catch (error) {
    console.error('加载统计信息失败:', error)
    // 显示友好的错误提示
    if (error.response?.data?.error) {
      const apiError = error.response.data.error
      console.error('API错误:', apiError.code, apiError.message)
    }
  }
}

onMounted(() => {
  loadStats()
})
</script>

<style scoped>
/* 使用全局 CSS 变量实现主题同步 */
.admin-container {
  height: 100%; /* 填满父容器 */
  overflow-y: auto; /* 允许垂直滚动 */
  background-color: var(--secondary-bg); /* 跟随主题背景 */
  color: var(--text-primary); /* 跟随主题文字颜色 */
  font-family: inherit;
}

.admin-layout {
  max-width: 1400px;
  margin: 0 auto;
  padding: 32px;
  min-height: 100%;
}

.admin-header {
  margin-bottom: 32px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 24px;
}

.header-left h1 {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 8px 0;
  color: var(--text-primary);
}

.header-subtitle {
  color: var(--text-secondary);
  margin: 0;
  font-size: 14px;
}

.admin-stats-grid {
  display: flex;
  gap: 20px;
}

.admin-stat-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px 24px;
  min-width: 180px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: var(--shadow-elev);
  transition: transform 0.2s, box-shadow 0.2s;
  backdrop-filter: blur(10px);
}

.admin-stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-elev-high);
  border-color: var(--border-strong);
}

.admin-stat-info {
  display: flex;
  flex-direction: column;
}

.admin-stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.admin-stat-label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.admin-body {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 32px;
  align-items: start;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--card-bg);
  padding: 16px;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-elev);
}

.nav-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s;
  text-align: left;
}

.nav-btn:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.nav-btn.active {
  background: var(--button-bg);
  color: var(--button-text);
  box-shadow: var(--shadow-elev);
}

.nav-icon {
  font-size: 18px;
}

.main-content {
  background: var(--card-bg);
  border-radius: 16px;
  min-height: 600px;
  padding: 32px;
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-elev);
}

.content-panel {
  width: 100%;
}

.fade-in {
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 响应式调整 */
@media (max-width: 1024px) {
  .admin-body {
    grid-template-columns: 1fr;
  }
  
  .sidebar-nav {
    flex-direction: row;
    overflow-x: auto;
    padding: 12px;
  }

  .nav-btn {
    white-space: nowrap;
    padding: 8px 16px;
  }
  
  .admin-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 24px;
  }
  
  .admin-stats-grid {
    width: 100%;
    flex-wrap: wrap;
  }
  
  .admin-stat-card {
    flex: 1;
  }
}

.admin-stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  background: #0f172a; /* Dark background for 3D asset */
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  overflow: hidden;
  border: 1px solid rgba(255,255,255,0.1);
}

.admin-stat-img {
  width: 140%;
  height: 140%;
  object-fit: contain;
  mix-blend-mode: screen;
}
</style>
