<template>
  <div class="settings-section">
    <h3>通用设置</h3>
    
    <div class="setting-item">
      <div class="setting-label">
        <label>主题</label>
        <p class="setting-description">选择您喜欢的界面主题</p>
      </div>
      <div class="setting-control">
        <button @click="$emit('toggle-theme')" class="theme-toggle-btn">
          <span v-if="currentTheme === 'light'">🌙 切换到深色模式</span>
          <span v-else>☀️ 切换到浅色模式</span>
        </button>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-label">
        <label>语言</label>
        <p class="setting-description">选择界面显示语言</p>
      </div>
      <div class="setting-control">
        <select v-model="language" @change="updateLanguage" class="setting-select">
          <option value="zh">简体中文</option>
          <option value="en">English</option>
        </select>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-label">
        <label>自动保存</label>
        <p class="setting-description">自动保存对话历史</p>
      </div>
      <div class="setting-control">
        <label class="switch">
          <input type="checkbox" v-model="autoSave" @change="updateAutoSave" />
          <span class="slider"></span>
        </label>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-label">
        <label>发送快捷键</label>
        <p class="setting-description">选择发送消息的快捷键</p>
      </div>
      <div class="setting-control">
        <select v-model="sendKey" @change="updateSendKey" class="setting-select">
          <option value="enter">Enter</option>
          <option value="ctrl-enter">Ctrl + Enter</option>
          <option value="shift-enter">Shift + Enter</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { userAPI } from '@/api/index';
import { STORAGE_KEYS } from '@/utils/config';

defineProps({
  currentTheme: String
});

defineEmits(['toggle-theme']);

const language = ref('zh');
const autoSave = ref(true);
const sendKey = ref('enter');
const isSaving = ref(false);

// 同步状态
const syncStatus = ref('idle'); // idle, syncing, success, error
const syncError = ref('');

/**
 * 保存设置到服务器和本地
 * 修复：先修改服务器，获得确认后再修改本地，添加同步状态反馈
 */
const saveSettings = async () => {
  // 保存旧值用于回滚
  const oldValues = {
    language: localStorage.getItem(STORAGE_KEYS.LANGUAGE),
    autoSave: localStorage.getItem(STORAGE_KEYS.AUTO_SAVE),
    sendKey: localStorage.getItem(STORAGE_KEYS.SEND_KEY)
  };
  
  isSaving.value = true;
  syncStatus.value = 'syncing';
  syncError.value = '';
  
  try {
    // 先尝试同步到服务器
    await userAPI.updatePreferences({
      language: language.value,
      auto_save: autoSave.value,
      send_key: sendKey.value
    });
    
    // 服务器同步成功后，再保存到本地
    localStorage.setItem(STORAGE_KEYS.LANGUAGE, language.value);
    localStorage.setItem(STORAGE_KEYS.AUTO_SAVE, autoSave.value);
    localStorage.setItem(STORAGE_KEYS.SEND_KEY, sendKey.value);
    
    syncStatus.value = 'success';
    console.log('[Settings] 设置已同步到服务器和本地');
    
    // 3秒后重置状态
    setTimeout(() => {
      if (syncStatus.value === 'success') {
        syncStatus.value = 'idle';
      }
    }, 3000);
    
  } catch (error) {
    // 服务器同步失败，回滚本地值
    syncStatus.value = 'error';
    syncError.value = error.message || '同步失败';
    
    // 恢复旧值
    if (oldValues.language) language.value = oldValues.language;
    if (oldValues.autoSave !== null) autoSave.value = oldValues.autoSave !== 'false';
    if (oldValues.sendKey) sendKey.value = oldValues.sendKey;
    
    console.warn('[Settings] 同步到服务器失败，已回滚:', error.message);
    
    // 5秒后重置错误状态
    setTimeout(() => {
      if (syncStatus.value === 'error') {
        syncStatus.value = 'idle';
        syncError.value = '';
      }
    }, 5000);
  } finally {
    isSaving.value = false;
  }
};

const updateLanguage = () => {
  saveSettings();
};

const updateAutoSave = () => {
  saveSettings();
};

const updateSendKey = () => {
  saveSettings();
};

/**
 * 从服务器加载设置
 */
const loadSettingsFromServer = async () => {
  try {
    const prefs = await userAPI.getPreferences();
    if (prefs) {
      if (prefs.language) language.value = prefs.language;
      if (prefs.auto_save !== undefined) autoSave.value = prefs.auto_save;
      if (prefs.send_key) sendKey.value = prefs.send_key;
      
      // 同步到本地存储
      localStorage.setItem(STORAGE_KEYS.LANGUAGE, language.value);
      localStorage.setItem(STORAGE_KEYS.AUTO_SAVE, autoSave.value);
      localStorage.setItem(STORAGE_KEYS.SEND_KEY, sendKey.value);
    }
  } catch (error) {
    console.warn('[Settings] 从服务器加载设置失败，使用本地设置');
  }
};

onMounted(async () => {
  // 先从本地加载
  language.value = localStorage.getItem(STORAGE_KEYS.LANGUAGE) || 'zh';
  autoSave.value = localStorage.getItem(STORAGE_KEYS.AUTO_SAVE) !== 'false';
  sendKey.value = localStorage.getItem(STORAGE_KEYS.SEND_KEY) || 'enter';
  
  // 然后尝试从服务器同步
  await loadSettingsFromServer();
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

.setting-item:last-child {
  border-bottom: none;
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

.theme-toggle-btn {
  padding: var(--spacing-sm) var(--spacing-lg);
  background: var(--button-bg);
  color: var(--button-text);
  border: none;
  border-radius: var(--radius-medium);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
}

.theme-toggle-btn:hover {
  background: var(--button-hover-bg);
  transform: translateY(-1px);
}

.setting-select {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--input-bg);
  color: var(--text-primary);
  border: 1px solid var(--input-border);
  border-radius: var(--radius-medium);
  font-size: 14px;
  cursor: pointer;
  min-width: 150px;
}

.setting-select:focus {
  outline: none;
  border-color: var(--input-focus-border);
}

/* Toggle Switch */
.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 28px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--secondary-bg);
  transition: 0.3s;
  border-radius: 28px;
  border: 2px solid var(--border-color);
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 2px;
  bottom: 2px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--accent-blue);
  border-color: var(--accent-blue);
}

input:checked + .slider:before {
  transform: translateX(22px);
}
</style>
