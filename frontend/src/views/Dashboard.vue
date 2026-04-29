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

      <div class="stats-grid">
        <div class="card stat-card">
          <div class="stat-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
          </div>
          <div class="stat-value">--</div>
          <div class="stat-label">健康评分</div>
        </div>
        <div class="card stat-card">
          <div class="stat-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
          </div>
          <div class="stat-value">0</div>
          <div class="stat-label">记录天数</div>
        </div>
        <div class="card stat-card">
          <div class="stat-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
          </div>
          <div class="stat-value">0</div>
          <div class="stat-label">健康指标</div>
        </div>
      </div>

      <div class="card-lg quick-actions">
        <h2>快速操作</h2>
        <div class="actions-grid">
          <button class="btn-outline">记录体重</button>
          <button class="btn-outline">记录血压</button>
          <button class="btn-outline">记录血糖</button>
          <button class="btn-outline">查看报告</button>
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

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

@media (max-width: 640px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1.5rem 1rem;
  text-align: center;
}

.stat-icon {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 50%;
  background: var(--muted);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
}

.stat-value {
  font-family: var(--font-serif);
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--foreground);
}

.stat-label {
  font-size: 0.75rem;
  color: #8a7b70;
  text-transform: uppercase;
  letter-spacing: 0.05em;
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
