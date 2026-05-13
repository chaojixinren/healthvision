import { ref } from 'vue'
import { defineStore } from 'pinia'
import {
  listMedicines,
  createMedicine as apiCreateMedicine,
  updateMedicine as apiUpdateMedicine,
  deleteMedicine as apiDeleteMedicine,
  listReminders,
  createReminder as apiCreateReminder,
  updateReminder as apiUpdateReminder,
  deleteReminder as apiDeleteReminder,
  listBindings,
  createBinding as apiCreateBinding,
  respondBinding as apiRespondBinding,
  deleteBinding as apiDeleteBinding,
  listConfirmations,
  confirmDose,
  type Binding,
  type Confirmation,
  type Medicine,
  type Reminder,
} from '../services/api'
import {
  countOutboxOperations,
  clearCareCache,
  clearOutboxOperations,
  enqueueOutboxOperation,
  listOutboxOperations,
  readCareCache,
  writeCareCache,
  type CareCacheData,
  type CareCacheRecord,
  type OutboxOperationType,
} from '../services/offline-db'
import { syncOutbox } from '../services/offline-sync'
import { scheduleAll } from '../services/notifications'
import { useAuthStore } from './auth'
import { useNetworkStore } from './network'

type MedicineInput = Omit<Medicine, 'id' | 'created_at' | 'updated_at'>
type ReminderInput = {
  medicine_id: number
  time: string
  target_user_id?: number
  repeat_type?: string
  interval_days?: number
  weekdays?: string
}
type ReminderUpdateInput = {
  time: string
  enabled: boolean
  repeat_type?: string
  interval_days?: number
  weekdays?: string
}

export const useCareStore = defineStore('care', () => {
  const auth = useAuthStore()
  const network = useNetworkStore()
  const medicines = ref<Medicine[]>([])
  const reminders = ref<Reminder[]>([])
  const bindings = ref<Binding[]>([])
  const confirmations = ref<Confirmation[]>([])
  const loading = ref(false)
  const error = ref('')
  const offlineMessage = ref('')
  const usingOfflineCache = ref(false)
  const lastSyncedAt = ref<string | null>(null)
  const pendingSyncCount = ref(0)
  const syncing = ref(false)
  const syncError = ref('')
  const pendingMedicineIds = ref<number[]>([])
  const pendingReminderIds = ref<number[]>([])
  const pendingConfirmationIds = ref<number[]>([])

  function currentUserId(): number | null {
    return auth.user?.id ?? null
  }

  function now(): string {
    return new Date().toISOString()
  }

  function tempId(): number {
    return -Date.now()
  }

  function isTransientMutationError(e: unknown): boolean {
    const message = messageFromError(e, '')
    return /Failed to fetch|NetworkError|Load failed|Request failed: 50[234]|ECONNREFUSED/i.test(message)
  }

  function shouldQueueMutation(e?: unknown): boolean {
    return !network.connected || (e !== undefined && isTransientMutationError(e))
  }

  function setQueuedMessage() {
    error.value = ''
    offlineMessage.value = '离线编辑已保存，将在网络恢复后自动同步'
    usingOfflineCache.value = true
  }

  async function refreshPendingSyncCount() {
    const userId = currentUserId()
    if (!userId) {
      pendingSyncCount.value = 0
      pendingMedicineIds.value = []
      pendingReminderIds.value = []
      pendingConfirmationIds.value = []
      return
    }
    const operations = await listOutboxOperations(userId).catch(() => null)
    if (!operations) {
      pendingSyncCount.value = await countOutboxOperations(userId).catch(() => pendingSyncCount.value)
      return
    }

    pendingSyncCount.value = operations.length
    pendingMedicineIds.value = operations
      .filter((operation) => operation.type.startsWith('medicine.'))
      .map((operation) => {
        const payload = operation.payload as { id?: number; clientId?: number }
        return payload.id ?? payload.clientId ?? 0
      })
      .filter((id) => id !== 0)
    pendingReminderIds.value = operations
      .filter((operation) => operation.type.startsWith('reminder.'))
      .map((operation) => {
        const payload = operation.payload as { id?: number; clientId?: number }
        return payload.id ?? payload.clientId ?? 0
      })
      .filter((id) => id !== 0)
    pendingConfirmationIds.value = operations
      .filter((operation) => operation.type === 'confirmation.confirm')
      .map((operation) => (operation.payload as { id?: number }).id ?? 0)
      .filter((id) => id !== 0)
  }

  async function enqueueMutation<TPayload>(
    type: OutboxOperationType,
    payload: TPayload,
  ) {
    const userId = currentUserId()
    if (!userId) throw new Error('请先登录')
    await enqueueOutboxOperation(userId, type, payload)
    await refreshPendingSyncCount()
    setQueuedMessage()
  }

  function applyCache(record: CareCacheRecord) {
    medicines.value = record.medicines ?? []
    reminders.value = record.reminders ?? []
    bindings.value = record.bindings ?? []
    confirmations.value = record.confirmations ?? []
    lastSyncedAt.value = record.updatedAt
    usingOfflineCache.value = true
  }

  async function hydrateFromCache(): Promise<boolean> {
    const userId = currentUserId()
    if (!userId) return false

    const record = await readCareCache(userId).catch(() => null)
    if (!record) return false

    applyCache(record)
    await scheduleAll(reminders.value).catch(() => {})
    return true
  }

  async function persistCache(data: CareCacheData = {}) {
    const userId = currentUserId()
    if (!userId) return

    const record = await writeCareCache(userId, {
      medicines: medicines.value,
      reminders: reminders.value,
      bindings: bindings.value,
      confirmations: confirmations.value,
      ...data,
    }).catch(() => null)

    if (record) {
      lastSyncedAt.value = record.updatedAt
    }
  }

  function messageFromError(e: unknown, fallback: string): string {
    return (e as Error).message || fallback
  }

  function keepCachedData(e: unknown, hadCache: boolean, fallback: string): boolean {
    if (!hadCache) {
      error.value = messageFromError(e, fallback)
      return false
    }
    error.value = ''
    offlineMessage.value = '离线模式，正在显示上次同步数据'
    usingOfflineCache.value = true
    return true
  }

  async function loadAll() {
    loading.value = true
    error.value = ''
    offlineMessage.value = ''
    const hadCache = await hydrateFromCache()
    try {
      const [mRes, rRes, bRes, cRes] = await Promise.all([
        listMedicines(),
        listReminders(),
        listBindings(),
        listConfirmations(),
      ])
      medicines.value = mRes.data || []
      reminders.value = rRes.data || []
      bindings.value = bRes.bindings || []
      confirmations.value = cRes.data || []
      usingOfflineCache.value = false
      await persistCache()
      await refreshPendingSyncCount()
      await scheduleAll(reminders.value).catch(() => {})
    } catch (e: unknown) {
      if (!keepCachedData(e, hadCache, '加载失败')) throw e
    } finally {
      await refreshPendingSyncCount()
      loading.value = false
    }
  }

  async function loadMedicines() {
    loading.value = true
    error.value = ''
    offlineMessage.value = ''
    const hadCache = await hydrateFromCache()
    try {
      const res = await listMedicines()
      medicines.value = res.data || []
      usingOfflineCache.value = false
      await persistCache({ medicines: medicines.value })
      await refreshPendingSyncCount()
    } catch (e: unknown) {
      if (!keepCachedData(e, hadCache, '加载药品失败')) throw e
    } finally {
      await refreshPendingSyncCount()
      loading.value = false
    }
  }

  async function loadReminders() {
    loading.value = true
    error.value = ''
    offlineMessage.value = ''
    const hadCache = await hydrateFromCache()
    try {
      const res = await listReminders()
      reminders.value = res.data || []
      usingOfflineCache.value = false
      await persistCache({ reminders: reminders.value })
      await refreshPendingSyncCount()
      await scheduleAll(reminders.value).catch(() => {})
    } catch (e: unknown) {
      if (!keepCachedData(e, hadCache, '加载提醒失败')) throw e
    } finally {
      await refreshPendingSyncCount()
      loading.value = false
    }
  }

  async function loadBindings() {
    loading.value = true
    error.value = ''
    offlineMessage.value = ''
    const hadCache = await hydrateFromCache()
    try {
      const res = await listBindings()
      bindings.value = res.bindings || []
      usingOfflineCache.value = false
      await persistCache({ bindings: bindings.value })
      await refreshPendingSyncCount()
    } catch (e: unknown) {
      if (!keepCachedData(e, hadCache, '加载绑定失败')) throw e
    } finally {
      await refreshPendingSyncCount()
      loading.value = false
    }
  }

  async function loadConfirmations(date?: string) {
    loading.value = true
    error.value = ''
    offlineMessage.value = ''
    const hadCache = await hydrateFromCache()
    try {
      const res = await listConfirmations(date)
      confirmations.value = res.data || []
      usingOfflineCache.value = false
      await persistCache({ confirmations: confirmations.value })
      await refreshPendingSyncCount()
    } catch (e: unknown) {
      if (!keepCachedData(e, hadCache, '加载服药确认失败')) throw e
    } finally {
      await refreshPendingSyncCount()
      loading.value = false
    }
  }

  async function createMedicine(data: MedicineInput) {
    if (network.connected) {
      try {
        const medicine = await apiCreateMedicine(data)
        medicines.value = [medicine, ...medicines.value]
        await persistCache()
        return medicine
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    const medicine: Medicine = {
      id: tempId(),
      ...data,
      created_at: now(),
      updated_at: now(),
    }
    medicines.value = [medicine, ...medicines.value]
    await persistCache()
    await enqueueMutation('medicine.create', { clientId: medicine.id, data })
    return medicine
  }

  async function updateMedicine(id: number, data: MedicineInput) {
    if (network.connected && id > 0) {
      try {
        const medicine = await apiUpdateMedicine(id, data)
        medicines.value = medicines.value.map((m) => (m.id === id ? medicine : m))
        await persistCache()
        return medicine
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    let updated: Medicine | null = null
    medicines.value = medicines.value.map((m) => {
      if (m.id !== id) return m
      updated = { ...m, ...data, updated_at: now() }
      return updated
    })
    await persistCache()
    await enqueueMutation('medicine.update', { id, data })
    return updated ?? { id, ...data, created_at: now(), updated_at: now() }
  }

  async function deleteMedicine(id: number) {
    if (network.connected && id > 0) {
      try {
        await apiDeleteMedicine(id)
        medicines.value = medicines.value.filter((m) => m.id !== id)
        reminders.value = reminders.value.filter((r) => r.medicine_id !== id)
        confirmations.value = confirmations.value.filter((c) => c.medicine_id !== id)
        await persistCache()
        await scheduleAll(reminders.value).catch(() => {})
        return
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    medicines.value = medicines.value.filter((m) => m.id !== id)
    reminders.value = reminders.value.filter((r) => r.medicine_id !== id)
    confirmations.value = confirmations.value.filter((c) => c.medicine_id !== id)
    await persistCache()
    await enqueueMutation('medicine.delete', { id })
    await scheduleAll(reminders.value).catch(() => {})
  }

  async function createReminder(data: ReminderInput) {
    if (network.connected) {
      try {
        const reminder = await apiCreateReminder(data)
        await loadReminders()
        await loadConfirmations()
        await persistCache()
        return reminder
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    const userId = currentUserId()
    const reminder: Reminder = {
      id: tempId(),
      user_id: data.target_user_id || userId || 0,
      medicine_id: data.medicine_id,
      medicine_name: medicines.value.find((m) => m.id === data.medicine_id)?.name || '药品',
      time: data.time,
      repeat_type: (data.repeat_type as Reminder['repeat_type']) || 'daily',
      interval_days: data.interval_days || 0,
      weekdays: data.weekdays || '',
      enabled: true,
      created_by: userId || 0,
      created_at: now(),
      updated_at: now(),
    }
    reminders.value = [reminder, ...reminders.value]
    await persistCache()
    await enqueueMutation('reminder.create', { clientId: reminder.id, data })
    await scheduleAll(reminders.value).catch(() => {})
    return reminder
  }

  async function updateReminder(id: number, data: ReminderUpdateInput) {
    if (network.connected && id > 0) {
      try {
        const reminder = await apiUpdateReminder(id, data)
        reminders.value = reminders.value.map((r) => (r.id === id ? reminder : r))
        await persistCache()
        await scheduleAll(reminders.value).catch(() => {})
        return reminder
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    let updated: Reminder | null = null
    reminders.value = reminders.value.map((r) => {
      if (r.id !== id) return r
      updated = {
        ...r,
        ...data,
        repeat_type: (data.repeat_type as Reminder['repeat_type']) || r.repeat_type,
        interval_days: data.interval_days ?? r.interval_days,
        weekdays: data.weekdays ?? r.weekdays,
        updated_at: now(),
      }
      return updated
    })
    await persistCache()
    await enqueueMutation('reminder.update', { id, data })
    await scheduleAll(reminders.value).catch(() => {})
    if (!updated) throw new Error('提醒不存在')
    return updated
  }

  async function deleteReminder(id: number) {
    if (network.connected && id > 0) {
      try {
        await apiDeleteReminder(id)
        reminders.value = reminders.value.filter((r) => r.id !== id)
        confirmations.value = confirmations.value.filter((c) => c.reminder_id !== id)
        await persistCache()
        await scheduleAll(reminders.value).catch(() => {})
        return
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    reminders.value = reminders.value.filter((r) => r.id !== id)
    confirmations.value = confirmations.value.filter((c) => c.reminder_id !== id)
    await persistCache()
    await enqueueMutation('reminder.delete', { id })
    await scheduleAll(reminders.value).catch(() => {})
  }

  async function confirm(id: number) {
    if (network.connected) {
      try {
        const res = await confirmDose(id)
        const target = confirmations.value.find((c) => c.id === id)
        if (target) {
          target.confirmed_at = res.confirmed_at
          target.confirmed_by = res.confirmed_by
        }
        await persistCache()
        return res
      } catch (e: unknown) {
        if (!shouldQueueMutation(e)) throw e
      }
    }

    const confirmedAt = now()
    const confirmedBy = currentUserId() || 0
    const target = confirmations.value.find((c) => c.id === id)
    if (target) {
      target.confirmed_at = confirmedAt
      target.confirmed_by = confirmedBy
    }
    await persistCache()
    await enqueueMutation('confirmation.confirm', { id })
    return { id, confirmed_at: confirmedAt, confirmed_by: confirmedBy }
  }

  async function syncQueuedChanges() {
    const userId = currentUserId()
    if (!userId || syncing.value) return

    syncing.value = true
    syncError.value = ''
    try {
      await syncOutbox(userId)
      await refreshPendingSyncCount()
      if (pendingSyncCount.value === 0) {
        offlineMessage.value = ''
      }
      await loadAll()
    } catch (e: unknown) {
      syncError.value = messageFromError(e, '同步失败')
      await refreshPendingSyncCount()
    } finally {
      syncing.value = false
    }
  }

  async function clearOfflineData(userId = currentUserId()) {
    if (!userId) return
    await Promise.all([
      clearCareCache(userId),
      clearOutboxOperations(userId),
    ]).catch(() => {})
    await refreshPendingSyncCount()
  }

  function isMedicinePending(id: number): boolean {
    return pendingMedicineIds.value.includes(id)
  }

  function isReminderPending(id: number): boolean {
    return pendingReminderIds.value.includes(id)
  }

  function isConfirmationPending(id: number): boolean {
    return pendingConfirmationIds.value.includes(id)
  }

  async function createBinding(email: string) {
    await apiCreateBinding(email)
    await loadBindings()
    await persistCache()
  }

  async function respondBinding(id: number, accept: boolean) {
    await apiRespondBinding(id, accept)
    await loadBindings()
    await persistCache()
  }

  async function deleteBinding(id: number) {
    await apiDeleteBinding(id)
    bindings.value = bindings.value.filter((b) => b.id !== id)
    await persistCache()
  }

  function reset() {
    medicines.value = []
    reminders.value = []
    bindings.value = []
    confirmations.value = []
    loading.value = false
    error.value = ''
    offlineMessage.value = ''
    usingOfflineCache.value = false
    lastSyncedAt.value = null
    pendingSyncCount.value = 0
    syncing.value = false
    syncError.value = ''
    pendingMedicineIds.value = []
    pendingReminderIds.value = []
    pendingConfirmationIds.value = []
  }

  return {
    medicines,
    reminders,
    bindings,
    confirmations,
    loading,
    error,
    offlineMessage,
    usingOfflineCache,
    lastSyncedAt,
    pendingSyncCount,
    syncing,
    syncError,
    pendingMedicineIds,
    pendingReminderIds,
    pendingConfirmationIds,
    loadAll,
    loadMedicines,
    loadReminders,
    loadBindings,
    loadConfirmations,
    createMedicine,
    updateMedicine,
    deleteMedicine,
    createReminder,
    updateReminder,
    deleteReminder,
    confirm,
    refreshPendingSyncCount,
    syncQueuedChanges,
    clearOfflineData,
    isMedicinePending,
    isReminderPending,
    isConfirmationPending,
    createBinding,
    respondBinding,
    deleteBinding,
    reset,
  }
})
