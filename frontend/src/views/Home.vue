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
      <div class="hero-badge">您的私人健康管家</div>
      <h1>按时用药<br>就是最好的治疗</h1>
      <p class="hero-sub">HealthVision 帮你管理药品、设定用药提醒，让每一次服药都准时无忧</p>
      <div class="hero-actions">
        <button class="btn-primary btn-lg" @click="cta">免费开始使用</button>
        <router-link to="/login" class="btn-ghost btn-lg">已有账户？登录</router-link>
      </div>
    </div>
    <div class="hero-wave">
      <svg viewBox="0 0 1440 120" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M0 60C240 120 480 0 720 60C960 120 1200 0 1440 60V120H0V60Z" fill="var(--background)"/>
      </svg>
    </div>
  </section>

  <section class="features container">
    <h2 class="section-title">帮你做到这三点</h2>
    <div class="features-grid">
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 10v4M6 4h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z"/>
            <path d="M6 10h12"/>
            <path d="M9 14h6"/>
            <path d="M9 18h3"/>
          </svg>
        </div>
        <h3>药品管理</h3>
        <p>集中记录所有药品信息，名称、图片、说明和备注一目了然，告别翻找药瓶的烦恼</p>
      </div>
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
        </div>
        <h3>用药提醒</h3>
        <p>为每种药品设置每日提醒时间，手机通知准时响起，再也不会漏服或重复用药</p>
      </div>
      <div class="card feature-card">
        <div class="feature-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>
          </svg>
        </div>
        <h3>健康关怀</h3>
        <p>关心自己，也关心家人。帮长辈管理用药计划，即使不在身边也能及时提醒</p>
      </div>
    </div>
  </section>

  <section class="steps">
    <div class="container">
      <h2 class="section-title">三步开始使用</h2>
      <div class="steps-grid">
        <div class="step">
          <div class="step-num">1</div>
          <h3>注册账户</h3>
          <p>几秒钟完成注册，进入你的专属健康空间</p>
        </div>
        <div class="step-arrow">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
          </svg>
        </div>
        <div class="step">
          <div class="step-num">2</div>
          <h3>添加药品</h3>
          <p>录入你正在服用的药品，写上用法用量</p>
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
    </div>
  </section>

  <section class="cta-section">
    <div class="container">
      <div class="card-lg cta-card">
        <h2>准备好开始了吗？</h2>
        <p>加入 HealthVision，让用药管理变得简单可靠</p>
        <button class="btn-primary btn-lg" @click="cta">立即开始</button>
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
/* Hero */
.hero {
  background: var(--hero-gradient);
  padding: 5rem 1.5rem 0;
  text-align: center;
  position: relative;
}

.hero-inner {
  max-width: var(--container-max);
  margin: 0 auto;
  padding-bottom: 4rem;
}

.hero-badge {
  display: inline-block;
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-full);
  padding: 0.375rem 1.25rem;
  font-size: 0.8125rem;
  margin-bottom: 1.5rem;
  backdrop-filter: blur(4px);
}

.hero h1 {
  font-size: 3.25rem;
  color: #ffffff;
  margin-bottom: 1.25rem;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.hero-sub {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.8);
  max-width: 30rem;
  margin: 0 auto 2.5rem;
  line-height: 1.7;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
}

.hero .btn-primary {
  color: var(--primary);
  background: #ffffff;
  font-weight: 600;
}

.hero .btn-primary:hover {
  opacity: 0.92;
}

.btn-ghost {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: rgba(255, 255, 255, 0.8);
  border-radius: var(--radius-full);
  padding: 0.875rem 2.5rem;
  font-weight: 500;
  font-size: 1rem;
  font-family: var(--font-sans);
  cursor: pointer;
  text-decoration: none;
  transition: color 0.2s;
}

.btn-ghost:hover {
  color: #ffffff;
  text-decoration: none;
}

.btn-lg {
  padding: 0.875rem 2.5rem;
  font-size: 1rem;
}

.hero-wave {
  line-height: 0;
}

.hero-wave svg {
  width: 100%;
  height: auto;
}

/* Features */
.features {
  padding: 5rem 1.5rem;
}

.section-title {
  text-align: center;
  font-size: 1.75rem;
  margin-bottom: 3rem;
  color: var(--foreground);
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.25rem;
}

.feature-card {
  text-align: center;
  padding: 2.5rem 1.5rem;
  transition: transform 0.2s, box-shadow 0.2s;
}

.feature-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 24px rgba(139, 34, 82, 0.08);
}

.feature-icon {
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 1rem;
  background: var(--primary-light);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1.25rem;
  color: var(--primary);
}

.feature-card h3 {
  font-size: 1.125rem;
  margin-bottom: 0.625rem;
}

.feature-card p {
  font-size: 0.875rem;
  color: #6b5e54;
  line-height: 1.6;
}

/* Steps */
.steps {
  padding: 0 1.5rem 5rem;
}

.steps-grid {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: 1rem;
}

.step {
  flex: 1;
  max-width: 14rem;
  text-align: center;
}

.step-num {
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 50%;
  background: var(--primary);
  color: #ffffff;
  font-family: var(--font-serif);
  font-size: 1.25rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.step h3 {
  font-size: 1.0625rem;
  margin-bottom: 0.5rem;
}

.step p {
  font-size: 0.8125rem;
  color: #6b5e54;
  line-height: 1.5;
}

.step-arrow {
  display: flex;
  align-items: center;
  padding-top: 0.75rem;
  color: #c9b8ad;
  flex-shrink: 0;
}

/* CTA */
.cta-section {
  padding: 0 1.5rem 5rem;
}

.cta-card {
  text-align: center;
  padding: 3.5rem 2rem;
  background: linear-gradient(135deg, #fdf6f0 0%, #f9ece1 50%, #f5e0d3 100%);
  border: 1px solid #ecd5c2;
}

.cta-card h2 {
  font-size: 1.75rem;
  margin-bottom: 0.75rem;
}

.cta-card p {
  font-size: 0.9375rem;
  color: #6b5e54;
  margin-bottom: 2rem;
}

/* Footer */
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

/* Responsive */
@media (max-width: 768px) {
  .hero h1 {
    font-size: 2.25rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
    max-width: 24rem;
    margin: 0 auto;
  }

  .steps-grid {
    flex-direction: column;
    align-items: center;
  }

  .step-arrow {
    transform: rotate(90deg);
    padding-top: 0;
  }

  .footer-inner {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
  }
}
</style>
