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
      <p class="hero-sub">用药提醒平台，关心你的健康</p>
      <div class="hero-actions">
        <button class="btn-primary btn-lg" @click="cta">开始使用</button>
        <router-link to="/login" class="btn-outline btn-lg">登录</router-link>
      </div>
    </div>
  </section>



  <footer class="footer">
    <div class="container footer-inner">
      <span class="footer-brand">HealthVision</span>
      <span class="footer-copy">&copy; 2026 HealthVision 保留所有权利</span>
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
