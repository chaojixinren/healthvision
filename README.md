# HealthVision

用药提醒平台，支持老人/子女账户绑定、跨用户用药提醒、AI 健康助手和智能药箱距离提醒。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go · Gin · GORM · JWT · CORS · Google ADK |
| 数据库 | MySQL |
| 前端 | Vue 3 · Vite · TypeScript · Pinia · Vue Router |
| 移动端 | Capacitor 8 · Android |
| 硬件 | ESP32 · GPS 模块（NEO-6M/7M/8M） |
| 部署 | Docker · Docker Hub |

## 项目结构

```
healthvision/
├── backend/
│   ├── cmd/server/main.go          # 入口
│   ├── internal/
│   │   ├── agent/                  # ADK agent 工厂 & 指令
│   │   │   ├── model/              # LLM 适配层（OpenAI 兼容）
│   │   │   └── tools/              # agent 工具（药品/提醒 CRUD）
│   │   ├── config/                 # 配置（从环境变量加载）
│   │   ├── database/               # 数据库连接 & 自动迁移
│   │   ├── handlers/               # 请求处理
│   │   ├── middleware/             # JWT 鉴权 + CORS 中间件
│   │   ├── models/                 # GORM 模型
│   │   ├── repository/             # 数据访问层
│   │   ├── router/                 # 路由注册
│   │   └── services/               # 业务逻辑层
│   ├── Dockerfile                  # 多阶段构建
│   ├── docker-compose.yml          # MySQL + Backend 一键部署
│   ├── go.mod / go.sum
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── views/                  # 页面组件
│   │   ├── stores/                 # Pinia 状态管理
│   │   ├── services/               # API 封装 + 距离检测
│   │   ├── router/                 # 前端路由
│   │   └── App.vue                 # 根组件
│   ├── android/                    # Capacitor Android 项目
│   ├── capacitor.config.ts         # Capacitor 配置
│   └── vite.config.ts              # Vite 配置（含 API 代理）
└── docs/
```

## 下载

[![GitHub Release](https://img.shields.io/github/v/release/chaojixinren/healthvision?label=latest)](https://github.com/chaojixinren/healthvision/releases/latest)

Android APK 请前往 [Releases](https://github.com/chaojixinren/healthvision/releases) 页面下载最新版本。

## 快速开始

### 后端（本地开发）

```bash
cd backend
cp .env.example .env   # 按需修改数据库和 JWT 配置
go mod tidy
go run ./cmd/server     # 监听 http://localhost:8080
```

### 前端（浏览器开发）

```bash
cd frontend
npm install
npm run dev             # 监听 http://localhost:5173，API 代理到 8080
```

### Docker 部署（云服务器）

```bash
# 使用 docker-compose（含 MySQL）
cd backend
cp .env.example .env.production   # 填入真实配置
docker compose --env-file .env.production up -d

# 或单独拉取后端镜像
docker pull chaojixinren/healthvision-backend:latest
docker run -d --name healthvision -p 8080:8080 \
  -e DB_DSN="user:pass@tcp(mysql:3306)/healthvision?charset=utf8mb4&parseTime=True&loc=Local" \
  -e JWT_SECRET="..." \
  -e LLM_API_KEY="..." \
  chaojixinren/healthvision-backend:latest
```

### Android APK 构建

```bash
cd frontend

# 1. 构建前端
VITE_API_URL=http://你的服务器IP:8080/api/v1 npm run build

# 2. 同步到 Android 项目
npx cap sync

# 3. 编译 APK
JAVA_HOME="/path/to/jdk" ANDROID_HOME="$HOME/Library/Android/sdk" \
  cd android && ./gradlew assembleDebug

# APK 输出位置：android/app/build/outputs/apk/debug/app-debug.apk

# 或用 Android Studio 打开
npx cap open android
```

> 生产环境 `VITE_API_URL` 已配置在 `frontend/.env.production`，直接 `npm run build` 即可。

## 环境变量

| 变量 | 说明 | 默认值 |
|---|---|---|
| `APP_ENV` | 运行环境（production 时强制检查 JWT_SECRET） | `development` |
| `PORT` | 后端端口 | `8080` |
| `DB_DRIVER` | 数据库驱动 | `mysql` |
| `DB_DSN` | 数据库连接串 | — |
| `JWT_SECRET` | JWT 签名密钥 | — |
| `JWT_ISSUER` | JWT 签发者 | `healthvision` |
| `ACCESS_TOKEN_TTL` | Token 有效期 | `24h` |
| `REFRESH_TOKEN_TTL` | 刷新令牌有效期 | `720h` |
| `MAX_SESSIONS_PER_USER` | 同一用户最大并发会话数（超出后废除最旧会话） | `5` |
| `LLM_MODEL` | LLM 模型名称 | `gpt-4o-mini` |
| `LLM_BASE_URL` | LLM API 地址 | `https://api.openai.com/v1` |
| `LLM_API_KEY` | LLM API 密钥 | — |
| `AGENT_REQUIRE_WRITE_TOOL_CONFIRMATION` | 写操作是否需要用户人工确认（`true`/`false`） | `true` |
| `CHAT_RETENTION_DAYS` | AI 会话消息保留天数，`0` 表示不按时间清理 | `30` |
| `CHAT_MAX_MESSAGES_PER_USER` | 单用户最多保留的 AI 会话消息数，`0` 表示不按数量清理 | `2000` |

## AI 助手

AI 对话基于 Google ADK（Agent Development Kit），模型通过 runner 驱动，自动调用工具读取或修改用户数据。

- **工具**：9 个，覆盖药品和提醒的 CRUD
- **写保护**：新增/修改/删除操作需要用户在前端弹窗确认（HITL），生产环境默认启用
- **动态指令**：每次对话自动注入当前用户的药品库和提醒快照，减少工具调用次数
- **流式输出**：通过 SSE 推送 token，支持中途下发确认弹窗
- **图片识别**：支持上传图片，由视觉模型分析
- **历史清理**：每天自动清理过期 AI 会话消息，并支持用户手动清空历史

## API 接口

### 公开

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/healthz` | 健康检查 |
| `POST` | `/api/v1/users` | 注册 |
| `POST` | `/api/v1/sessions` | 登录 |
| `POST` | `/api/v1/sessions/refresh` | 刷新登录令牌 |
| `DELETE` | `/api/v1/sessions` | 退出登录并撤销刷新令牌 |

### 需认证（JWT）

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/v1/users/me` | 当前用户信息 |
| `PUT` | `/api/v1/users/me/identity` | 切换老人/子女身份 |
| `GET` | `/api/v1/users/search?q=` | 搜索用户 |
| `POST` | `/api/v1/medicines` | 新增药品 |
| `GET` | `/api/v1/medicines` | 药品列表（分页） |
| `GET` | `/api/v1/medicines/:id` | 药品详情 |
| `PUT` | `/api/v1/medicines/:id` | 更新药品 |
| `DELETE` | `/api/v1/medicines/:id` | 删除药品 |
| `POST` | `/api/v1/reminders` | 新增用药提醒 |
| `GET` | `/api/v1/reminders` | 提醒列表 |
| `GET` | `/api/v1/reminders/:id` | 提醒详情 |
| `PUT` | `/api/v1/reminders/:id` | 更新提醒 |
| `DELETE` | `/api/v1/reminders/:id` | 删除提醒 |
| `POST` | `/api/v1/chat/send` | 发送聊天消息（SSE 流式，支持图片 & 工具确认） |
| `GET` | `/api/v1/chat/conversations` | 会话列表 |
| `POST` | `/api/v1/chat/messages` | 获取会话消息 |
| `POST` | `/api/v1/chat/delete` | 删除会话 |
| `POST` | `/api/v1/chat/clear` | 清空当前用户所有 AI 会话 |
| `POST` | `/api/v1/confirmations/generate` | 生成今日确认记录 |
| `GET` | `/api/v1/confirmations` | 确认记录列表 |
| `POST` | `/api/v1/confirmations/:id/confirm` | 确认服药 |
| `POST` | `/api/v1/bindings` | 创建账户绑定 |
| `GET` | `/api/v1/bindings` | 绑定列表 |
| `PUT` | `/api/v1/bindings/:id` | 接受/拒绝绑定 |
| `DELETE` | `/api/v1/bindings/:id` | 删除绑定 |
| `POST` | `/api/v1/locations` | 上报设备位置（ESP32 药箱） |
| `GET` | `/api/v1/locations/latest` | 获取自己最新设备位置 |

所有需认证的接口需携带 Header：

```http
Authorization: Bearer ***
```

## 智能药箱

通过 ESP32 + GPS 模块实现药箱位置追踪，当老人与药箱距离超过 50 米时自动提醒携带。

### 工作原理

```
ESP32 药箱                        服务器                         手机 App
   │                                │                               │
   │── POST /locations ────────────>│                               │
   │   (GPS坐标 + JWT)              │── 存储药箱位置                 │
   │                                │                               │
   │                                │<── GET /locations/latest ─────│
   │                                │    (返回药箱坐标)              │
   │                                │                               │
   │                                │            本地 Haversine 比较 │
   │                                │            手机坐标 vs 药箱坐标 │
   │                                │                               │
   │                                │            距离 > 50m ───────>│ 通知：请携带药箱
```

- ESP32 每 30 秒上报一次 GPS 位置
- 手机 App 每 60 秒获取药箱坐标，本地计算距离
- 距离超过 50 米触发本地通知提醒
- 超过 2 分钟未上报显示药箱离线

### ESP32 固件

ESP32 固件代码独立维护（不包含在本仓库中），功能：

- WiFi 连接 + 串口配置
- GPS 定位（NEO-6M/7M/8M 模块）
- 用户登录获取 JWT（与 App 共用同一账号）
- 定时上报位置到后端
- 同步服务端提醒并在 OLED 屏幕显示
- 蜂鸣器提醒服药

### 前端药箱状态

老人端"个人中心"页面展示：

- 药箱连接状态（在线/离线）
- 最后上报时间
- 药箱与手机的距离
- 连接说明指引
