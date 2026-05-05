<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { isAuthenticated, isOld } from './services/auth'
import { ref, computed } from 'vue'

const router = useRouter()
const route = useRoute()
const authenticated = ref(isAuthenticated())
const elderly = ref(isOld())

const isLanding = computed(() => route.path === '/')

router.afterEach(() => {
  authenticated.value = isAuthenticated()
  elderly.value = isOld()
})

const tabs = computed(() => {
  if (!authenticated.value) return []
  if (elderly.value) {
    return [
      { path: '/reminders', label: '提醒', icon: 'reminder' },
      { path: '/profile', label: '我的', icon: 'profile' },
    ]
  }
  return [
    { path: '/medicines', label: '药品', icon: 'medicine' },
    { path: '/reminders', label: '提醒', icon: 'reminder' },
    { path: '/chat', label: '助手', icon: 'chat' },
    { path: '/profile', label: '我的', icon: 'profile' },
  ]
})
</script>

<template>
  <header v-if="!isLanding" class="navbar">
    <div class="container navbar-inner" :class="{ centered: authenticated }">
      <router-link v-if="!authenticated" to="/" class="logo">HealthVision</router-link>
      <nav class="nav-links">
        <template v-if="authenticated && !elderly">
          <router-link to="/medicines">药品</router-link>
          <router-link to="/reminders">提醒</router-link>
          <router-link to="/chat">助手</router-link>
          <router-link to="/profile">我的</router-link>
        </template>
        <template v-else-if="authenticated && elderly">
          <router-link to="/reminders">提醒</router-link>
          <router-link to="/profile">我的</router-link>
        </template>
        <template v-else>
          <router-link to="/login">登录</router-link>
          <router-link to="/register">
            <button class="btn-primary btn-sm">注册</button>
          </router-link>
        </template>
      </nav>
    </div>
  </header>

  <main>
    <router-view />
  </main>

  <nav v-if="authenticated && !isLanding" class="bottom-bar">
    <router-link
      v-for="tab in tabs"
      :key="tab.path"
      :to="tab.path"
      class="bottom-tab"
      active-class="bottom-tab--active"
    >
      <svg v-if="tab.icon === 'medicine'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="3"/>
        <path d="M9 3v18"/>
        <path d="M3 9h18"/>
        <path d="M3 15h18"/>
      </svg>
      <svg v-else-if="tab.icon === 'reminder'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"/>
        <polyline points="12 6 12 12 16 14"/>
      </svg>
      <svg v-else-if="tab.icon === 'chat'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        <path d="M8 9h8"/>
        <path d="M8 13h5"/>
      </svg>
      <svg v-else-if="tab.icon === 'profile'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="8" r="4"/>
        <path d="M20 21a8 8 0 1 0-16 0"/>
      </svg>
      <span class="bottom-tab-label">{{ tab.label }}</span>
    </router-link>
  </nav>
</template>

<style scoped>
.navbar {
  background: var(--card);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 50;
}

.navbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 3.5rem;
}

.navbar-inner.centered {
  justify-content: center;
}

.logo {
  font-family: var(--font-serif);
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--primary);
  text-decoration: none;
}

.nav-links {
  display: flex;
  align-items: center;
  gap: 1.25rem;
}

.nav-links a {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--foreground);
  text-decoration: none;
}

.nav-links a:hover {
  color: var(--primary);
}

.btn-sm {
  padding: 0.5rem 1.25rem;
  font-size: 0.8125rem;
}

/* ── Bottom tab bar (mobile only) ── */
.bottom-bar {
  display: none;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 50;
  background: var(--card);
  border-top: 1px solid var(--border);
  padding-bottom: env(safe-area-inset-bottom);
}

.bottom-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 6px 0;
  color: #8a7b70;
  text-decoration: none;
  transition: color 0.15s;
  -webkit-tap-highlight-color: transparent;
}

.bottom-tab--active {
  color: var(--primary);
}

.bottom-tab-label {
  font-size: 0.625rem;
  font-weight: 500;
}

@media (max-width: 768px) {
  .navbar {
    display: none;
  }

  .bottom-bar {
    display: flex;
  }

  main {
    padding-bottom: calc(56px + env(safe-area-inset-bottom));
  }
}
</style>
