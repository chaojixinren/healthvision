import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../services/auth'
import Home from '../views/Home.vue'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import Profile from '../views/Profile.vue'
import Medicines from '../views/Medicines.vue'
import Reminders from '../views/Reminders.vue'
import Chat from '../views/Chat.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/login', component: Login, meta: { guest: true } },
    { path: '/register', component: Register, meta: { guest: true } },
    { path: '/medicines', component: Medicines, meta: { auth: true } },
    { path: '/reminders', component: Reminders, meta: { auth: true } },
    { path: '/chat', component: Chat, meta: { auth: true } },
    { path: '/profile', component: Profile, meta: { auth: true } },
  ],
})

router.beforeEach((to) => {
  if (to.meta.auth && !isAuthenticated()) return '/login'
  if (to.meta.guest && isAuthenticated()) return '/medicines'
})

export default router
