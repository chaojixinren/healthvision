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
    const { exact_alarm } = await LocalNotifications.checkExactNotificationSetting()
    if (exact_alarm !== 'granted') {
      await LocalNotifications.changeExactNotificationSetting()
    }
  } catch { /* best-effort */ }
}

function matchesRepeat(r: Reminder, date: Date): boolean {
  switch (r.repeat_type) {
    case 'daily':
      return true
    case 'interval': {
      if (!r.interval_days || r.interval_days <= 0) return true
      const created = new Date(r.created_at)
      const daysSince = Math.floor((date.getTime() - created.getTime()) / (1000 * 60 * 60 * 24))
      return daysSince % r.interval_days === 0
    }
    case 'weekly': {
      if (!r.weekdays) return true
      const weekday = date.getDay()
      return r.weekdays.split(',').map((s) => parseInt(s.trim())).includes(weekday)
    }
    default:
      return true
  }
}

export async function scheduleAll(reminders: Reminder[]): Promise<void> {
  const enabled = reminders.filter((r) => r.enabled)
  if (enabled.length === 0) return

  const pending = await LocalNotifications.getPending()

  const now = new Date()
  const notifications: LocalNotificationSchema[] = []

  for (const r of enabled) {
    const [h, m] = r.time.split(':').map(Number)
    for (let day = 0; day < DAYS_AHEAD; day++) {
      const at = new Date(now.getFullYear(), now.getMonth(), now.getDate() + day, h, m)
      if (at <= now) continue
      if (!matchesRepeat(r, at)) continue

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
