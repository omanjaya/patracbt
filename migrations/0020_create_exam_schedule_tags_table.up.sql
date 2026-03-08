CREATE TABLE exam_schedule_tags (
    exam_schedule_id BIGINT NOT NULL REFERENCES exam_schedules(id) ON DELETE CASCADE,
    tag_id           BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (exam_schedule_id, tag_id)
);
