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
import { scheduleAll } from '../services/notifications'

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
  const medicines = ref<Medicine[]>([])
  const reminders = ref<Reminder[]>([])
  const bindings = ref<Binding[]>([])
  const confirmations = ref<Confirmation[]>([])
  const loading = ref(false)
  const error = ref('')

  async function run<T>(work: () => Promise<T>): Promise<T> {
    loading.value = true
    error.value = ''
    try {
      return await work()
    } catch (e: unknown) {
      error.value = (e as Error).message || '加载失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function loadAll() {
    await run(async () => {
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
      await scheduleAll(reminders.value).catch(() => {})
    })
  }

  async function loadMedicines() {
    await run(async () => {
      const res = await listMedicines()
      medicines.value = res.data || []
    })
  }

  async function loadReminders() {
    await run(async () => {
      const res = await listReminders()
      reminders.value = res.data || []
      await scheduleAll(reminders.value).catch(() => {})
    })
  }

  async function loadBindings() {
    await run(async () => {
      const res = await listBindings()
      bindings.value = res.bindings || []
    })
  }

  async function loadConfirmations(date?: string) {
    await run(async () => {
      const res = await listConfirmations(date)
      confirmations.value = res.data || []
    })
  }

  async function createMedicine(data: MedicineInput) {
    const medicine = await apiCreateMedicine(data)
    medicines.value = [medicine, ...medicines.value]
    return medicine
  }

  async function updateMedicine(id: number, data: MedicineInput) {
    const medicine = await apiUpdateMedicine(id, data)
    medicines.value = medicines.value.map((m) => (m.id === id ? medicine : m))
    return medicine
  }

  async function deleteMedicine(id: number) {
    await apiDeleteMedicine(id)
    medicines.value = medicines.value.filter((m) => m.id !== id)
    reminders.value = reminders.value.filter((r) => r.medicine_id !== id)
    confirmations.value = confirmations.value.filter((c) => c.medicine_id !== id)
    await scheduleAll(reminders.value).catch(() => {})
  }

  async function createReminder(data: ReminderInput) {
    const reminder = await apiCreateReminder(data)
    await loadReminders()
    await loadConfirmations()
    return reminder
  }

  async function updateReminder(id: number, data: ReminderUpdateInput) {
    const reminder = await apiUpdateReminder(id, data)
    reminders.value = reminders.value.map((r) => (r.id === id ? reminder : r))
    await scheduleAll(reminders.value).catch(() => {})
    return reminder
  }

  async function deleteReminder(id: number) {
    await apiDeleteReminder(id)
    reminders.value = reminders.value.filter((r) => r.id !== id)
    confirmations.value = confirmations.value.filter((c) => c.reminder_id !== id)
    await scheduleAll(reminders.value).catch(() => {})
  }

  async function confirm(id: number) {
    const res = await confirmDose(id)
    const target = confirmations.value.find((c) => c.id === id)
    if (target) {
      target.confirmed_at = res.confirmed_at
      target.confirmed_by = res.confirmed_by
    }
    return res
  }

  async function createBinding(email: string) {
    await apiCreateBinding(email)
    await loadBindings()
  }

  async function respondBinding(id: number, accept: boolean) {
    await apiRespondBinding(id, accept)
    await loadBindings()
  }

  async function deleteBinding(id: number) {
    await apiDeleteBinding(id)
    bindings.value = bindings.value.filter((b) => b.id !== id)
  }

  function reset() {
    medicines.value = []
    reminders.value = []
    bindings.value = []
    confirmations.value = []
    loading.value = false
    error.value = ''
  }

  return {
    medicines,
    reminders,
    bindings,
    confirmations,
    loading,
    error,
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
    createBinding,
    respondBinding,
    deleteBinding,
    reset,
  }
})
