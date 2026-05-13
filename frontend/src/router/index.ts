import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
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
  const auth = useAuthStore()
  const authenticated = auth.isAuthenticated

  if (to.meta.auth && !authenticated) return '/login'

  if (to.meta.guest && authenticated) {
    return auth.isOld ? '/reminders' : '/medicines'
  }

  if (authenticated && auth.isOld && (to.path === '/medicines' || to.path === '/chat')) {
    return '/reminders'
  }
})

export default router
