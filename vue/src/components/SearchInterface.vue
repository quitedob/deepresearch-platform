<template>
  <div class="search-interface">
    <div class="search-header">
      <h2>智能搜索</h2>
      <div class="header-actions">
        <button @click="showAdvancedSearch = !showAdvancedSearch" class="btn btn-outline">
          {{ showAdvancedSearch ? '简单搜索' : '高级搜索' }}
        </button>
        <button @click="showSearchHistory = true" class="btn btn-outline">
          📜 搜索历史
        </button>
      </div>
    </div>

    <div class="search-content">
      <!-- 搜索输入区域 -->
      <div class="search-input-section">
        <div class="search-container">
          <div class="search-input-wrapper">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="输入搜索内容..."
              class="search-input"
              @keyup.enter="performSearch"
              @input="handleInputChange"
            />
            <div class="search-suggestions" v-if="suggestions.length > 0 && showSuggestions">
              <div
                v-for="(suggestion, index) in suggestions"
                :key="index"
                class="suggestion-item"
                @click="selectSuggestion(suggestion)"
              >
                <span class="suggestion-icon">🔍</span>
                <span class="suggestion-text">{{ suggestion }}</span>
              </div>
            </div>
          </div>
          <div class="search-controls">
            <select v-model="searchScope" class="search-scope">
              <option value="all">全部内容</option>
              <option value="documents">文档</option>
              <option value="knowledge_base">知识库</option>
              <option value="web">网络</option>
              <option value="images">图片</option>
            </select>
            <button @click="performSearch" class="btn btn-primary" :disabled="!searchQuery.trim() || searching">
              {{ searching ? '搜索中...' : '搜索' }}
            </button>
          </div>
        </div>

        <!-- 高级搜索选项 -->
        <div v-if="showAdvancedSearch" class="advanced-search">
          <div class="advanced-filters">
            <div class="filter-group">
              <label>时间范围</label>
              <select v-model="advancedFilters.timeRange" class="filter-select">
                <option value="">不限</option>
                <option value="today">今天</option>
                <option value="week">最近一周</option>
                <option value="month">最近一个月</option>
                <option value="year">最近一年</option>
              </select>
            </div>

            <div class="filter-group">
              <label>内容类型</label>
              <select v-model="advancedFilters.contentType" class="filter-select">
                <option value="">全部类型</option>
                <option value="text">文本</option>
                <option value="image">图片</option>
                <option value="video">视频</option>
                <option value="audio">音频</option>
                <option value="document">文档</option>
              </select>
            </div>

            <div class="filter-group">
              <label>语言</label>
              <select v-model="advancedFilters.language" class="filter-select">
                <option value="">全部语言</option>
                <option value="zh">中文</option>
                <option value="en">英文</option>
                <option value="ja">日文</option>
                <option value="ko">韩文</option>
              </select>
            </div>

            <div class="filter-group">
              <label>排序方式</label>
              <select v-model="advancedFilters.sortBy" class="filter-select">
                <option value="relevance">相关性</option>
                <option value="date">日期</option>
                <option value="popularity">热度</option>
                <option value="rating">评分</option>
              </select>
            </div>

            <div class="filter-group">
              <label>结果数量</label>
              <select v-model="advancedFilters.limit" class="filter-select">
                <option value="10">10条</option>
                <option value="20">20条</option>
                <option value="50">50条</option>
                <option value="100">100条</option>
              </select>
            </div>
          </div>

          <div class="advanced-options">
            <div class="option-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="advancedFilters.exactMatch" />
                精确匹配
              </label>
              <label class="checkbox-label">
                <input type="checkbox" v-model="advancedFilters.includeSynonyms" />
                包含同义词
              </label>
              <label class="checkbox-label">
                <input type="checkbox" v-model="advancedFilters.safeSearch" />
                安全搜索
              </label>
            </div>
          </div>
        </div>
      </div>

      <!-- 搜索结果区域 -->
      <div class="search-results-section">
        <div v-if="searching" class="searching-state">
          <div class="searching-spinner">⟳</div>
          <p>正在搜索中...</p>
        </div>

        <div v-else-if="!searchPerformed && !searchResults.length" class="search-welcome">
          <div class="welcome-icon">🔍</div>
          <h3>开始智能搜索</h3>
          <p>输入关键词搜索文档、知识库、网络内容等</p>
          <div class="quick-searches">
            <h4>热门搜索</h4>
            <div class="quick-search-tags">
              <span
                v-for="tag in popularSearches"
                :key="tag"
                class="quick-search-tag"
                @click="searchQuery = tag; performSearch()"
              >
                {{ tag }}
              </span>
            </div>
          </div>
        </div>

        <div v-else-if="searchResults.length === 0" class="no-results">
          <div class="no-results-icon">📭</div>
          <h3>未找到相关结果</h3>
          <p>尝试使用不同的关键词或调整搜索条件</p>
          <div class="search-suggestions">
            <h4>搜索建议</h4>
            <ul>
              <li>检查拼写是否正确</li>
              <li>使用更通用的关键词</li>
              <li>尝试相关词汇或同义词</li>
              <li>减少搜索条件限制</li>
            </ul>
          </div>
        </div>

        <div v-else class="search-results">
          <div class="results-header">
            <div class="results-info">
              <span class="results-count">找到 {{ totalResults }} 条结果</span>
              <span class="search-time">耗时 {{ searchTime }}ms</span>
            </div>
            <div class="results-actions">
              <button @click="saveSearch" class="btn btn-sm btn-outline">
                💾 保存搜索
              </button>
              <button @click="exportResults" class="btn btn-sm btn-outline">
                📥 导出结果
              </button>
            </div>
          </div>

          <div class="results-list">
            <div
              v-for="(result, index) in searchResults"
              :key="result.id"
              class="result-item"
              :class="{ 'result-expanded': expandedResults.includes(result.id) }"
            >
              <div class="result-header">
                <div class="result-rank">{{ index + 1 }}</div>
                <div class="result-info">
                  <h3 class="result-title" v-html="highlightText(result.title)"></h3>
                  <div class="result-meta">
                    <span class="result-type" :class="result.type">
                      {{ getTypeIcon(result.type) }} {{ getTypeText(result.type) }}
                    </span>
                    <span class="result-source">{{ result.source }}</span>
                    <span class="result-date">{{ formatDate(result.date) }}</span>
                    <span class="result-score">相关度: {{ Math.round(result.score * 100) }}%</span>
                  </div>
                </div>
                <div class="result-actions">
                  <button @click="toggleResultExpand(result.id)" class="expand-btn">
                    {{ expandedResults.includes(result.id) ? '收起' : '展开' }}
                  </button>
                </div>
              </div>

              <div class="result-content">
                <p class="result-description" v-html="highlightText(result.description)"></p>

                <div v-if="result.snippets && result.snippets.length > 0" class="result-snippets">
                  <div
                    v-for="(snippet, snippetIndex) in result.snippets"
                    :key="snippetIndex"
                    class="snippet"
                    v-html="highlightText(snippet)"
                  ></div>
                </div>

                <div v-if="expandedResults.includes(result.id)" class="result-expanded-content">
                  <div v-if="result.fullContent" class="full-content">
                    <h4>完整内容</h4>
                    <div v-html="highlightText(result.fullContent)"></div>
                  </div>

                  <div v-if="result.tags && result.tags.length > 0" class="result-tags">
                    <h4>标签</h4>
                    <div class="tags-list">
                      <span v-for="tag in result.tags" :key="tag" class="tag">
                        {{ tag }}
                      </span>
                    </div>
                  </div>

                  <div v-if="result.metadata" class="result-metadata">
                    <h4>元数据</h4>
                    <div class="metadata-grid">
                      <div
                        v-for="(value, key) in result.metadata"
                        :key="key"
                        class="metadata-item"
                      >
                        <span class="metadata-key">{{ key }}:</span>
                        <span class="metadata-value">{{ value }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="result-footer">
                <div class="result-links">
                  <a v-if="result.url" :href="result.url" target="_blank" class="result-link">
                    🔗 查看原文
                  </a>
                  <button @click="viewDetails(result)" class="result-link">
                    📄 查看详情
                  </button>
                  <button @click="shareResult(result)" class="result-link">
                    🔗 分享
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="totalPages > 1" class="pagination">
            <button
              @click="prevPage"
              :disabled="currentPage === 1"
              class="btn btn-sm btn-outline"
            >
              ← 上一页
            </button>
            <div class="page-numbers">
              <button
                v-for="page in visiblePages"
                :key="page"
                class="btn btn-sm"
                :class="{ 'btn-primary': page === currentPage, 'btn-outline': page !== currentPage }"
                @click="goToPage(page)"
              >
                {{ page }}
              </button>
            </div>
            <button
              @click="nextPage"
              :disabled="currentPage === totalPages"
              class="btn btn-sm btn-outline"
            >
              下一页 →
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 搜索历史模态框 -->
    <div v-if="showSearchHistory" class="modal-overlay" @click="closeSearchHistory">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>搜索历史</h3>
          <button @click="closeSearchHistory" class="btn-close">×</button>
        </div>
        <div class="modal-body">
          <div class="history-header">
            <button @click="clearSearchHistory" class="btn btn-sm btn-outline">
              清空历史
            </button>
          </div>
          <div class="history-list">
            <div
              v-for="(item, index) in searchHistory"
              :key="index"
              class="history-item"
            >
              <div class="history-query" @click="searchFromHistory(item.query)">
                {{ item.query }}
              </div>
              <div class="history-meta">
                <span class="history-time">{{ formatTime(item.timestamp) }}</span>
                <span class="history-results">{{ item.results }} 条结果</span>
              </div>
              <button @click="removeHistoryItem(index)" class="remove-history">
                ×
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 保存搜索模态框 -->
    <div v-if="showSaveSearchModal" class="modal-overlay" @click="closeSaveSearchModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>保存搜索</h3>
          <button @click="closeSaveSearchModal" class="btn-close">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>搜索名称</label>
            <input
              v-model="savedSearchName"
              type="text"
              placeholder="输入搜索名称"
              class="form-input"
            />
          </div>
          <div class="form-group">
            <label>描述（可选）</label>
            <textarea
              v-model="savedSearchDescription"
              placeholder="输入搜索描述"
              class="form-textarea"
              rows="3"
            ></textarea>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeSaveSearchModal" class="btn btn-outline">取消</button>
          <button @click="confirmSaveSearch" class="btn btn-primary" :disabled="!savedSearchName.trim()">
            保存
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'

// 响应式数据
const searchQuery = ref('')
const searchScope = ref('all')
const searching = ref(false)
const searchPerformed = ref(false)
const searchResults = ref([])
const totalResults = ref(0)
const searchTime = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const expandedResults = ref([])

// 高级搜索
const showAdvancedSearch = ref(false)
const advancedFilters = ref({
  timeRange: '',
  contentType: '',
  language: '',
  sortBy: 'relevance',
  limit: '20',
  exactMatch: false,
  includeSynonyms: false,
  safeSearch: false
})

// 搜索建议
const suggestions = ref([])
const showSuggestions = ref(false)
const suggestionTimer = ref(null)

// 搜索历史 - 从 localStorage 加载
const showSearchHistory = ref(false)
const searchHistory = ref([])

// 从 localStorage 加载搜索历史
const loadSearchHistory = () => {
  try {
    const saved = localStorage.getItem('search_history')
    if (saved) {
      searchHistory.value = JSON.parse(saved).map(item => ({
        ...item,
        timestamp: new Date(item.timestamp)
      }))
    }
  } catch (e) {
    searchHistory.value = []
  }
}

// 保存搜索历史到 localStorage
const saveSearchHistory = () => {
  try {
    localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
  } catch (e) {
    console.warn('保存搜索历史失败:', e)
  }
}

// 保存搜索
const showSaveSearchModal = ref(false)
const savedSearchName = ref('')
const savedSearchDescription = ref('')

// 热门搜索
const popularSearches = ref([
  '人工智能',
  '机器学习',
  'Vue.js',
  'React',
  'Python编程',
  '数据分析',
  'Web开发',
  '云计算'
])

// 计算属性
const totalPages = computed(() => {
  return Math.ceil(totalResults.value / pageSize.value)
})

const visiblePages = computed(() => {
  const pages = []
  const start = Math.max(1, currentPage.value - 2)
  const end = Math.min(totalPages.value, currentPage.value + 2)

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  return pages
})

// 方法
const performSearch = async () => {
  if (!searchQuery.value.trim()) return

  searching.value = true
  searchPerformed.value = true
  showSuggestions.value = false

  const startTime = Date.now()

  try {
    // TODO: 接入实际搜索 API，当前为占位实现
    // const response = await searchAPI.search({
    //   query: searchQuery.value,
    //   scope: searchScope.value,
    //   filters: showAdvancedSearch.value ? advancedFilters.value : {},
    //   page: currentPage.value,
    //   page_size: pageSize.value
    // })
    // searchResults.value = response.data.results
    // totalResults.value = response.data.total

    // 占位：无后端搜索 API 时返回空结果
    searchResults.value = []
    totalResults.value = 0
    searchTime.value = Date.now() - startTime

    // 添加到搜索历史
    if (!searchHistory.value.find(item => item.query === searchQuery.value)) {
      searchHistory.value.unshift({
        query: searchQuery.value,
        timestamp: new Date(),
        results: totalResults.value
      })
      saveSearchHistory()
    }

    currentPage.value = 1
  } catch (error) {
    console.error('搜索失败:', error)
  } finally {
    searching.value = false
  }
}

const handleInputChange = () => {
  showSuggestions.value = true

  if (suggestionTimer.value) {
    clearTimeout(suggestionTimer.value)
  }

  suggestionTimer.value = setTimeout(() => {
    if (searchQuery.value.trim()) {
      // 模拟搜索建议
      suggestions.value = [
        searchQuery.value + ' 教程',
        searchQuery.value + ' 示例',
        searchQuery.value + ' 最佳实践',
        searchQuery.value + ' 入门指南'
      ].slice(0, 5)
    } else {
      suggestions.value = []
    }
  }, 300)
}

const selectSuggestion = (suggestion) => {
  searchQuery.value = suggestion
  showSuggestions.value = false
  performSearch()
}

const highlightText = (text) => {
  if (!searchQuery.value.trim()) return text

  const regex = new RegExp(`(${searchQuery.value})`, 'gi')
  return text.replace(regex, '<mark>$1</mark>')
}

const toggleResultExpand = (resultId) => {
  const index = expandedResults.value.indexOf(resultId)
  if (index > -1) {
    expandedResults.value.splice(index, 1)
  } else {
    expandedResults.value.push(resultId)
  }
}

const viewDetails = (result) => {
  console.log('查看详情:', result)
}

const shareResult = (result) => {
  console.log('分享结果:', result)
}

const saveSearch = () => {
  showSaveSearchModal.value = true
  savedSearchName.value = searchQuery.value
}

const exportResults = () => {
  console.log('导出结果')

  const exportData = searchResults.value.map(result => ({
    标题: result.title,
    描述: result.description,
    类型: getTypeText(result.type),
    来源: result.source,
    日期: formatDate(result.date),
    相关度: Math.round(result.score * 100) + '%',
    链接: result.url || ''
  }))

  const csvContent = [
    Object.keys(exportData[0]).join(','),
    ...exportData.map(row => Object.values(row).map(value => `"${value}"`).join(','))
  ].join('\n')

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `search_results_${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(link.href)
}

const searchFromHistory = (query) => {
  searchQuery.value = query
  closeSearchHistory()
  performSearch()
}

const removeHistoryItem = (index) => {
  searchHistory.value.splice(index, 1)
}

const clearSearchHistory = () => {
  if (confirm('确定要清空搜索历史吗？')) {
    searchHistory.value = []
    saveSearchHistory()
  }
}

const confirmSaveSearch = () => {
  console.log('保存搜索:', {
    name: savedSearchName.value,
    description: savedSearchDescription.value,
    query: searchQuery.value,
    filters: advancedFilters.value
  })

  closeSaveSearchModal()
}

// 分页方法
const prevPage = () => {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

const nextPage = () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

const goToPage = (page) => {
  currentPage.value = page
}

// 模态框控制方法
const closeSearchHistory = () => {
  showSearchHistory.value = false
}

const closeSaveSearchModal = () => {
  showSaveSearchModal.value = false
  savedSearchName.value = ''
  savedSearchDescription.value = ''
}

// 工具方法
const getTypeIcon = (type) => {
  const icons = {
    document: '📄',
    web: '🌐',
    image: '🖼️',
    video: '🎥',
    audio: '🎵',
    knowledge_base: '📚'
  }
  return icons[type] || '📄'
}

const getTypeText = (type) => {
  const typeMap = {
    document: '文档',
    web: '网页',
    image: '图片',
    video: '视频',
    audio: '音频',
    knowledge_base: '知识库'
  }
  return typeMap[type] || type
}

const formatDate = (date) => {
  return date.toLocaleDateString('zh-CN')
}

const formatTime = (date) => {
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  return date.toLocaleDateString('zh-CN')
}

// 监听点击外部关闭搜索建议
document.addEventListener('click', (e) => {
  if (!e.target.closest('.search-input-wrapper')) {
    showSuggestions.value = false
  }
})

// 生命周期
onMounted(() => {
  loadSearchHistory()
})
</script>

<style scoped>
.search-interface {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #f5f7fa;
}

.search-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 2rem;
  background: white;
  border-bottom: 1px solid #e1e8ed;
}

.search-header h2 {
  margin: 0;
  color: #2c3e50;
  font-size: 1.5rem;
}

.header-actions {
  display: flex;
  gap: 1rem;
}

.search-content {
  flex: 1;
  overflow-y: auto;
  padding: 2rem;
}

.search-input-section {
  background: white;
  border-radius: 12px;
  padding: 2rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
}

.search-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.search-input-wrapper {
  position: relative;
  flex: 1;
}

.search-input {
  width: 100%;
  padding: 1rem 1.5rem;
  font-size: 1.1rem;
  border: 2px solid #e1e8ed;
  border-radius: 8px;
  transition: border-color 0.3s ease;
}

.search-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.search-suggestions {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: white;
  border: 1px solid #e1e8ed;
  border-top: none;
  border-radius: 0 0 8px 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  max-height: 200px;
  overflow-y: auto;
}

.suggestion-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  cursor: pointer;
  transition: background-color 0.3s ease;
}

.suggestion-item:hover {
  background: #f8f9fa;
}

.suggestion-icon {
  color: #5a6c7d;
}

.suggestion-text {
  color: #2c3e50;
}

.search-controls {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.search-scope {
  padding: 0.75rem;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  font-size: 0.9rem;
}

.advanced-search {
  border-top: 1px solid #e1e8ed;
  padding-top: 1.5rem;
}

.advanced-filters {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.filter-group label {
  font-size: 0.9rem;
  font-weight: 600;
  color: #2c3e50;
}

.filter-select {
  padding: 0.5rem;
  border: 1px solid #e1e8ed;
  border-radius: 4px;
  font-size: 0.9rem;
}

.advanced-options {
  display: flex;
  justify-content: center;
}

.option-group {
  display: flex;
  gap: 2rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.9rem;
  color: #2c3e50;
}

.search-results-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.searching-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #5a6c7d;
}

.searching-spinner {
  font-size: 2rem;
  margin-bottom: 1rem;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.search-welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 400px;
  text-align: center;
  padding: 2rem;
}

.welcome-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.search-welcome h3 {
  margin: 0 0 0.5rem 0;
  color: #2c3e50;
}

.search-welcome p {
  margin: 0 0 2rem 0;
  color: #5a6c7d;
}

.quick-searches h4 {
  margin: 0 0 1rem 0;
  color: #2c3e50;
}

.quick-search-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  justify-content: center;
}

.quick-search-tag {
  padding: 0.5rem 1rem;
  background: #f8f9fa;
  border: 1px solid #e1e8ed;
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.3s ease;
  color: #2c3e50;
}

.quick-search-tag:hover {
  background: #667eea;
  color: white;
  border-color: #667eea;
}

.no-results {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 400px;
  text-align: center;
  padding: 2rem;
}

.no-results-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.no-results h3 {
  margin: 0 0 0.5rem 0;
  color: #2c3e50;
}

.no-results p {
  margin: 0 0 2rem 0;
  color: #5a6c7d;
}

.search-suggestions h4 {
  margin: 0 0 1rem 0;
  color: #2c3e50;
}

.search-suggestions ul {
  margin: 0;
  padding-left: 1.5rem;
  text-align: left;
}

.search-suggestions li {
  margin-bottom: 0.5rem;
  color: #5a6c7d;
}

.search-results {
  padding: 1.5rem;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e1e8ed;
}

.results-info {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.results-count {
  font-weight: 600;
  color: #2c3e50;
}

.search-time {
  font-size: 0.9rem;
  color: #5a6c7d;
}

.results-actions {
  display: flex;
  gap: 0.5rem;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.result-item {
  border: 1px solid #e1e8ed;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;
}

.result-item:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.result-item.result-expanded {
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
}

.result-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.5rem;
  background: #f8f9fa;
}

.result-rank {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: #667eea;
  color: white;
  border-radius: 50%;
  font-weight: 600;
  font-size: 0.9rem;
}

.result-info {
  flex: 1;
}

.result-title {
  margin: 0 0 0.75rem 0;
  color: #2c3e50;
  font-size: 1.2rem;
  cursor: pointer;
}

.result-title:hover {
  color: #667eea;
}

.result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 0.8rem;
  color: #5a6c7d;
}

.result-type {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: #e3f2fd;
  color: #1976d2;
  border-radius: 12px;
}

.result-actions {
  display: flex;
  align-items: flex-start;
}

.expand-btn {
  padding: 0.5rem 1rem;
  background: none;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.8rem;
  color: #5a6c7d;
  transition: all 0.3s ease;
}

.expand-btn:hover {
  background: #e9ecef;
  color: #2c3e50;
}

.result-content {
  padding: 1.5rem;
}

.result-description {
  margin: 0 0 1rem 0;
  color: #2c3e50;
  line-height: 1.6;
}

.result-snippets {
  margin-bottom: 1rem;
}

.snippet {
  padding: 0.75rem;
  background: #f8f9fa;
  border-left: 4px solid #667eea;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  line-height: 1.5;
  color: #5a6c7d;
}

.result-expanded-content {
  border-top: 1px solid #e1e8ed;
  padding-top: 1.5rem;
  margin-top: 1.5rem;
}

.result-expanded-content h4 {
  margin: 0 0 1rem 0;
  color: #2c3e50;
  font-size: 1rem;
}

.full-content {
  margin-bottom: 1.5rem;
  line-height: 1.6;
  color: #2c3e50;
}

.result-tags {
  margin-bottom: 1.5rem;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.tag {
  padding: 0.25rem 0.75rem;
  background: #e3f2fd;
  color: #1976d2;
  border-radius: 12px;
  font-size: 0.8rem;
}

.result-metadata {
  margin-bottom: 1.5rem;
}

.metadata-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 0.75rem;
}

.metadata-item {
  display: flex;
  gap: 0.5rem;
  font-size: 0.9rem;
}

.metadata-key {
  font-weight: 600;
  color: #5a6c7d;
}

.metadata-value {
  color: #2c3e50;
}

.result-footer {
  padding: 1rem 1.5rem;
  background: #f8f9fa;
  border-top: 1px solid #e1e8ed;
}

.result-links {
  display: flex;
  gap: 1rem;
}

.result-link {
  padding: 0.5rem 1rem;
  background: none;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.8rem;
  color: #5a6c7d;
  text-decoration: none;
  transition: all 0.3s ease;
}

.result-link:hover {
  background: #667eea;
  color: white;
  border-color: #667eea;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  background: #f8f9fa;
  border-top: 1px solid #e1e8ed;
}

.page-numbers {
  display: flex;
  gap: 0.25rem;
}

/* 高亮样式 */
:deep(mark) {
  background: #ffeb3b;
  color: #000;
  padding: 0.1rem 0.2rem;
  border-radius: 2px;
}

/* 按钮样式 */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  border: none;
  font-size: 0.9rem;
  transition: all 0.3s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 0.25rem 0.75rem;
  font-size: 0.8rem;
}

.btn-primary {
  background: #667eea;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #5a6fd8;
}

.btn-outline {
  background: transparent;
  color: #667eea;
  border: 1px solid #667eea;
}

.btn-outline:hover:not(:disabled) {
  background: #667eea;
  color: white;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #e1e8ed;
}

.modal-header h3 {
  margin: 0;
  color: #2c3e50;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #5a6c7d;
  padding: 0.25rem;
  border-radius: 4px;
}

.btn-close:hover {
  background: #f1f3f4;
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #e1e8ed;
}

/* 表单样式 */
.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: #2c3e50;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #e1e8ed;
  border-radius: 6px;
  font-size: 0.9rem;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.2);
}

.history-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 6px;
}

.history-query {
  flex: 1;
  font-weight: 600;
  color: #2c3e50;
  cursor: pointer;
}

.history-query:hover {
  color: #667eea;
}

.history-meta {
  display: flex;
  gap: 1rem;
  font-size: 0.8rem;
  color: #5a6c7d;
}

.remove-history {
  background: #dc3545;
  color: white;
  border: none;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  cursor: pointer;
  font-size: 0.8rem;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .search-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .header-actions {
    justify-content: center;
  }

  .search-content {
    padding: 1rem;
  }

  .search-input-section {
    padding: 1.5rem;
  }

  .search-controls {
    flex-direction: column;
  }

  .advanced-filters {
    grid-template-columns: 1fr;
  }

  .option-group {
    flex-direction: column;
    gap: 1rem;
  }

  .results-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .result-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .result-meta {
    flex-direction: column;
    gap: 0.5rem;
  }

  .result-links {
    flex-wrap: wrap;
    justify-content: center;
  }

  .pagination {
    flex-direction: column;
    gap: 1rem;
  }

  .page-numbers {
    flex-wrap: wrap;
    justify-content: center;
  }

  .modal-content {
    width: 95%;
    margin: 1rem;
  }
}

@media (max-width: 480px) {
  .search-input {
    font-size: 1rem;
    padding: 0.75rem 1rem;
  }

  .result-title {
    font-size: 1rem;
  }

  .result-content {
    padding: 1rem;
  }

  .result-header {
    padding: 1rem;
  }

  .result-footer {
    padding: 0.75rem 1rem;
  }

  .metadata-grid {
    grid-template-columns: 1fr;
  }
}
</style>