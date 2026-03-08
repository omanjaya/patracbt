-- user_rooms is a join table linking users to rooms
-- Note: user_profiles already has room_id, but this allows many-to-many if needed
CREATE TABLE user_rooms (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, room_id)
);
