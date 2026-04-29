<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { isAuthenticated, removeToken } from './services/auth'
import { ref, computed } from 'vue'

const router = useRouter()
const route = useRoute()
const authenticated = ref(isAuthenticated())

const isLanding = computed(() => route.path === '/')

function logout() {
  removeToken()
  authenticated.value = false
  router.push('/')
}

router.afterEach(() => {
  authenticated.value = isAuthenticated()
})
</script>

<template>
  <header v-if="!isLanding" class="navbar">
    <div class="container navbar-inner">
      <router-link to="/" class="logo">HealthVision</router-link>
      <nav class="nav-links">
        <template v-if="authenticated">
          <router-link to="/dashboard">Dashboard</router-link>
          <router-link to="/profile">Profile</router-link>
          <button class="btn-outline btn-sm" @click="logout">Logout</button>
        </template>
        <template v-else>
          <router-link to="/login">Login</router-link>
          <router-link to="/register">
            <button class="btn-primary btn-sm">Sign Up</button>
          </router-link>
        </template>
      </nav>
    </div>
  </header>

  <main>
    <router-view />
  </main>
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
</style>
