<template>
  <div class="paper-view-page">
    <!-- 侧边栏 -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <button class="back-btn" @click="$router.push('/paper')">← 返回</button>
      </div>
      <div class="paper-info">
        <h2 class="paper-title-nav">{{ result?.title }}</h2>
        <div class="paper-meta-nav">
          <span>{{ totalWords.toLocaleString() }} 字</span>
          <span>{{ result?.citation_count }} 条引用</span>
        </div>
      </div>
      <nav class="chapter-nav">
        <a
          v-for="ch in result?.chapters"
          :key="ch.id"
          class="chapter-nav-item"
          :class="{ active: activeChapter === ch.id }"
          @click.prevent="scrollToChapter(ch.id)"
        >
          {{ ch.title }}
        </a>
      </nav>
      <div class="sidebar-actions">
        <div class="export-group">
          <button class="export-btn" @click="openPreview('markdown')">👁 预览 Markdown</button>
          <button class="export-btn" @click="openPreview('latex')">👁 预览 LaTeX</button>
        </div>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content" ref="mainRef" @scroll="onScroll">
      <div v-if="loading" class="loading-state">
        <div class="spinner"></div> 加载中...
      </div>

      <template v-else-if="result">
        <header class="paper-header">
          <h1 class="paper-main-title">{{ result.title }}</h1>
          <div class="paper-badges">
            <span class="badge">{{ result.paper_type === 'liberal_arts' ? '文科' : '理科' }}</span>
            <span class="badge">{{ totalWords.toLocaleString() }} 字</span>
            <span class="badge">{{ result.review_rounds }} 轮审查</span>
          </div>
        </header>

        <!-- 章节 -->
        <section
          v-for="ch in result.chapters"
          :key="ch.id"
          :id="`chapter-${ch.id}`"
          class="chapter-section"
        >
          <div class="chapter-header">
            <h2 class="chapter-title">{{ ch.title }}</h2>
            <div class="chapter-actions">
              <span class="word-count">{{ chapterWords(ch) }} 字</span>
              <!-- 编辑模式切换 -->
              <template v-if="editingId === ch.id">
                <button class="action-btn save" :disabled="saving" @click="saveEdit(ch)">
                  {{ saving ? '保存中…' : '✓ 保存' }}
                </button>
                <button class="action-btn cancel" @click="cancelEdit">取消</button>
              </template>
              <template v-else>
                <button class="action-btn edit" @click="startEdit(ch)">✏ 编辑</button>
                <button
                  class="action-btn regen"
                  :disabled="regenLoading === ch.id"
                  @click="openRegenModal(ch)"
                >
                  {{ regenLoading === ch.id ? '生成中…' : '↺ 重新生成' }}
                </button>
              </template>
            </div>
          </div>

          <!-- 编辑模式：textarea -->
          <div v-if="editingId === ch.id" class="editor-wrap">
            <textarea
              v-model="editContent"
              class="chapter-editor"
              spellcheck="false"
              @keydown.ctrl.s.prevent="saveEdit(ch)"
              @keydown.meta.s.prevent="saveEdit(ch)"
            ></textarea>
            <p class="editor-hint">Ctrl+S 保存 · 支持 Markdown 语法</p>
          </div>

          <!-- 预览模式：渲染 Markdown -->
          <div v-else class="chapter-content" v-html="renderMarkdown(ch.content)"></div>
        </section>

        <!-- 参考文献 -->
        <section v-if="result.citations?.length" class="chapter-section">
          <h2 class="chapter-title">参考文献</h2>
          <ol class="citation-list">
            <li v-for="c in result.citations" :key="c.id" class="citation-item">
              <span class="citation-ref">{{ c.formatted_ref }}</span>
              <a v-if="c.url" :href="c.url" target="_blank" class="citation-link">🔗</a>
            </li>
          </ol>
        </section>
      </template>
    </main>

    <!-- 重新生成弹窗 -->
    <div v-if="regenModal" class="modal-overlay" @click.self="regenModal = null">
      <div class="modal">
        <h3>重新生成：{{ regenModal.title }}</h3>
        <div class="form-group">
          <label>修改建议（可选）</label>
          <textarea
            v-model="regenFeedback"
            placeholder="描述你希望如何改进这个章节..."
            rows="3"
          ></textarea>
        </div>
        <div class="modal-actions">
          <button @click="regenModal = null">取消</button>
          <button class="primary" @click="doRegen">开始重新生成</button>
        </div>
      </div>
    </div>

    <!-- 预览弹窗 -->
    <div v-if="previewVisible" class="preview-overlay" @click.self="previewVisible = false">
      <div class="preview-modal">
        <div class="preview-header">
          <div class="preview-tabs">
            <button
              :class="{ active: previewFormat === 'markdown' }"
              @click="switchPreview('markdown')"
            >Markdown 预览</button>
            <button
              :class="{ active: previewFormat === 'latex' }"
              @click="switchPreview('latex')"
            >LaTeX 源码</button>
          </div>
          <div class="preview-header-actions">
            <button class="download-btn" @click="exportPaper(previewFormat)">
              ⬇ 下载 {{ previewFormat === 'latex' ? '.tex' : '.md' }}
            </button>
            <button class="close-btn" @click="previewVisible = false">✕</button>
          </div>
        </div>

        <div class="preview-body">
          <div v-if="previewLoading" class="preview-loading">
            <div class="spinner"></div> 加载中...
          </div>
          <!-- Markdown：渲染效果 -->
          <div
            v-else-if="previewFormat === 'markdown'"
            class="preview-rendered"
            v-html="renderMarkdown(previewContent)"
          ></div>
          <!-- LaTeX：源码高亮 -->
          <pre v-else class="preview-source"><code>{{ previewContent }}</code></pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { paperAPI, createPaperSSE } from '@/api/paper'
import toast from '@/utils/toast'
import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({ html: false, linkify: true, typographer: true })

const route = useRoute()
const paperId = route.params.id

const result = ref(null)
const loading = ref(true)
const activeChapter = ref('')
const mainRef = ref(null)

// 编辑状态
const editingId = ref(null)
const editContent = ref('')
const saving = ref(false)

// 重新生成
const regenModal = ref(null)
const regenFeedback = ref('')
const regenLoading = ref(null)

// 预览
const previewVisible = ref(false)
const previewFormat = ref('markdown')
const previewContent = ref('')
const previewLoading = ref(false)

// 与后端 paper.CountWords 对齐：过滤空白、标点、Markdown 标记
const countWords = (text) => {
  if (!text) return 0
  return [...text].filter(c => {
    if (c.codePointAt(0) <= 0x20) return false
    if ('#*_`>-|~'.includes(c)) return false
    return !c.match(/\p{P}|\p{S}/u)
  }).length
}

// 计算总字数（实时反映编辑后的变化）
const totalWords = computed(() => {
  if (!result.value?.chapters) return result.value?.total_words || 0
  return result.value.chapters.reduce((sum, ch) => {
    if (editingId.value === ch.id) return sum + countWords(editContent.value)
    return sum + (ch.word_count || 0)
  }, 0)
})

const chapterWords = (ch) => {
  if (editingId.value === ch.id) return countWords(editContent.value).toLocaleString()
  return (ch.word_count || 0).toLocaleString()
}

// 与后端 paper.CountWords 对齐：过滤空白、标点、Markdown 标记
const countWords = (text) => {
  if (!text) return 0
  return [...text].filter(c => {
    const code = c.codePointAt(0)
    if (code <= 0x20) return false                          // 空白控制字符
    if ('#*_`>-|~'.includes(c)) return false               // Markdown 标记
    const cat = c.match(/\p{P}|\p{S}/u)                    // 标点/符号
    return !cat
  }).length
}

const renderMarkdown = (content) => {
  if (!content) return ''
  try { return md.render(content) } catch { return content }
}

const scrollToChapter = (id) => {
  document.getElementById(`chapter-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  activeChapter.value = id
}

const onScroll = () => {
  if (!result.value?.chapters) return
  for (const ch of [...result.value.chapters].reverse()) {
    const el = document.getElementById(`chapter-${ch.id}`)
    if (el && el.getBoundingClientRect().top <= 140) {
      activeChapter.value = ch.id
      break
    }
  }
}

// ── 内联编辑 ──────────────────────────────────────────
const startEdit = (ch) => {
  editingId.value = ch.id
  editContent.value = ch.content || ''
}

const cancelEdit = () => {
  editingId.value = null
  editContent.value = ''
}

const saveEdit = async (ch) => {
  if (saving.value) return
  saving.value = true
  try {
    await paperAPI.updateChapter(ch.id, paperId, editContent.value)
    // 本地更新，不重新请求
    const idx = result.value.chapters.findIndex(c => c.id === ch.id)
    if (idx !== -1) {
      result.value.chapters[idx] = {
        ...result.value.chapters[idx],
        content: editContent.value,
        word_count: countWords(editContent.value)
      }
    }
    editingId.value = null
    toast.success('保存成功')
  } catch (e) {
    toast.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

// ── 导出 ──────────────────────────────────────────────
const exportPaper = async (format) => {
  try {
    const blob = await paperAPI.export(paperId, format)
    const ext = format === 'latex' ? '.tex' : '.md'
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${result.value?.title || 'paper'}${ext}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    toast.error('导出失败')
  }
}

// ── 预览 ──────────────────────────────────────────────
const openPreview = async (format) => {
  previewFormat.value = format
  previewVisible.value = true
  previewLoading.value = true
  previewContent.value = ''
  try {
    const data = await paperAPI.preview(paperId, format)
    previewContent.value = data.content
  } catch (e) {
    toast.error('预览加载失败')
    previewVisible.value = false
  } finally {
    previewLoading.value = false
  }
}

const switchPreview = async (format) => {
  if (format === previewFormat.value) return
  previewFormat.value = format
  previewLoading.value = true
  previewContent.value = ''
  try {
    const data = await paperAPI.preview(paperId, format)
    previewContent.value = data.content
  } catch (e) {
    toast.error('切换预览失败')
  } finally {
    previewLoading.value = false
  }
}

// ── 重新生成 ──────────────────────────────────────────
const openRegenModal = (ch) => {
  regenModal.value = ch
  regenFeedback.value = ''
}

const doRegen = async () => {
  const ch = regenModal.value
  regenModal.value = null
  regenLoading.value = ch.id

  try {
    await paperAPI.regenerateChapter({
      paper_id: paperId,
      chapter_id: ch.id,
      feedback: regenFeedback.value
    })
    toast.info('重新生成已启动，请稍候…')

    // 10 分钟超时保护
    const timeout = setTimeout(() => {
      close()
      regenLoading.value = null
      toast.warning('章节重新生成超时，请稍后刷新查看结果')
    }, 10 * 60 * 1000)

    const close = createPaperSSE(paperId, async (evt) => {
      if (evt.type === 'chapter_regenerated') {
        clearTimeout(timeout)
        close()
        regenLoading.value = null
        toast.success(`${ch.title} 重新生成完成`)
        const data = await paperAPI.getResult(paperId)
        result.value = data
      } else if (evt.type === 'error') {
        clearTimeout(timeout)
        close()
        regenLoading.value = null
        toast.error('章节重新生成失败')
      }
    }, () => {
      clearTimeout(timeout)
      regenLoading.value = null
    })
  } catch (e) {
    regenLoading.value = null
    toast.error(e.message || '操作失败')
  }
}

onMounted(async () => {
  try {
    const data = await paperAPI.getResult(paperId)
    result.value = data
    if (data.chapters?.length) activeChapter.value = data.chapters[0].id
  } catch (e) {
    toast.error('加载论文失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.paper-view-page {
  display: flex;
  height: 100vh;
  background: var(--primary-bg, #0a0a0a);
  color: var(--text-primary, #f0f0f0);
  overflow: hidden;
}

/* ── 侧边栏 ── */
.sidebar {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid rgba(255,255,255,0.07);
  display: flex;
  flex-direction: column;
  padding: 20px 0;
  overflow-y: auto;
}
.sidebar-header { padding: 0 16px 16px; }
.back-btn {
  background: none;
  border: 1px solid rgba(255,255,255,0.1);
  color: var(--text-secondary, #aaa);
  padding: 7px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}
.back-btn:hover { color: #fff; border-color: rgba(255,255,255,0.25); }

.paper-info { padding: 0 16px 16px; border-bottom: 1px solid rgba(255,255,255,0.06); }
.paper-title-nav { font-size: 13px; font-weight: 600; margin: 0 0 8px; line-height: 1.4; }
.paper-meta-nav { display: flex; gap: 10px; font-size: 11px; color: var(--text-secondary, #aaa); }

.chapter-nav { flex: 1; padding: 10px 0; }
.chapter-nav-item {
  display: block;
  padding: 7px 16px;
  font-size: 12px;
  color: var(--text-secondary, #aaa);
  text-decoration: none;
  cursor: pointer;
  border-left: 2px solid transparent;
  transition: all 0.15s;
}
.chapter-nav-item:hover { color: #fff; background: rgba(255,255,255,0.04); }
.chapter-nav-item.active { color: #60a5fa; border-left-color: #3b82f6; background: rgba(59,130,246,0.06); }

.sidebar-actions { padding: 14px 16px; border-top: 1px solid rgba(255,255,255,0.06); }
.export-group { display: flex; flex-direction: column; gap: 6px; }
.export-btn {
  width: 100%;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-secondary, #aaa);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}
.export-btn:hover { border-color: rgba(255,255,255,0.25); color: #fff; }

/* ── 主内容 ── */
.main-content {
  flex: 1;
  overflow-y: auto;
  padding: 40px 60px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 0;
  color: var(--text-secondary, #aaa);
}
.spinner {
  width: 20px; height: 20px;
  border: 2px solid rgba(255,255,255,0.1);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.paper-header { margin-bottom: 40px; padding-bottom: 24px; border-bottom: 1px solid rgba(255,255,255,0.07); }
.paper-main-title { font-size: 26px; font-weight: 700; margin: 0 0 14px; line-height: 1.3; }
.paper-badges { display: flex; gap: 8px; flex-wrap: wrap; }
.badge {
  font-size: 11px;
  padding: 3px 10px;
  border-radius: 20px;
  background: rgba(59,130,246,0.12);
  color: #60a5fa;
  border: 1px solid rgba(59,130,246,0.2);
}

/* ── 章节 ── */
.chapter-section { margin-bottom: 48px; scroll-margin-top: 80px; }
.chapter-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  gap: 12px;
}
.chapter-title { font-size: 19px; font-weight: 600; margin: 0; flex: 1; }
.chapter-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.word-count { font-size: 11px; color: var(--text-secondary, #aaa); }

.action-btn {
  padding: 5px 12px;
  border-radius: 7px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.action-btn.edit { color: #94a3b8; }
.action-btn.edit:hover { border-color: #3b82f6; color: #60a5fa; }
.action-btn.regen { color: #94a3b8; }
.action-btn.regen:hover:not(:disabled) { border-color: #7c3aed; color: #a78bfa; }
.action-btn.regen:disabled { opacity: 0.4; cursor: not-allowed; }
.action-btn.save { border-color: rgba(34,197,94,0.4); color: #4ade80; }
.action-btn.save:hover:not(:disabled) { background: rgba(34,197,94,0.1); }
.action-btn.save:disabled { opacity: 0.5; cursor: not-allowed; }
.action-btn.cancel { color: #94a3b8; }
.action-btn.cancel:hover { border-color: rgba(239,68,68,0.4); color: #f87171; }

/* ── 编辑器 ── */
.editor-wrap { position: relative; }
.chapter-editor {
  width: 100%;
  min-height: 320px;
  padding: 16px;
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(59,130,246,0.3);
  border-radius: 10px;
  color: var(--text-primary, #f0f0f0);
  font-size: 14px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  line-height: 1.7;
  resize: vertical;
  box-sizing: border-box;
  outline: none;
  transition: border-color 0.2s;
}
.chapter-editor:focus { border-color: #3b82f6; }
.editor-hint {
  font-size: 11px;
  color: rgba(255,255,255,0.25);
  margin: 6px 0 0;
  text-align: right;
}

/* ── 内容渲染 ── */
.chapter-content {
  line-height: 1.85;
  font-size: 15px;
  color: rgba(255,255,255,0.85);
}
.chapter-content :deep(h2), .chapter-content :deep(h3) {
  color: var(--text-primary, #f0f0f0);
  margin: 1.4em 0 0.5em;
}
.chapter-content :deep(p) { margin: 0 0 1em; }
.chapter-content :deep(ul), .chapter-content :deep(ol) { padding-left: 1.5em; margin: 0 0 1em; }
.chapter-content :deep(strong) { color: #e2e8f0; }
.chapter-content :deep(code) {
  background: rgba(255,255,255,0.08);
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 13px;
}

/* ── 参考文献 ── */
.citation-list { padding-left: 1.5em; }
.citation-item {
  margin-bottom: 10px;
  font-size: 13px;
  color: rgba(255,255,255,0.65);
  line-height: 1.6;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.citation-link { color: #60a5fa; text-decoration: none; flex-shrink: 0; }

/* ── 弹窗 ── */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal {
  background: #1a1a2e;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 16px;
  padding: 28px;
  width: 420px;
}
.modal h3 { margin: 0 0 20px; font-size: 17px; }
.form-group label { display: block; font-size: 13px; color: var(--text-secondary, #aaa); margin-bottom: 8px; }
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  background: rgba(255,255,255,0.05);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px;
  color: var(--text-primary, #f0f0f0);
  font-size: 14px;
  resize: vertical;
  box-sizing: border-box;
  outline: none;
}
.form-group textarea:focus { border-color: #3b82f6; }
.modal-actions { display: flex; gap: 12px; justify-content: flex-end; margin-top: 20px; }
.modal-actions button {
  padding: 9px 20px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-primary, #f0f0f0);
  cursor: pointer;
  font-size: 14px;
  transition: all 0.15s;
}
.modal-actions button:hover { border-color: rgba(255,255,255,0.25); }
.modal-actions button.primary {
  background: linear-gradient(135deg, #3b82f6, #7c3aed);
  border-color: transparent;
}

@media (max-width: 768px) {
  .sidebar { display: none; }
  .main-content { padding: 24px 20px; }
  .chapter-header { flex-wrap: wrap; }
}

/* ── 预览弹窗 ── */
.preview-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1100;
  padding: 24px;
}

.preview-modal {
  background: #0f172a;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 16px;
  width: 100%;
  max-width: 860px;
  height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255,255,255,0.07);
  flex-shrink: 0;
}

.preview-tabs {
  display: flex;
  gap: 4px;
  background: rgba(255,255,255,0.05);
  border-radius: 8px;
  padding: 3px;
}
.preview-tabs button {
  padding: 6px 16px;
  border-radius: 6px;
  border: none;
  background: none;
  color: var(--text-secondary, #aaa);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.preview-tabs button.active {
  background: rgba(59,130,246,0.25);
  color: #60a5fa;
}

.preview-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.download-btn {
  padding: 7px 16px;
  border-radius: 8px;
  border: 1px solid rgba(59,130,246,0.4);
  background: rgba(59,130,246,0.1);
  color: #60a5fa;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}
.download-btn:hover { background: rgba(59,130,246,0.2); }

.close-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: none;
  color: var(--text-secondary, #aaa);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}
.close-btn:hover { border-color: rgba(239,68,68,0.4); color: #f87171; }

.preview-body {
  flex: 1;
  overflow-y: auto;
  padding: 28px 36px;
}

.preview-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 100%;
  color: var(--text-secondary, #aaa);
}

/* Markdown 渲染样式 */
.preview-rendered {
  line-height: 1.85;
  font-size: 15px;
  color: rgba(255,255,255,0.88);
}
.preview-rendered :deep(h1) { font-size: 22px; font-weight: 700; margin: 0 0 20px; padding-bottom: 12px; border-bottom: 1px solid rgba(255,255,255,0.08); }
.preview-rendered :deep(h2) { font-size: 18px; font-weight: 600; margin: 2em 0 0.6em; color: #e2e8f0; }
.preview-rendered :deep(h3) { font-size: 15px; font-weight: 600; margin: 1.5em 0 0.5em; color: #cbd5e1; }
.preview-rendered :deep(p) { margin: 0 0 1em; }
.preview-rendered :deep(ul), .preview-rendered :deep(ol) { padding-left: 1.5em; margin: 0 0 1em; }
.preview-rendered :deep(li) { margin-bottom: 4px; }
.preview-rendered :deep(strong) { color: #f1f5f9; }
.preview-rendered :deep(blockquote) {
  border-left: 3px solid #3b82f6;
  padding-left: 16px;
  margin: 0 0 1em;
  color: rgba(255,255,255,0.6);
}
.preview-rendered :deep(code) {
  background: rgba(255,255,255,0.08);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: monospace;
}
.preview-rendered :deep(pre) {
  background: rgba(255,255,255,0.05);
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  margin: 0 0 1em;
}

/* LaTeX 源码 */
.preview-source {
  margin: 0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.7;
  color: #a5f3fc;
  white-space: pre-wrap;
  word-break: break-all;
}
.preview-source code { font-family: inherit; color: inherit; background: none; }
</style>
