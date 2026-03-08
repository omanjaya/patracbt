.PHONY: dev stop build logs restart

# ─── Development (Docker All-in-One) ─────────
# Jalankan semua service: PostgreSQL, Redis, MinIO, Go (Air), Vue (Vite)
dev:
	docker compose up --build

# Jalankan di background
dev-bg:
	docker compose up --build -d

# Stop semua
stop:
	docker compose down

# Restart hanya Go API (setelah edit docker config, dll)
restart-api:
	docker compose restart api

# Logs semua / per service
logs:
	docker compose logs -f

logs-api:
	docker compose logs -f api

logs-web:
	docker compose logs -f web

# ─── Production Build ────────────────────────
build-web:
	cd web && npm run build

build:
	$(MAKE) build-web
	CGO_ENABLED=0 go build -ldflags='-w -s' -o cbt-patra ./cmd/server

# ─── Utilities ───────────────────────────────
tidy:
	go mod tidy

# Masuk ke shell container
shell-api:
	docker compose exec api sh

shell-db:
	docker compose exec postgres psql -U cbt_user -d cbt_patra
