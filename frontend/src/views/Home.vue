<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

function cta() {
  router.push(auth.isAuthenticated ? (auth.isOld ? '/reminders' : '/medicines') : '/register')
}
</script>

<template>
  <div class="page">
    <section class="hero">
      <div class="hero-inner">
        <h1>HealthVision</h1>
        <p class="hero-sub">按时用药 就是最好的治疗</p>
        <div class="hero-actions">
          <button class="btn-primary btn-lg" @click="cta">免费开始使用</button>
          <router-link to="/login" class="btn-primary btn-lg">已有账户？登录</router-link>
        </div>
      </div>
    </section>

    <section class="steps">
      <div class="steps-grid">
        <div class="step">
          <div class="step-num">1</div>
          <h3>注册账户</h3>
          <p>几秒钟完成注册，进入专属健康空间</p>
        </div>
        <div class="step-arrow">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
          </svg>
        </div>
        <div class="step">
          <div class="step-num">2</div>
          <h3>添加药品</h3>
          <p>录入正在服用的药品，写上用法用量</p>
        </div>
        <div class="step-arrow">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
          </svg>
        </div>
        <div class="step">
          <div class="step-num">3</div>
          <h3>设定提醒</h3>
          <p>选择药品和时间，每天准时收到用药通知</p>
        </div>
      </div>
    </section>

    <footer class="footer">
      <span class="footer-brand">HealthVision</span>
      <span class="footer-copy">&copy; 2026 HealthVision</span>
    </footer>
  </div>
</template>

<style scoped>
.page {
  height: 100dvh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Hero */
.hero {
  background: var(--hero-gradient);
  padding: 3.5rem 1.5rem 2rem;
  text-align: center;
  flex-shrink: 0;
}

.hero-inner {
  max-width: var(--container-max);
  margin: 0 auto;
}

.hero h1 {
  font-size: 2.75rem;
  color: #ffffff;
  margin-bottom: 0.75rem;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.hero-sub {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.8);
  max-width: 30rem;
  margin: 0 auto 1.75rem;
  line-height: 1.7;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
}

.btn-lg {
  padding: 0.75rem 2rem;
  font-size: 0.9375rem;
}

/* Steps */
.steps {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem 1.5rem;
}

.steps-grid {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: 0.75rem;
}

.step {
  flex: 1;
  max-width: 12rem;
  text-align: center;
}

.step-num {
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  background: var(--primary);
  color: #ffffff;
  font-family: var(--font-serif);
  font-size: 1.375rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.step h3 {
  font-size: 1.125rem;
  margin-bottom: 0.5rem;
}

.step p {
  font-size: 0.875rem;
  color: #6b5e54;
  line-height: 1.5;
}

.step-arrow {
  display: flex;
  align-items: center;
  padding-top: 0.5rem;
  color: #c9b8ad;
  flex-shrink: 0;
}

/* Footer */
.footer {
  background: var(--muted);
  border-top: 1px solid var(--border);
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
}

.footer-brand {
  font-family: var(--font-serif);
  font-weight: 600;
  color: var(--primary);
  font-size: 0.8125rem;
}

.footer-copy {
  font-size: 0.75rem;
  color: #8a7b70;
}

/* Responsive */
@media (max-width: 768px) {
  .hero {
    padding: 1.75rem 1.25rem 1rem;
  }

  .hero h1 {
    font-size: 2rem;
  }

  .hero-sub {
    font-size: 0.9375rem;
    margin-bottom: 1.25rem;
  }

  .btn-lg {
    padding: 0.625rem 1.5rem;
    font-size: 0.875rem;
  }

  .steps {
    padding: 0.75rem 1rem;
  }

  .steps-grid {
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
  }

  .step {
    max-width: 14rem;
  }

  .step-arrow {
    transform: rotate(90deg);
    padding-top: 0;
  }

  .footer {
    flex-direction: column;
    gap: 0.25rem;
    text-align: center;
    padding: 0.75rem 1.5rem;
    padding-bottom: calc(0.75rem + env(safe-area-inset-bottom));
  }
}
</style>
