import { LocalNotifications, type ScheduleOptions, type LocalNotificationSchema, type ActionPerformed } from '@capacitor/local-notifications'
import { Capacitor } from '@capacitor/core'
import type { Reminder } from './api'

const CHANNEL_ID = 'reminders'
const DAYS_AHEAD = 7

export async function requestPermissions(): Promise<void> {
  const result = await LocalNotifications.checkPermissions()
  if (result.display !== 'granted') {
    await LocalNotifications.requestPermissions()
  }
}

export async function ensureExactAlarms(): Promise<void> {
  if (Capacitor.getPlatform() !== 'android') return

  try {
    const { granted } = await LocalNotifications.checkExactNotificationSetting()
    if (!granted) {
      await LocalNotifications.changeExactNotificationSetting()
    }
  } catch { /* best-effort */ }
}

export async function scheduleAll(reminders: Reminder[]): Promise<void> {
  const enabled = reminders.filter((r) => r.enabled)
  if (enabled.length === 0) return

  const pending = await LocalNotifications.getPending()
  const pendingReminderIds = new Set(
    pending.notifications
      .map((n) => n.extra?.reminder_id)
      .filter((id): id is number => typeof id === 'number'),
  )

  const now = new Date()
  const notifications: LocalNotificationSchema[] = []

  for (const r of enabled) {
    const [h, m] = r.time.split(':').map(Number)
    for (let day = 0; day < DAYS_AHEAD; day++) {
      const at = new Date(now.getFullYear(), now.getMonth(), now.getDate() + day, h, m)
      if (at <= now) continue

      notifications.push({
        id: r.id * 1000 + day,
        title: r.medicine_name,
        body: `该服用 ${r.medicine_name} 了`,
        schedule: { at, allowWhileIdle: true },
        extra: { medicine_id: r.medicine_id, reminder_id: r.id },
        channelId: CHANNEL_ID,
        smallIcon: 'ic_launcher',
        sound: 'default',
      })
    }
  }

  if (notifications.length === 0) return

  // Only cancel and reschedule notifications for reminders that changed
  const toCancel = pending.notifications.filter((n) => {
    const rid = n.extra?.reminder_id
    return typeof rid === 'number' && !enabled.some((r) => r.id === rid)
  })
  if (toCancel.length > 0) {
    await LocalNotifications.cancel({ notifications: toCancel })
  }

  const opts: ScheduleOptions = { notifications }
  await LocalNotifications.schedule(opts)
}

let listenersRegistered = false

export function addListeners(onTap: () => void): void {
  if (listenersRegistered) return
  listenersRegistered = true

  LocalNotifications.addListener(
    'localNotificationActionPerformed',
    (_ev: ActionPerformed) => {
      onTap()
    },
  )
}

export async function removeAllListeners(): Promise<void> {
  listenersRegistered = false
  await LocalNotifications.removeAllListeners()
}
