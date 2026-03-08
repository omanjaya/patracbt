# System Design - CBT Patra (Golang + Vue.js)

**Tanggal:** 2026-03-05
**Status:** Draft

---

## Vision & Core Objectives

Membangun ulang ExamPatra dengan stack modern yang lebih performan, mudah di-deploy, dan scalable.

**Objectives:**
1. Single binary deployment (tidak perlu multiple runtime)
2. Real-time monitoring built-in tanpa server Node.js terpisah
3. SPA frontend (tidak ada page reload saat ujian berlangsung)
4. Server-side timer (tidak bisa dimanipulasi client)
5. Clean Architecture yang maintainable jangka panjang
6. API-first untuk mendukung mobile app di masa depan

---

## High-Level Architecture

```
+----------------------------------+
|          CLIENT LAYER            |
|  Vue.js 3 SPA (Vite + Pinia)    |
|  - Admin Panel                   |
|  - Guru Interface                |
|  - Peserta Interface             |
|  - Pengawas Dashboard            |
+----------------------------------+
           |              |
      REST API        WebSocket
           |              |
+----------------------------------+
|        BACKEND LAYER             |
|    Golang (Gin Framework)        |
|                                  |
|  +----------+  +-----------+    |
|  | HTTP API |  | WS Handler|    |
|  +----------+  +-----------+    |
|       |               |         |
|  +----------------------------+  |
|  |    Application Layer       |  |
|  |  (Use Cases / Services)    |  |
|  +----------------------------+  |
|  |    Domain Layer            |  |
|  |  (Entities + Rules)        |  |
|  +----------------------------+  |
|  |   Infrastructure Layer     |  |
|  |  (GORM, Redis, MinIO)      |  |
|  +----------------------------+  |
+----------------------------------+
           |         |        |
    +------+   +-----+   +----+
    |           |             |
+-------+  +-------+  +----------+
|  PgSQL|  | Redis |  |  MinIO   |
|(Data) |  |(Cache)|  | (Files)  |
+-------+  +-------+  +----------+
```

---

## Technology Stack

| Layer | Teknologi | Versi | Rationale |
|-------|-----------|-------|-----------|
| Backend Language | Golang | 1.23+ | Performa tinggi, goroutine, single binary |
| HTTP Framework | Gin | v1.9+ | Mature, fast, middleware rich |
| ORM | GORM | v2 | Idiomatic Go ORM, migrate support |
| WebSocket | Gorilla WebSocket | v1.5 | Standard, reliable |
| Auth | JWT (golang-jwt) | v5 | Stateless, scalable |
| Database | PostgreSQL | 16+ | ACID, JSON support, performance |
| Cache/Queue | Redis | 7+ | Session, pub/sub WebSocket, job queue |
| File Storage | MinIO (S3-compat) | Latest | Self-hosted, S3 API |
| Frontend Framework | Vue.js 3 | Latest | Composition API, reactivity |
| Frontend Build | Vite | 5+ | Cepat, HMR |
| State Management | Pinia | 2+ | Ringan, TypeScript-friendly |
| HTTP Client | Axios | Latest | Interceptors, typed |
| UI Components | shadcn-vue / PrimeVue | Latest | Rich components |
| PDF Export | go-wkhtmltopdf / chromedp | - | PDF generation |
| AI | HTTP Client ke AI API | - | Fleksibel, bisa Gemini/OpenAI |
| Containerization | Docker + Docker Compose | Latest | Consistent environment |
| Deployment | Single VPS / Docker Swarm | - | Simple ops |

---

## Core Subsystems

### 1. Auth Subsystem
```
Client
  |
  v
POST /api/auth/login
  |
  v
Validate credential (bcrypt)
  |
  v
Issue JWT (access_token 15m + refresh_token 7d)
  |
  v
Client store token (localStorage / httpOnly cookie)
  |
  v
Request with Bearer header
  |
  v
JWT Middleware validate + inject user context
```

**JWT Payload:**
```json
{
  "sub": "user_id",
  "role": "guru",
  "tenant_id": "school_id",
  "exp": 1234567890
}
```

### 2. Exam Engine Subsystem
```
Peserta: POST /api/exam/{schedule_id}/start
  |
  v
ExamService.StartSession()
  |-- Cek eligibility (jadwal aktif, peserta terdaftar, belum terminate)
  |-- Cek session existing (resume jika ada)
  |-- Generate question_order (shuffle jika aktif, acak per bank soal)
  |-- Simpan ke ExamSession
  |
  v
Return: session_id + questions + timer_end (server-calculated)

Client: Tampil soal, timer dari server

Peserta: POST /api/exam/session/{session_id}/answer
  |
  v
AnswerService.Save()
  |-- Validate session ownership + status ongoing
  |-- Upsert ExamAnswer
  |-- Return: OK

Timer habis / Peserta klik finish:
POST /api/exam/session/{session_id}/finish
  |
  v
ExamService.FinishSession()
  |-- Set status = completed, end_time = now
  |-- Trigger ScoreCalculation (async via goroutine atau sync)
  |
  v
Return: score (jika show_immediately)
```

### 3. Real-time WebSocket Subsystem
```
Hub (WebSocket Manager)
  |
  +-- Room management (per exam_schedule_id)
  |
  +-- Client map: {user_id -> *websocket.Conn}
  |
  +-- Broadcast: emit ke semua client dalam 1 room

Events:
  PESERTA -> SERVER:
    - join_room: {schedule_id, role}
    - heartbeat: {} (setiap 30 detik)
    - violation: {type}

  SERVER -> PESERTA:
    - lock_client: {} (dari pengawas)
    - time_sync: {server_time} (sinkronisasi waktu)

  PENGAWAS -> SERVER:
    - join_room: {schedule_id, token}
    - lock_client: {target_user_id}

  SERVER -> PENGAWAS:
    - student_joined: {user_id, name}
    - student_left: {user_id}
    - answer_saved: {user_id, question_count}
    - violation_logged: {user_id, count, type}
    - session_finished: {user_id}
```

### 4. Score Calculation Subsystem
```
Trigger: ExamSession.finish()
  |
  v
ScoreService.Calculate(session_id)
  |
  v
Load ExamSession + ExamAnswers + Questions
  |
  v
Per Question:
  |-- PG: jawaban == kunci → skor penuh
  |-- PGK: semua kunci harus ada, tidak boleh lebih → skor proporsional
  |-- Benar-Salah: sama dengan PG
  |-- Menjodohkan: cek pasangan yang benar
  |-- Isian Singkat: normalize text, compare
  |-- Matrix: per sel yang benar / total sel
  |-- Esai: skip (manual/AI grading)
  |
  v
Calculate final score = (total_mark / max_mark) * 100
  |
  v
Update ExamSession.score
```

---

## Security Architecture

### Authentication & Authorization
- JWT RS256 (asymmetric) untuk production security
- Refresh token via Redis (bisa di-revoke)
- Role-based middleware: Admin > Guru > Pengawas > Peserta
- Resource ownership check (peserta hanya bisa akses session miliknya)

### Anti-Cheat
- Violation counter di server (bukan client)
- Window blur event dikirim via API tiap event (bukan WebSocket saja)
- Session hash ID (tidak bisa brute force session orang lain)
- Rate limiting per endpoint per user

### Data Security
- Password: bcrypt cost 12
- File upload: validasi mime type, limit ukuran
- SQL injection: GORM parameterized query
- XSS: output escape di frontend (Vue auto-escape)
- CORS: whitelist domain

---

## Integration Architecture

```
CBT Patra Backend
  |
  +-- PostgreSQL (primary datastore)
  |
  +-- Redis
  |     |-- Cache soal (QuestionCache)
  |     |-- Refresh token storage
  |     |-- WebSocket pub/sub (jika horizontal scale)
  |     |-- Job queue (async tasks)
  |
  +-- MinIO (optional)
  |     |-- Storage gambar soal
  |     |-- Storage audio soal
  |     |-- PDF laporan
  |
  +-- AI API (optional)
        |-- Gemini API / OpenAI API
        |-- Auto-grading esai
        |-- Generate soal
```

---

## Scalability Design

### Horizontal Scaling
```
Load Balancer (Nginx/Caddy)
  |
  +-- CBT Instance 1 (Golang binary)
  +-- CBT Instance 2 (Golang binary)
  +-- CBT Instance N (Golang binary)
  |
  +-- Shared: PostgreSQL + Redis + MinIO

WebSocket Scaling:
  - Redis Pub/Sub sebagai message broker antar instance
  - Client terhubung ke instance manapun
  - Event dari satu instance di-broadcast via Redis ke instance lain
```

### Caching Strategy
```
Layer 1: In-memory (Go sync.Map)
  - Question cache per session (short TTL: 5 menit)
  
Layer 2: Redis
  - Question bank cache (TTL: 1 jam, invalidate on update)
  - User session data (TTL: 24 jam)
  - WebSocket room state
```

---

## Monitoring & Observability

| Tool | Purpose |
|------|---------|
| Prometheus + Grafana | Metrics (request rate, latency, goroutine count) |
| Zap logger | Structured logging (JSON) |
| Sentry (optional) | Error tracking |
| /health endpoint | Health check untuk load balancer |

---

## Disaster Recovery

- Database backup: pg_dump scheduled (cron) ke MinIO
- Soft delete semua entitas penting
- Redis persistence: RDB + AOF
- Docker volume untuk data persistence

---

## Development Phases

---

## Performance Tuning — 4-Core CPU

> Concern: server 4 vCPU. Ini CUKUP untuk CBT sekolah 300-1000 peserta concurrent jika dioptimasi dengan benar.
> Golang + Goroutine jauh lebih efisien dari PHP + FPM pada jumlah CPU yang sama.

### Profil Beban CBT (Golang vs PHP)

```
Skenario: 300 peserta concurrent, 40 soal, auto-save per jawaban

PHP (ExamPatra Original):
  - Setiap request buka proses PHP baru (FPM)
  - 300 concurrent = 300 proses PHP di RAM
  - CPU spike tinggi saat banyak yang submit serentak

Golang (CBT Patra):
  - 1 proses binary, ribuan goroutine ringan
  - 300 concurrent = 300 goroutine (tiap goroutine ~2-8 KB stack)
  - CPU lebih stabil, latency lebih rendah

Estimasi kapasitas server 4 Core / 8 GB RAM:
  - Golang: 500-1000 concurrent users AMAN
  - PHP FPM: 100-200 concurrent users (tergantung tuning)
```

### Strategi Golang Concurrency

```go
// GOMAXPROCS = jumlah CPU cores
// Default: otomatis set ke jumlah CPU
// Untuk 4 core: GOMAXPROCS=4 (sudah otomatis, tidak perlu manual)

// Gin: default menggunakan goroutine per request
// Tidak perlu konfigurasi tambahan untuk concurrency dasar

// Worker Pool untuk heavy task (scoring, PDF)
type WorkerPool struct {
    jobs chan Job
    wg   sync.WaitGroup
}

// Max 4 goroutine untuk heavy CPU task
const HEAVY_TASK_WORKERS = 4
```

### Konfigurasi PostgreSQL (4-Core)

```ini
# postgresql.conf — tuning untuk 4 core, 8 GB RAM

# Connections
max_connections = 100               # 4 core bisa handle ~100 concurrent conn

# Memory
shared_buffers = 2GB                # 25% dari total RAM
effective_cache_size = 6GB          # 75% dari total RAM
work_mem = 64MB                     # Per sort/hash operation
maintenance_work_mem = 512MB        # VACUUM, CREATE INDEX

# Parallel Query
max_parallel_workers = 4            # = jumlah CPU core
max_parallel_workers_per_gather = 2 # Parallelisasi query besar

# WAL & Checkpoint
checkpoint_completion_target = 0.9
wal_buffers = 64MB
default_statistics_target = 100

# Query Planner
random_page_cost = 1.1              # Jika pakai SSD (lebih kecil dari default 4.0)
effective_io_concurrency = 200      # Untuk SSD
```

### Konfigurasi GORM (Connection Pool)

```go
// infrastructure/persistence/postgres/db.go

func NewDB(cfg *config.Config) *gorm.DB {
    dsn := fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=disable",
        cfg.DBHost, cfg.DBUser, cfg.DBName, cfg.DBPassword)

    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Warn),
        // Disable default transaction per-query (performance!)
        SkipDefaultTransaction: true,
        // Cache prepared statements
        PrepareStmt: true,
    })

    sqlDB, _ := db.DB()

    // === CONNECTION POOL TUNING ===
    // Untuk 4 core, max_connections PostgreSQL = 100
    // Sisakan 10 untuk koneksi lain (psql manual, backup, dll)
    sqlDB.SetMaxOpenConns(80)           // max concurrent DB connections dari app
    sqlDB.SetMaxIdleConns(20)           // idle connections tetap terbuka
    sqlDB.SetConnMaxLifetime(time.Hour) // recycle connection setiap 1 jam
    sqlDB.SetConnMaxIdleTime(30 * time.Minute)

    return db
}
```

### Konfigurasi Redis

```ini
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru      # Hapus key lama jika memori penuh

# Persistence: kombinasi RDB + AOF
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec               # Tidak terlalu agresif flush ke disk

# Connections
maxclients 500
tcp-backlog 511
```

### Strategi Caching untuk CBT

```
Caching Priority (paling berdampak ke performa):

1. Question Cache (PALING PENTING)
   Key: question_bank:{bank_id}
   Value: JSON array semua soal (sudah termasuk options, correct_answer_text)
   TTL: 1 jam, invalidate saat soal diupdate
   Impact: Eliminasi N+1 query soal per sesi peserta
   
   // Saat 300 peserta mulai ujian bersamaan:
   // Tanpa cache: 300 x DB query soal = 300 query heavy
   // Dengan cache: 1 x DB query + 299 x Redis get (~1ms)

2. User Session Cache
   Key: user_session:{user_id}
   Value: user data (role, name, profile)
   TTL: 24 jam
   Impact: Tidak query users table per request

3. Schedule Eligibility Cache
   Key: eligibility:{schedule_id}:{user_id}
   Value: true/false
   TTL: 5 menit
   Impact: Saat banyak peserta cek eligibility bersamaan

4. Live Monitor Cache
   Key: live_status:{schedule_id}
   Value: JSON status semua peserta
   TTL: 10 detik (refresh oleh server)
   Impact: Dashboard pengawas tidak terus query DB
```

### Optimasi Query Kritis

```go
// ANTI-PATTERN (N+1 query): JANGAN LAKUKAN INI
for _, session := range sessions {
    _ = session.User // setiap ini trigger 1 DB query baru
}

// GOOD: Preload semua relasi sekaligus (1 query per relasi)
db.Preload("User").Preload("Answers").Find(&sessions)

// GOOD: SELECT only kolom yang diperlukan
db.Select("id, user_id, score, status").
    Where("exam_schedule_id = ?", scheduleID).
    Find(&sessions)

// Untuk Live Score: gunakan SQL aggregate, JANGAN load semua data ke Go
var stats []struct {
    UserID    uint
    Answered  int64
    Score     float64
}
db.Raw(`
    SELECT ea.exam_session_id,
           COUNT(ea.answer) as answered,
           es.score
    FROM exam_answers ea
    JOIN exam_sessions es ON ea.exam_session_id = es.id
    WHERE es.exam_schedule_id = ?
    GROUP BY ea.exam_session_id, es.score
`, scheduleID).Scan(&stats)
```

### WebSocket Hub — Efficient Broadcasting

```go
// JANGAN lock global saat broadcast
// Pattern: tiered channel broadcasting

type Hub struct {
    rooms   map[uint]*Room           // room per schedule_id
    mu      sync.RWMutex             // RWMutex: banyak reader, 1 writer
}

type Room struct {
    students   map[uint]*Client      // user_id -> client
    supervisors map[uint]*Client     // user_id -> supervisor client
    broadcast  chan []byte            // buffered channel
}

// Broadcast ke room: non-blocking
func (r *Room) Broadcast(msg []byte) {
    select {
    case r.broadcast <- msg:
    default:
        // Drop jika channel penuh (tidak blok goroutine lain)
    }
}

// Buffered channel mencegah slow client memblok semua
clientSend: make(chan []byte, 256)
```

### Ginnya HTTP Server Tuning

```go
// cmd/server/main.go

srv := &http.Server{
    Addr:         ":8080",
    Handler:      router,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
    // MaxHeaderBytes default sudah 1MB, OK
}

// Untuk 4 core: GOMAXPROCS otomatis = 4
// Gin sudah thread-safe by default
```

### Rate Limiting (Lindungi Server)

```go
// Menggunakan golang.org/x/time/rate per user
// Atau middleware rate limit dari Redis

// Endpoint kritis yang perlu rate limit ketat:
// POST /auth/login      : 10 req/menit per IP (cegah brute force)
// POST /exam/.../answer : 60 req/menit per user (1 jawaban/detik = wajar)
// GET /monitoring/...   : 30 req/menit (dashboard pengawas polling)
```

### Estimasi Kapasitas (4 Core, 8 GB RAM)

```
Komponen          RAM Usage    Keterangan
-----------       ---------    ----------
Golang binary     ~50 MB       proses utama
Goroutine x500    ~4 MB        500 concurrent users x 8KB/goroutine
PostgreSQL        ~3 GB        shared_buffers 2GB + overhead
Redis             ~2 GB        maxmemory 2GB
OS + lainnya      ~500 MB
-----------       ---------
Total             ~5.5 GB      dari 8 GB (buffer aman)

CPU:
- Golang HTTP handling: ~10-20% dari 1 core untuk 500 req/detik
- PostgreSQL query: ~20-30% untuk heavy analitik
- Background jobs: ~10% untuk scoring async
- Total idle: ~40-60% (headroom untuk spike)

Kesimpulan: 4 core CUKUP untuk 300-500 peserta serentak
Bottleneck pertama biasanya: disk I/O (untuk DB write), bukan CPU.
```

### Build Flags untuk Production

```bash
# Build dengan optimasi
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -trimpath \
    -o cbt-patra ./cmd/server

# -w: hapus DWARF debug info (lebih kecil binary)
# -s: hapus symbol table
# -trimpath: hapus info path file (security)
# Hasil: binary ~10-15 MB, startup < 100ms
```

---

## Development Phases

### Phase 1 - Core MVP (4-6 minggu)
- Setup project structure (Golang + Vue.js)
- Auth system (JWT)
- User & Role management
- Bank soal CRUD (semua tipe)
- Jadwal ujian CRUD
- Flow ujian peserta (start, answer, finish)
- Score calculation (PG, PGK, Benar-Salah)
- WebSocket monitoring peserta

### Phase 2 - Enhancement (3-4 minggu)
- Import soal Excel
- Laporan PDF
- Regrade
- Deteksi kecurangan lengkap
- Serial exam
- Export laporan

### Phase 3 - Advanced (4-6 minggu)
- AI Grading esai
- AI Generate soal
- Audio streaming soal
- Analisis soal (daya pembeda, tingkat kesukaran)
- Multi-tenant
- Notifikasi email
