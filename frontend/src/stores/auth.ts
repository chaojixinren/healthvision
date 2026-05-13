import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { AuthResponse, User } from '../services/api'
import {
  AUTH_CHANGED_EVENT,
  getRefreshToken,
  getToken,
  getUser,
  removeToken,
  setAuthTokens,
  setUser,
} from '../services/auth'

let authListenerRegistered = false

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(getToken())
  const refreshToken = ref<string | null>(getRefreshToken())
  const user = ref<User | null>(getUser())

  const isAuthenticated = computed(() => !!accessToken.value)
  const isOld = computed(() => user.value?.is_old ?? false)

  function syncFromStorage() {
    accessToken.value = getToken()
    refreshToken.value = getRefreshToken()
    user.value = getUser()
  }

  function setSession(auth: AuthResponse) {
    accessToken.value = auth.access_token
    refreshToken.value = auth.refresh_token
    user.value = auth.user
    setAuthTokens(auth.access_token, auth.refresh_token)
    setUser(auth.user)
  }

  function setCurrentUser(nextUser: User) {
    user.value = nextUser
    setUser(nextUser)
  }

  function clearSession() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    removeToken()
  }

  if (!authListenerRegistered) {
    authListenerRegistered = true
    window.addEventListener(AUTH_CHANGED_EVENT, syncFromStorage)
    window.addEventListener('storage', syncFromStorage)
  }

  return {
    accessToken,
    refreshToken,
    user,
    isAuthenticated,
    isOld,
    syncFromStorage,
    setSession,
    setCurrentUser,
    clearSession,
  }
})
