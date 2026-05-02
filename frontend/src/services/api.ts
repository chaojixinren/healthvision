import { getToken, removeToken } from './auth'

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers })

  if (res.status === 401) {
    removeToken()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || body.message || `Request failed: ${res.status}`)
  }

  return res.json()
}

export function get<T>(path: string): Promise<T> {
  return request<T>(path)
}

export function post<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, { method: 'POST', body: JSON.stringify(body) })
}

export interface User {
  id: number
  name: string
  email: string
  created_at: string
}

export interface AuthResponse {
  access_token: string
  user: User
}

export function register(data: { name: string; email: string; password: string }): Promise<AuthResponse> {
  return post<AuthResponse>('/users', data)
}

export function login(data: { email: string; password: string }): Promise<AuthResponse> {
  return post<AuthResponse>('/sessions', data)
}

export function getMe(): Promise<User> {
  return get<{ user: User }>('/users/me').then((res) => res.user)
}

// --- Medicines ---

export interface Medicine {
  id: number
  name: string
  image_url: string
  description: string
  notes: string
  created_at: string
  updated_at: string
}

export interface PaginationInfo {
  page: number
  per_page: number
  total: number
}

export interface ListMedicinesResponse {
  data: Medicine[]
  pagination: PaginationInfo
}

export function listMedicines(page = 1, perPage = 20): Promise<ListMedicinesResponse> {
  return get<ListMedicinesResponse>(`/medicines?page=${page}&per_page=${perPage}`)
}

export function getMedicine(id: number): Promise<Medicine> {
  return get<Medicine>(`/medicines/${id}`)
}

export function createMedicine(data: Omit<Medicine, 'id' | 'created_at' | 'updated_at'>): Promise<Medicine> {
  return post<Medicine>('/medicines', data)
}

export function updateMedicine(
  id: number,
  data: Omit<Medicine, 'id' | 'created_at' | 'updated_at'>,
): Promise<Medicine> {
  return request<Medicine>(`/medicines/${id}`, { method: 'PUT', body: JSON.stringify(data) })
}

export function deleteMedicine(id: number): Promise<void> {
  return request<void>(`/medicines/${id}`, { method: 'DELETE' })
}

// --- Reminders ---

export interface Reminder {
  id: number
  medicine_id: number
  time: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ListRemindersResponse {
  data: Reminder[]
  pagination: PaginationInfo
}

export function listReminders(medicineID?: number, page = 1, perPage = 50): Promise<ListRemindersResponse> {
  const params = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  if (medicineID) params.set('medicine_id', String(medicineID))
  return get<ListRemindersResponse>(`/reminders?${params}`)
}

export function createReminder(data: { medicine_id: number; time: string }): Promise<Reminder> {
  return post<Reminder>('/reminders', data)
}

export function updateReminder(id: number, data: { time: string; enabled: boolean }): Promise<Reminder> {
  return request<Reminder>(`/reminders/${id}`, { method: 'PUT', body: JSON.stringify(data) })
}

export function deleteReminder(id: number): Promise<void> {
  return request<void>(`/reminders/${id}`, { method: 'DELETE' })
}

// --- Chat ---

export interface SSEChatEvent {
  id?: string
  partial?: boolean
  content: { role: 'user' | 'model'; parts: Array<{ text: string }> }
  turnComplete?: boolean
  finishReason?: string
  actions?: unknown
}

export interface ChatCallbacks {
  onToken: (text: string) => void
  onComplete: () => void
  onError: (error: string) => void
}

export function createSession(userId: string, sessionId: string): Promise<void> {
  return post<void>(`/agent/apps/healthvision/users/${userId}/sessions/${sessionId}`, {})
}

export function streamChat(
  userId: string,
  sessionId: string,
  message: string,
  callbacks: ChatCallbacks,
): AbortController {
  const controller = new AbortController()
  const token = getToken()

  fetch(`${BASE_URL}/agent/run_sse`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      appName: 'healthvision',
      userId,
      sessionId,
      newMessage: {
        role: 'user',
        parts: [{ text: message }],
      },
    }),
    signal: controller.signal,
  })
    .then(async (res) => {
      if (res.status === 401) {
        removeToken()
        window.location.href = '/login'
        return
      }
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as Record<string, string>))
        callbacks.onError(body.error || `请求失败 (${res.status})`)
        return
      }
      const reader = res.body?.getReader()
      if (!reader) {
        callbacks.onError('无法读取响应流')
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      try {
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const events = buffer.split('\n\n')
          buffer = events.pop() || ''

          for (const raw of events) {
            if (!raw.trim()) continue
            let eventType = ''
            let data = ''

            for (const line of raw.split('\n')) {
              if (line.startsWith('event: ')) eventType = line.slice(7).trim()
              else if (line.startsWith('data: ')) data = line.slice(6)
            }

            if (eventType === 'error') {
              try {
                const parsed = JSON.parse(data) as { error?: string }
                callbacks.onError(parsed.error || '流式响应错误')
              } catch {
                callbacks.onError('流式响应错误')
              }
              return
            }
            if (!data) continue

            try {
              const parsed = JSON.parse(data) as SSEChatEvent
              if (parsed.partial) {
                const text = parsed.content?.parts?.[0]?.text
                if (text) callbacks.onToken(text)
              }
            } catch {
              /* skip unparseable frames */
            }
          }
        }
        callbacks.onComplete()
      } catch (err: unknown) {
        if ((err as Error).name !== 'AbortError') {
          callbacks.onError((err as Error).message || '流式响应中断')
        }
      }
    })
    .catch((err: unknown) => {
      if ((err as Error).name !== 'AbortError') {
        callbacks.onError((err as Error).message || '请求失败')
      }
    })

  return controller
}
