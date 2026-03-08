# Implementation Roadmap - CBT Patra

**Tanggal:** 2026-03-05
**Developer:** Solo
**Estimasi Total:** 10-12 minggu

---

## Prinsip Implementasi

1. **Vertikal dulu, horizontal kemudian** — Selesaikan 1 fitur end-to-end (DB → API → UI) sebelum pindah fitur lain
2. **MVP first** — Fitur inti dulu, polish belakangan
3. **Test as you go** — Tulis unit test minimal untuk domain logic, jangan tumpuk di akhir
4. **Commit sering** — Setiap fitur selesai = 1 commit yang bisa di-demo

---

## Sprint 1 — Foundation (Week 1-2) ✅ DONE

**Goal:** Project bisa jalan, user bisa login, ada layout dasar.

### Backend
- [x] `go mod init`, setup folder structure sesuai 03-LAYERED-ARCHITECTURE
- [x] `config/config.go` — load .env
- [x] `pkg/logger/logger.go` — Zap logger
- [x] `pkg/response/response.go` — Standard JSON response
- [x] `internal/infrastructure/persistence/postgres/db.go` — GORM setup + connection pool
- [x] `internal/infrastructure/cache/redis/client.go` — Redis connection
- [x] Migration: `users`, `user_profiles`, `settings`
- [x] Entity: `User`, `UserProfile`, `Setting`
- [x] Repository: `UserRepository`, `SettingRepository`
- [x] UseCase: `LoginUseCase`, `LogoutUseCase`, `RefreshTokenUseCase`
- [x] `pkg/jwt/jwt.go` — Generate & validate JWT
- [x] `pkg/bcrypt/bcrypt.go` — Hash & verify password
- [x] Middleware: `AuthMiddleware`, `RoleMiddleware`
- [x] Handler: `AuthHandler` (login, logout, refresh, me)
- [x] Seeder: admin user default (`admin` / `password`)
- [x] `docker-compose.yml` untuk dev (PostgreSQL + Redis + MinIO)
- [x] `Makefile` shortcuts

### Frontend (web/)
- [x] `npm create vite` dengan Vue 3 + TypeScript
- [x] Setup: Vue Router, Pinia, Axios
- [x] `useApi.ts` composable (interceptors, token inject)
- [x] `auth.store.ts` (login state, token management)
- [x] `LoginPage.vue` — form login
- [x] `AdminLayout.vue` — sidebar + header + content area
- [x] `PesertaLayout.vue` — simple layout tanpa sidebar
- [x] Route guards (redirect jika belum login)

### Deliverable
```
✓ docker compose up → PostgreSQL + Redis jalan
✓ go run ./cmd/server → API di :8080
✓ npm run dev → Vue di :5173
✓ Login sebagai admin → masuk dashboard kosong
```

---

## Sprint 2 — Master Data CRUD (Week 3-4) ✅ DONE

**Goal:** Admin bisa manage semua data master.

### Backend
- [x] Migration: `rombels`, `user_rombels`, `subjects`, `tags`, `user_tags`, `rooms`
- [x] Entity + Repository + UseCase + Handler untuk:
  - Rombel (CRUD + assign user)
  - Subject (CRUD)
  - Tag (CRUD + assign user)
  - Room (CRUD)
  - User (CRUD + import Excel)
  - Setting (GET/UPDATE)
- [x] `pkg/pagination/pagination.go` — reusable pagination
- [ ] `pkg/validator/validator.go` — request validation
- [ ] `import_users_usecase.go` — parse Excel, bulk create

### Frontend
- [x] `BaseButton.vue`, `BaseInput.vue`, `BaseModal.vue`, `BaseTable.vue`, `BasePagination.vue`, `BaseBadge.vue`
- [x] Halaman Admin:
  - Dashboard (statistik ringkas)
  - Users list + create/edit modal
  - Rombels list + create/edit
  - Subjects list + create/edit
  - Tags list + create/edit
  - Rooms list + create/edit
  - Settings page (identitas, AI config, panic mode)
  - Roles (Hak & Izin) list + create/edit
  - Pengaitan Rombel (assign users ke rombel)
  - Pengaitan Ruangan (assign users ke ruangan)
  - Pengaitan Grup/Tag (assign users ke tag)
  - Cetak Kartu Peserta

### Deliverable
```
✓ Admin bisa CRUD user, rombel, subject, tag, room, role
✓ Admin bisa assign user ke rombel, ruangan, dan tag
✓ Settings bisa disimpan (nama app, AI config, panic mode)
✓ Kartu peserta bisa dicetak
```

---

## Sprint 3 — Bank Soal (Week 5-6) ✅ DONE

**Goal:** Guru bisa buat bank soal dan isi soal semua tipe.

### Backend
- [x] Migration: `question_banks`, `stimuli`, `questions`
- [x] Entity: `QuestionBank`, `Stimulus`, `Question`
- [x] `pkg/types/json.go` — custom JSONB type for GORM
- [x] Repository: `QuestionBankRepository`, `QuestionRepository`
- [x] UseCase: `QuestionBankUseCase`, `QuestionUseCase` (CRUD + Stimulus CRUD)
- [x] Handler: `QuestionBankHandler`, `QuestionHandler`
- [x] Routes: GET/POST/PUT/DELETE `/question-banks`, `/question-banks/:bankId/questions`, `/questions/:id`, `/stimuli`
- [ ] ImportQuestionsUseCase (Excel/CSV) — post-MVP
- [ ] Upload gambar soal ke MinIO — post-MVP

### Frontend
- [x] `QuestionBanksPage.vue` — list + filter by subject + create/edit/delete
- [x] `QuestionBankDetailPage.vue` — soal list + expand preview + stimulus manager
- [x] Form soal semua 7 tipe:
  - PG — pilihan + kunci jawaban
  - PGK — pilihan + multi-kunci + bobot per opsi
  - Benar/Salah — 2 opsi fixed + kunci
  - Menjodohkan — prompts + answers + kunci pairs
  - Isian Singkat — accepted answers list
  - Matrix — rows + columns + kunci per baris
  - Esai — body saja
- [x] Route `/admin/question-banks/:id`

### Deliverable
```
✓ Admin/Guru bisa buat bank soal
✓ Admin/Guru bisa tambah soal semua 7 tipe
✓ Preview soal expand/collapse
✓ Stimulus/wacana bisa dikelola per bank
```

---

## Sprint 4 — Jadwal Ujian + Flow Peserta (Week 7-8) ✅ DONE

**Goal:** Peserta bisa mengerjakan ujian dari awal sampai selesai.

### Backend
- [ ] Migration: `exam_schedules`, `exam_schedule_question_banks`, `exam_schedule_rombels`, `exam_schedule_tags`, `exam_schedule_users`, `exam_sessions`, `exam_answers`
- [ ] Entity: `ExamSchedule`, `ExamSession`, `ExamAnswer`
- [ ] Domain Service: `ScoreCalculator`, `EligibilityChecker`
- [ ] UseCase Jadwal:
  - `CreateScheduleUseCase` (+ assign banks, rombels, tags, users)
  - `UpdateScheduleUseCase`, `DeleteScheduleUseCase`, `ListSchedulesUseCase`
- [ ] UseCase Peserta Flow:
  - `ConfirmExamUseCase` — info ujian + rules
  - `StartExamUseCase` — buat session, generate question_order (shuffle), hitung end_time
  - `LoadSessionUseCase` — load soal + jawaban existing
  - `SaveAnswerUseCase` — upsert jawaban (auto-save)
  - `ToggleFlagUseCase` — toggle ragu-ragu
  - `FinishExamUseCase` — selesaikan sesi, hitung skor semua tipe
  - `LogViolationUseCase` — catat pelanggaran, terminate jika > limit
- [ ] `pkg/hashid/hashid.go` — Hash session ID untuk URL aman
- [ ] `ScoreCalculator` harus handle semua 7 tipe soal:
  - PG: cocok opsi weight = 1.0
  - PGK: sum weight semua opsi yang dipilih
  - Menjodohkan: jumlah pair benar / total pair
  - Matrix: jumlah row benar / total row
  - Singkat: case-insensitive match accepted answers
  - Esai: manual/AI score
  - BS: sama seperti PG
- [ ] Cache soal di Redis saat start exam (performance)

### Frontend
- [ ] Halaman Guru:
  - Jadwal Ujian list (filter: akan datang, berlangsung, selesai)
  - Form buat/edit jadwal ujian (pilih bank soal, rombel, konfigurasi)
- [ ] Halaman Peserta:
  - Dashboard peserta (ujian aktif, mendatang, selesai)
  - Confirm page (info ujian + input token)
  - **ExamPage.vue** (halaman utama mengerjakan ujian):
    - Panel soal (70%) — render komponen per tipe
    - Panel navigasi (30%) — grid nomor soal dengan warna status
    - Timer countdown (dari server end_time)
    - Auto-save saat jawaban berubah
    - Tombol flag ragu-ragu
    - Tombol prev/next
    - Konfirmasi selesai
  - Result page (tampil nilai jika diizinkan)
- [ ] Komponen soal: `QuestionPG.vue`, `QuestionPGK.vue`, `QuestionEssay.vue`, `QuestionMatching.vue`, `QuestionFillIn.vue`, `QuestionMatrix.vue`, `QuestionTrueFalse.vue`
- [ ] `useExamTimer.ts` — countdown dari server end_time
- [ ] Deteksi tab switch (visibilitychange event → POST log-violation)

### Deliverable
```
✓ Guru bisa buat jadwal ujian
✓ Peserta bisa lihat jadwal aktif di dashboard
✓ Peserta bisa mulai ujian (input token)
✓ Peserta bisa jawab semua 7 tipe soal
✓ Auto-save jawaban
✓ Timer countdown
✓ Peserta bisa selesaikan ujian → skor dihitung
✓ Deteksi pindah tab → violation tercatat
```

---

## Sprint 5 — Real-time Monitoring (Week 9-10) ✅ DONE

**Goal:** Pengawas bisa pantau peserta real-time.

### Backend
- [x] `internal/infrastructure/websocket/hub.go` — Room-based WebSocket hub
- [x] `internal/infrastructure/websocket/client.go` — Client wrapper
- [x] `internal/infrastructure/websocket/message.go` — Message types
- [x] `internal/presentation/http/handler/ws_handler.go` — Upgrade HTTP → WebSocket
- [x] Events: heartbeat, student_joined, student_left, answer_saved, violation_logged, session_finished, lock_client, time_sync
- [x] Handler emit di ExamSessionHandler: answer_saved, violation_logged, session_finished
- [x] Routes: GET /ws/exam/:scheduleId, GET /monitoring/:scheduleId/clients, POST /monitoring/:scheduleId/lock

### Frontend
- [x] `useWebSocket.ts` composable (auto-reconnect, event handling, heartbeat 30s)
- [x] `SupervisionPage.vue` — grid peserta real-time, filter status, lock client, WS status indicator, offline detection 90s
- [x] ExamPage.vue — WS connect saat exam, handle lock_client → overlay, disconnect on unmount

### Deliverable
```
✓ Pengawas bisa lihat semua peserta real-time
✓ Progress (25/40) dan status update live
✓ Pelanggaran muncul langsung di dashboard pengawas
✓ Pengawas bisa kunci client peserta
✓ Peserta yang offline terdeteksi
```

---

## Sprint 6 — Report, Analysis & Polish (Week 11-12) ✅ DONE

**Goal:** Fitur laporan lengkap, analisis soal, backup, dan finishing.

### Backend
- [x] Migration: `regrade_logs`
- [x] `internal/application/usecase/report/report_usecase.go`:
  - `GetScheduleReport` — rekap semua peserta + nilai + statistik (mean, median, std dev)
  - `GetPersonalReport` — detail jawaban per soal dengan earned score
  - `GetExamAnalysis` — difficulty index (p-value) + discrimination index (D-value) per soal + quality label
  - `RegradeSchedule` — hitung ulang nilai semua sesi finished/terminated
  - `SetEssayScore` — grading manual/AI untuk soal esai
- [x] `internal/presentation/http/handler/report_handler.go` — semua endpoint
- [x] Routes: GET /reports/:id, GET /reports/:id/analysis, GET /reports/:id/sessions/:sid, POST /reports/:id/regrade, POST /reports/sessions/:sid/grade-essay

### Frontend
- [x] `web/src/api/report.api.ts` — semua tipe + API calls
- [x] `ReportsPage.vue` — schedule selector, stats bar, tab Rekap Nilai (tabel dengan rank + progress bar), tab Analisis Butir Soal (difficulty index, discrimination index, quality label), personal report detail (jawaban per soal correct/wrong), print PDF via window.print()
- [x] `LiveScorePage.vue` — scoreboard real-time TV mode, fullscreen mode, WS auto-refresh
- [x] `web/src/stores/toast.store.ts` — toast notification system (Pinia store)
- [x] `web/src/components/ui/ToastContainer.vue` — toast UI dengan TransitionGroup animasi
- [x] `App.vue` — include ToastContainer global
- [x] `SettingsPage.vue` — toast feedback saat save
- [x] `AdminLayout.vue` — mobile responsive (hamburger menu, overlay, sidebar slide-in)

### Deliverable
```
✓ Guru bisa lihat rekap nilai + download PDF
✓ Peserta bisa lihat laporan personal (jika diizinkan)
✓ Analisis butir soal berfungsi
✓ AI grading esai berfungsi (jika API tersedia)
✓ Admin bisa backup/restore .patrabak
✓ Panic mode bisa mengunci semua peserta
✓ Aplikasi terasa polished dan production-ready
```

---

## Nice-to-Have (Post-MVP)

Fitur ini dikerjakan **setelah** 6 sprint di atas selesai:

| Fitur | Prioritas | Estimasi |
|-------|-----------|----------|
| Serial exam (ujian berantai) | Medium | 3-5 hari |
| Multi-stage exam (beberapa bank soal dalam 1 jadwal) | Medium | 3-5 hari |
| Kartu peserta PDF | Low | 1-2 hari |
| QR Login untuk peserta | Low | 2-3 hari |
| Import/export bank soal antar instansi | Low | 2-3 hari |
| Audio question player + limit | Low | 1-2 hari |
| PWA mode (installable, offline-first cache) | Low | 3-5 hari |
| Dark mode | Low | 1-2 hari |

---

## Checklist Pre-Production

Sebelum deploy ke server production:

- [ ] Semua `.env` secrets sudah diganti dari default
- [ ] CORS hanya izinkan domain production
- [ ] Rate limit aktif di semua endpoint sensitif
- [ ] HTTPS aktif (Caddy auto-cert)
- [ ] Log level diset ke `warn` (bukan `debug`)
- [ ] `GORM SkipDefaultTransaction: true`
- [ ] `GORM PrepareStmt: true`
- [ ] PostgreSQL tuning sesuai spec server (5-DATABASE-SCHEMA)
- [ ] Redis `maxmemory` diset
- [ ] Backup cron aktif
- [ ] Health check endpoint (`GET /api/v1/health`) berfungsi
- [ ] Admin default password sudah diganti
