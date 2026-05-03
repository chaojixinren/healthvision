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
    const message = body?.error?.message || body?.message || `Request failed: ${res.status}`
    throw new Error(message)
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
  is_old: boolean
  created_at: string
}

export interface AuthResponse {
  access_token: string
  user: User
}

export function register(data: { name: string; email: string; password: string; is_old: boolean }): Promise<AuthResponse> {
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
  user_id: number
  medicine_id: number
  medicine_name: string
  time: string
  enabled: boolean
  created_by: number
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

export function createReminder(data: { medicine_id: number; time: string; target_user_id?: number }): Promise<Reminder> {
  return post<Reminder>('/reminders', data)
}

export function updateReminder(id: number, data: { time: string; enabled: boolean }): Promise<Reminder> {
  return request<Reminder>(`/reminders/${id}`, { method: 'PUT', body: JSON.stringify(data) })
}

export function deleteReminder(id: number): Promise<void> {
  return request<void>(`/reminders/${id}`, { method: 'DELETE' })
}

// --- Chat ---

export interface Conversation {
  id: number
  user_id: number
  title: string
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  id: number
  user_id: number
  conversation_id: number
  role: string
  content: string
  images?: string
  created_at: string
}

export function listConversations(): Promise<{ data: Conversation[] }> {
  return get<{ data: Conversation[] }>('/chat/conversations')
}

export function getMessages(conversationID: number): Promise<{ data: ChatMessage[] }> {
  return post<{ data: ChatMessage[] }>('/chat/messages', { conversation_id: conversationID })
}

export function deleteConversation(id: number): Promise<void> {
  return post<void>('/chat/delete', { conversation_id: id })
}

// --- Bindings ---

export interface Binding {
  id: number
  elder_id: number
  child_id: number
  status: 'pending' | 'accepted' | 'rejected'
  created_at: string
  updated_at: string
  elder?: User
  child?: User
}

export function searchUsers(q: string): Promise<{ users: User[] }> {
  return get<{ users: User[] }>(`/users/search?q=${encodeURIComponent(q)}`)
}

export function createBinding(to_email: string): Promise<{ binding: Binding }> {
  return post<{ binding: Binding }>('/bindings', { to_email })
}

export function listBindings(): Promise<{ bindings: Binding[] }> {
  return get<{ bindings: Binding[] }>('/bindings')
}

export function respondBinding(id: number, accept: boolean): Promise<{ binding: Binding }> {
  return request<{ binding: Binding }>(`/bindings/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ accept }),
  })
}

export function deleteBinding(id: number): Promise<{ message: string }> {
  return request<{ message: string }>(`/bindings/${id}`, { method: 'DELETE' })
}

export function changeIdentity(): Promise<{ user: User; message: string }> {
  return request<{ user: User; message: string }>('/users/me/identity', { method: 'PUT' })
}
