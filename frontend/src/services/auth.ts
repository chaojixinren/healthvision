import type { User } from './api'

const TOKEN_KEY = 'hv_token'
const REFRESH_TOKEN_KEY = 'hv_refresh_token'
const USER_KEY = 'hv_user'
export const AUTH_CHANGED_EVENT = 'healthvision:auth-changed'

function notifyAuthChanged(): void {
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT))
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
  notifyAuthChanged()
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function setRefreshToken(token: string): void {
  localStorage.setItem(REFRESH_TOKEN_KEY, token)
  notifyAuthChanged()
}

export function setAuthTokens(accessToken: string, refreshToken: string): void {
  localStorage.setItem(TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  notifyAuthChanged()
}

export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  notifyAuthChanged()
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function getUser(): User | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

export function setUser(user: User): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
  notifyAuthChanged()
}

export function isOld(): boolean {
  return getUser()?.is_old ?? false
}
