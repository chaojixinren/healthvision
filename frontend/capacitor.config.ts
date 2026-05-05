import { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.healthvision.app',
  appName: 'HealthVision',
  webDir: 'dist',
  server: {
    androidScheme: 'http',
    // 开发时开启，指向 Vite dev server 的局域网地址
    // url: 'http://192.168.x.x:5173',
    // cleartext: true,
  },
}

export default config
