<script setup lang="ts">
import { isAuthenticated } from '../services/auth'
import { useRouter } from 'vue-router'

const router = useRouter()

function cta() {
  router.push(isAuthenticated() ? '/dashboard' : '/register')
}
</script>

<template>
  <section class="hero">
    <div class="hero-inner">
      <h1>HealthVision</h1>
      <p class="hero-sub">智能健康管理平台，用数据守护你的每一天</p>
      <div class="hero-actions">
        <button class="btn-primary btn-lg" @click="cta">开始使用</button>
        <router-link to="/login" class="btn-outline btn-lg">登录</router-link>
      </div>
    </div>
  </section>

  <section class="features container">
    <div class="features-grid">
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
        </div>
        <h3>健康监测</h3>
        <p>实时追踪关键健康指标，全面掌握身体状态</p>
      </div>
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        </div>
        <h3>智能分析</h3>
        <p>AI 驱动的数据分析，提供个性化健康建议</p>
      </div>
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <h3>隐私安全</h3>
        <p>端到端加密存储，你的健康数据只属于你</p>
      </div>
    </div>
  </section>

  <section class="details container">
    <h2>为什么选择 HealthVision</h2>
    <ul class="detail-list">
      <li><span class="bullet"></span><span>一站式整合多维度健康数据，告别信息碎片化</span></li>
      <li><span class="bullet"></span><span>可视化趋势图表，让健康变化一目了然</span></li>
      <li><span class="bullet"></span><span>智能预警机制，提前发现潜在健康风险</span></li>
      <li><span class="bullet"></span><span>开放 API 接口，灵活对接第三方设备与服务</span></li>
    </ul>
  </section>

  <footer class="footer">
    <div class="container footer-inner">
      <span class="footer-brand">HealthVision</span>
      <span class="footer-copy">&copy; 2026 HealthVision. All rights reserved.</span>
    </div>
  </footer>
</template>

<style scoped>
.hero {
  background: var(--hero-gradient);
  padding: 6rem 1.5rem 5rem;
  text-align: center;
}

.hero-inner {
  max-width: var(--container-max);
  margin: 0 auto;
}

.hero h1 {
  font-size: 3rem;
  color: #ffffff;
  margin-bottom: 1rem;
}

.hero-sub {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.85);
  max-width: 28rem;
  margin: 0 auto 2.5rem;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

.btn-lg {
  padding: 0.875rem 2.5rem;
  font-size: 1rem;
}

.hero .btn-primary {
  color: #ffffff;
}

.hero .btn-outline {
  color: #ffffff;
  border-color: rgba(255, 255, 255, 0.6);
}

.hero .btn-outline:hover {
  background: rgba(255, 255, 255, 0.1);
}

.features {
  margin-top: -2.5rem;
  position: relative;
  z-index: 1;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

@media (max-width: 640px) {
  .features-grid {
    grid-template-columns: 1fr;
  }
}

.feature-card {
  text-align: center;
  padding: 2rem 1.25rem;
}

.feature-icon {
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  background: var(--muted);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
  color: var(--primary);
}

.feature-card h3 {
  font-size: 1.125rem;
  margin-bottom: 0.5rem;
}

.feature-card p {
  font-size: 0.875rem;
  color: #6b5e54;
  line-height: 1.5;
}

.details {
  padding: 4rem 1.5rem;
}

.details h2 {
  font-size: 1.5rem;
  text-align: center;
  margin-bottom: 2rem;
}

.detail-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 32rem;
  margin: 0 auto;
}

.detail-list li {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 1rem;
}

.footer {
  background: var(--muted);
  border-top: 1px solid var(--border);
  padding: 2rem 0;
}

.footer-inner {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-brand {
  font-family: var(--font-serif);
  font-weight: 600;
  color: var(--primary);
  font-size: 1rem;
}

.footer-copy {
  font-size: 0.75rem;
  color: #8a7b70;
}

@media (max-width: 640px) {
  .hero h1 {
    font-size: 2.25rem;
  }

  .footer-inner {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
  }
}
</style>
