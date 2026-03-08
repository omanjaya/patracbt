package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

// CachedSession holds essential session data stored in Redis.
type CachedSession struct {
	ID             uint            `json:"id"`
	UserID         uint            `json:"user_id"`
	Status         string          `json:"status"`
	EndTime        *time.Time      `json:"end_time"`
	ExamScheduleID uint            `json:"exam_schedule_id"`
	QuestionOrder  json.RawMessage `json:"question_order"`
}

// CachedAnswer holds a single answer stored in the Redis hash buffer.
type CachedAnswer struct {
	QuestionID uint            `json:"question_id"`
	Answer     json.RawMessage `json:"answer"`
	IsFlagged  bool            `json:"is_flagged"`
	AnsweredAt time.Time       `json:"answered_at"`
}

// ExamCache provides Redis-backed caching for active exam sessions and answer buffering.
type ExamCache struct {
	rdb *redis.Client
}

// NewExamCache creates a new ExamCache.
func NewExamCache(rdb *redis.Client) *ExamCache {
	return &ExamCache{rdb: rdb}
}

// ─── Session Cache ─────────────────────────────────────────────

func sessionKey(sessionID uint) string {
	return fmt.Sprintf("exam:session:%d", sessionID)
}

// CacheSession stores essential session data in Redis with the given TTL.
func (c *ExamCache) CacheSession(session *entity.ExamSession, ttl time.Duration) error {
	cached := CachedSession{
		ID:             session.ID,
		UserID:         session.UserID,
		Status:         session.Status,
		EndTime:        session.EndTime,
		ExamScheduleID: session.ExamScheduleID,
		QuestionOrder:  json.RawMessage(session.QuestionOrder),
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return c.rdb.Set(context.Background(), sessionKey(session.ID), data, ttl).Err()
}

// GetSession retrieves a cached session from Redis. Returns redis.Nil error on cache miss.
func (c *ExamCache) GetSession(sessionID uint) (*CachedSession, error) {
	data, err := c.rdb.Get(context.Background(), sessionKey(sessionID)).Bytes()
	if err != nil {
		return nil, err
	}
	var s CachedSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// InvalidateSession removes the cached session from Redis.
func (c *ExamCache) InvalidateSession(sessionID uint) {
	ctx := context.Background()
	if err := c.rdb.Del(ctx, sessionKey(sessionID)).Err(); err != nil {
		logger.Log.Errorf("InvalidateSession %d: %v", sessionID, err)
	}
}

// ─── Answer Buffer ─────────────────────────────────────────────

func answerKey(sessionID uint) string {
	return fmt.Sprintf("exam:answers:%d", sessionID)
}

// SaveAnswer writes a single answer to the Redis hash buffer (HSET).
func (c *ExamCache) SaveAnswer(sessionID, questionID uint, answer json.RawMessage, isFlagged bool) error {
	cached := CachedAnswer{
		QuestionID: questionID,
		Answer:     answer,
		IsFlagged:  isFlagged,
		AnsweredAt: time.Now(),
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	field := fmt.Sprintf("%d", questionID)
	key := answerKey(sessionID)
	if err := c.rdb.HSet(context.Background(), key, field, data).Err(); err != nil {
		return err
	}
	// Safety-net TTL on answer buffer keys to prevent orphan accumulation
	// if the flusher fails to clean up. 24h is generous — flusher runs every 5s.
	c.rdb.Expire(context.Background(), key, 24*time.Hour)
	return nil
}

// SaveAnswerBatch writes multiple answers to the Redis hash buffer using a pipeline.
func (c *ExamCache) SaveAnswerBatch(sessionID uint, answers []CachedAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	ctx := context.Background()
	pipe := c.rdb.Pipeline()
	key := answerKey(sessionID)
	for _, a := range answers {
		data, err := json.Marshal(a)
		if err != nil {
			logger.Log.Warnf("SaveAnswerBatch: failed to marshal answer for question %d in session %d: %v", a.QuestionID, sessionID, err)
			continue
		}
		field := fmt.Sprintf("%d", a.QuestionID)
		pipe.HSet(ctx, key, field, data)
	}
	_, err := pipe.Exec(ctx)
	// Safety-net TTL for answer buffer
	if err == nil {
		c.rdb.Expire(ctx, key, 24*time.Hour)
	}
	return err
}

// GetAnswerCount returns HLEN of the answer buffer (number of distinct questions answered).
// GetBufferedAnswer returns a single cached answer from the Redis hash buffer.
func (c *ExamCache) GetBufferedAnswer(sessionID, questionID uint) (*CachedAnswer, error) {
	field := fmt.Sprintf("%d", questionID)
	data, err := c.rdb.HGet(context.Background(), answerKey(sessionID), field).Result()
	if err != nil {
		return nil, err
	}
	var a CachedAnswer
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *ExamCache) GetAnswerCount(sessionID uint) (int64, error) {
	return c.rdb.HLen(context.Background(), answerKey(sessionID)).Result()
}

// GetAllBufferedAnswers returns all answers from the Redis hash buffer.
func (c *ExamCache) GetAllBufferedAnswers(sessionID uint) ([]CachedAnswer, error) {
	result, err := c.rdb.HGetAll(context.Background(), answerKey(sessionID)).Result()
	if err != nil {
		return nil, err
	}
	answers := make([]CachedAnswer, 0, len(result))
	for _, v := range result {
		var a CachedAnswer
		if err := json.Unmarshal([]byte(v), &a); err != nil {
			continue
		}
		answers = append(answers, a)
	}
	return answers, nil
}

// ClearAnswerBuffer removes the entire answer hash for a session.
func (c *ExamCache) ClearAnswerBuffer(sessionID uint) {
	if err := c.rdb.Del(context.Background(), answerKey(sessionID)).Err(); err != nil {
		logger.Log.Errorf("ClearAnswerBuffer %d: %v", sessionID, err)
	}
}

// GetDirtySessionIDs returns all session IDs that have buffered answers in Redis.
// Uses SCAN instead of KEYS to avoid blocking Redis on large datasets.
func (c *ExamCache) GetDirtySessionIDs(ctx context.Context) ([]uint, error) {
	var ids []uint
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, "exam:answers:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			var id uint
			if _, err := fmt.Sscanf(k, "exam:answers:%d", &id); err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return ids, nil
}

// IsAvailable returns true if Redis is reachable.
func (c *ExamCache) IsAvailable() bool {
	return c.rdb.Ping(context.Background()).Err() == nil
}

// ToEntity converts a CachedSession to an entity.ExamSession (minimal fields).
func (cs *CachedSession) ToEntity() *entity.ExamSession {
	return &entity.ExamSession{
		ID:             cs.ID,
		UserID:         cs.UserID,
		Status:         cs.Status,
		EndTime:        cs.EndTime,
		ExamScheduleID: cs.ExamScheduleID,
		QuestionOrder:  types.JSON(cs.QuestionOrder),
	}
}
