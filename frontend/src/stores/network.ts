import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { PluginListenerHandle } from '@capacitor/core'
import { Network, type ConnectionStatus, type ConnectionType } from '@capacitor/network'
import { Capacitor } from '@capacitor/core'

function browserStatus(): ConnectionStatus {
  const connected = navigator.onLine
  return {
    connected,
    connectionType: connected ? 'unknown' : 'none',
  }
}

/**
 * 通过实际发请求检测后端是否可达，
 * 避免 Capacitor Network 插件报告"有网络"但后端不可达的误判
 */
async function probeBackend(): Promise<boolean> {
  try {
    const baseUrl = import.meta.env.VITE_API_URL || '/api/v1'
    // 只做 HEAD 请求检测连通性，不消耗带宽
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 5000)
    const res = await fetch(`${baseUrl}/../healthz`, {
      method: 'HEAD',
      cache: 'no-store',
      signal: controller.signal,
    })
    clearTimeout(timer)
    return res.ok || res.status < 500
  } catch {
    return false
  }
}

export const useNetworkStore = defineStore('network', () => {
  const connected = ref(true)
  const connectionType = ref<ConnectionType>('unknown')
  const initialized = ref(false)
  let listener: PluginListenerHandle | null = null

  function applyStatus(status: ConnectionStatus) {
    connected.value = status.connected
    connectionType.value = status.connectionType
  }

  async function init() {
    if (initialized.value) return
    initialized.value = true

    if (Capacitor.isNativePlatform()) {
      // 原生平台：优先用 Capacitor Network 插件
      try {
        const status = await Network.getStatus()
        applyStatus(status)

        // 如果插件报告离线，但实际可能是误判，做一次后端探测
        if (!status.connected) {
          const reachable = await probeBackend()
          if (reachable) {
            connected.value = true
            connectionType.value = 'unknown'
          }
        }

        listener = await Network.addListener('networkStatusChange', async (newStatus) => {
          applyStatus(newStatus)
          // 网络恢复时也做后端探测，避免"WiFi已连但后端不可达"的假阳性
          if (newStatus.connected) {
            const reachable = await probeBackend()
            if (!reachable) {
              // 有网络但后端不通，仍标记为离线
              connected.value = false
            }
          }
        })
      } catch {
        // Capacitor 桥未就绪，回退到浏览器 API
        applyStatus(browserStatus())
        window.addEventListener('online', handleBrowserChange)
        window.addEventListener('offline', handleBrowserChange)
      }
    } else {
      // 浏览器/Web 平台
      applyStatus(browserStatus())
      window.addEventListener('online', handleBrowserChange)
      window.addEventListener('offline', handleBrowserChange)
    }
  }

  function handleBrowserChange() {
    applyStatus(browserStatus())
  }

  async function dispose() {
    if (listener) {
      await listener.remove()
      listener = null
    }
    window.removeEventListener('online', handleBrowserChange)
    window.removeEventListener('offline', handleBrowserChange)
    initialized.value = false
  }

  return {
    connected,
    connectionType,
    initialized,
    init,
    dispose,
  }
})
