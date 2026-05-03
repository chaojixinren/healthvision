<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../services/api'
import { setToken } from '../services/auth'

const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res = await register({ name: username.value, email: email.value, password: password.value })
    setToken(res.access_token)
    router.push('/medicines')
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
  max-width: 24rem;
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
