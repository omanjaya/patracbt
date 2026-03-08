CREATE TABLE exam_schedule_rombels (
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    rombel_id        BIGINT NOT NULL REFERENCES rombels(id) ON DELETE CASCADE,
    PRIMARY KEY (exam_schedule_id, rombel_id)
);
