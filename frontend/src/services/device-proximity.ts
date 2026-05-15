import { Capacitor } from '@capacitor/core'
import { LocalNotifications } from '@capacitor/local-notifications'
import { getLatestLocation } from './api'
import { haversine } from './geo'

const ALERT_DISTANCE_METERS = 50
const LOCATION_STALE_MS = 2 * 60_000
const NOTIFICATION_ID = 99999
const NOTIFICATION_CHANNEL = 'device-proximity'

let watcherId: string | null = null
let lastAlertState: 'far' | 'stale' | 'near' | null = null

// Cache for the device location — ESP32 only updates every 30s,
// so there's no point hitting the server on every GPS callback.
let cachedDeviceLoc: { latitude: number; longitude: number; timestamp: string } | null = null
let cachedDeviceLocAt = 0  // when the cache was populated (ms)
const DEVICE_LOC_CACHE_TTL = 60_000  // refresh at most once per minute

/**
 * Start background proximity monitoring (elderly user only).
 * Uses a foreground service on Android so the OS won't kill the process.
 * On web it's a no-op.
 */
export async function startProximityWatch(): Promise<void> {
  stopProximityWatch()

  if (Capacitor.getPlatform() !== 'android') return

  // Lazy-load the native plugin — avoids Vite trying to resolve it in the browser.
  // The dynamic import is guarded by the platform check above, so it only
  // executes on real Android devices where the plugin is available.
  const { BackgroundGeolocation } = await loadBackgroundGeolocation()

  watcherId = await BackgroundGeolocation.addWatcher(
    {
      backgroundMessage: '正在监测您与药箱的距离',
      backgroundTitle: '药箱距离监测',
      requestPermissions: true,
      stale: false,
      distanceFilter: 10,
    },
    (position, error) => {
      if (error) return
      if (!position) return
      performCheck(position.latitude, position.longitude)
    },
  )
}

export async function stopProximityWatch(): Promise<void> {
  if (watcherId !== null) {
    const { BackgroundGeolocation } = await loadBackgroundGeolocation()
    await BackgroundGeolocation.removeWatcher({ id: watcherId })
    watcherId = null
  }
  lastAlertState = null
  cachedDeviceLoc = null
  cachedDeviceLocAt = 0
}

/**
 * Load the BackgroundGeolocation plugin at runtime.
 * Uses a string-based import() so Vite won't statically analyse it.
 */
function loadBackgroundGeolocation(): Promise<typeof import('@capacitor-community/background-geolocation')> {
  // The variable prevents Vite from tracing the import at build time.
  const mod = '@capacitor-community/background-geolocation'
  return import(/* @vite-ignore */ mod)
}

async function performCheck(phoneLat: number, phoneLng: number): Promise<void> {
  try {
    // Use cached device location if still fresh (within TTL)
    const now = Date.now()
    let deviceLoc = cachedDeviceLoc
    if (!deviceLoc || (now - cachedDeviceLocAt) > DEVICE_LOC_CACHE_TTL) {
      deviceLoc = await getLatestLocation()
      cachedDeviceLoc = deviceLoc
      cachedDeviceLocAt = now
    }

    const deviceTime = new Date(deviceLoc.timestamp).getTime()
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

    const dist = haversine(phoneLat, phoneLng, deviceLoc.latitude, deviceLoc.longitude)
    const newState = dist > ALERT_DISTANCE_METERS ? 'far' : 'near'

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
    // Silently ignore
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
