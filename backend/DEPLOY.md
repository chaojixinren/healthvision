# HealthVision Backend Deployment

This directory contains a production Docker Compose setup for the backend and MySQL.

## Server setup

Install Docker and the Compose plugin on the cloud server, then copy or pull this repository.

Create production environment variables:

```bash
cd backend
cp .env.production.example .env.production
```

Edit `.env.production` and set real values for:

- `MYSQL_ROOT_PASSWORD`
- `JWT_SECRET`
- `LLM_API_KEY`
- `BACKEND_PORT` if the server should expose a port other than `8080`

The template defaults Docker builds to `GOPROXY=https://goproxy.cn,direct` and
`GOSUMDB=sum.golang.google.cn`, which is usually more reliable on mainland
China cloud servers. Override them in `.env.production` if your server can use
the upstream Go proxy directly.

## Build and start

If you are deploying from the DockerHub image, keep `BACKEND_IMAGE` set to
`chaojixinren/healthvision-backend:latest` in `.env.production`, then run:

```bash
cd backend
docker compose --env-file .env.production -f docker-compose.prod.yml pull backend
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

If you are building directly on the server from source, run:

```bash
cd backend
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Check status:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
curl http://127.0.0.1:${BACKEND_PORT:-8080}/healthz
```

## Upgrade

```bash
git pull
cd backend
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

## Notes

- The production compose file does not publish MySQL to the public network. Only the backend port is exposed.
- MySQL data is stored in the named volume `mysql_data`.
- The backend health check calls `/healthz`.
- The Android app calls the backend from a WebView origin such as `http://localhost`. If a reverse proxy is placed in front of the backend, make sure it forwards `OPTIONS` requests and does not strip CORS headers.
