# API Design - CBT Patra

**Tanggal:** 2026-03-05
**Base URL:** https://cbt.domain.com/api/v1
**Auth:** Bearer Token (JWT)
**Format:** JSON

---

## Standard Response Format

### Success Response
```json
{
    "success": true,
    "message": "OK",
    "data": { ... } | [ ... ] | null,
    "meta": {
        "page": 1,
        "per_page": 20,
        "total": 150,
        "total_pages": 8
    }
}
```

### Error Response
```json
{
    "success": false,
    "code": "SESSION_NOT_FOUND",
    "message": "Sesi ujian tidak ditemukan"
}
```

### Validation Error Response
```json
{
    "success": false,
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "errors": {
        "name": "Nama jadwal wajib diisi",
        "start_time": "Waktu mulai harus lebih dari sekarang"
    }
}
```

---

## AUTH Endpoints

### POST /api/v1/auth/login
Request:
```json
{
    "login": "admin",
    "password": "secret",
    "force_login": false
}
```

Response:
```json
{
    "success": true,
    "data": {
        "access_token": "eyJ...",
        "refresh_token": "eyJ...",
        "expires_in": 900,
        "user": {
            "id": 1,
            "name": "Admin Sekolah",
            "username": "admin",
            "role": "admin",
            "avatar_url": null
        }
    }
}
```

### POST /api/v1/auth/logout
Header: Authorization: Bearer {token}

Response:
```json
{ "success": true, "message": "Logout berhasil" }
```

### POST /api/v1/auth/refresh
Request:
```json
{ "refresh_token": "eyJ..." }
```

Response:
```json
{
    "success": true,
    "data": {
        "access_token": "eyJ...",
        "expires_in": 900
    }
}
```

### GET /api/v1/auth/me
Response:
```json
{
    "success": true,
    "data": {
        "id": 1,
        "name": "Administrator",
        "username": "admin",
        "role": "admin",
        "profile": {
            "nis": null,
            "class": null,
            "major": null
        }
    }
}
```

---

## USER Endpoints (Admin)

### GET /api/v1/admin/users
Query: ?page=1&per_page=20&search=budi&role=peserta&rombel_id=1

Response: list users + pagination meta

### POST /api/v1/admin/users
Request:
```json
{
    "name": "Budi Santoso",
    "username": "budi2024",
    "password": "password123",
    "role": "peserta",
    "rombel_ids": [1, 2],
    "profile": {
        "nis": "1234567890",
        "class": "XII IPA 1",
        "year": 2024
    }
}
```

### PUT /api/v1/admin/users/{id}
### DELETE /api/v1/admin/users/{id}
### POST /api/v1/admin/users/import (multipart/form-data)

---

## QUESTION BANK Endpoints (Guru/Admin)

### GET /api/v1/banks
Query: ?search=matematika&subject_id=1

### POST /api/v1/banks
```json
{
    "name": "Bank Soal Matematika",
    "subject_id": 1,
    "description": "Soal-soal untuk Matematika kelas XII"
}
```

### GET /api/v1/banks/{id}
### PUT /api/v1/banks/{id}
### DELETE /api/v1/banks/{id}

---

## QUESTION Endpoints

### GET /api/v1/banks/{bank_id}/questions
Query: ?page=1&type=pg

Response:
```json
{
    "success": true,
    "data": [{
        "id": 1,
        "type": "pg",
        "question_body": "<p>Berapa hasil 2+2?</p>",
        "audio_path": null,
        "options": [
            {"key": "A", "text": "3"},
            {"key": "B", "text": "4"},
            {"key": "C", "text": "5"}
        ],
        "correct_answer_text": {"key": "B"},
        "sort_order": 0,
        "default_mark": 1.0
    }],
    "meta": { "total": 50 }
}
```

### POST /api/v1/banks/{bank_id}/questions
```json
{
    "type": "pg",
    "question_body": "<p>Teks soal...</p>",
    "options": [
        {"key": "A", "text": "Opsi A"},
        {"key": "B", "text": "Opsi B"}
    ],
    "correct_answer_text": {"key": "A"},
    "default_mark": 1.0,
    "sort_order": 0
}
```

### PUT /api/v1/banks/{bank_id}/questions/{id}
### DELETE /api/v1/banks/{bank_id}/questions/{id}
### POST /api/v1/banks/{bank_id}/questions/import (multipart/form-data)
### POST /api/v1/banks/{bank_id}/questions/ai-generate
```json
{
    "topic": "Persamaan Kuadrat",
    "count": 5,
    "type": "pg",
    "difficulty": "medium"
}
```

---

## EXAM SCHEDULE Endpoints

### GET /api/v1/schedules
Query: ?status=active&page=1

### POST /api/v1/schedules
```json
{
    "name": "UH Matematika Kelas XII",
    "question_banks": [
        {"id": 1, "duration_minutes": 90, "weight": 1.0, "sort_order": 0}
    ],
    "duration_minutes": 90,
    "start_time": "2026-03-10T08:00:00+08:00",
    "end_time": "2026-03-10T10:00:00+08:00",
    "token": "ABC123",
    "late_policy": "cut_time",
    "detect_cheating": true,
    "cheating_limit": 3,
    "show_score_after": "immediately",
    "show_report": true,
    "shuffle_questions": true,
    "shuffle_options": true,
    "is_locked_by_pengawas": false,
    "min_working_time": 15,
    "rombels": [{"id": 1, "type": "include"}],
    "tags": [],
    "users": []
}
```

### GET /api/v1/schedules/{id}
### PUT /api/v1/schedules/{id}
### DELETE /api/v1/schedules/{id}

---

## PESERTA Endpoints (Exam Flow)

### GET /api/v1/peserta/dashboard
Response:
```json
{
    "success": true,
    "data": {
        "active_exams": [{
            "schedule_id": 1,
            "name": "UH Matematika",
            "end_time": "2026-03-10T10:00:00Z",
            "has_session": false,
            "session_status": null
        }],
        "upcoming_exams": [],
        "past_exams": []
    }
}
```

### GET /api/v1/peserta/exam/{schedule_id}/confirm
Response:
```json
{
    "success": true,
    "data": {
        "schedule_id": 1,
        "name": "UH Matematika Kelas XII",
        "duration_minutes": 90,
        "question_count": 40,
        "requires_token": true,
        "is_locked_by_pengawas": false,
        "detect_cheating": true,
        "cheating_limit": 3,
        "has_existing_session": false,
        "rules": [
            "Jangan berpindah tab selama ujian berlangsung",
            "Pelanggaran 3x akan mengakhiri ujian otomatis"
        ]
    }
}
```

### POST /api/v1/peserta/exam/{schedule_id}/start
Request:
```json
{ "token": "ABC123" }
```

Response:
```json
{
    "success": true,
    "data": {
        "session_id": "abc123hash",
        "exam_end_time": "2026-03-10T09:47:00Z",
        "question_count": 40
    }
}
```

### GET /api/v1/peserta/exam/session/{session_hash}
Response:
```json
{
    "success": true,
    "data": {
        "session_id": "abc123hash",
        "status": "ongoing",
        "exam_end_time": "2026-03-10T09:47:00Z",
        "violation_count": 0,
        "questions": [{
            "id": 5,
            "number": 1,
            "type": "pg",
            "question_body": "<p>Teks soal...</p>",
            "options": [{"key": "A", "text": "..."}, ...],
            "stimulus": null,
            "audio_path": null
        }],
        "answer_map": {
            "5": {"answer": "A", "is_doubtful": false}
        }
    }
}
```

### POST /api/v1/peserta/exam/session/{session_hash}/answer
Request:
```json
{
    "question_id": 5,
    "answer": "A",
    "is_doubtful": false
}
```

Response:
```json
{ "success": true, "data": { "saved": true } }
```

### POST /api/v1/peserta/exam/session/{session_hash}/toggle-flag
Request:
```json
{ "question_id": 5, "is_doubtful": true }
```

### POST /api/v1/peserta/exam/session/{session_hash}/log-violation
Request:
```json
{ "type": "tab_switch" }
```

Response:
```json
{
    "success": true,
    "data": {
        "result": "LOGGED",
        "violation_count": 1,
        "max_violations": 3
    }
}
```
atau jika terminasi:
```json
{
    "success": true,
    "data": {
        "result": "TERMINATED",
        "violation_count": 3
    }
}
```

### POST /api/v1/peserta/exam/session/{session_hash}/finish
Response:
```json
{
    "success": true,
    "data": {
        "session_id": "abc123hash",
        "status": "completed",
        "score_display": {
            "status": "show",
            "value": 87.5,
            "message": null
        }
    }
}
```

---

## MONITORING Endpoints (Pengawas)

### GET /api/v1/monitoring/{schedule_id}/students
Header: X-Supervision-Token: {token ruang}

Response:
```json
{
    "success": true,
    "data": [{
        "user_id": 100,
        "name": "Budi Santoso",
        "avatar_url": null,
        "status": "ongoing",
        "is_online": true,
        "answered_count": 25,
        "total_questions": 40,
        "violation_count": 1,
        "session_id": "abc123hash"
    }]
}
```

### POST /api/v1/monitoring/{schedule_id}/lock/{user_id}
(Trigger lock via WebSocket ke client peserta)

---

## REPORT Endpoints (Guru/Admin)

### GET /api/v1/report/schedules/{schedule_id}
Response: rekap nilai semua peserta

### POST /api/v1/report/schedules/{schedule_id}/regrade
Trigger regrade semua sesi dalam jadwal

### POST /api/v1/report/sessions/{session_id}/ai-grade-essay
Request:
```json
{ "question_id": 10 }
```

### GET /api/v1/report/schedules/{schedule_id}/export-pdf
Response: binary PDF file

### GET /api/v1/report/schedules/{schedule_id}/analysis
Query: ?rombel_id=1&include_ongoing=false

Response:
```json
{
    "success": true,
    "data": {
        "general_stats": {
            "mean": 72.5,
            "median": 75.0,
            "mode": 80.0,
            "min": 30.0,
            "max": 100.0,
            "std_dev": 15.3,
            "participants": 35
        },
        "item_analysis": [{
            "question_id": 1,
            "number": 1,
            "type": "pg",
            "difficulty_index": 0.72,
            "difficulty_label": "Mudah",
            "discriminant_index": 0.45,
            "discriminant_label": "Sangat Baik",
            "correct_count": 25,
            "upper_group_correct": 9,
            "lower_group_correct": 4
        }]
    }
}
```

---

## PERSONAL REPORT Endpoints (Peserta)

### GET /api/v1/peserta/exam/session/{session_hash}/report
Response:
```json
{
    "success": true,
    "data": {
        "session": {
            "id": "abc123hash",
            "score": 87.5,
            "status": "completed"
        },
        "is_multi_stage": false,
        "report": [{
            "question": {
                "id": 1,
                "type": "pg",
                "question_body": "<p>...</p>",
                "options": [...]
            },
            "student_answer": {"option_index": 1},
            "score": 1.0,
            "max_score": 1.0,
            "is_correct": true
        }],
        "grouped_report": null
    }
}
```

---

## SETTINGS Endpoints (Admin)

### GET /api/v1/admin/settings
Response:
```json
{
    "success": true,
    "data": {
        "app_name": "CBT Patra",
        "login_qr_enabled": "1",
        "login_method": "normal",
        "websocket_enabled": "1",
        "exam_nav_style": "classic",
        "panic_mode_active": "0",
        "ai_api_url": "",
        "ai_api_header": "Authorization"
    }
}
```

### POST /api/v1/admin/settings
Request:
```json
{
    "app_name": "CBT Sekolah XYZ",
    "login_qr_enabled": "1",
    "websocket_enabled": "1"
}
```

### POST /api/v1/admin/settings/panic-mode
Request:
```json
{ "active": true }
```

### POST /api/v1/admin/settings/login-method
Request:
```json
{ "method": "redis" }
```

---

## BACKUP & RESTORE Endpoints (Admin)

### POST /api/v1/admin/backup
Request:
```json
{ "batch_id": "bk_20260310_1234" }
```

Response:
```json
{
    "success": true,
    "data": {
        "download_url": "/api/v1/admin/backup/download/backup-patra-2026-03-10-123456.patrabak"
    }
}
```

### GET /api/v1/admin/backup/download/{filename}
Response: binary .patrabak file

### POST /api/v1/admin/backup/progress
Request:
```json
{ "batch_id": "bk_20260310_1234" }
```

Response:
```json
{ "success": true, "data": { "percent": 65, "message": "Mengarsipkan File: 120/200" } }
```

### POST /api/v1/admin/restore (multipart/form-data)
Fields: backup_file (.patrabak), batch_id

### POST /api/v1/admin/restore/chunk (multipart/form-data)
Fields: chunk (binary), batch_id, chunk_index

### POST /api/v1/admin/restore/process
Request:
```json
{ "batch_id": "rs_20260310_5678" }
```

---

## MASTER DATA Endpoints (Admin)

### Rombels
```
GET    /api/v1/admin/rombels
POST   /api/v1/admin/rombels
PUT    /api/v1/admin/rombels/{id}
DELETE /api/v1/admin/rombels/{id}
```

### Subjects
```
GET    /api/v1/admin/subjects
POST   /api/v1/admin/subjects
PUT    /api/v1/admin/subjects/{id}
DELETE /api/v1/admin/subjects/{id}
```

### Tags
```
GET    /api/v1/admin/tags
POST   /api/v1/admin/tags
PUT    /api/v1/admin/tags/{id}
DELETE /api/v1/admin/tags/{id}
```

### Rooms
```
GET    /api/v1/admin/rooms
POST   /api/v1/admin/rooms
PUT    /api/v1/admin/rooms/{id}
DELETE /api/v1/admin/rooms/{id}
```

---

## HEALTH CHECK

### GET /api/v1/health
Response:
```json
{
    "success": true,
    "data": {
        "status": "healthy",
        "postgres": "connected",
        "redis": "connected",
        "uptime": "72h15m",
        "version": "1.0.0"
    }
}
```

---

## WEBSOCKET Endpoint

### WS /ws/exam/{schedule_id}
Header: Authorization: Bearer {token}

#### Client -> Server Events

```json
// join room
{ "type": "join", "data": {"role": "peserta"|"pengawas", "token": "..."} }

// heartbeat (setiap 30 detik)
{ "type": "heartbeat" }
```

#### Server -> Client Events

```json
// ke peserta: dikunci pengawas
{ "type": "lock_client", "data": {} }

// ke peserta: sinkronisasi waktu server
{ "type": "time_sync", "data": {"server_time": "2026-03-10T..."} }

// ke pengawas: peserta join
{ "type": "student_joined", "data": {"user_id": 100, "name": "Budi"} }

// ke pengawas: peserta offline
{ "type": "student_left", "data": {"user_id": 100} }

// ke pengawas: jawaban tersimpan
{ "type": "answer_saved", "data": {"user_id": 100, "answered_count": 26} }

// ke pengawas: pelanggaran
{ "type": "violation_logged", "data": {"user_id": 100, "count": 2, "type": "tab_switch"} }

// ke pengawas: peserta selesai
{ "type": "session_finished", "data": {"user_id": 100, "status": "completed"} }
```

---

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| POST /auth/login | 10 req/menit per IP |
| POST /exam/session/*/answer | 60 req/menit per user |
| POST /exam/session/*/log-violation | 20 req/menit per user |
| POST /banks/*/questions/ai-generate | 3 req/menit per user |
| Semua endpoint lain | 120 req/menit per user |
