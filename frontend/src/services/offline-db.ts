import type { Binding, Confirmation, Medicine, Reminder } from './api'

const DB_NAME = 'healthvision-offline'
const DB_VERSION = 1
const CARE_STORE = 'care-cache'
const OUTBOX_STORE = 'outbox'

export interface CareCacheData {
  medicines?: Medicine[]
  reminders?: Reminder[]
  bindings?: Binding[]
  confirmations?: Confirmation[]
}

export interface CareCacheRecord extends CareCacheData {
  key: string
  userId: number
  updatedAt: string
}

export type OutboxOperationType =
  | 'medicine.create'
  | 'medicine.update'
  | 'medicine.delete'
  | 'reminder.create'
  | 'reminder.update'
  | 'reminder.delete'
  | 'confirmation.confirm'

export interface OutboxOperation<TPayload = unknown> {
  id: string
  userId: number
  type: OutboxOperationType
  payload: TPayload
  createdAt: string
  retryCount: number
  lastError?: string
}

type EntityPayload = { id?: number; clientId?: number; data?: { medicine_id?: number } }
type CompactionResult = {
  operations: OutboxOperation[]
  enqueue: boolean
}

function careKey(userId: number): string {
  return `user:${userId}:care`
}

function toPlain<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)

    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(CARE_STORE)) {
        db.createObjectStore(CARE_STORE, { keyPath: 'key' })
      }
      if (!db.objectStoreNames.contains(OUTBOX_STORE)) {
        db.createObjectStore(OUTBOX_STORE, { keyPath: 'id' })
      }
    }

    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('无法打开离线缓存'))
  })
}

function withStore<T>(
  mode: IDBTransactionMode,
  work: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  return openDB().then((db) => new Promise<T>((resolve, reject) => {
    const tx = db.transaction(CARE_STORE, mode)
    const store = tx.objectStore(CARE_STORE)
    const request = work(store)

    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('离线缓存操作失败'))
    tx.oncomplete = () => db.close()
    tx.onerror = () => {
      db.close()
      reject(tx.error ?? new Error('离线缓存事务失败'))
    }
  }))
}

function withNamedStore<T>(
  storeName: string,
  mode: IDBTransactionMode,
  work: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  return openDB().then((db) => new Promise<T>((resolve, reject) => {
    const tx = db.transaction(storeName, mode)
    const store = tx.objectStore(storeName)
    const request = work(store)

    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('离线缓存操作失败'))
    tx.oncomplete = () => db.close()
    tx.onerror = () => {
      db.close()
      reject(tx.error ?? new Error('离线缓存事务失败'))
    }
  }))
}

export async function readCareCache(userId: number): Promise<CareCacheRecord | null> {
  const record = await withStore<CareCacheRecord | undefined>('readonly', (store) => (
    store.get(careKey(userId)) as IDBRequest<CareCacheRecord | undefined>
  ))
  return record ?? null
}

export async function writeCareCache(userId: number, data: CareCacheData): Promise<CareCacheRecord> {
  const existing = await readCareCache(userId)
  const record: CareCacheRecord = {
    key: careKey(userId),
    userId,
    medicines: toPlain(data.medicines ?? existing?.medicines ?? []),
    reminders: toPlain(data.reminders ?? existing?.reminders ?? []),
    bindings: toPlain(data.bindings ?? existing?.bindings ?? []),
    confirmations: toPlain(data.confirmations ?? existing?.confirmations ?? []),
    updatedAt: new Date().toISOString(),
  }

  await withStore<IDBValidKey>('readwrite', (store) => store.put(record))
  return record
}

export async function clearCareCache(userId: number): Promise<void> {
  await withStore<undefined>('readwrite', (store) => (
    store.delete(careKey(userId)) as IDBRequest<undefined>
  ))
}

export async function enqueueOutboxOperation<TPayload>(
  userId: number,
  type: OutboxOperationType,
  payload: TPayload,
): Promise<OutboxOperation<TPayload>> {
  const operations = await listOutboxOperations(userId)
  const compacted = compactOutbox(operations, type, toPlain(payload) as EntityPayload)
  await replaceOutboxOperations(userId, compacted.operations)

  if (!compacted.enqueue) {
    const existing = compacted.operations.find((operation) => sameTarget(operation, operation.type, payload as EntityPayload))
    if (existing) return existing as OutboxOperation<TPayload>
    return {
      id: '',
      userId,
      type,
      payload: toPlain(payload),
      createdAt: new Date().toISOString(),
      retryCount: 0,
    }
  }

  const operation: OutboxOperation<TPayload> = {
    id: `${Date.now()}-${crypto.randomUUID()}`,
    userId,
    type,
    payload: toPlain(payload),
    createdAt: new Date().toISOString(),
    retryCount: 0,
  }
  await withNamedStore<IDBValidKey>('outbox', 'readwrite', (store) => store.put(operation))
  return operation
}

function targetId(payload: EntityPayload): number {
  return payload.id ?? payload.clientId ?? 0
}

function sameTarget(operation: OutboxOperation, type: OutboxOperationType, payload: EntityPayload): boolean {
  if (operation.type !== type) return false
  return targetId(operation.payload as EntityPayload) === targetId(payload)
}

function compactOutbox(
  operations: OutboxOperation[],
  type: OutboxOperationType,
  payload: EntityPayload,
): CompactionResult {
  const id = targetId(payload)
  if (!id) return { operations, enqueue: true }

  if (type === 'medicine.update') {
    const create = operations.find((operation) => (
      operation.type === 'medicine.create' && targetId(operation.payload as EntityPayload) === id
    ))
    if (create) {
      create.payload = {
        ...(create.payload as { clientId: number; data: unknown }),
        data: (payload as { data: unknown }).data,
      }
      return {
        operations: operations.filter((operation) => !(
          operation.type === 'medicine.update' && targetId(operation.payload as EntityPayload) === id
        )),
        enqueue: false,
      }
    }
    return {
      operations: operations.filter((operation) => !(
        operation.type === 'medicine.update' && targetId(operation.payload as EntityPayload) === id
      )),
      enqueue: true,
    }
  }

  if (type === 'medicine.delete') {
    const hasCreate = operations.some((operation) => (
      operation.type === 'medicine.create' && targetId(operation.payload as EntityPayload) === id
    ))
    const withoutMedicineOps = operations.filter((operation) => {
      if (operation.type.startsWith('medicine.') && targetId(operation.payload as EntityPayload) === id) {
        return false
      }
      if (operation.type.startsWith('reminder.')) {
        return (operation.payload as EntityPayload).data?.medicine_id !== id
      }
      return true
    })
    return { operations: withoutMedicineOps, enqueue: !hasCreate }
  }

  if (type === 'reminder.update') {
    const create = operations.find((operation) => (
      operation.type === 'reminder.create' && targetId(operation.payload as EntityPayload) === id
    ))
    if (create) {
      create.payload = {
        ...(create.payload as { clientId: number; data: Record<string, unknown> }),
        data: {
          ...((create.payload as { data: Record<string, unknown> }).data),
          ...((payload as { data: Record<string, unknown> }).data),
        },
      }
      return {
        operations: operations.filter((operation) => !(
          operation.type === 'reminder.update' && targetId(operation.payload as EntityPayload) === id
        )),
        enqueue: false,
      }
    }
    return {
      operations: operations.filter((operation) => !(
        operation.type === 'reminder.update' && targetId(operation.payload as EntityPayload) === id
      )),
      enqueue: true,
    }
  }

  if (type === 'reminder.delete') {
    const hasCreate = operations.some((operation) => (
      operation.type === 'reminder.create' && targetId(operation.payload as EntityPayload) === id
    ))
    const withoutReminderOps = operations.filter((operation) => !(
      operation.type.startsWith('reminder.') && targetId(operation.payload as EntityPayload) === id
    ))
    return { operations: withoutReminderOps, enqueue: !hasCreate }
  }

  if (type === 'confirmation.confirm') {
    return {
      operations: operations.filter((operation) => !(
        operation.type === 'confirmation.confirm' && targetId(operation.payload as EntityPayload) === id
      )),
      enqueue: true,
    }
  }

  return { operations, enqueue: true }
}

async function replaceOutboxOperations(userId: number, operations: OutboxOperation[]): Promise<void> {
  const current = await listOutboxOperations(userId)
  await Promise.all(current.map((operation) => deleteOutboxOperation(operation.id)))
  await Promise.all(operations.map((operation) => (
    withNamedStore<IDBValidKey>('outbox', 'readwrite', (store) => store.put(toPlain(operation)))
  )))
}

export async function listOutboxOperations(userId: number): Promise<OutboxOperation[]> {
  const operations = await withNamedStore<OutboxOperation[]>('outbox', 'readonly', (store) => (
    store.getAll() as IDBRequest<OutboxOperation[]>
  ))
  return operations
    .filter((operation) => operation.userId === userId)
    .sort((a, b) => a.createdAt.localeCompare(b.createdAt))
}

export async function deleteOutboxOperation(id: string): Promise<void> {
  await withNamedStore<undefined>('outbox', 'readwrite', (store) => (
    store.delete(id) as IDBRequest<undefined>
  ))
}

export async function markOutboxOperationFailed(id: string, error: string): Promise<void> {
  const operation = await withNamedStore<OutboxOperation | undefined>('outbox', 'readonly', (store) => (
    store.get(id) as IDBRequest<OutboxOperation | undefined>
  ))
  if (!operation) return

  await withNamedStore<IDBValidKey>('outbox', 'readwrite', (store) => store.put({
    ...operation,
    retryCount: operation.retryCount + 1,
    lastError: error,
  }))
}

export async function countOutboxOperations(userId: number): Promise<number> {
  return listOutboxOperations(userId).then((operations) => operations.length)
}

export async function clearOutboxOperations(userId: number): Promise<void> {
  const operations = await listOutboxOperations(userId)
  await Promise.all(operations.map((operation) => deleteOutboxOperation(operation.id)))
}
