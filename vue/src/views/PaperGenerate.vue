<template>
  <div class="paper-generate-page">
    <div class="page-header">
      <button class="back-btn" @click="$router.push('/paper')">← 返回</button>
      <h1 class="page-title">新建论文</h1>
    </div>

    <form class="generate-form" @submit.prevent="submit">
      <!-- 基本信息 -->
      <section class="form-section">
        <h2 class="section-title">基本信息</h2>

        <div class="form-group">
          <label>论文标题 <span class="required">*</span></label>
          <input
            v-model="form.title"
            type="text"
            placeholder="例：人工智能在医疗诊断中的应用研究"
            maxlength="200"
            required
          />
          <span class="char-count">{{ form.title.length }}/200</span>
        </div>

        <div class="form-group">
          <label>研究主题 <span class="required">*</span></label>
          <textarea
            v-model="form.topic"
            placeholder="详细描述论文的研究方向、核心问题、研究背景等（至少10字）"
            rows="4"
            maxlength="5000"
            required
          ></textarea>
          <span class="char-count">{{ form.topic.length }}/5000</span>
        </div>

        <div class="form-group">
          <label>参考内容（可选）</label>
          <textarea
            v-model="form.input_content"
            placeholder="粘贴已有的研究笔记、摘要或参考资料，AI 将参考这些内容生成论文"
            rows="3"
          ></textarea>
        </div>
      </section>

      <!-- 论文设置 -->
      <section class="form-section">
        <h2 class="section-title">论文设置</h2>

        <div class="form-row">
          <div class="form-group">
            <label>论文类型 <span class="required">*</span></label>
            <div class="type-selector">
              <button
                type="button"
                :class="{ active: form.paper_type === 'liberal_arts' }"
                @click="form.paper_type = 'liberal_arts'"
              >
                📚 文科论文
              </button>
              <button
                type="button"
                :class="{ active: form.paper_type === 'science' }"
                @click="form.paper_type = 'science'"
              >
                🔬 理科论文
              </button>
            </div>
          </div>

          <div class="form-group">
            <label>目标字数 <span class="required">*</span></label>
            <div class="word-count-selector">
              <button
                v-for="opt in wordCountOptions"
                :key="opt.value"
                type="button"
                :class="{ active: form.target_words === opt.value }"
                @click="form.target_words = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
            <input
              v-model.number="form.target_words"
              type="number"
              min="1000"
              max="100000"
              placeholder="自定义字数"
              class="custom-words-input"
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label>引用格式</label>
            <select v-model="form.options.citation_style">
              <option v-for="s in citationStyles" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
          </div>

          <div class="form-group">
            <label>审查轮数</label>
            <select v-model.number="form.options.max_review_rounds">
              <option :value="1">1轮（快速）</option>
              <option :value="2">2轮（标准）</option>
              <option :value="3">3轮（精细）</option>
            </select>
          </div>
        </div>
      </section>

      <!-- 预览章节结构 -->
      <section v-if="previewChapters.length" class="form-section">
        <h2 class="section-title">章节结构预览</h2>
        <div class="chapter-preview">
          <div v-for="ch in previewChapters" :key="ch.type" class="chapter-item">
            <span class="chapter-title">{{ ch.title }}</span>
            <span class="chapter-words">{{ ch.min_words }}–{{ ch.max_words }} 字</span>
          </div>
        </div>
      </section>

      <div class="form-footer">
        <p class="estimate-note">
          预计生成时间：{{ estimateTime }} 分钟（取决于网络和模型响应速度）
        </p>
        <button type="submit" class="submit-btn" :disabled="submitting">
          <span v-if="submitting">
            <span class="btn-spinner"></span> 启动中...
          </span>
          <span v-else>🚀 开始生成论文</span>
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { paperAPI } from '@/api/paper'
import toast from '@/utils/toast'

const router = useRouter()
const submitting = ref(false)
const citationStyles = ref([])
const templates = ref([])

const form = ref({
  title: '',
  topic: '',
  input_content: '',
  target_words: 5000,
  paper_type: 'liberal_arts',
  options: {
    citation_style: 'chinese-gb',
    max_review_rounds: 2
  }
})

const wordCountOptions = [
  { label: '3000字', value: 3000 },
  { label: '5000字', value: 5000 },
  { label: '8000字', value: 8000 },
  { label: '10000字', value: 10000 },
]

const previewChapters = computed(() => {
  const tmpl = templates.value.find(t => t.type === form.value.paper_type)
  return tmpl?.chapters || []
})

const estimateTime = computed(() => {
  const base = Math.ceil(form.value.target_words / 1000) * 2
  return Math.max(5, Math.min(30, base))
})

const submit = async () => {
  if (!form.value.title.trim() || !form.value.topic.trim()) {
    toast.error('请填写标题和研究主题')
    return
  }
  submitting.value = true
  try {
    const session = await paperAPI.start(form.value)
    toast.success('论文生成已启动')
    router.push(`/paper/${session.id}/progress`)
  } catch (e) {
    toast.error(e.message || '启动失败')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  try {
    const [styles, tmplData] = await Promise.all([
      paperAPI.getCitationStyles(),
      paperAPI.getTemplates()
    ])
    citationStyles.value = styles
    templates.value = tmplData
  } catch {
    toast.warning('引用格式和章节模板加载失败，将使用默认配置')
    // 提供兜底默认值，确保表单可用
    citationStyles.value = [
      { id: 'chinese-gb', name: 'GB/T 7714 国标格式' },
      { id: 'apa', name: 'APA 格式' },
      { id: 'mla', name: 'MLA 格式' },
      { id: 'latex', name: 'LaTeX/BibTeX 格式' }
    ]
  }
})
</script>

<style scoped>
.paper-generate-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 32px 24px;
  min-height: 100vh;
  background: var(--primary-bg, #0a0a0a);
  color: var(--text-primary, #f0f0f0);
}

.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
}

.back-btn {
  background: none;
  border: 1px solid rgba(255,255,255,0.15);
  color: var(--text-secondary, #aaa);
  padding: 8px 14px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}
.back-btn:hover { border-color: rgba(255,255,255,0.3); color: #fff; }

.page-title { font-size: 24px; font-weight: 700; margin: 0; }

.form-section {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 14px;
  padding: 24px;
  margin-bottom: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 20px;
  color: #60a5fa;
}

.form-group {
  margin-bottom: 20px;
  position: relative;
}
.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 8px;
  color: var(--text-secondary, #ccc);
}
.required { color: #f87171; }

.form-group input[type="text"],
.form-group input[type="number"],
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 11px 14px;
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 10px;
  color: var(--text-primary, #f0f0f0);
  font-size: 14px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}
.form-group input:focus,
.form-group textarea:focus,
.form-group select:focus {
  outline: none;
  border-color: #3b82f6;
}
.form-group textarea { resize: vertical; }
.form-group select option { background: #1a1a2e; }

.char-count {
  position: absolute;
  right: 12px;
  bottom: 10px;
  font-size: 11px;
  color: rgba(255,255,255,0.3);
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.type-selector, .word-count-selector {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}
.type-selector button, .word-count-selector button {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-secondary, #aaa);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.type-selector button.active, .word-count-selector button.active {
  background: rgba(59,130,246,0.2);
  border-color: #3b82f6;
  color: #60a5fa;
}

.custom-words-input {
  margin-top: 8px;
}

.chapter-preview {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 8px;
}
.chapter-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(255,255,255,0.03);
  border-radius: 8px;
  font-size: 13px;
}
.chapter-title { color: var(--text-primary, #f0f0f0); }
.chapter-words { color: var(--text-secondary, #aaa); font-size: 11px; }

.form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}
.estimate-note { font-size: 13px; color: var(--text-secondary, #aaa); margin: 0; }

.submit-btn {
  background: linear-gradient(135deg, #3b82f6, #7c3aed);
  color: #fff;
  border: none;
  padding: 13px 32px;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}
.submit-btn:hover:not(:disabled) { opacity: 0.85; }
.submit-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-spinner {
  display: inline-block;
  width: 14px; height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

@media (max-width: 600px) {
  .form-row { grid-template-columns: 1fr; }
  .form-footer { flex-direction: column; gap: 16px; align-items: stretch; }
  .submit-btn { justify-content: center; }
}
</style>
