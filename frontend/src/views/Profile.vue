<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  getMe,
  searchUsers,
  changeIdentity,
  logoutSession,
  type User,
  type Binding,
} from '../services/api'
import { useAuthStore } from '../stores/auth'
import { useCareStore } from '../stores/care'

const router = useRouter()
const auth = useAuthStore()
const care = useCareStore()
const user = computed(() => auth.user)
const bindings = computed(() => care.bindings)
const loading = ref(true)

const searchQuery = ref('')
const searchResult = ref<User | null>(null)
const searchError = ref('')
const searching = ref(false)
const bindingLoading = ref(false)
const identityLoading = ref(false)

onMounted(async () => {
  try {
    const [u] = await Promise.all([getMe(), care.loadBindings()])
    auth.setCurrentUser(u)
  } catch {
    // auth guard handles redirect
  } finally {
    loading.value = false
  }
})

async function handleSearch() {
  const q = searchQuery.value.trim()
  if (!q) return
  searchError.value = ''
  searchResult.value = null
  searching.value = true
  try {
    const res = await searchUsers(q)
    searchResult.value = res.users[0] || null
    if (!searchResult.value) searchError.value = '未找到该用户'
  } catch (e: any) {
    searchError.value = e.message || '搜索失败'
  } finally {
    searching.value = false
  }
}

async function handleCreateBinding() {
  if (!searchResult.value) return
  bindingLoading.value = true
  try {
    await care.createBinding(searchResult.value.email)
    searchResult.value = null
    searchQuery.value = ''
  } catch (e: any) {
    searchError.value = e.message || '发起绑定失败'
  } finally {
    bindingLoading.value = false
  }
}

async function handleRespond(id: number, accept: boolean) {
  try {
    await care.respondBinding(id, accept)
  } catch (e: any) {
    alert(e.message || '操作失败')
  }
}

async function handleDeleteBinding(id: number) {
  try {
    await care.deleteBinding(id)
  } catch (e: any) {
    alert(e.message || '删除失败')
  }
}

async function handleChangeIdentity() {
  if (!user.value) return
  const newIdentity = user.value.is_old ? '子女端' : '老人端'
  if (!confirm(`确认切换到「${newIdentity}」？\n\n如有绑定关系需先解除。`)) return
  identityLoading.value = true
  try {
    const res = await changeIdentity()
    await care.clearOfflineData(user.value.id)
    auth.setCurrentUser(res.user)
    care.reset()
    router.push('/reminders')
  } catch (e: any) {
    alert(e.message || '切换失败')
  } finally {
    identityLoading.value = false
  }
}

async function logout() {
  const userId = auth.user?.id ?? null
  await logoutSession().catch(() => {})
  await care.clearOfflineData(userId)
  auth.clearSession()
  care.reset()
  router.push('/')
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

function bindingLabel(b: Binding): string {
  if (user.value?.is_old) {
    return b.child?.name || '子女用户'
  } else {
    return b.elder?.name || '老人用户'
  }
}

function bindingEmail(b: Binding): string {
  if (user.value?.is_old) {
    return b.child?.email || ''
  } else {
    return b.elder?.email || ''
  }
}

const pendingBindings = computed(() => bindings.value.filter(b => b.status === 'pending'))
const activeBindings = computed(() => bindings.value.filter(b => b.status === 'accepted'))
</script>

<template>
  <div class="container profile">
    <div v-if="loading" class="loading">加载中...</div>

    <template v-else-if="user">
      <h1>个人中心</h1>

      <!-- 用户信息卡片 -->
      <div class="card-lg profile-card">
        <div class="avatar">
          {{ user.name.charAt(0).toUpperCase() }}
        </div>
        <div class="profile-info">
          <div class="profile-header">
            <h2>{{ user.name }}</h2>
            <span class="identity-badge" :class="user.is_old ? 'elder' : 'child'">
              {{ user.is_old ? '老人端' : '子女端' }}
            </span>
          </div>
          <p class="profile-email">{{ user.email }}</p>
        </div>
      </div>

      <!-- 账户信息 -->
      <div class="card-lg details-card">
        <h3>账户信息</h3>
        <ul class="detail-list">
          <li>
            <span class="detail-label">用户名</span>
            <span class="detail-value">{{ user.name }}</span>
          </li>
          <li>
            <span class="detail-label">邮箱</span>
            <span class="detail-value">{{ user.email }}</span>
          </li>
          <li>
            <span class="detail-label">账户类型</span>
            <span class="detail-value">{{ user.is_old ? '老人端' : '子女端' }}</span>
          </li>
          <li>
            <span class="detail-label">注册时间</span>
            <span class="detail-value">{{ formatDate(user.created_at) }}</span>
          </li>
        </ul>
      </div>

      <!-- 身份切换 -->
      <div class="card-lg identity-card">
        <div class="identity-row">
          <div>
            <h3>身份切换</h3>
            <p class="identity-hint">当前为「{{ user.is_old ? '老人端' : '子女端' }}」，切换前需解除所有绑定关系</p>
          </div>
          <button
            class="btn-outline"
            :disabled="identityLoading"
            @click="handleChangeIdentity"
          >
            {{ identityLoading ? '切换中...' : '切换身份' }}
          </button>
        </div>
      </div>

      <!-- 发起绑定 -->
      <div class="card-lg">
        <h3>添加绑定</h3>
        <p class="section-hint">
          {{ user.is_old ? '搜索子女账号并发送绑定请求' : '搜索老人账号并发送绑定请求' }}
        </p>
        <div class="search-row">
          <input
            v-model="searchQuery"
            type="email"
            class="search-input"
            placeholder="输入对方邮箱搜索..."
            maxlength="254"
            @keyup.enter="handleSearch"
          />
          <button class="btn-primary" :disabled="searching" @click="handleSearch">
            {{ searching ? '搜索中...' : '搜索' }}
          </button>
        </div>
        <p v-if="searchError" class="error-msg">{{ searchError }}</p>
        <div v-if="searchResult" class="search-result">
          <div class="result-info">
            <span class="result-name">{{ searchResult.name }}</span>
            <span class="result-email">{{ searchResult.email }}</span>
            <span class="identity-badge small" :class="searchResult.is_old ? 'elder' : 'child'">
              {{ searchResult.is_old ? '老人端' : '子女端' }}
            </span>
          </div>
          <button class="btn-primary btn-sm" :disabled="bindingLoading" @click="handleCreateBinding">
            {{ bindingLoading ? '发送中...' : '发送绑定请求' }}
          </button>
        </div>
      </div>

      <!-- 绑定列表 -->
      <div class="card-lg">
        <h3>我的绑定</h3>

        <!-- 待处理请求 -->
        <div v-if="pendingBindings.length > 0" class="binding-section">
          <h4 class="binding-section-title pending-title">待处理请求</h4>
          <div v-for="b in pendingBindings" :key="b.id" class="binding-item">
            <div class="binding-info">
              <span class="binding-name">{{ bindingLabel(b) }}</span>
              <span class="binding-email">{{ bindingEmail(b) }}</span>
            </div>
            <div v-if="user.is_old" class="binding-actions">
              <button class="btn-primary btn-sm" @click="handleRespond(b.id, true)">接受</button>
              <button class="btn-outline btn-sm" @click="handleRespond(b.id, false)">拒绝</button>
            </div>
            <span v-else class="binding-status pending">等待对方确认</span>
          </div>
        </div>

        <!-- 已绑定 -->
        <div v-if="activeBindings.length > 0" class="binding-section">
          <h4 class="binding-section-title active-title">已绑定</h4>
          <div v-for="b in activeBindings" :key="b.id" class="binding-item">
            <div class="binding-info">
              <span class="binding-name">{{ bindingLabel(b) }}</span>
              <span class="binding-email">{{ bindingEmail(b) }}</span>
            </div>
            <button class="btn-outline btn-sm danger" @click="handleDeleteBinding(b.id)">解除绑定</button>
          </div>
        </div>

        <p v-if="bindings.length === 0" class="empty-hint">暂无绑定关系</p>
      </div>

      <button class="btn-outline logout-btn" @click="logout">退出登录</button>
    </template>
  </div>
</template>

<style scoped>
.profile {
  padding: 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 32rem;
}

.profile h1 {
  font-size: 1.875rem;
  margin-bottom: 0.5rem;
}

.loading {
  text-align: center;
  padding: 4rem 0;
  color: #8a7b70;
}

/* 用户信息卡片 */
.profile-card {
  display: flex;
  align-items: center;
  gap: 1.25rem;
}

.avatar {
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  font-family: var(--font-serif);
  font-size: 1.5rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.profile-info {
  flex: 1;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.125rem;
}

.profile-header h2 {
  font-size: 1.25rem;
}

.identity-badge {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  white-space: nowrap;
}

.identity-badge.elder {
  background: #fff3e0;
  color: #e65100;
}

.identity-badge.child {
  background: #e3f2fd;
  color: #1565c0;
}

.identity-badge.small {
  font-size: 0.625rem;
}

.profile-email {
  font-size: 0.875rem;
  color: #8a7b70;
}

/* 账户信息 */
.details-card h3 {
  font-size: 1.125rem;
  margin-bottom: 0.75rem;
}

.detail-list {
  list-style: none;
  display: flex;
  flex-direction: column;
}

.detail-list li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.625rem 0;
  border-bottom: 1px solid var(--border);
}

.detail-list li:last-child {
  border-bottom: none;
}

.detail-label {
  font-size: 0.875rem;
  color: #8a7b70;
}

.detail-value {
  font-size: 0.875rem;
  font-weight: 500;
}

/* 身份切换 */
.identity-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.identity-row h3 {
  font-size: 1.125rem;
  margin-bottom: 0.25rem;
}

.identity-hint {
  font-size: 0.8125rem;
  color: #8a7b70;
}

/* 搜索绑定 */
h3 {
  font-size: 1.125rem;
  margin-bottom: 0.25rem;
}

.section-hint {
  font-size: 0.8125rem;
  color: #8a7b70;
  margin-bottom: 0.75rem;
}

.search-row {
  display: flex;
  gap: 0.5rem;
}

.search-input {
  flex: 1;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-size: 0.875rem;
  outline: none;
  transition: border-color 0.2s;
}

.search-input:focus {
  border-color: var(--primary);
}

.search-result {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: var(--bg);
  border-radius: var(--radius);
}

.result-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.result-name {
  font-size: 0.875rem;
  font-weight: 600;
}

.result-email {
  font-size: 0.8125rem;
  color: #8a7b70;
}

.error-msg {
  color: var(--danger, #d32f2f);
  font-size: 0.8125rem;
  margin-top: 0.5rem;
}

/* 绑定列表 */
.binding-section {
  margin-top: 0.75rem;
}

.binding-section-title {
  font-size: 0.8125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.pending-title {
  color: #e65100;
}

.active-title {
  color: var(--primary);
}

.binding-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border);
}

.binding-item:last-child {
  border-bottom: none;
}

.binding-info {
  display: flex;
  flex-direction: column;
}

.binding-name {
  font-size: 0.875rem;
  font-weight: 500;
}

.binding-email {
  font-size: 0.75rem;
  color: #8a7b70;
}

.binding-actions {
  display: flex;
  gap: 0.375rem;
}

.binding-status {
  font-size: 0.75rem;
  color: #8a7b70;
}

.binding-status.pending {
  color: #e65100;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.danger {
  border-color: var(--danger, #d32f2f);
  color: var(--danger, #d32f2f);
}

.danger:hover {
  background: #fff5f5;
}

.empty-hint {
  text-align: center;
  color: #8a7b70;
  font-size: 0.875rem;
  padding: 1rem 0;
}

.logout-btn {
  width: 100%;
  padding: 0.75rem;
  margin-top: 0.5rem;
}
</style>
