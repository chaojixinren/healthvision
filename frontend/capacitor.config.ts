import { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.healthvision.app',
  appName: 'HealthVision',
  webDir: 'dist',
  server: {
    // 使用 http scheme：后端 API 是 http://，如果用 https scheme 会导致
    // 混合内容策略拦截所有 API 请求。离线检测问题通过 probeBackend() 修正。
    androidScheme: 'http',
    // 开发时开启，指向 Vite dev server 的局域网地址
    // url: 'http://192.168.x.x:5173',
  },
}

export default config
