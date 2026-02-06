<template>
  <div class="help-center">
    <h3>帮助中心</h3>
    
    <div class="help-section">
      <h4>🚀 快速入门</h4>
      <div class="help-card">
        <p>Deep Research 是一个AI驱动的智能对话和研究平台。</p>
        <ul>
          <li>在左侧输入框输入问题，开始与AI对话</li>
          <li>点击"+"按钮创建新对话</li>
          <li>历史对话会自动保存在左侧列表中</li>
        </ul>
      </div>
    </div>

    <div class="help-section">
      <h4>💬 对话功能</h4>
      <div class="help-card">
        <ul>
          <li><strong>普通对话</strong> - 与AI进行日常问答交流</li>
          <li><strong>联网搜索</strong> - AI会搜索最新网络信息回答问题</li>
          <li><strong>深度思考</strong> - 使用推理模型进行复杂问题分析</li>
          <li><strong>深度研究</strong> - 多步骤研究，生成详细报告</li>
        </ul>
      </div>
    </div>

    <div class="help-section">
      <h4>⚙️ 设置说明</h4>
      <div class="help-card">
        <ul>
          <li><strong>通用设置</strong> - 选择AI模型和服务提供商</li>
          <li><strong>聊天记忆</strong> - 开启/关闭对话记忆，设置系统提示词</li>
          <li><strong>订阅</strong> - 查看配额使用情况，激活高级版</li>
        </ul>
      </div>
    </div>

    <div class="help-section">
      <h4>📊 订阅版本</h4>
      <div class="help-card">
        <div class="plan-info">
          <div class="plan">
            <strong>基础版（免费）</strong>
            <p>普通对话 10次/月，深度研究 1次/月</p>
          </div>
          <div class="plan">
            <strong>高级版</strong>
            <p>普通对话 50次/月，深度研究 10次/月</p>
            <p class="hint">通过激活码兑换</p>
          </div>
        </div>
      </div>
    </div>

    <div class="help-section">
      <h4>❓ 常见问题</h4>
      <div class="faq-list">
        <div class="faq-item" v-for="(faq, index) in faqs" :key="index">
          <div class="faq-question" @click="toggleFaq(index)">
            <span>{{ faq.q }}</span>
            <span class="faq-icon">{{ expandedFaq === index ? '−' : '+' }}</span>
          </div>
          <div class="faq-answer" v-if="expandedFaq === index">{{ faq.a }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';

const expandedFaq = ref(null);

const faqs = [
  { q: '如何切换深色/浅色模式？', a: '点击右上角头像，在下拉菜单中选择"亮色模式"或"暗色模式"。' },
  { q: '对话配额用完了怎么办？', a: '配额每月1日重置。您也可以通过激活码升级到高级版获得更多配额。' },
  { q: '如何使用深度研究功能？', a: '在输入框中输入研究问题，点击"深度研究"按钮。AI会进行多步骤研究并生成详细报告。' },
  { q: '聊天记录会保存多久？', a: '您的聊天记录会永久保存在服务器上，可以随时查看历史对话。' }
];

const toggleFaq = (index) => {
  expandedFaq.value = expandedFaq.value === index ? null : index;
};
</script>

<style scoped>
.help-center {
  max-width: 700px;
}

.help-center h3 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 24px;
  color: var(--text-primary);
}

.help-section {
  margin-bottom: 24px;
}

.help-section h4 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.help-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
}

.help-card p {
  margin: 0 0 12px 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.help-card ul {
  margin: 0;
  padding-left: 20px;
}

.help-card li {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.8;
}

.help-card li strong {
  color: var(--text-primary);
}

.plan-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.plan {
  padding: 12px;
  background: var(--secondary-bg);
  border-radius: 8px;
}

.plan strong {
  color: var(--text-primary);
  display: block;
  margin-bottom: 4px;
}

.plan p {
  margin: 0;
  font-size: 13px;
}

.plan .hint {
  color: var(--accent-blue);
  font-size: 12px;
  margin-top: 4px;
}

.faq-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.faq-item {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.faq-question {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-primary);
  transition: background 0.2s;
}

.faq-question:hover {
  background: var(--hover-bg);
}

.faq-icon {
  font-size: 18px;
  color: var(--text-secondary);
}

.faq-answer {
  padding: 0 16px 12px 16px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  border-top: 1px solid var(--border-color);
  padding-top: 12px;
  margin-top: 0;
}
</style>
