<template>
  <div class="settings-section">
    <h3>聊天记忆设置</h3>
    <div class="setting-item">
      <div class="setting-label">
        <label>启用聊天记忆</label>
        <p class="setting-description">开启后，AI将记住当前会话的所有对话内容</p>
      </div>
      <div class="setting-control">
        <label class="switch">
          <input type="checkbox" v-model="memoryEnabled" @change="updateMemoryEnabled" />
          <span class="slider"></span>
        </label>
      </div>
    </div>
    <div class="setting-item full-width">
      <div class="setting-label">
        <label>自定义系统提示词</label>
        <p class="setting-description">每次对话都会自动添加的提示词</p>
      </div>
      <div class="setting-control">
        <textarea v-model="customSystemPrompt" @blur="updateCustomPrompt" class="setting-textarea" placeholder="例如：请用简洁专业的语言回答问题..." rows="4"></textarea>
        <div class="char-count">{{ customSystemPrompt.length }} / 5000</div>
      </div>
    </div>
    <div class="setting-actions">
      <button @click="saveAllSettings" class="save-btn" :disabled="saving">{{ saving ? '保存中...' : '保存设置' }}</button>
      <button @click="resetToDefaults" class="reset-btn">恢复默认</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { userAPI } from '@/api/index';

const memoryEnabled = ref(true);
const customSystemPrompt = ref('');
const saving = ref(false);

const loadSettings = async () => {
  try {
    const response = await userAPI.getPreferences();
    if (response) {
      memoryEnabled.value = response.memory_enabled ?? true;
      customSystemPrompt.value = response.custom_system_prompt || '';
    }
  } catch (error) { console.error('加载记忆设置失败:', error); }
};

const updateMemoryEnabled = async () => {
  try { await userAPI.updatePreferences({ memory_enabled: memoryEnabled.value }); }
  catch (error) { console.error('更新记忆开关失败:', error); }
};

const updateCustomPrompt = async () => {
  if (customSystemPrompt.value.length > 5000) customSystemPrompt.value = customSystemPrompt.value.substring(0, 5000);
  try { await userAPI.updatePreferences({ custom_system_prompt: customSystemPrompt.value }); }
  catch (error) { console.error('更新自定义提示词失败:', error); }
};

const saveAllSettings = async () => {
  saving.value = true;
  try {
    await userAPI.updatePreferences({ memory_enabled: memoryEnabled.value, custom_system_prompt: customSystemPrompt.value });
    alert('设置保存成功！');
  } catch (error) { console.error('保存设置失败:', error); alert('保存设置失败，请重试'); }
  finally { saving.value = false; }
};

const resetToDefaults = () => { memoryEnabled.value = true; customSystemPrompt.value = ''; saveAllSettings(); };

onMounted(() => { loadSettings(); });
</script>

<style scoped>
.settings-section { max-width: 700px; }
.settings-section h3 { font-size: 24px; font-weight: 600; margin-bottom: 32px; color: var(--text-primary); }
.setting-item { display: flex; justify-content: space-between; align-items: flex-start; padding: 24px 0; border-bottom: 1px solid var(--border-color); }
.setting-item.full-width { flex-direction: column; gap: 16px; }
.setting-item.full-width .setting-control { width: 100%; margin-left: 0; }
.setting-item:last-child { border-bottom: none; }
.setting-label { flex: 1; }
.setting-label label { display: block; font-size: 16px; font-weight: 500; color: var(--text-primary); margin-bottom: 4px; }
.setting-description { font-size: 14px; color: var(--text-secondary); margin: 0; }
.setting-control { flex-shrink: 0; margin-left: 24px; }
.setting-textarea { width: 100%; padding: 16px; background: var(--input-bg); color: var(--text-primary); border: 1px solid var(--input-border); border-radius: 12px; font-size: 14px; resize: vertical; min-height: 100px; }
.setting-textarea:focus { outline: none; border-color: var(--input-focus-border); }
.char-count { text-align: right; font-size: 12px; color: var(--text-secondary); margin-top: 4px; }
.switch { position: relative; display: inline-block; width: 50px; height: 28px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: var(--secondary-bg); transition: 0.3s; border-radius: 28px; border: 2px solid var(--border-color); }
.slider:before { position: absolute; content: ''; height: 20px; width: 20px; left: 2px; bottom: 2px; background-color: white; transition: 0.3s; border-radius: 50%; }
input:checked + .slider { background-color: var(--accent-blue); border-color: var(--accent-blue); }
input:checked + .slider:before { transform: translateX(22px); }
.setting-actions { display: flex; gap: 16px; margin-top: 32px; padding-top: 24px; border-top: 1px solid var(--border-color); }
.save-btn { padding: 8px 32px; background: var(--accent-blue); color: white; border: none; border-radius: 12px; font-size: 14px; cursor: pointer; }
.save-btn:hover:not(:disabled) { opacity: 0.9; }
.save-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.reset-btn { padding: 8px 32px; background: transparent; color: var(--text-secondary); border: 1px solid var(--border-color); border-radius: 12px; font-size: 14px; cursor: pointer; }
.reset-btn:hover { background: var(--hover-bg); }
</style>