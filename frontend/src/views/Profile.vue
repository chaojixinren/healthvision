<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getMe, type User } from '../services/api'
import { removeToken } from '../services/auth'

const router = useRouter()
const user = ref<User | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    user.value = await getMe()
  } catch {
    // auth guard handles redirect
  } finally {
    loading.value = false
  }
})

function logout() {
  removeToken()
  router.push('/')
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
</script>

<template>
  <div class="container profile">
    <div v-if="loading" class="loading">加载中...</div>

    <template v-else-if="user">
      <h1>个人中心</h1>

      <div class="card-lg profile-card">
        <div class="avatar">
          {{ user.name.charAt(0).toUpperCase() }}
        </div>
        <div class="profile-info">
          <h2>{{ user.name }}</h2>
          <p class="profile-email">{{ user.email }}</p>
        </div>
      </div>

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
            <span class="detail-label">用户 ID</span>
            <span class="detail-value mono">{{ user.id }}</span>
          </li>
          <li>
            <span class="detail-label">注册时间</span>
            <span class="detail-value">{{ formatDate(user.created_at) }}</span>
          </li>
        </ul>
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
  gap: 1.5rem;
}

.profile h1 {
  font-size: 1.875rem;
}

.loading {
  text-align: center;
  padding: 4rem 0;
  color: #8a7b70;
}

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
  color: #ffffff;
  font-family: var(--font-serif);
  font-size: 1.5rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.profile-info h2 {
  font-size: 1.25rem;
  margin-bottom: 0.125rem;
}

.profile-email {
  font-size: 0.875rem;
  color: #8a7b70;
}

.details-card h3 {
  font-size: 1.125rem;
  margin-bottom: 1rem;
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
  padding: 0.75rem 0;
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

.mono {
  font-family: var(--font-mono);
  font-size: 0.8125rem;
}

.logout-btn {
  width: 100%;
  padding: 0.75rem;
}
</style>
