<template>
  <div class="sidebar-wrapper">
    <!-- 移动端汉堡菜单按钮 -->
    <button class="mobile-menu-btn" @click="toggleMobileSidebar" aria-label="打开菜单">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="3" y1="6" x2="21" y2="6"></line>
        <line x1="3" y1="12" x2="21" y2="12"></line>
        <line x1="3" y1="18" x2="21" y2="18"></line>
      </svg>
    </button>

    <!-- 移动端遮罩层 -->
    <div v-if="isMobileSidebarOpen" class="sidebar-overlay" @click="closeMobileSidebar"></div>

    <div class="sidebar" :class="{ 'mobile-open': isMobileSidebarOpen }">
    <div class="sidebar-header">
      <div class="logo-row">
        <div class="logo">Deep Research</div>
        <div class="header-actions">
          <NotificationBell />
          <router-link v-if="isAdmin" to="/admin" class="admin-btn" title="管理员控制台">
            ⚙️
          </router-link>
          <!-- 移动端关闭按钮 -->
          <button class="mobile-close-btn" @click="closeMobileSidebar" aria-label="关闭菜单">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </div>
      
      <!-- 导航切换按钮 -->
      <div class="nav-tabs">
        <router-link to="/home" class="nav-tab" :class="{ active: $route.path === '/home' }" @click="closeMobileSidebar">
          <span class="nav-icon">💬</span>
          <span class="nav-label">对话</span>
        </router-link>
        <router-link to="/ai-space" class="nav-tab" :class="{ active: $route.path === '/ai-space' }" @click="closeMobileSidebar">
          <span class="nav-icon">🎯</span>
          <span class="nav-label">AI出题</span>
        </router-link>
      </div>
      
      <MembershipInfo v-if="showMembership" />
    </div>
    
    <div class="list-container">
      <!-- 根据路由显示不同的历史列表 -->
      <HistoryList v-if="$route.path === '/home'" />
      <AIQuestionHistoryList v-else-if="$route.path === '/ai-space'" />
    </div>
    </div>
  </div>
</template>


<script setup>
import { useChatStore } from '@/store';
import { useRoute } from 'vue-router';
import HistoryList from './HistoryList.vue';
import AIQuestionHistoryList from './AIQuestionHistoryList.vue';
import NotificationBell from './NotificationBell.vue';
import MembershipInfo from './MembershipInfo.vue';
import { onMounted, ref, computed, watch } from 'vue';

const chatStore = useChatStore();
const route = useRoute();
const showMembership = ref(false);
const isMobileSidebarOpen = ref(false);

const toggleMobileSidebar = () => {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value;
};

const closeMobileSidebar = () => {
  isMobileSidebarOpen.value = false;
};

// 检查是否是管理员
const isAdmin = computed(() => {
  try {
    const userStr = localStorage.getItem('user') || sessionStorage.getItem('user');
    if (userStr) {
      const user = JSON.parse(userStr);
      return user.is_admin === true;
    }
  } catch (e) {
    console.error('检查管理员权限失败:', e);
  }
  return false;
});

onMounted(async () => {
  if (route.path === '/home') {
    // 等待一个微任务周期，确保 token 已保存到 storage
    await new Promise(resolve => setTimeout(resolve, 0));
    const token = localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
    if (token) {
      chatStore.fetchSessions();
    }
  }
});

// 监听路由变化，加载对应的数据
watch(() => route.path, (newPath) => {
  if (newPath === '/home') {
    chatStore.fetchSessions();
  }
});
</script>

<style scoped>
.sidebar-wrapper {
  display: contents;
}

.sidebar {
  width: 260px;
  height: 100vh;
  background-color: var(--secondary-bg);
  display: flex;
  flex-direction: column;
  padding: 16px;
  box-sizing: border-box;
  flex-shrink: 0;
  border-right: 1px solid var(--border-color);
  transition: margin-left 0.3s ease, width 0.3s ease;
}

.sidebar.collapsed {
  margin-left: -260px;
}

.sidebar-header {
  margin-bottom: 16px;
}

.logo-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.logo {
  font-size: 20px;
  font-weight: bold;
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.admin-btn {
  font-size: 18px;
  text-decoration: none;
  padding: 4px;
  border-radius: 4px;
  transition: background 0.2s;
}

.admin-btn:hover {
  background: var(--hover-bg);
}

/* 导航切换标签 */
.nav-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  padding: 4px;
  background: var(--primary-bg);
  border-radius: 10px;
}

.nav-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 12px;
  border-radius: 8px;
  text-decoration: none;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-tab:hover {
  background: var(--hover-bg);
}

.nav-tab.active,
.nav-tab.router-link-active.router-link-exact-active {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
  font-weight: 600;
}

.nav-icon {
  font-size: 16px;
}

.nav-label {
  font-size: 13px;
}

/* 亮色模式下的导航标签 */
:global(body.light) .nav-tab {
  color: #1d1d1f;
}

:global(body.light) .nav-tab.active,
:global(body.light) .nav-tab.router-link-active.router-link-exact-active {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
}

.list-container {
  flex-grow: 1;
  overflow-y: auto;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  padding: 0 8px;
  margin-bottom: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.explore-title {
  margin-top: 24px;
}

.explore-items {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.explore-item {
  padding: 8px 12px;
  border-radius: 6px;
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 14px;
  transition: background-color 0.2s, color 0.2s;
}
.explore-item:hover {
  background-color: var(--hover-bg);
  color: var(--text-primary);
}

/* 移动端汉堡菜单按钮 */
.mobile-menu-btn {
  display: none;
  position: fixed;
  top: 12px;
  left: 12px;
  z-index: 1001;
  background: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 8px;
  color: var(--text-primary);
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transition: background 0.2s;
  min-width: 44px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
}

.mobile-menu-btn:hover {
  background: var(--hover-bg);
}

/* 移动端关闭按钮 */
.mobile-close-btn {
  display: none;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  min-width: 44px;
  min-height: 44px;
  align-items: center;
  justify-content: center;
}

.mobile-close-btn:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

/* 移动端遮罩层 */
.sidebar-overlay {
  display: none;
}

/* 移动端响应式：侧边栏抽屉式导航 */
@media (max-width: 768px) {
  .mobile-menu-btn {
    display: flex;
  }

  .mobile-close-btn {
    display: flex;
  }

  .sidebar-overlay {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1099;
    animation: fadeIn 0.2s ease;
  }

  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    height: 100dvh;
    z-index: 1100;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    box-shadow: 4px 0 20px rgba(0, 0, 0, 0.2);
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
}
</style>