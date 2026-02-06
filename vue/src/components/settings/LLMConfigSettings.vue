<template>
  <div class="settings-section">
    <h3>AI模型配置</h3>
    
    <div class="setting-item">
      <div class="setting-label">
        <label>默认模型提供商</label>
        <p class="setting-description">选择默认使用的AI模型提供商</p>
      </div>
      <div class="setting-control">
        <select v-model="defaultProvider" class="setting-select" @change="onProviderChange">
          <option v-for="provider in providers" :key="provider.name" :value="provider.name">
            {{ provider.display_name || provider.name }}
          </option>
        </select>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-label">
        <label>默认模型</label>
        <p class="setting-description">选择默认使用的具体模型</p>
      </div>
      <div class="setting-control">
        <select v-model="defaultModel" class="setting-select">
          <option v-for="model in availableModels" :key="model.name" :value="model.name">
            {{ model.display_name || model.name }}
          </option>
        </select>
      </div>
    </div>

    <div class="actions">
      <button @click="saveSettings" class="test-btn" :disabled="loading">
        {{ loading ? '加载中...' : '保存设置' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { getProviders } from '@/api/model';

const loading = ref(false);
const providers = ref([]);
const defaultProvider = ref('');
const defaultModel = ref('');

// 当前 provider 的可用模型
const availableModels = computed(() => {
  const provider = providers.value.find(p => p.name === defaultProvider.value);
  return provider?.models || [];
});

// provider 变更时，自动选择默认模型
const onProviderChange = () => {
  const provider = providers.value.find(p => p.name === defaultProvider.value);
  if (provider) {
    defaultModel.value = provider.default_model || (provider.models?.[0]?.name || '');
  }
};

// 加载 providers
const loadProviders = async () => {
  loading.value = true;
  try {
    const response = await getProviders();
    providers.value = response.data?.providers || [];
    
    // 从 localStorage 恢复设置
    const savedProvider = localStorage.getItem('defaultProvider');
    const savedModel = localStorage.getItem('defaultModel');
    
    if (savedProvider && providers.value.find(p => p.name === savedProvider)) {
      defaultProvider.value = savedProvider;
    } else if (providers.value.length > 0) {
      defaultProvider.value = providers.value[0].name;
    }
    
    if (savedModel) {
      defaultModel.value = savedModel;
    } else {
      onProviderChange();
    }
  } catch (error) {
    console.error('加载提供商列表失败:', error);
  } finally {
    loading.value = false;
  }
};

const saveSettings = () => {
  localStorage.setItem('defaultProvider', defaultProvider.value);
  localStorage.setItem('defaultModel', defaultModel.value);
  alert('设置已保存');
};

onMounted(() => {
  loadProviders();
});
</script>

<style scoped>
.settings-section {
  max-width: 600px;
}

.settings-section h3 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: var(--spacing-xl);
  color: var(--text-primary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-lg) 0;
  border-bottom: 1px solid var(--border-color);
}

.setting-label {
  flex: 1;
}

.setting-label label {
  display: block;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: var(--spacing-xs);
}

.setting-description {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.setting-control {
  flex-shrink: 0;
  margin-left: var(--spacing-lg);
}

.setting-select {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--input-bg);
  color: var(--text-primary);
  border: 1px solid var(--input-border);
  border-radius: var(--radius-medium);
  font-size: 14px;
  cursor: pointer;
  min-width: 200px;
}

.actions {
  margin-top: var(--spacing-xl);
}

.test-btn {
  padding: var(--spacing-sm) var(--spacing-lg);
  background: var(--gradient-blue);
  color: white;
  border: none;
  border-radius: var(--radius-medium);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.test-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
