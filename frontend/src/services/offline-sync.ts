import {
  createMedicine,
  createReminder,
  deleteMedicine,
  deleteReminder,
  updateMedicine,
  updateReminder,
  confirmDose,
} from './api'
import {
  deleteOutboxOperation,
  listOutboxOperations,
  markOutboxOperationFailed,
  type OutboxOperation,
} from './offline-db'

type MedicineInput = {
  name: string
  image_url: string
  description: string
  notes: string
}

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

type MedicineCreatePayload = { clientId: number; data: MedicineInput }
type MedicineUpdatePayload = { id: number; data: MedicineInput }
type MedicineDeletePayload = { id: number }
type ReminderCreatePayload = { clientId: number; data: ReminderInput }
type ReminderUpdatePayload = { id: number; data: ReminderUpdateInput }
type ReminderDeletePayload = { id: number }
type ConfirmationPayload = { id: number }

export interface SyncResult {
  synced: number
}

function errorMessage(e: unknown): string {
  return (e as Error).message || '同步失败'
}

function mappedId(id: number, map: Map<number, number>): number {
  return id < 0 ? map.get(id) ?? id : id
}

export async function syncOutbox(userId: number): Promise<SyncResult> {
  const operations = await listOutboxOperations(userId)
  const medicineIds = new Map<number, number>()
  const reminderIds = new Map<number, number>()
  let synced = 0

  for (const operation of operations) {
    try {
      await syncOperation(operation, medicineIds, reminderIds)
      await deleteOutboxOperation(operation.id)
      synced += 1
    } catch (e: unknown) {
      await markOutboxOperationFailed(operation.id, errorMessage(e))
      throw e
    }
  }

  return { synced }
}

async function syncOperation(
  operation: OutboxOperation,
  medicineIds: Map<number, number>,
  reminderIds: Map<number, number>,
): Promise<void> {
  switch (operation.type) {
    case 'medicine.create': {
      const payload = operation.payload as MedicineCreatePayload
      const medicine = await createMedicine(payload.data)
      medicineIds.set(payload.clientId, medicine.id)
      return
    }
    case 'medicine.update': {
      const payload = operation.payload as MedicineUpdatePayload
      await updateMedicine(mappedId(payload.id, medicineIds), payload.data)
      return
    }
    case 'medicine.delete': {
      const payload = operation.payload as MedicineDeletePayload
      await deleteMedicine(mappedId(payload.id, medicineIds))
      return
    }
    case 'reminder.create': {
      const payload = operation.payload as ReminderCreatePayload
      const reminder = await createReminder({
        ...payload.data,
        medicine_id: mappedId(payload.data.medicine_id, medicineIds),
      })
      reminderIds.set(payload.clientId, reminder.id)
      return
    }
    case 'reminder.update': {
      const payload = operation.payload as ReminderUpdatePayload
      await updateReminder(mappedId(payload.id, reminderIds), payload.data)
      return
    }
    case 'reminder.delete': {
      const payload = operation.payload as ReminderDeletePayload
      await deleteReminder(mappedId(payload.id, reminderIds))
      return
    }
    case 'confirmation.confirm': {
      const payload = operation.payload as ConfirmationPayload
      await confirmDose(payload.id)
      return
    }
  }
}
