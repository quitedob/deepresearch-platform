<template>
  <div class="model-selector-wrapper" ref="selectorRef">
    <button class="current-model-btn" @click="toggleDropdown" :disabled="loading">
      <span class="model-display">
        {{ getCurrentProviderDisplay() }}
      </span>
      <span v-if="loading" class="loading-icon">⏳</span>
      <span v-else class="dropdown-icon" :class="{ 'is-open': isDropdownOpen }">▼</span>
    </button>

    <div v-if="isDropdownOpen" class="model-dropdown">
      <div class="dropdown-header">
        <div class="dropdown-title">选择AI服务商</div>
      </div>

      <ul class="provider-list">
        <li
          v-for="provider in providers"
          :key="provider.name"
          class="provider-option"
          @click="selectProvider(provider)"
        >
          <div class="provider-info">
            <div class="provider-icon">{{ provider.icon || '🤖' }}</div>
            <div class="provider-details">
              <div class="provider-name">{{ provider.display_name || provider.name }}</div>
              <div class="provider-description">{{ provider.description }}</div>
            </div>
          </div>
          <span v-if="chatStore.currentProvider === provider.name" class="checkmark">✔</span>
        </li>
      </ul>

      <div class="dropdown-footer" v-if="currentProvider">
        <div class="model-info-text">
          <span class="info-label">默认模型:</span>
          <span class="info-value">{{ getModelDisplayName(currentProvider.default_model) }}</span>
        </div>
        <div class="model-info-text">
          <span class="info-label">深度思考:</span>
          <span class="info-value">{{ getModelDisplayName(currentProvider.deep_think_model) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useChatStore } from '@/store';
import { getProviders } from '@/api/model';

const chatStore = useChatStore();
const isDropdownOpen = ref(false);
const selectorRef = ref(null);
const loading = ref(false);
const providers = ref([]);

// 当前选中的 provider
const currentProvider = computed(() => {
  const providerId = chatStore.currentProvider;
  return providers.value.find(p => p.name === providerId) || null;
});

// 获取当前服务商显示名称
const getCurrentProviderDisplay = () => {
  if (currentProvider.value) {
    return currentProvider.value.display_name || currentProvider.value.name;
  }
  return '选择服务商';
};

// 从 API 加载 providers
const loadProviders = async () => {
  loading.value = true;
  try {
    const response = await getProviders();
    providers.value = response.data?.providers || [];
    
    // 如果没有设置当前服务商，设置第一个可用的
    if (!chatStore.currentProvider && providers.value.length > 0) {
      const firstProvider = providers.value[0];
      chatStore.setProvider(firstProvider.name);
      chatStore.setModel(firstProvider.default_model);
    }
  } catch (error) {
    console.error('加载服务商列表失败:', error);
  } finally {
    loading.value = false;
  }
};

// 切换下拉菜单
const toggleDropdown = () => {
  if (!loading.value) {
    isDropdownOpen.value = !isDropdownOpen.value;
  }
};

// 选择服务商
const selectProvider = (provider) => {
  chatStore.setProvider(provider.name);
  chatStore.setModel(provider.default_model);
  isDropdownOpen.value = false;
};

// 根据模型名获取显示名称
const getModelDisplayName = (modelName) => {
  if (!modelName || !currentProvider.value) return modelName || '';
  const models = currentProvider.value.models || [];
  const model = models.find(m => m.name === modelName);
  return model?.display_name || modelName;
};

// 处理外部点击
const handleClickOutside = (event) => {
  if (selectorRef.value && !selectorRef.value.contains(event.target)) {
    isDropdownOpen.value = false;
  }
};

// 组件挂载
onMounted(() => {
  document.addEventListener('mousedown', handleClickOutside);
  loadProviders();
});

// 组件卸载
onUnmounted(() => {
  document.removeEventListener('mousedown', handleClickOutside);
});

// 导出方法供外部使用
defineExpose({
  getDeepThinkModel: () => currentProvider.value?.deep_think_model || '',
  getDefaultModel: () => currentProvider.value?.default_model || '',
  getCurrentProvider: () => currentProvider.value,
  refreshProviders: loadProviders,
});
</script>

<style scoped>
.model-selector-wrapper {
  position: relative;
  z-index: 20;
}

.current-model-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 160px;
  justify-content: space-between;
}

.current-model-btn:hover:not(:disabled) {
  background-color: var(--hover-bg);
  border-color: var(--accent-color);
}

.current-model-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.loading-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.dropdown-icon {
  font-size: 10px;
  transition: transform 0.2s;
  opacity: 0.6;
}

.dropdown-icon.is-open {
  transform: rotate(180deg);
}

.model-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 8px;
  width: 280px;
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.dropdown-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--primary-bg);
}

.dropdown-title {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 600;
}

.provider-list {
  list-style: none;
  padding: 8px;
  margin: 0;
  max-height: 300px;
  overflow-y: auto;
}

.provider-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.provider-option:hover {
  background-color: var(--hover-bg);
}

.provider-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.provider-icon {
  font-size: 24px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-bg);
  border-radius: 8px;
}

.provider-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.provider-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.provider-description {
  font-size: 11px;
  color: var(--text-secondary);
}

.checkmark {
  font-size: 16px;
  color: var(--accent-color);
}

.dropdown-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--border-color);
  background: var(--primary-bg);
}

.model-info-text {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  margin-bottom: 4px;
}

.model-info-text:last-child {
  margin-bottom: 0;
}

.info-label {
  color: var(--text-secondary);
}

.info-value {
  color: var(--text-primary);
  font-weight: 500;
  font-family: monospace;
}
</style>
