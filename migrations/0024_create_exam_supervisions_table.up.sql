CREATE TABLE exam_supervisions (
    id               BIGSERIAL PRIMARY KEY,
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    room_id          BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    token            VARCHAR(10) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_supervision_schedule_room ON exam_supervisions (exam_schedule_id, room_id);
