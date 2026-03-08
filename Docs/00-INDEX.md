# Dokumentasi Project - CBT Patra (Golang + Vue.js)

**Dibuat:** 2026-03-05
**Diperbarui:** 2026-03-05 (v2)
**Stack:** Golang (Gin + GORM) | Vue.js 3 (Composition API) | PostgreSQL | Redis | WebSocket
**Referensi:** ExamPatra (Laravel) - https://github.com/arimartana/patra
**Struktur:** Single project (Go di root, Vue.js di `web/`)

---

## Daftar Dokumen

| No | File | Deskripsi | Status |
|----|------|-----------|--------|
| 01 | [01-RESEARCH-ANALYSIS.md](./01-RESEARCH-ANALYSIS.md) | Analisis 20 fitur ExamPatra & kompetitor | ✅ DONE |
| 02 | [02-SYSTEM-DESIGN.md](./02-SYSTEM-DESIGN.md) | High-level architecture, tech stack & performance tuning 4-core | ✅ DONE |
| 03 | [03-LAYERED-ARCHITECTURE.md](./03-LAYERED-ARCHITECTURE.md) | Clean Architecture, folder structure, use case listing | ✅ DONE |
| 04 | [04-CLEAN-CODE-PRACTICES.md](./04-CLEAN-CODE-PRACTICES.md) | Coding standards Golang | ✅ DONE |
| 05 | [05-DATABASE-SCHEMA.md](./05-DATABASE-SCHEMA.md) | 15+ tabel PostgreSQL, indexes, GORM model | ✅ DONE |
| 06 | [06-UI-UX-DESIGN.md](./06-UI-UX-DESIGN.md) | Design system, color palette, layout per role | ✅ DONE |
| 07 | [07-SHARED-COMPONENTS.md](./07-SHARED-COMPONENTS.md) | Komponen Vue.js reusable + composables + TypeScript types | ✅ DONE |
| 08 | [08-API-DESIGN.md](./08-API-DESIGN.md) | REST API endpoints, WebSocket events, rate limits | ✅ DONE |
| 09 | [09-DEPLOYMENT.md](./09-DEPLOYMENT.md) | Docker multi-stage, Caddy, backup strategy | ✅ DONE |
| 10 | [10-IMPLEMENTATION-ROADMAP.md](./10-IMPLEMENTATION-ROADMAP.md) | Sprint plan 6 fase (12 minggu) + checklist | ✅ DONE |

---

## Ringkasan Project

### Apa itu CBT Patra?
CBT Patra adalah aplikasi Computer-Based Testing (ujian online) yang di-rebuild dari ExamPatra (Laravel)
menggunakan **Golang sebagai backend** dan **Vue.js 3 sebagai frontend** dalam satu project.
Tujuan utama: performa lebih tinggi di server 4-core, maintainability lebih baik, dan single binary deployment.

### Target User
- **Admin Sekolah/Instansi** — manage user, settings, backup/restore
- **Guru/Pengajar** — buat bank soal, jadwal ujian, laporan & analisis
- **Pengawas** — pantau peserta real-time via WebSocket
- **Peserta** — mengerjakan ujian

### Fitur Utama (20 fitur)
- Manajemen bank soal (7 tipe: PG, PGK, Esai, Menjodohkan, Isian Singkat, Matrix, Benar-Salah)
- Jadwal ujian dengan konfigurasi lengkap (token, shuffle, late policy, cheating detection)
- Real-time monitoring via WebSocket (built-in Go, tanpa Node.js)
- Deteksi kecurangan (tab switch, window blur) + auto-terminate
- Scoring otomatis semua tipe + AI Grading untuk esai
- Analisis butir soal (Difficulty Index, Discrimination Index)
- Personal report peserta (detail per soal)
- Laporan hasil ujian (PDF export)
- Backup & Restore (.patrabak)
- Import soal massal (Excel/CSV)
- Multi-rombel & tag system
- Serial exam (ujian berantai)
- Settings lanjutan (Panic Mode, QR Login, Login Optimization)

### Cara Develop (Solo Dev)
```bash
make infra       # Nyalain PostgreSQL + Redis + MinIO (Docker)
make dev-api     # Terminal 1: Go API di :8080
make dev-web     # Terminal 2: Vue dev di :5173
make build       # Production: 1 binary (Go + Vue embedded)
```
