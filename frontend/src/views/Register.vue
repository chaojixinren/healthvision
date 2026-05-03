<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../services/api'
import { setToken, setUser } from '../services/auth'

const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const isOld = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res = await register({ name: username.value, email: email.value, password: password.value, is_old: isOld.value })
    setToken(res.access_token)
    setUser(res.user)
    router.push(res.user.is_old ? '/reminders' : '/medicines')
  } catch (e: any) {
    error.value = e.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="card auth-card">
      <h2>创建账户</h2>
      <p class="auth-sub">注册一个新的 HealthVision 账户</p>

      <form @submit.prevent="submit" class="auth-form">
        <div class="field">
          <label for="username">用户名</label>
          <input id="username" v-model="username" type="text" placeholder="请设置用户名" required />
        </div>
        <div class="field">
          <label for="email">邮箱</label>
          <input id="email" v-model="email" type="email" placeholder="请输入邮箱" required />
        </div>
        <div class="field">
          <label for="password">密码</label>
          <input id="password" v-model="password" type="password" placeholder="请设置密码（至少 6 位）" required minlength="6" />
        </div>

        <div class="field">
          <label>账户类型</label>
          <div class="identity-selector">
            <div
              class="identity-card"
              :class="{ active: isOld }"
              @click="isOld = true"
            >
              <span class="identity-icon">👴</span>
              <span class="identity-label">老人端</span>
              <span class="identity-desc">用药提醒管理</span>
            </div>
            <div
              class="identity-card"
              :class="{ active: !isOld }"
              @click="isOld = false"
            >
              <span class="identity-icon">🧑</span>
              <span class="identity-label">子女端</span>
              <span class="identity-desc">管理家人健康</span>
            </div>
          </div>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary btn-full" :disabled="loading">
          {{ loading ? '注册中...' : '注册' }}
        </button>
      </form>

      <p class="auth-footer">
        已有账户？<router-link to="/login">登录</router-link>
      </p>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: calc(100vh - 3.5rem);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem 1.5rem;
}

.auth-card {
  width: 100%;
  max-width: 26rem;
  padding: 2rem;
}

.auth-card h2 {
  font-size: 1.5rem;
  text-align: center;
  margin-bottom: 0.25rem;
}

.auth-sub {
  text-align: center;
  font-size: 0.875rem;
  color: #8a7b70;
  margin-bottom: 1.5rem;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
}

.identity-selector {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.identity-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.875rem 0.5rem;
  border: 2px solid var(--border);
  border-radius: 0.75rem;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.identity-card:hover {
  border-color: var(--primary);
}

.identity-card.active {
  border-color: var(--primary);
  background: #f0faf4;
}

.identity-icon {
  font-size: 1.5rem;
}

.identity-label {
  font-size: 0.875rem;
  font-weight: 600;
}

.identity-desc {
  font-size: 0.75rem;
  color: #8a7b70;
}

.btn-full {
  width: 100%;
  margin-top: 0.5rem;
}

.auth-footer {
  text-align: center;
  font-size: 0.875rem;
  margin-top: 1.5rem;
  color: #8a7b70;
}
</style>
