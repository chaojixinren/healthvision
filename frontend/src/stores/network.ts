import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { PluginListenerHandle } from '@capacitor/core'
import { Network, type ConnectionStatus, type ConnectionType } from '@capacitor/network'

function browserStatus(): ConnectionStatus {
  const connected = navigator.onLine
  return {
    connected,
    connectionType: connected ? 'unknown' : 'none',
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

    try {
      applyStatus(await Network.getStatus())
      listener = await Network.addListener('networkStatusChange', applyStatus)
    } catch {
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
