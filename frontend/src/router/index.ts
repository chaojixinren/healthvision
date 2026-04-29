import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../services/auth'
import Home from '../views/Home.vue'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Dashboard from '../views/Dashboard.vue'
import Profile from '../views/Profile.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/login', component: Login, meta: { guest: true } },
    { path: '/register', component: Register, meta: { guest: true } },
    { path: '/dashboard', component: Dashboard, meta: { auth: true } },
    { path: '/profile', component: Profile, meta: { auth: true } },
  ],
})

router.beforeEach((to) => {
  if (to.meta.auth && !isAuthenticated()) return '/login'
  if (to.meta.guest && isAuthenticated()) return '/dashboard'
})

export default router
