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
