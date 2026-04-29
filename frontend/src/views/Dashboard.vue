<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMe, type User } from '../services/api'

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
</script>

<template>
  <div class="container dashboard">
    <div v-if="loading" class="loading">Loading...</div>

    <template v-else-if="user">
      <div class="card-lg welcome-card">
        <h1>Welcome, {{ user.name }}</h1>
        <p class="welcome-sub">这是你的健康仪表盘，开始记录和追踪你的健康数据吧</p>
      </div>

      <div class="card-lg quick-actions">
        <h2>快速操作</h2>
        <div class="actions-grid">
          <router-link to="/medicines">
            <button class="btn-outline">药品管理</button>
          </router-link>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.loading {
  text-align: center;
  padding: 4rem 0;
  color: #8a7b70;
}

.welcome-card h1 {
  font-size: 1.875rem;
  margin-bottom: 0.5rem;
}

.welcome-sub {
  font-size: 0.875rem;
  color: #8a7b70;
}


.quick-actions h2 {
  font-size: 1.25rem;
  margin-bottom: 1rem;
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

@media (max-width: 480px) {
  .actions-grid {
    grid-template-columns: 1fr;
  }
}

.actions-grid .btn-outline {
  padding: 0.625rem 1.5rem;
  font-size: 0.8125rem;
}
</style>
