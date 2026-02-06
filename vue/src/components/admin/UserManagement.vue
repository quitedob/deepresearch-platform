<template>
  <div class="user-management">
    <div class="section-header">
      <h2>用户管理</h2>
      <div class="header-actions">
        <div class="search-box">
          <input v-model="searchQuery" placeholder="搜索用户..." @input="handleSearch" />
        </div>
        <!-- 批量操作按钮 -->
        <div v-if="selectedUsers.length > 0" class="batch-actions">
          <span class="selected-count">已选择 {{ selectedUsers.length }} 个用户</span>
          <button class="btn-batch" @click="batchResetQuota">批量重置配额</button>
          <button class="btn-batch btn-warning" @click="batchBanUsers">批量禁用</button>
          <button class="btn-batch btn-success" @click="batchActivateUsers">批量启用</button>
        </div>
      </div>
    </div>

    <div class="users-table">
      <table>
        <thead>
          <tr>
            <th class="checkbox-col">
              <input type="checkbox" v-model="selectAll" @change="toggleSelectAll" />
            </th>
            <th>用户名</th>
            <th>邮箱</th>
            <th>会员类型</th>
            <th>状态</th>
            <th>聊天配额</th>
            <th>研究配额</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id" :class="{ selected: selectedUsers.includes(user.id) }">
            <td class="checkbox-col">
              <input type="checkbox" :value="user.id" v-model="selectedUsers" />
            </td>
            <td>{{ user.username }}</td>
            <td>{{ user.email }}</td>
            <td>
              <span :class="['badge', user.membership_type === 'premium' ? 'premium' : 'free']">
                {{ user.membership_type === 'premium' ? '高级会员' : '普通用户' }}
              </span>
            </td>
            <td>
              <span :class="['badge', user.status === 'active' ? 'active' : 'banned']">
                {{ user.status === 'active' ? '正常' : '已禁用' }}
              </span>
            </td>
            <td>{{ getMembershipChatUsage(user) }}</td>
            <td>{{ getMembershipResearchUsage(user) }}</td>
            <td>{{ formatDate(user.created_at) }}</td>
            <td class="actions">
              <button class="btn-icon" @click="viewChatHistory(user)" title="查看聊天记录">📋</button>
              <button class="btn-icon" @click="exportChatHistory(user)" title="导出聊天记录">📥</button>
              <button class="btn-icon" @click="editUser(user)" title="编辑用户">✏️</button>
              <button class="btn-icon btn-danger-icon" @click="confirmDeleteUser(user)" title="删除用户">🗑️</button>
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

    <!-- 编辑用户弹窗 -->
    <div v-if="showEditModal" class="modal-overlay" @click.self="showEditModal = false">
      <div class="modal">
        <h3>编辑用户: {{ editingUser?.username }}</h3>
        <div class="form-group">
          <label>用户状态</label>
          <select v-model="editForm.status">
            <option value="active">正常</option>
            <option value="banned">禁用</option>
          </select>
        </div>
        <div class="form-group">
          <label>会员类型</label>
          <select v-model="editForm.membershipType">
            <option value="free">普通用户</option>
            <option value="premium">高级会员</option>
          </select>
        </div>
        <div v-if="editForm.membershipType === 'premium'" class="form-group">
          <label>会员有效天数</label>
          <input type="number" v-model.number="editForm.validDays" min="1" />
        </div>
        <div class="form-group">
          <label>聊天配额</label>
          <input type="number" v-model.number="editForm.chatLimit" min="0" />
        </div>
        <div class="form-group">
          <label>研究配额</label>
          <input type="number" v-model.number="editForm.researchLimit" min="0" />
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showEditModal = false">取消</button>
          <button class="btn-danger" @click="resetQuota">重置配额</button>
          <button class="btn-primary" @click="saveUser">保存</button>
        </div>
      </div>
    </div>

    <!-- 聊天记录弹窗 -->
    <div v-if="showChatModal" class="modal-overlay" @click.self="showChatModal = false">
      <div class="modal modal-large">
        <h3>{{ viewingUser?.username }} 的聊天记录</h3>
        <div class="chat-sessions">
          <div v-for="session in chatSessions" :key="session.id" class="chat-session">
            <div class="session-header">
              <span class="session-title">{{ session.title || '未命名会话' }}</span>
              <span class="session-meta">{{ session.provider }} / {{ session.model }}</span>
              <span class="session-date">{{ formatDate(session.created_at) }}</span>
            </div>
            <div class="session-messages">
              <div v-for="msg in session.messages" :key="msg.id" :class="['message', msg.role]">
                <span class="role">{{ msg.role === 'user' ? '用户' : 'AI' }}:</span>
                <span class="content">{{ msg.content }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showChatModal = false">关闭</button>
        </div>
      </div>
    </div>

    <!-- 删除用户确认弹窗 -->
    <div v-if="showDeleteConfirmModal" class="modal-overlay" @click.self="cancelDelete">
      <div class="modal modal-danger">
        <h3>⚠️ 确认删除用户</h3>
        <div class="delete-warning">
          <p><strong>您即将删除用户：</strong>{{ userToDelete?.username }}</p>
          <p class="warning-text">此操作将会：</p>
          <ul class="warning-list">
            <li>删除该用户的所有聊天记录</li>
            <li>删除该用户的所有会话</li>
            <li>删除该用户的会员信息</li>
            <li>删除该用户的激活记录</li>
          </ul>
          <p class="warning-note">💡 系统将执行软删除，数据可在30天内恢复。</p>
        </div>
        <div class="form-group">
          <label>请输入用户名 <strong>{{ userToDelete?.username }}</strong> 以确认删除：</label>
          <input 
            type="text" 
            v-model="deleteConfirmText" 
            :placeholder="userToDelete?.username"
            autocomplete="off"
          />
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="cancelDelete" :disabled="isDeleting">取消</button>
          <button 
            class="btn-danger" 
            @click="executeDeleteUser" 
            :disabled="isDeleting || deleteConfirmText !== userToDelete?.username"
          >
            {{ isDeleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { apiClient } from '@/api/index'
import { 
  listUsers, 
  updateUserStatus, 
  updateUserMembership, 
  resetUserQuota, 
  setUserQuota,
  getUserChatHistory,
  exportUserChatHistory,
  batchUpdateUserStatus,
  batchResetUserQuotas
} from '@/api/admin'

const users = ref([])
const total = ref(0)
const limit = ref(20)
const offset = ref(0)
const searchQuery = ref('')
const showEditModal = ref(false)
const showChatModal = ref(false)
const editingUser = ref(null)
const viewingUser = ref(null)
const chatSessions = ref([])

// 批量选择相关
const selectedUsers = ref([])
const selectAll = ref(false)

const editForm = ref({
  status: 'active',
  membershipType: 'free',
  validDays: 30,
  chatLimit: 10,
  researchLimit: 1
})

const currentPage = computed(() => Math.floor(offset.value / limit.value) + 1)
const totalPages = computed(() => Math.ceil(total.value / limit.value))

const loadUsers = async () => {
  try {
    const response = await listUsers(limit.value, offset.value)
    const data = response.data || response
    users.value = data.users || []
    total.value = data.total || 0
  } catch (error) {
    console.error('加载用户列表失败:', error)
  }
}

const getMembershipChatUsage = (user) => {
  if (!user.membership) return '-'
  const m = user.membership
  if (m.membership_type === 'premium') {
    return `${m.premium_chat_used}/${m.premium_chat_limit}`
  }
  return `${m.normal_chat_used}/${m.normal_chat_limit}`
}

const getMembershipResearchUsage = (user) => {
  if (!user.membership) return '-'
  const m = user.membership
  if (m.membership_type === 'premium') {
    return `${m.premium_research_used}/${m.premium_research_limit}`
  }
  return `${m.research_used}/${m.research_limit}`
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

const handleSearch = () => {
  offset.value = 0
  loadUsers()
}

const prevPage = () => {
  if (offset.value > 0) {
    offset.value -= limit.value
    loadUsers()
  }
}

const nextPage = () => {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    loadUsers()
  }
}

const editUser = (user) => {
  editingUser.value = user
  editForm.value = {
    status: user.status,
    membershipType: user.membership_type || 'free',
    validDays: 30,
    chatLimit: user.membership?.normal_chat_limit || 10,
    researchLimit: user.membership?.research_limit || 1
  }
  showEditModal.value = true
}

const saveUser = async () => {
  try {
    if (editForm.value.status !== editingUser.value.status) {
      await updateUserStatus(editingUser.value.id, editForm.value.status)
    }
    if (editForm.value.membershipType !== editingUser.value.membership_type) {
      await updateUserMembership(editingUser.value.id, editForm.value.membershipType, editForm.value.validDays)
    }
    await setUserQuota(editingUser.value.id, editForm.value.chatLimit, editForm.value.researchLimit)
    showEditModal.value = false
    loadUsers()
  } catch (error) {
    console.error('保存用户失败:', error)
    alert('保存失败')
  }
}

const resetQuota = async () => {
  try {
    await resetUserQuota(editingUser.value.id)
    alert('配额已重置')
    loadUsers()
  } catch (error) {
    console.error('重置配额失败:', error)
  }
}

const viewChatHistory = async (user) => {
  viewingUser.value = user
  try {
    const response = await getUserChatHistory(user.id)
    chatSessions.value = (response.data || response).sessions || []
    showChatModal.value = true
  } catch (error) {
    console.error('获取聊天记录失败:', error)
  }
}

const exportChatHistory = async (user) => {
  try {
    const response = await exportUserChatHistory(user.id)
    const blob = new Blob([response], { type: 'application/json' })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `chat_history_${user.username}.json`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('导出聊天记录失败:', error)
  }
}

// 批量操作相关
const toggleSelectAll = () => {
  if (selectAll.value) {
    selectedUsers.value = users.value.map(u => u.id)
  } else {
    selectedUsers.value = []
  }
}

const batchResetQuota = async () => {
  if (!confirm(`确定要重置 ${selectedUsers.value.length} 个用户的配额吗？`)) return
  
  try {
    await batchResetUserQuotas(selectedUsers.value)
    alert('批量重置配额成功')
    selectedUsers.value = []
    selectAll.value = false
    loadUsers()
  } catch (error) {
    console.error('批量重置配额失败:', error)
    alert('批量重置配额失败')
  }
}

const batchBanUsers = async () => {
  if (!confirm(`确定要禁用 ${selectedUsers.value.length} 个用户吗？`)) return
  
  try {
    await batchUpdateUserStatus(selectedUsers.value, 'banned')
    alert('批量禁用成功')
    selectedUsers.value = []
    selectAll.value = false
    loadUsers()
  } catch (error) {
    console.error('批量禁用失败:', error)
    alert('批量禁用失败')
  }
}

const batchActivateUsers = async () => {
  if (!confirm(`确定要启用 ${selectedUsers.value.length} 个用户吗？`)) return
  
  try {
    await batchUpdateUserStatus(selectedUsers.value, 'active')
    alert('批量启用成功')
    selectedUsers.value = []
    selectAll.value = false
    loadUsers()
  } catch (error) {
    console.error('批量启用失败:', error)
    alert('批量启用失败')
  }
}

// ==================== 删除用户（带二次确认） ====================
const showDeleteConfirmModal = ref(false)
const userToDelete = ref(null)
const deleteConfirmText = ref('')
const isDeleting = ref(false)

/**
 * 显示删除确认对话框
 * 修复：添加二次确认，防止误删除
 */
const confirmDeleteUser = (user) => {
  userToDelete.value = user
  deleteConfirmText.value = ''
  showDeleteConfirmModal.value = true
}

/**
 * 执行删除用户
 * 修复：实现软删除，保留恢复机制
 */
const executeDeleteUser = async () => {
  if (!userToDelete.value) return
  
  // 验证确认文本
  if (deleteConfirmText.value !== userToDelete.value.username) {
    alert('请输入正确的用户名以确认删除')
    return
  }
  
  isDeleting.value = true
  
  try {
    // 使用集中化的apiClient进行软删除
    await apiClient.delete(`/admin/users/${userToDelete.value.id}`, {
      data: {
        soft_delete: true,
        confirm: true
      }
    })
    
    alert(`用户 "${userToDelete.value.username}" 已被删除（软删除，可恢复）`)
    showDeleteConfirmModal.value = false
    userToDelete.value = null
    deleteConfirmText.value = ''
    loadUsers()
  } catch (error) {
    console.error('删除用户失败:', error)
    alert(`删除失败: ${error.message}`)
  } finally {
    isDeleting.value = false
  }
}

/**
 * 取消删除
 */
const cancelDelete = () => {
  showDeleteConfirmModal.value = false
  userToDelete.value = null
  deleteConfirmText.value = ''
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.user-management { width: 100%; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 12px; }
.section-header h2 { font-size: 18px; font-weight: 600; }
.header-actions { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.search-box input { padding: 8px 16px; border: 1px solid var(--border-color); border-radius: 8px; width: 250px; background: var(--input-bg); color: var(--text-primary); }
.batch-actions { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--secondary-bg); border-radius: 8px; }
.selected-count { font-size: 13px; color: var(--text-secondary); margin-right: 8px; }
.btn-batch { padding: 6px 12px; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; background: var(--button-bg); color: var(--button-text); }
.btn-batch.btn-warning { background: var(--accent-orange); color: white; }
.btn-batch.btn-success { background: var(--accent-green); color: white; }
.btn-batch:hover { opacity: 0.9; }
.checkbox-col { width: 40px; text-align: center; }
tr.selected { background: var(--hover-bg); }
.users-table { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 12px; text-align: left; border-bottom: 1px solid var(--border-color); color: var(--text-primary); }
th { background: var(--secondary-bg); font-weight: 600; font-size: 13px; color: var(--text-secondary); }
td { font-size: 14px; }
.badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
.badge.premium { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
.badge.free { background: var(--tertiary-bg); color: var(--text-secondary); }
.badge.active { background: rgba(52, 199, 89, 0.1); color: var(--accent-green); }
.badge.banned { background: rgba(255, 59, 48, 0.1); color: var(--accent-red); }
.actions { display: flex; gap: 8px; }
.btn-icon { background: none; border: none; cursor: pointer; font-size: 16px; padding: 4px; }
.btn-icon:hover { background: var(--hover-bg); border-radius: 4px; }
.pagination { display: flex; justify-content: center; align-items: center; gap: 16px; margin-top: 20px; }
.pagination button { padding: 8px 16px; border: 1px solid var(--border-color); background: var(--card-bg); border-radius: 6px; cursor: pointer; color: var(--text-primary); }
.pagination button:disabled { opacity: 0.5; cursor: not-allowed; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--card-bg); border-radius: 12px; padding: 24px; min-width: 400px; max-width: 90vw; max-height: 90vh; overflow-y: auto; color: var(--text-primary); border: 1px solid var(--border-color); }
.modal-large { min-width: 800px; }
.modal h3 { margin-bottom: 20px; font-size: 18px; color: var(--text-primary); }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; color: var(--text-secondary); }
.form-group input, .form-group select { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--input-bg); color: var(--text-primary); }
.modal-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 20px; }
.btn-primary { padding: 8px 16px; background: var(--button-bg); color: var(--button-text); border: none; border-radius: 6px; cursor: pointer; }
.btn-secondary { padding: 8px 16px; background: var(--secondary-bg); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 6px; cursor: pointer; }
.btn-danger { padding: 8px 16px; background: var(--accent-red); color: white; border: none; border-radius: 6px; cursor: pointer; }
.chat-sessions { max-height: 500px; overflow-y: auto; }
.chat-session { border: 1px solid var(--border-color); border-radius: 8px; margin-bottom: 12px; padding: 12px; background: var(--primary-bg); }
.session-header { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 13px; }
.session-title { font-weight: 600; }
.session-meta { color: var(--text-secondary); }
.session-messages { font-size: 13px; }
.message { padding: 4px 0; }
.message.user .role { color: var(--accent-blue); }
.message.assistant .role { color: var(--accent-green); }
.message .content { margin-left: 8px; }
.btn-danger-icon:hover { background: rgba(255, 59, 48, 0.1) !important; color: var(--accent-red) !important; }
.modal-danger h3 { color: var(--accent-red); }
.delete-warning { background: rgba(255, 59, 48, 0.05); border: 1px solid rgba(255, 59, 48, 0.2); border-radius: 8px; padding: 16px; margin-bottom: 16px; }
.delete-warning p { margin: 8px 0; font-size: 14px; }
.warning-text { font-weight: 600; color: var(--accent-red); }
.warning-list { margin: 8px 0 8px 20px; font-size: 13px; color: var(--accent-red); opacity: 0.8; }
.warning-list li { margin: 4px 0; }
.warning-note { font-size: 13px; color: var(--accent-green); background: rgba(52, 199, 89, 0.1); padding: 8px; border-radius: 4px; margin-top: 12px; }
.btn-danger { padding: 8px 16px; background: var(--accent-red); color: white; border: none; border-radius: 6px; cursor: pointer; }
.btn-danger:hover { opacity: 0.9; }
.btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
