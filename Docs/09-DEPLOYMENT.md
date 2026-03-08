# Deployment Strategy - CBT Patra

**Tanggal:** 2026-03-05
**Diperbarui:** 2026-03-05 (v2 - single project structure)

---

## Infrastructure Overview

### Development (Local)
```
Docker Compose:
  - cbt-backend (Golang binary, port 8080)
  - cbt-frontend (Vite dev server, port 5173)
  - postgres (port 5432)
  - redis (port 6379)
  - minio (port 9000/9001)
```

### Production
```
VPS (Ubuntu 22.04, min 2 vCPU, 4GB RAM):
  - Caddy (reverse proxy, HTTPS auto via Let's Encrypt)
  - cbt-backend (Golang binary, systemd service)
  - cbt-frontend (Nginx serving static files, atau served by Caddy)
  - PostgreSQL 16
  - Redis 7
  - MinIO (optional, bisa pakai S3)

Alternatif: Docker Compose pada VPS (untuk simplisitas)
```

---

## Docker Setup

### Dockerfile (Single Project — Go + Vue.js)

```dockerfile
# Build stage 1: Build Vue.js
FROM node:20-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build
# Output: /app/web/dist/

# Build stage 2: Build Golang binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy Vue build output ke dalam Go project (untuk embed.FS)
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-w -s' -o cbt-patra ./cmd/server
# Binary sudah include static Vue files via //go:embed

# Production stage: minimal image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/cbt-patra .
COPY --from=go-builder /app/migrations ./migrations
EXPOSE 8080
CMD ["./cbt-patra"]
```

### docker-compose.yml (Production — Single Service)
```yaml
version: '3.9'

services:
  app:
    image: cbt-patra:latest        # Single image: Go binary + Vue embedded
    restart: unless-stopped
    environment:
      - APP_ENV=production
    env_file: .env
    ports:
      - "8080:8080"                # Semua traffic: API + Vue SPA
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=${DB_DATABASE}
      - POSTGRES_USER=${DB_USERNAME}
      - POSTGRES_PASSWORD=${DB_PASSWORD}

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    ports:
      - "9000:9000"
      - "9001:9001"

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

---

## Caddy Configuration (Reverse Proxy)
```
cbt.domain.com {
    # Frontend
    root * /var/www/cbt-frontend
    file_server
    try_files {path} /index.html

    # Backend API
    handle /api/* {
        reverse_proxy localhost:8080
    }

    # WebSocket
    handle /ws/* {
        reverse_proxy localhost:8080
    }

    encode gzip
    log
}
```

---

## Environment Variables

```env
# App
APP_ENV=production
APP_PORT=8080
APP_SECRET=superrandom256bitsecret

# JWT
JWT_ACCESS_SECRET=accesssecret
JWT_REFRESH_SECRET=refreshsecret
JWT_ACCESS_TTL=900          # 15 menit (detik)
JWT_REFRESH_TTL=604800      # 7 hari (detik)

# Database
DB_HOST=postgres
DB_PORT=5432
DB_DATABASE=cbt_patra
DB_USERNAME=cbt_user
DB_PASSWORD=strongpassword
DB_MAX_OPEN=25
DB_MAX_IDLE=5

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioaccess
MINIO_SECRET_KEY=miniosecret
MINIO_BUCKET=cbt-patra
MINIO_USE_SSL=false

# AI (Optional)
AI_API_URL=
AI_API_KEY=
AI_API_HEADER=Authorization

# Frontend URL (CORS)
ALLOWED_ORIGINS=https://cbt.domain.com
```

---

## Database Migration Strategy

Menggunakan golang-migrate atau GORM AutoMigrate:

```
Option A: GORM AutoMigrate (simple, dev-friendly)
  - Run AutoMigrate saat startup
  - GORM tambah kolom baru, tidak hapus kolom lama

Option B: golang-migrate (production-recommended)
  - SQL migration files dengan versi
  - Up/Down migration
  - Riwayat migration di tabel schema_migrations
  - Command: ./cbt-patra migrate up
```

---

## Deployment Process

### Initial Setup
```bash
# 1. Clone repo
git clone https://github.com/yourname/cbt-patra.git
cd cbt-patra

# 2. Setup .env
cp .env.example .env
nano .env  # isi semua konfigurasi

# 3. Build dan jalankan
docker compose up -d --build

# 4. Run migration
docker compose exec backend ./cbt-patra migrate up

# 5. Seed data awal (admin user, settings)
docker compose exec backend ./cbt-patra seed
```

### Update/Deploy Ulang
```bash
git pull
docker compose build backend frontend
docker compose up -d backend frontend
docker compose exec backend ./cbt-patra migrate up
```

---

## Backup Strategy

### Database Backup (cron)
```bash
# /etc/cron.d/cbt-backup
0 2 * * * docker compose exec postgres pg_dump -U cbt_user cbt_patra | gzip > /backup/cbt_$(date +%Y%m%d).sql.gz

# Hapus backup > 30 hari
0 3 * * * find /backup -name "cbt_*.sql.gz" -mtime +30 -delete
```

### File Storage
- Gambar soal, audio soal, PDF laporan: tersimpan di MinIO
- MinIO sync ke S3/Backblaze setiap malam (opsional)

---

## Monitoring

| Service | Tool | URL |
|---------|------|-----|
| Health check | GET /api/v1/health | Cek DB, Redis connection |
| Metrics | Prometheus /metrics | Request rate, latency, goroutines |
| Logs | Docker logs / Grafana Loki | Structured JSON logs |
| Uptime | UptimeRobot / Grafana | Alert jika down |

---

## Hostinger VPS (sesuai SSH config)

```
Host: 185.214.124.85
Port: 65002
User: u212852160
Key:  ~/.ssh/hostinger_scriptsis

Deploy command:
ssh -p 65002 -i ~/.ssh/hostinger_scriptsis u212852160@185.214.124.85 "cd /home/u212852160/cbt-patra && git pull && docker compose build && docker compose up -d"
```
