# Database Schema - CBT Patra

**Tanggal:** 2026-03-05
**Database:** PostgreSQL 16+
**ORM:** GORM v2

---

## Entity Relationship Diagram (ASCII)

```
users
  |
  +--< user_profiles (1:1)
  |
  +--< user_rombels (N:M)  >-- rombels
  |
  +--< user_tags (N:M)  >-- tags
  |
  +--< exam_sessions (1:N)
  |       |
  |       +--< exam_answers (1:N)
  |
  +--< question_banks (1:N, as creator)
  |       |
  |       +--< questions (1:N)
  |       |       |
  |       |       +-- stimuli (N:1, optional)
  |       |
  |       +--< exam_schedule_question_banks (N:M)
  |
  +--< exam_schedules (1:N, as creator)
          |
          +--< exam_schedule_rombels (N:M)  >-- rombels
          |
          +--< exam_schedule_tags (N:M)  >-- tags
          |
          +--< exam_schedule_users (N:M)  >-- users
          |
          +--< exam_sessions (1:N)
          |
          +--< exam_supervisions (1:N)
          |       |
          |       +-- rooms (N:1)
          |
          +--< regrade_logs (1:N)

subjects
  |
  +--< question_banks (1:N)

settings (key-value store)
```

---

## Table Definitions

### users
```sql
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    username        VARCHAR(100) UNIQUE NOT NULL,
    email           VARCHAR(255) UNIQUE,
    password        VARCHAR(255) NOT NULL,               -- bcrypt hash
    role            VARCHAR(50) NOT NULL DEFAULT 'peserta', -- admin, guru, pengawas, peserta
    avatar_path     VARCHAR(500),
    last_login_at   TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,                         -- soft delete
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
```

### user_profiles
```sql
CREATE TABLE user_profiles (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nis         VARCHAR(50),                             -- Nomor Induk Siswa
    nip         VARCHAR(50),                             -- Nomor Induk Pegawai
    class       VARCHAR(50),                             -- Kelas (XII IPA 1)
    major       VARCHAR(100),                            -- Jurusan
    year        SMALLINT,                                -- Angkatan
    phone       VARCHAR(20),
    address     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### rombels
```sql
CREATE TABLE rombels (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    grade_level VARCHAR(50),                             -- X, XI, XII, etc.
    description TEXT,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### user_rombels (pivot)
```sql
CREATE TABLE user_rombels (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rombel_id   BIGINT NOT NULL REFERENCES rombels(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, rombel_id)
);

CREATE INDEX idx_user_rombels_rombel_id ON user_rombels(rombel_id);
```

### tags
```sql
CREATE TABLE tags (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    color       VARCHAR(20) DEFAULT '#6B7280',           -- hex color
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### user_tags (pivot)
```sql
CREATE TABLE user_tags (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, tag_id)
);
```

### subjects
```sql
CREATE TABLE subjects (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    code        VARCHAR(50),
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### question_banks
```sql
CREATE TABLE question_banks (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    subject_id  BIGINT REFERENCES subjects(id) ON DELETE SET NULL,
    user_id     BIGINT NOT NULL REFERENCES users(id),    -- creator
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT FALSE,          -- false = draft, true = published
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_question_banks_subject_id ON question_banks(subject_id);
CREATE INDEX idx_question_banks_user_id ON question_banks(user_id);
```

### stimuli
```sql
CREATE TABLE stimuli (
    id              BIGSERIAL PRIMARY KEY,
    question_bank_id BIGINT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,                       -- HTML narasi/bacaan
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### questions
```sql
CREATE TABLE questions (
    id                  BIGSERIAL PRIMARY KEY,
    question_bank_id    BIGINT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    stimulus_id         BIGINT REFERENCES stimuli(id) ON DELETE SET NULL,
    type                VARCHAR(50) NOT NULL,            -- pg, pgk, esai, menjodohkan, singkat, matrix, bs
    question_body       TEXT NOT NULL,                   -- HTML soal
    audio_path          VARCHAR(500),                    -- path file audio
    audio_limit         SMALLINT DEFAULT 0,              -- 0 = unlimited
    options             JSONB,                           -- opsi jawaban
    correct_answer_text JSONB,                           -- kunci jawaban
    sort_order          INTEGER DEFAULT 0,
    default_mark        DECIMAL(8,2) DEFAULT 1.0,        -- bobot nilai
    options_updated_at  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_questions_bank_id ON questions(question_bank_id);
CREATE INDEX idx_questions_stimulus_id ON questions(stimulus_id);
CREATE INDEX idx_questions_sort_order ON questions(question_bank_id, sort_order);
```

**Format JSONB `options` (contoh PG):**
```json
[
    {"key": "A", "text": "Jakarta", "image": null},
    {"key": "B", "text": "Surabaya", "image": null},
    {"key": "C", "text": "Bandung", "image": null}
]
```

**Format JSONB `correct_answer_text`:**
```json
{"key": "A"}                          -- PG (single answer)
{"keys": ["A", "C"]}                   -- PGK (multiple answer)
{"text": "Soekarno"}                   -- Isian Singkat
{"pairs": {"A": "1", "B": "3"}}        -- Menjodohkan
{"grid": {"Fakta": [0], "Opini": [1]}} -- Matrix
```

### exam_schedules
```sql
CREATE TABLE exam_schedules (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(255) NOT NULL,
    user_id                 BIGINT NOT NULL REFERENCES users(id),   -- creator
    duration_minutes        INTEGER NOT NULL DEFAULT 90,
    start_time              TIMESTAMPTZ NOT NULL,
    end_time                TIMESTAMPTZ NOT NULL,
    token                   VARCHAR(100),                           -- optional entry token
    late_policy             VARCHAR(50) DEFAULT 'cut_time',         -- cut_time | full_time
    detect_cheating         BOOLEAN NOT NULL DEFAULT FALSE,
    cheating_limit          SMALLINT DEFAULT 3,
    show_score_after        VARCHAR(50) DEFAULT 'after_end_time',   -- immediately | after_end_time | manual
    show_report             BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_questions       BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_options         BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked_by_pengawas   BOOLEAN NOT NULL DEFAULT FALSE,
    min_working_time        INTEGER DEFAULT 0,                      -- menit minimal sebelum boleh selesai
    next_exam_schedule_id   BIGINT REFERENCES exam_schedules(id),   -- serial exam
    last_graded_at          TIMESTAMPTZ,
    deleted_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exam_schedules_time ON exam_schedules(start_time, end_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_exam_schedules_user_id ON exam_schedules(user_id);
```

### exam_schedule_question_banks (pivot many-to-many)
```sql
CREATE TABLE exam_schedule_question_banks (
    exam_schedule_id    BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    question_bank_id    BIGINT NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    duration_minutes    INTEGER,                         -- durasi khusus untuk bank soal ini
    min_duration_minutes INTEGER,                        -- waktu minimal khusus
    description         TEXT,
    sort_order          INTEGER DEFAULT 0,               -- urutan tampil bank soal
    weight              DECIMAL(5,2) DEFAULT 1.0,        -- bobot bank soal dalam nilai akhir
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (exam_schedule_id, question_bank_id)
);
```

### exam_schedule_rombels (pivot)
```sql
CREATE TABLE exam_schedule_rombels (
    exam_schedule_id    BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    rombel_id           BIGINT NOT NULL REFERENCES rombels(id) ON DELETE CASCADE,
    type                VARCHAR(20) DEFAULT 'include',   -- include | exclude
    PRIMARY KEY (exam_schedule_id, rombel_id)
);
```

### exam_schedule_tags (pivot)
```sql
CREATE TABLE exam_schedule_tags (
    exam_schedule_id    BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    tag_id              BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    type                VARCHAR(20) DEFAULT 'include',   -- include | exclude
    PRIMARY KEY (exam_schedule_id, tag_id)
);
```

### exam_schedule_users (pivot)
```sql
CREATE TABLE exam_schedule_users (
    exam_schedule_id    BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type                VARCHAR(20) DEFAULT 'include',   -- include | exclude
    PRIMARY KEY (exam_schedule_id, user_id)
);
```

### exam_sessions
```sql
CREATE TABLE exam_sessions (
    id                      BIGSERIAL PRIMARY KEY,
    exam_schedule_id        BIGINT NOT NULL REFERENCES exam_schedules(id),
    user_id                 BIGINT NOT NULL REFERENCES users(id),
    start_time              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_time                TIMESTAMPTZ,                            -- NULL = masih ongoing
    score                   DECIMAL(8,2),
    status                  VARCHAR(50) NOT NULL DEFAULT 'ongoing', -- ongoing | completed | terminated
    violation_count         INTEGER NOT NULL DEFAULT 0,
    session_details         JSONB,                                  -- {question_order: [1,5,3,...], option_orders: {...}}
    regraded_at             TIMESTAMPTZ,
    original_score          DECIMAL(8,2),                          -- backup score sebelum regrade
    score_change_notified   BOOLEAN DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exam_sessions_schedule_id ON exam_sessions(exam_schedule_id);
CREATE INDEX idx_exam_sessions_user_id ON exam_sessions(user_id);
CREATE INDEX idx_exam_sessions_status ON exam_sessions(status);
CREATE UNIQUE INDEX idx_exam_sessions_user_schedule ON exam_sessions(user_id, exam_schedule_id)
    WHERE status != 'terminated';
```

### exam_answers
```sql
CREATE TABLE exam_answers (
    id              BIGSERIAL PRIMARY KEY,
    exam_session_id BIGINT NOT NULL REFERENCES exam_sessions(id) ON DELETE CASCADE,
    question_id     BIGINT NOT NULL REFERENCES questions(id),
    answer          JSONB,                                          -- jawaban siswa (flexible format)
    is_doubtful     BOOLEAN NOT NULL DEFAULT FALSE,
    score           DECIMAL(8,2),                                  -- NULL = belum dinilai (esai)
    graded_by_ai    BOOLEAN DEFAULT FALSE,
    ai_reason       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exam_session_id, question_id)
);

CREATE INDEX idx_exam_answers_session_id ON exam_answers(exam_session_id);
CREATE INDEX idx_exam_answers_question_id ON exam_answers(question_id);
```

### rooms
```sql
CREATE TABLE rooms (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    capacity    INTEGER DEFAULT 30,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### exam_supervisions
```sql
CREATE TABLE exam_supervisions (
    id                  BIGSERIAL PRIMARY KEY,
    exam_schedule_id    BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    room_id             BIGINT REFERENCES rooms(id),
    token               VARCHAR(100) NOT NULL UNIQUE,               -- token untuk pengawas masuk
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### regrade_logs
```sql
CREATE TABLE regrade_logs (
    id              BIGSERIAL PRIMARY KEY,
    exam_session_id BIGINT NOT NULL REFERENCES exam_sessions(id),
    old_score       DECIMAL(8,2),
    new_score       DECIMAL(8,2),
    reason          TEXT,
    regraded_by     BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### settings
```sql
CREATE TABLE settings (
    id          BIGSERIAL PRIMARY KEY,
    key         VARCHAR(100) UNIQUE NOT NULL,
    value       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed data awal
INSERT INTO settings (key, value) VALUES
    ('app_name', 'CBT Patra'),
    ('ai_api_url', ''),
    ('ai_api_key', ''),
    ('ai_api_header', 'Authorization'),
    ('ai_model_params', '{}');
```

---

## Indexes Summary

| Table | Index | Reason |
|-------|-------|--------|
| users | username, email | Login lookup |
| question_banks | subject_id, user_id | Filter by subject/creator |
| questions | question_bank_id, sort_order | List soal dalam bank |
| exam_schedules | start_time, end_time | Filter jadwal aktif |
| exam_sessions | schedule_id, user_id, status | Dashboard & monitoring queries |
| exam_answers | session_id, question_id | Load semua jawaban sesi |

---

## GORM Model (Contoh)

```go
// internal/domain/entity/exam_session.go

package entity

import (
    "time"
    "gorm.io/gorm"
)

type ExamSession struct {
    ID                  uint           `gorm:"primaryKey"`
    ExamScheduleID      uint           `gorm:"not null;index"`
    UserID              uint           `gorm:"not null;index"`
    StartTime           time.Time      `gorm:"not null;default:now()"`
    EndTime             *time.Time
    Score               *float64       `gorm:"type:decimal(8,2)"`
    Status              string         `gorm:"not null;default:'ongoing'"`
    ViolationCount      int            `gorm:"not null;default:0"`
    SessionDetails      SessionDetails `gorm:"type:jsonb;serializer:json"`
    RegradadAt          *time.Time
    OriginalScore       *float64       `gorm:"type:decimal(8,2)"`
    ScoreChangeNotified bool           `gorm:"default:false"`
    CreatedAt           time.Time
    UpdatedAt           time.Time

    // Relations
    ExamSchedule ExamSchedule `gorm:"foreignKey:ExamScheduleID"`
    User         User         `gorm:"foreignKey:UserID"`
    Answers      []ExamAnswer `gorm:"foreignKey:ExamSessionID"`
}

type SessionDetails struct {
    QuestionOrder []uint            `json:"question_order"`
    OptionOrders  map[string][]int  `json:"option_orders,omitempty"`
}

func (s *ExamSession) IsOngoing() bool  { return s.Status == "ongoing" }
func (s *ExamSession) IsCompleted() bool { return s.Status == "completed" }
func (s *ExamSession) IsTerminated() bool { return s.Status == "terminated" }

func (s *ExamSession) MarkAsCompleted() {
    now := time.Now()
    s.Status = "completed"
    s.EndTime = &now
}

func (s *ExamSession) MarkAsTerminated() {
    now := time.Now()
    s.Status = "terminated"
    s.EndTime = &now
}
```
