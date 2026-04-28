# HealthVision

## Backend

Gin + GORM backend scaffold with JWT authentication.

### Run

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/server
```

The server listens on `http://localhost:8080` by default.

### Endpoints

- `GET /healthz`
- `POST /api/v1/users`
- `POST /api/v1/sessions`
- `GET /api/v1/users/me`

Register and login return a Bearer token. Send it as:

```http
Authorization: Bearer <access_token>
```

### Database

Default config uses SQLite at `backend/data/healthvision.db`.

For MySQL:

```env
DB_DRIVER=mysql
DB_DSN=user:password@tcp(127.0.0.1:3306)/healthvision?charset=utf8mb4&parseTime=True&loc=Local
```
