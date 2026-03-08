# Research & Analisis - CBT Patra

**Tanggal:** 2026-03-05
**Diperbarui:** 2026-03-05 (v2 - coverage lengkap)
**Tujuan:** Analisis mendalam fitur ExamPatra (Laravel) untuk di-rebuild dengan Golang + Vue.js
**Status:** Research Complete

---

## Gambaran Umum

ExamPatra (nama resmi: Patra Anyar Gen) adalah aplikasi ujian online berbasis web yang dibangun
menggunakan Laravel + Node.js + Socket.IO + Redis + FFmpeg. Dibuat untuk kebutuhan CBT (Computer
Based Testing) di lingkungan sekolah/instansi, dengan fitur real-time monitoring, deteksi
kecurangan, streaming audio soal, dan sistem chat pengawas-peserta.

**Versi:** 4.0
**Domain:** demo.exam.patra.co.id
**Database:** PostgreSQL (bukan MySQL, sudah migrasi)
**Stack Asli:** Laravel (Octane/FrankenPHP) + Node.js Socket.IO + Redis + Minio + FFmpeg

---

## Platform & Teknologi (ExamPatra Asli)

| Komponen | Teknologi | Keterangan |
|----------|-----------|------------|
| Backend Web | Laravel 11 + Octane (FrankenPHP) | Server utama |
| Real-time | Node.js + Socket.IO | WebSocket server terpisah |
| Antrian | Redis Queue + Laravel Jobs | Proses async |
| Session/Cache | Redis | Auth WebSocket |
| Database | PostgreSQL | Data utama |
| Storage | Minio (S3-compatible) | Backup storage |
| Audio | FFmpeg | Streaming soal listening |
| Frontend | Blade + Vite + Alpine.js | Server-side rendered |
| Auth Peserta | Sanctum (REST API Token) | Untuk mobile/native |

---

## Target Market

- Sekolah menengah (SMP/SMA/SMK) yang butuh CBT mandiri
- Lembaga bimbingan belajar
- Instansi pemerintah/swasta (ujian masuk kerja, tes internal)
- Guru individual yang butuh alat ujian online

**Skala:** Single-tenant (1 instansi per deployment), potensi multi-tenant
**Model:** Self-hosted / open source

---

## Fitur Utama & Spesifikasi

### A. MANAJEMEN PENGGUNA

#### 1. Role & Permission
- Deskripsi: Sistem RBAC menggunakan Spatie Laravel Permission
- Sub-fitur:
  - [x] Role: Admin, Guru, Pengawas, Peserta
  - [x] Permission granular per fitur (CRUD bank soal, kelola jadwal, dll)
  - [x] Assign permission per user (override role)
  - [x] Middleware proteksi route per role

#### 2. Manajemen User
- Deskripsi: CRUD pengguna dengan profil lengkap
- Sub-fitur:
  - [x] CRUD user (nama, username, email, password, foto)
  - [x] Import user massal (Excel)
  - [x] Assign ke Rombel (kelas/kelompok)
  - [x] Assign Tag (label khusus: Remedial, Sesi 1, dll)
  - [x] Force login (admin bisa pindah sesi aktif)
  - [x] Last login tracking

#### 3. Profil Peserta
- Sub-fitur:
  - [x] UserProfile: NIS/NIP, kelas, jurusan, angkatan, foto
  - [x] Rombel (kelas) assignment

### B. BANK SOAL

#### 4. Tipe Soal
- Deskripsi: Mendukung 7 tipe soal berbeda
- Sub-fitur:
  - [x] PG (Pilihan Ganda biasa, 1 jawaban benar)
  - [x] PGK (Pilihan Ganda Kompleks, jawaban > 1)
  - [x] Benar-Salah (A=Benar, B=Salah)
  - [x] Menjodohkan (pasangan premis-jawaban)
  - [x] Isian Singkat (jawaban pendek pasti)
  - [x] Matrix/Tabel (baris vs kolom, pemetaan angka)
  - [x] Esai (uraian panjang, dinilai manual/AI)

#### 5. Bank Soal (Question Bank)
- Deskripsi: Wadah kumpulan soal per mata pelajaran
- Sub-fitur:
  - [x] CRUD bank soal
  - [x] Link ke Subject (Mata Pelajaran)
  - [x] CRUD soal di dalam bank
  - [x] Upload gambar/audio per soal
  - [x] Stimulus (narasi/teks induk untuk soal berkelompok)
  - [x] Sort order soal
  - [x] Bobot nilai per soal (default_mark)
  - [x] AI Generate Soal (via API key AI)
  - [x] Import soal massal (Excel/CSV dengan parser khusus)
  - [x] Export soal ke format portable
  - [x] Cache soal (QuestionCacheService)

#### 6. Mata Pelajaran (Subject)
- Sub-fitur:
  - [x] CRUD subject
  - [x] Link ke bank soal

### C. JADWAL UJIAN

#### 7. Konfigurasi Jadwal Ujian
- Deskripsi: Panel lengkap konfigurasi ujian
- Sub-fitur:
  - [x] Nama jadwal
  - [x] Multi bank soal (many-to-many dengan bobot & urutan)
  - [x] Durasi per bank soal
  - [x] Waktu mulai & selesai (start_time, end_time)
  - [x] Token masuk ujian (opsional, bisa kosong)
  - [x] Token pengawas (is_locked_by_pengawas)
  - [x] Kebijakan keterlambatan: potong waktu / full time
  - [x] Deteksi kecurangan on/off
  - [x] Batas pelanggaran (cheating_limit)
  - [x] Kapan nilai tampil: langsung / setelah end_time / manual
  - [x] Tampilkan laporan ke siswa: on/off
  - [x] Acak urutan soal (shuffle_questions)
  - [x] Acak opsi jawaban (shuffle_options)
  - [x] Waktu minimal pengerjaan (min_working_time menit)
  - [x] Serial exam (next_exam_schedule_id - ujian berantai)

#### 8. Target Peserta Ujian
- Deskripsi: Sistem include/exclude peserta ujian
- Sub-fitur:
  - [x] Assign per Rombel (include/exclude)
  - [x] Assign per Tag (include/exclude)
  - [x] Assign per User individu (include/exclude)
  - [x] Status jadwal: Akan Datang, Berlangsung, Selesai (otomatis dari waktu)

### D. PELAKSANAAN UJIAN (PESERTA)

#### 9. Flow Pengerjaan Ujian
- Deskripsi: Alur lengkap peserta dari login hingga selesai
- Sub-fitur:
  - [x] Dashboard peserta (active, upcoming, past exams)
  - [x] Konfirmasi sebelum mulai (info jadwal, durasi, aturan)
  - [x] Start exam (create ExamSession, generate question order)
  - [x] Resume exam (lanjut sesi yang sudah ada)
  - [x] Tampil soal satu per satu / navigasi bebas
  - [x] Simpan jawaban real-time (auto-save)
  - [x] Flag soal (ragu-ragu)
  - [x] Timer hitung mundur
  - [x] Log pelanggaran (tab switch, window blur)
  - [x] Terminasi otomatis (melebihi batas pelanggaran)
  - [x] Selesai ujian (tombol finish)
  - [x] Tampil nilai sesuai konfigurasi

#### 10. Sesi Ujian (ExamSession)
- Sub-fitur:
  - [x] Status: ongoing, completed, terminated
  - [x] session_details JSON (urutan soal yang sudah diacak khusus per siswa)
  - [x] violation_count tracking
  - [x] Hash ID untuk URL aman
  - [x] Regrade support (original_score backup)

### E. MONITORING & PENGAWASAN

#### 11. Dashboard Pengawas Real-time
- Deskripsi: Panel monitoring siswa saat ujian berlangsung via WebSocket
- Sub-fitur:
  - [x] List peserta + status online/offline real-time
  - [x] Progress pengerjaan per siswa
  - [x] Violation count per siswa
  - [x] Filter per room/ruangan
  - [x] Kunci klien (logout paksa real-time via Socket.IO)
  - [x] Chat pengawas-peserta
  - [x] Token ruang ujian (ExamSupervision)

#### 12. Deteksi Kecurangan
- Sub-fitur:
  - [x] Deteksi window blur (ganti tab/aplikasi)
  - [x] Auto-terminasi setelah batas pelanggaran
  - [x] Log violation type

### F. LAPORAN & ANALISIS

#### 13. Laporan Hasil Ujian
- Deskripsi: Laporan komprehensif setelah ujian selesai
- Sub-fitur:
  - [x] Live score monitoring (LiveScoreService)
  - [x] Rekap nilai semua peserta per jadwal
  - [x] Detail jawaban per peserta
  - [x] Analisis soal (ExamAnalysisService): daya pembeda, tingkat kesukaran
  - [x] Export PDF (kartu ujian, laporan nilai)
  - [x] Regrade (hitung ulang nilai setelah kunci diubah)
  - [x] Show/hide report ke peserta (show_report flag)

#### 14. AI Grading
- Deskripsi: Penilaian otomatis soal esai via AI API
- Sub-fitur:
  - [x] Integrasi API AI fleksibel (Gemini format / OpenAI format / custom)
  - [x] Prompt konstruksi otomatis dari soal + kunci
  - [x] Normalisasi skor 0-100
  - [x] Alasan penilaian (reason)

### G. PENGATURAN SISTEM

#### 15. Personal Report Peserta
- Deskripsi: Laporan hasil ujian pribadi peserta setelah ujian selesai
- Sub-fitur:
  - [x] Detail soal + jawaban siswa + skor per soal
  - [x] Hitung benar/salah/belum dijawab per soal
  - [x] Tampil per section (multi-bank soal)
  - [x] Kontrol akses: hanya tampil jika show_report = true & nilai sudah bisa dilihat
  - [x] Hash ID untuk URL aman (tidak bisa akses laporan orang lain)

#### 16. Analisis Butir Soal (ExamAnalysis)
- Deskripsi: Analisis psikometri soal setelah ujian selesai
- Sub-fitur:
  - [x] Statistik umum: Mean, Median, Modus, Standar Deviasi, Min, Max
  - [x] Difficulty Index (P): tingkat kesulitan soal (0.0 - 1.0)
  - [x] Discrimination Index (D): daya pembeda soal (upper group vs lower group, Kelly 27%)
  - [x] Label kualitas soal: Sangat Baik, Baik, Cukup, Kurang, Jelek
  - [x] Filter per rombel
  - [x] Include ongoing sessions (opsional)

#### 17. Settings Sistem (Lanjutan)
- Deskripsi: Konfigurasi global aplikasi yang lebih lengkap
- Sub-fitur:
  - [x] Konfigurasi AI (URL, API key, header, model params JSON)
  - [x] Nama aplikasi, logo, favicon (upload via Cropper.js)
  - [x] Login QR Code (enable/disable)
  - [x] PWA Restricted Mode
  - [x] App Access Key (kunci akses bergaya 8 karakter)
  - [x] Tag Alerts (pesan notifikasi per tag peserta)
  - [x] Login Method: Normal (DB) vs Redis (cache warm-up untuk performa)
  - [x] WebSocket enable/disable
  - [x] Exam Navigation Style: classic / (style lain)
  - [x] Panic Mode (lock semua akses peserta darurat)
  - [x] Timezone setting
  - [x] Monitor Scheduler (cek cron aktif)
  - [x] Log Viewer (baca log harian via UI)

#### 18. Backup & Restore (.patrabak)
- Deskripsi: Fitur backup dan restore data aplikasi dalam format .patrabak (ZIP)
- Sub-fitur:
  - [x] Backup database (pg_dump / mysqldump otomatis)
  - [x] Backup storage (gambar soal, audio, dll)
  - [x] Download file backup (.patrabak)
  - [x] Upload restore chunked (untuk file besar)
  - [x] Restore database + storage dari file .patrabak
  - [x] Backup ke MinIO (opsional)
  - [x] Progress tracking via Redis (batch_id)

#### 19. Kartu Peserta
- Sub-fitur:
  - [x] Generate kartu peserta PDF (CardGeneratorController)
  - [x] Template kartu

#### 20. Bank Soal - is_active Flag
- Sub-fitur:
  - [x] Status draft/publish (is_active) per bank soal
  - [x] Validasi: bank soal tidak boleh dihapus jika sudah dipakai jadwal

---

## User Flows & Workflows

### Flow 1: Guru Membuat Ujian
```
START
  |
  v
Login (guru)
  |
  v
Buka menu Bank Soal
  |
  v
Pilih / Buat Bank Soal
  |
  v
Tambah Soal (pilih tipe, tulis soal, isi opsi, set kunci)
  |
  v
Buka menu Jadwal Ujian
  |
  v
Buat Jadwal Baru
  |-- Isi nama, pilih bank soal, set durasi
  |-- Set waktu mulai & selesai
  |-- Set konfigurasi (token, shuffle, deteksi curang, dll)
  |-- Assign rombel/tag/user target
  |
  v
Simpan Jadwal
  |
  v
END (Ujian siap diakses peserta)
```

### Flow 2: Peserta Mengerjakan Ujian
```
START
  |
  v
Login (peserta)
  |
  v
Dashboard Peserta (lihat ujian aktif / upcoming)
  |
  v
Pilih Ujian Aktif → Klik Mulai
  |
  v
Halaman Konfirmasi (info ujian, aturan)
  |
  v
Input Token (jika diperlukan)
  |
  v
API: POST /exam/{id}/start
  |-- Sistem cek eligibility peserta
  |-- Sistem generate question order (shuffle jika aktif)
  |-- Buat ExamSession (status: ongoing)
  |
  v
Halaman Pengerjaan Ujian
  |-- Timer hitung mundur berjalan
  |-- Tampil soal satu per satu
  |-- Peserta jawab soal → API: POST /session/{id}/save-answer
  |-- Bisa flag ragu-ragu → API: POST /session/{id}/toggle-flag
  |-- Tab switch terdeteksi → API: POST /session/{id}/log-violation
  |      |
  |      v
  |   Jika violation >= limit → session terminated → redirect keluar
  |
  v
Klik Selesai (atau timer habis → auto finish)
  |
  v
API: POST /session/{id}/finish
  |-- Sistem hitung skor
  |-- Status session: completed
  |
  v
Halaman Hasil (tampil nilai jika show_score_after = immediately)
  |
  v
END
```

### Flow 3: Pengawas Monitoring Real-time
```
START
  |
  v
Login (pengawas)
  |
  v
Dashboard Pengawas → Pilih Jadwal Ujian
  |
  v
Input Token Ruang (ExamSupervision token)
  |
  v
Panel Monitoring Real-time
  |-- WebSocket connect ke Socket.IO server
  |-- Terima event: peserta_online, peserta_offline, answer_saved, violation_logged
  |-- Tampil grid peserta + status warna (hijau=online, merah=offline)
  |-- Klik peserta → lihat detail progress
  |
  v
Aksi Pengawas:
  |-- Chat ke peserta tertentu
  |-- Kunci klien (emit event: lock_client → peserta logout paksa)
  |
  v
END
```

### Flow 4: Admin Import User
```
START
  |
  v
Login (admin)
  |
  v
Menu User Management → Import
  |
  v
Download template Excel
  |
  v
Isi data user (nama, username, password, rombel, dll)
  |
  v
Upload file Excel
  |
  v
Sistem validasi, import, buat akun, assign rombel
  |
  v
Laporan hasil import (berhasil/gagal per baris)
  |
  v
END
```

### Flow 5: Guru Regrade (Hitung Ulang Nilai)
```
START
  |
  v
Login (guru) → Buka Jadwal Ujian Selesai
  |
  v
Edit kunci jawaban di bank soal
  |
  v
Buka Laporan Jadwal → Klik Regrade
  |
  v
Sistem hitung ulang semua skor ExamSession berdasarkan kunci baru
  |-- Simpan original_score backup (jika belum pernah regrade)
  |-- Update score baru
  |-- Set score_change_notified = false
  |
  v
Sistem notifikasi perubahan nilai ke peserta
  |
  v
END
```

### Flow 6: AI Grading Esai
```
START
  |
  v
Ujian selesai, ada soal tipe esai
  |
  v
Guru buka laporan → Klik "Grading AI" per jawaban
  |
  v
Sistem ambil: soal, kunci/kriteria, jawaban siswa
  |
  v
Kirim ke AI API (Gemini/OpenAI/Custom)
  |
  v
AI return JSON: { score: 75, reason: "..." }
  |
  v
Sistem simpan skor, update total nilai sesi
  |
  v
END
```

---

## Integrasi & Ekosistem

| Integrasi | Purpose | Required |
|-----------|---------|----------|
| Redis | Session, WebSocket auth, Queue, Cache | YES |
| Socket.IO (Node.js) | Real-time monitoring & chat | YES |
| FFmpeg | Audio streaming soal listening | Optional |
| Minio/S3 | Backup storage file | Optional |
| AI API (Gemini/OpenAI) | Auto-grading esai, generate soal | Optional |
| Mailer (SMTP) | Notifikasi email | Optional |

---

## Kekurangan ExamPatra Saat Ini

### A. Technical Issues
- Dua server terpisah (Laravel + Node.js) → kompleks deployment & maintenance
- FrankenPHP Octane: edge case di shared hosting, kompleks konfigurasi
- Tidak ada rate limiting di API peserta
- Tidak ada audit log aktivitas admin
- Session shuffle soal disimpan di JSON → tidak ada validasi integritas

### B. Missing Features
- Tidak ada multi-tenancy (1 app = 1 instansi)
- Tidak ada notifikasi push / email otomatis saat jadwal mulai
- Timer tidak sinkron di sisi server (bisa dimanipulasi via client)
- Tidak ada backup otomatis
- Tidak ada mode offline/progressive web app
- Tidak ada fitur live Q&A peserta-guru saat ujian

### C. UX Issues
- Halaman soal masih server-side rendered Blade (bukan SPA)
- Tidak ada preview soal sebelum ujian dimulai
- Import Excel error message tidak detail

### D. Comparative Analysis

| Fitur | ExamPatra | Google Forms | WinExam | Moodle Quiz |
|-------|-----------|-------------|---------|-------------|
| Real-time monitoring | GOOD | BAD | GOOD | BAD |
| Deteksi kecurangan | GOOD | BAD | GOOD | PARTIAL |
| AI Grading | GOOD | BAD | BAD | BAD |
| Multi-tenant | BAD | GOOD | BAD | GOOD |
| Mobile App | BAD | GOOD | BAD | PARTIAL |
| Audio Soal | GOOD | BAD | GOOD | PARTIAL |
| Import Soal Massal | GOOD | PARTIAL | GOOD | GOOD |
| Serial Exam | GOOD | BAD | BAD | GOOD |
| Laporan PDF | GOOD | PARTIAL | GOOD | GOOD |
| Gratis/OSS | GOOD | GOOD | BAD | GOOD |

---

## Kesempatan & Differentiator (CBT Patra Golang)

1. **Single binary deployment** - Golang compile ke binary tunggal, tidak perlu PHP, tidak perlu Node.js terpisah
2. **WebSocket built-in** - Golang memiliki net/http WebSocket native (Gorilla/Fiber), tidak perlu Socket.IO terpisah
3. **Performance** - Goroutine lebih efisien dari PHP thread untuk concurrent users
4. **Vue.js SPA** - Frontend lebih interaktif, tidak ada page reload saat ujian berlangsung
5. **Server-side timer** - Timer dipegang server, tidak bisa dimanipulasi client
6. **Multi-tenant siap** - Arsitektur didesain dari awal untuk multi-tenant
7. **API-first** - Backend pure REST API, bisa dikonsumsi app mobile nanti

---

## Development Priority

### Phase 1: Core (MVP)
- Auth (login, logout, session)
- Manajemen user & role
- Bank soal (CRUD soal semua tipe)
- Jadwal ujian (CRUD + konfigurasi dasar)
- Flow pengerjaan ujian peserta (start, answer, finish)
- Dashboard pengawas real-time (WebSocket)

### Phase 2: Enhancement
- Import/export soal massal
- Laporan PDF
- Regrade
- Deteksi kecurangan lengkap
- Serial exam

### Phase 3: Advanced
- AI Grading & AI Generate Soal
- Multi-tenant
- Audio streaming soal
- Notifikasi email
- Analytics soal (daya pembeda, tingkat kesukaran)
