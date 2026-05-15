import { Geolocation, type Position } from '@capacitor/geolocation'
import { Capacitor } from '@capacitor/core'
import { LocalNotifications } from '@capacitor/local-notifications'
import { getLatestLocation } from './api'
import { haversine } from './geo'

const CHECK_INTERVAL_MS = 60_000       // 60 seconds
const ALERT_DISTANCE_METERS = 50
const LOCATION_STALE_MS = 2 * 60_000   // 2 minutes — same as backend LocationExpiry
const NOTIFICATION_ID = 99999          // fixed ID so we don't stack duplicates
const NOTIFICATION_CHANNEL = 'device-proximity'

let timer: ReturnType<typeof setInterval> | null = null
let lastAlertState: 'far' | 'stale' | 'near' | null = null

/**
 * Start periodic proximity checks (elderly user only).
 * Works on Android via Capacitor; on web it's a no-op.
 */
export async function startProximityWatch(): Promise<void> {
  stopProximityWatch()

  if (Capacitor.getPlatform() !== 'android') return

  // Request location permission first
  const perm = await Geolocation.checkPermissions()
  if (perm.location === 'prompt' || perm.location === 'prompt-with-rationale') {
    const req = await Geolocation.requestPermissions()
    if (req.location !== 'granted') return
  }
  if (perm.location !== 'granted') return

  // Run first check immediately, then periodically
  await performCheck()
  timer = setInterval(performCheck, CHECK_INTERVAL_MS)
}

export function stopProximityWatch(): void {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
  lastAlertState = null
}

async function performCheck(): Promise<void> {
  try {
    const position: Position = await Geolocation.getCurrentPosition({
      enableHighAccuracy: true,
      timeout: 10000,
    })

    const phoneLat = position.coords.latitude
    const phoneLng = position.coords.longitude

    // Fetch device (pillbox) location from server
    const deviceLoc = await getLatestLocation()

    // Check staleness — device location too old means ESP32 is offline
    const deviceTime = new Date(deviceLoc.timestamp).getTime()
    const now = Date.now()
    if (now - deviceTime > LOCATION_STALE_MS) {
      const newState = 'stale'
      if (newState !== lastAlertState) {
        await sendNotification(
          '药箱设备离线',
          '无法获取药箱位置，请检查设备是否开机',
        )
        lastAlertState = newState
      }
      return
    }

    // Calculate distance locally (Haversine)
    const dist = haversine(phoneLat, phoneLng, deviceLoc.latitude, deviceLoc.longitude)
    const newState = dist > ALERT_DISTANCE_METERS ? 'far' : 'near'

    // Only notify on state transitions to avoid spamming
    if (newState !== lastAlertState) {
      if (newState === 'far') {
        await sendNotification(
          '请携带药箱',
          `您与药箱距离约 ${Math.round(dist)} 米，请记得带上药箱`,
        )
      } else if (newState === 'near') {
        await sendNotification(
          '已靠近药箱',
          `您与药箱距离约 ${Math.round(dist)} 米`,
        )
      }
      lastAlertState = newState
    }
  } catch {
    // Silently ignore — GPS might be unavailable, network down, etc.
  }
}

async function sendNotification(title: string, body: string): Promise<void> {
  try {
    await LocalNotifications.schedule({
      notifications: [
        {
          id: NOTIFICATION_ID,
          title,
          body,
          channelId: NOTIFICATION_CHANNEL,
          smallIcon: 'ic_launcher',
          sound: 'default',
        },
      ],
    })
  } catch {
    // Best-effort
  }
}
