<template>
  <div class="activation-code-management">
    <div class="section-header">
      <h2>激活码管理</h2>
      <div class="header-actions">
        <button class="btn-outline" @click="exportCodes" title="导出激活码">📥 导出</button>
        <button class="btn-outline" @click="showBatchCreateModal = true" title="批量创建">📦 批量创建</button>
        <button class="btn-primary" @click="showCreateModal = true">+ 创建激活码</button>
      </div>
    </div>

    <div class="codes-table">
      <table>
        <thead>
          <tr>
            <th>激活码</th>
            <th>最大激活数</th>
            <th>已激活数</th>
            <th>会员天数</th>
            <th>状态</th>
            <th>过期时间</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="code in codes" :key="code.id">
            <td class="code-cell">
              <span class="code-text">{{ code.code }}</span>
              <button class="btn-copy" @click="copyCode(code.code)" title="复制">📋</button>
            </td>
            <td>{{ code.max_activations }}</td>
            <td>{{ code.used_activations }}</td>
            <td>{{ code.valid_days }}天</td>
            <td>
              <span :class="['badge', code.is_active ? 'active' : 'inactive']">
                {{ code.is_active ? '有效' : '已禁用' }}
              </span>
            </td>
            <td>{{ code.expires_at ? formatDate(code.expires_at) : '永不过期' }}</td>
            <td>{{ formatDate(code.created_at) }}</td>
            <td class="actions">
              <button class="btn-icon" @click="viewDetails(code)" title="查看详情">👁️</button>
              <button class="btn-icon" @click="toggleCodeStatus(code)" :title="code.is_active ? '禁用' : '启用'">
                {{ code.is_active ? '🚫' : '✅' }}
              </button>
              <button class="btn-icon" @click="deleteCode(code)" title="删除">🗑️</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination">
      <button :disabled="offset === 0" @click="prevPage">上一页</button>
      <span>第 {{ currentPage }} 页 / 共 {{ totalPages }} 页</span>
      <button :disabled="offset + limit >= total" @click="nextPage">下一页</button>
    </div>

    <!-- 创建激活码弹窗 -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal">
        <h3>创建激活码</h3>
        <div class="form-group">
          <label>最大激活人数 *</label>
          <input type="number" v-model.number="createForm.maxActivations" min="1" placeholder="可激活的用户数量" />
        </div>
        <div class="form-group">
          <label>会员有效天数</label>
          <input type="number" v-model.number="createForm.validDays" min="1" placeholder="默认30天" />
        </div>
        <div class="form-group">
          <label>激活码过期天数</label>
          <input type="number" v-model.number="createForm.expiresInDays" min="0" placeholder="0表示永不过期" />
        </div>
        <div class="form-group">
          <label>自定义激活码（可选）</label>
          <input type="text" v-model="createForm.code" placeholder="留空自动生成" />
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showCreateModal = false">取消</button>
          <button class="btn-primary" @click="createCode">创建</button>
        </div>
      </div>
    </div>

    <!-- 批量创建激活码弹窗 -->
    <div v-if="showBatchCreateModal" class="modal-overlay" @click.self="showBatchCreateModal = false">
      <div class="modal">
        <h3>批量创建激活码</h3>
        <div class="form-group">
          <label>创建数量 *</label>
          <input type="number" v-model.number="batchCreateForm.count" min="1" max="100" placeholder="1-100" />
        </div>
        <div class="form-group">
          <label>每个激活码最大激活人数</label>
          <input type="number" v-model.number="batchCreateForm.maxActivations" min="1" placeholder="默认1" />
        </div>
        <div class="form-group">
          <label>会员有效天数</label>
          <input type="number" v-model.number="batchCreateForm.validDays" min="1" placeholder="默认30天" />
        </div>
        <div class="form-group">
          <label>激活码过期天数</label>
          <input type="number" v-model.number="batchCreateForm.expiresInDays" min="0" placeholder="0表示永不过期" />
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showBatchCreateModal = false">取消</button>
          <button class="btn-primary" @click="batchCreateCodes">批量创建</button>
        </div>
      </div>
    </div>

    <!-- 激活详情弹窗 -->
    <div v-if="showDetailsModal" class="modal-overlay" @click.self="showDetailsModal = false">
      <div class="modal">
        <h3>激活码详情: {{ viewingCode?.code }}</h3>
        <div class="details-info">
          <p><strong>已激活用户:</strong> {{ viewingCode?.used_activations }} / {{ viewingCode?.max_activations }}</p>
          <p v-if="consistencyWarning" class="consistency-warning">{{ consistencyWarning }}</p>
        </div>
        <div class="activation-records">
          <h4>激活记录</h4>
          <div v-if="activationRecords.length === 0" class="empty-state">暂无激活记录</div>
          <div v-else class="records-list">
            <div v-for="record in activationRecords" :key="record.id" class="record-item">
              <span class="record-user">{{ record.username }} ({{ record.email }})</span>
              <span class="record-time">{{ formatDate(record.activated_at) }}</span>
            </div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showDetailsModal = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { 
  listActivationCodes, 
  createActivationCode, 
  getActivationCodeDetails, 
  updateActivationCode, 
  deleteActivationCode 
} from '@/api/admin'

const codes = ref([])
const total = ref(0)
const limit = ref(20)
const offset = ref(0)
const showCreateModal = ref(false)
const showDetailsModal = ref(false)
const viewingCode = ref(null)
const activationRecords = ref([])

const createForm = ref({
  maxActivations: 1,
  validDays: 30,
  expiresInDays: 0,
  code: ''
})

const showBatchCreateModal = ref(false)
const batchCreateForm = ref({
  count: 10,
  maxActivations: 1,
  validDays: 30,
  expiresInDays: 0
})

const currentPage = computed(() => Math.floor(offset.value / limit.value) + 1)
const totalPages = computed(() => Math.ceil(total.value / limit.value))

const loadCodes = async () => {
  try {
    const response = await listActivationCodes(limit.value, offset.value)
    const data = response.data || response
    codes.value = data.codes || []
    total.value = data.total || 0
  } catch (error) {
    console.error('加载激活码列表失败:', error)
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

const copyCode = (code) => {
  navigator.clipboard.writeText(code)
  alert('已复制到剪贴板')
}

const prevPage = () => {
  if (offset.value > 0) {
    offset.value -= limit.value
    loadCodes()
  }
}

const nextPage = () => {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    loadCodes()
  }
}

const createCode = async () => {
  try {
    await createActivationCode({
      max_activations: createForm.value.maxActivations,
      valid_days: createForm.value.validDays || 30,
      expires_in_days: createForm.value.expiresInDays || 0,
      code: createForm.value.code || ''
    })
    showCreateModal.value = false
    createForm.value = { maxActivations: 1, validDays: 30, expiresInDays: 0, code: '' }
    loadCodes()
  } catch (error) {
    console.error('创建激活码失败:', error)
    alert('创建失败')
  }
}

// 一致性警告
const consistencyWarning = ref('')

const viewDetails = async (code) => {
  viewingCode.value = code
  consistencyWarning.value = ''
  
  try {
    const response = await getActivationCodeDetails(code.id)
    const data = response.data || response
    activationRecords.value = data.records || []
    
    // 一致性检查：验证used_activations与实际记录数是否一致
    const actualUsedCount = activationRecords.value.length
    if (code.used_activations !== actualUsedCount) {
      consistencyWarning.value = `⚠️ 数据不一致：显示已激活 ${code.used_activations} 次，但实际记录 ${actualUsedCount} 条。建议刷新数据。`
      console.warn('[ActivationCode] 一致性检查失败:', {
        codeId: code.id,
        displayedCount: code.used_activations,
        actualCount: actualUsedCount
      })
      
      // 自动修正本地显示（实时计算）
      code.used_activations = actualUsedCount
    }
    
    showDetailsModal.value = true
  } catch (error) {
    console.error('获取激活码详情失败:', error)
  }
}

const toggleCodeStatus = async (code) => {
  try {
    await updateActivationCode(code.id, { is_active: !code.is_active })
    code.is_active = !code.is_active
  } catch (error) {
    console.error('更新激活码状态失败:', error)
  }
}

const deleteCode = async (code) => {
  if (!confirm('确定要删除这个激活码吗？')) return
  try {
    await deleteActivationCode(code.id)
    loadCodes()
  } catch (error) {
    console.error('删除激活码失败:', error)
  }
}

// 批量创建激活码
const batchCreateCodes = async () => {
  try {
    const createdCodes = []
    for (let i = 0; i < batchCreateForm.value.count; i++) {
      const response = await createActivationCode({
        max_activations: batchCreateForm.value.maxActivations,
        valid_days: batchCreateForm.value.validDays || 30,
        expires_in_days: batchCreateForm.value.expiresInDays || 0,
        code: '' // 自动生成
      })
      if (response.code) {
        createdCodes.push(response.code)
      }
    }
    
    showBatchCreateModal.value = false
    batchCreateForm.value = { count: 10, maxActivations: 1, validDays: 30, expiresInDays: 0 }
    loadCodes()
    
    // 显示创建结果
    alert(`成功创建 ${createdCodes.length} 个激活码`)
  } catch (error) {
    console.error('批量创建激活码失败:', error)
    alert('批量创建失败')
  }
}

// 导出激活码
const exportCodes = () => {
  if (codes.value.length === 0) {
    alert('没有可导出的激活码')
    return
  }
  
  const exportData = codes.value.map(code => ({
    code: code.code,
    max_activations: code.max_activations,
    used_activations: code.used_activations,
    valid_days: code.valid_days,
    is_active: code.is_active,
    expires_at: code.expires_at,
    created_at: code.created_at
  }))
  
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `activation_codes_${new Date().toISOString().split('T')[0]}.json`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  loadCodes()
})
</script>

<style scoped>
.activation-code-management { width: 100%; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 12px; }
.section-header h2 { font-size: 18px; font-weight: 600; }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.btn-outline { padding: 8px 16px; background: var(--card-bg); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-outline:hover { background: var(--hover-bg); }
.codes-table { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 12px; text-align: left; border-bottom: 1px solid var(--border-color); }
th { background: var(--secondary-bg); font-weight: 600; font-size: 13px; color: var(--text-secondary); }
td { font-size: 14px; }
.code-cell { display: flex; align-items: center; gap: 8px; }
.code-text { font-family: monospace; background: var(--secondary-bg); padding: 4px 8px; border-radius: 4px; }
.btn-copy { background: none; border: none; cursor: pointer; font-size: 14px; }
.badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
.badge.active { background: #d1fae5; color: #065f46; }
.badge.inactive { background: #fee2e2; color: #991b1b; }
.actions { display: flex; gap: 8px; }
.btn-icon { background: none; border: none; cursor: pointer; font-size: 16px; padding: 4px; }
.btn-icon:hover { background: var(--hover-bg); border-radius: 4px; }
.pagination { display: flex; justify-content: center; align-items: center; gap: 16px; margin-top: 20px; }
.pagination button { padding: 8px 16px; border: 1px solid var(--border-color); background: var(--card-bg); border-radius: 6px; cursor: pointer; color: var(--text-primary); }
.pagination button:disabled { opacity: 0.5; cursor: not-allowed; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--card-bg); border-radius: 12px; padding: 24px; min-width: 400px; max-width: 90vw; color: var(--text-primary); }
.modal h3 { margin-bottom: 20px; font-size: 18px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; color: var(--text-secondary); }
.form-group input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--input-bg); color: var(--text-primary); }
.modal-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 20px; }
.btn-primary { padding: 8px 16px; background: var(--button-bg); color: var(--button-text); border: none; border-radius: 6px; cursor: pointer; }
.btn-secondary { padding: 8px 16px; background: var(--secondary-bg); color: var(--text-primary); border: none; border-radius: 6px; cursor: pointer; }
.details-info { margin-bottom: 16px; }
.activation-records h4 { font-size: 14px; margin-bottom: 12px; }
.empty-state { color: var(--text-tertiary); font-size: 14px; text-align: center; padding: 20px; }
.records-list { max-height: 300px; overflow-y: auto; }
.record-item { display: flex; justify-content: space-between; padding: 8px 12px; background: var(--secondary-bg); border-radius: 6px; margin-bottom: 8px; font-size: 13px; }
.record-time { color: var(--text-secondary); }
.consistency-warning { color: #f59e0b; background: #fef3c7; padding: 8px 12px; border-radius: 6px; font-size: 13px; margin-top: 8px; }
</style>
