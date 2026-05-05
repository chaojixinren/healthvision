import { LocalNotifications, type ScheduleOptions, type LocalNotificationSchema, type ActionPerformed } from '@capacitor/local-notifications'
import type { Reminder } from './api'

const CHANNEL_ID = 'reminders'
const DAYS_AHEAD = 7

export async function requestPermissions(): Promise<void> {
  const result = await LocalNotifications.checkPermissions()
  if (result.display !== 'granted') {
    await LocalNotifications.requestPermissions()
  }
}

export async function scheduleAll(reminders: Reminder[]): Promise<void> {
  const pending = await LocalNotifications.getPending()
  if (pending.notifications.length > 0) {
    await LocalNotifications.cancel({ notifications: pending.notifications })
  }

  const enabled = reminders.filter((r) => r.enabled)
  if (enabled.length === 0) return

  const now = new Date()
  const notifications: LocalNotificationSchema[] = []
  let id = 1

  for (const r of enabled) {
    const [h, m] = r.time.split(':').map(Number)
    for (let day = 0; day < DAYS_AHEAD; day++) {
      const at = new Date(now.getFullYear(), now.getMonth(), now.getDate() + day, h, m)
      if (at <= now) continue

      notifications.push({
        id,
        title: r.medicine_name,
        body: `该服用 ${r.medicine_name} 了`,
        schedule: { at },
        extra: { medicine_id: r.medicine_id, reminder_id: r.id },
        channelId: CHANNEL_ID,
        smallIcon: 'ic_launcher',
        sound: 'default',
      })
      id++
    }
  }

  if (notifications.length === 0) return

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
