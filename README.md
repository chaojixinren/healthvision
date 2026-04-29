# HealthVision

用药提醒平台。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go · Gin · GORM · JWT |
| 数据库 | MySQL |
| 前端 | Vue 3 · Vite · TypeScript · Vue Router |

## 项目结构

```
healthvision/
├── backend/
│   ├── cmd/server/main.go          # 入口
│   ├── internal/
│   │   ├── config/                 # 配置
│   │   ├── database/               # 数据库连接 & 迁移
│   │   ├── handlers/               # 请求处理
│   │   ├── httputil/               # HTTP 错误工具
│   │   ├── middleware/             # JWT 鉴权中间件
│   │   ├── models/                 # GORM 模型
│   │   ├── repository/             # 数据访问层
│   │   ├── router/                 # 路由注册
│   │   └── services/               # 业务逻辑层
│   ├── go.mod / go.sum
│   └── .env.example
├── frontend/
│   └── src/
│       ├── views/                  # 页面组件
│       │   ├── Home.vue            # Landing 页
│       │   ├── Login.vue           # 登录
│       │   ├── Register.vue        # 注册
│       │   ├── Dashboard.vue       # 仪表盘
│       │   ├── Medicines.vue       # 药品管理
│       │   └── Profile.vue         # 个人中心
│       ├── router/                 # 前端路由
│       ├── services/               # API 封装
│       └── App.vue                 # 根组件
└── docs/
    └── 设计图.jpg
```

## 快速开始

### 后端

```bash
cd backend
cp .env.example .env   # 按需修改数据库和 JWT 配置
go mod tidy
go run ./cmd/server
```

服务默认监听 `http://localhost:8080`。

### 前端

```bash
cd frontend
npm install
npm run dev
```

开发服务器默认监听 `http://localhost:5173`。

## 环境变量

| 变量 | 说明 | 默认值 |
|---|---|---|
| `APP_ENV` | 运行环境 | `development` |
| `PORT` | 后端端口 | `8080` |
| `DB_DRIVER` | 数据库驱动 | `sqlite` |
| `DB_DSN` | 数据库连接串 | — |
| `JWT_SECRET` | JWT 签名密钥 | — |
| `JWT_ISSUER` | JWT 签发者 | `healthvision` |
| `ACCESS_TOKEN_TTL` | Token 有效期 | `24h` |

### MySQL 配置示例

```env
DB_DRIVER=mysql
DB_DSN=user:password@tcp(127.0.0.1:3306)/healthvision?charset=utf8mb4&parseTime=True&loc=Local
```

## API 接口

### 公开

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/healthz` | 健康检查 |
| `POST` | `/api/v1/users` | 注册 |
| `POST` | `/api/v1/sessions` | 登录 |

注册和登录返回 Bearer token，后续请求通过 Header 携带：

```http
Authorization: Bearer <access_token>
```

### 需认证（JWT）

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/v1/users/me` | 获取当前用户 |
| `POST` | `/api/v1/medicines` | 新增药品 |
| `GET` | `/api/v1/medicines` | 药品列表（分页） |
| `GET` | `/api/v1/medicines/:id` | 药品详情 |
| `PUT` | `/api/v1/medicines/:id` | 更新药品 |
| `DELETE` | `/api/v1/medicines/:id` | 删除药品 |
