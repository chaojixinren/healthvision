<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../services/api'
import { setToken } from '../services/auth'

const router = useRouter()
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res = await login({ email: email.value, password: password.value })
    setToken(res.access_token)
    router.push('/dashboard')
  } catch (e: any) {
    error.value = e.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="card auth-card">
      <h2>Welcome Back</h2>
      <p class="auth-sub">登录你的 HealthVision 账户</p>

      <form @submit.prevent="submit" class="auth-form">
        <div class="field">
          <label for="email">Email</label>
          <input id="email" v-model="email" type="email" placeholder="you@example.com" required />
        </div>
        <div class="field">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" placeholder="Enter your password" required />
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary btn-full" :disabled="loading">
          {{ loading ? 'Logging in...' : 'Login' }}
        </button>
      </form>

      <p class="auth-footer">
        还没有账户？<router-link to="/register">注册</router-link>
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
