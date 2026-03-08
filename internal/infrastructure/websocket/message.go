package websocket

const (
	EventJoin            = "join"
	EventHeartbeat       = "heartbeat"
	EventStudentJoined   = "student_joined"
	EventStudentLeft     = "student_left"
	EventAnswerSaved     = "answer_saved"
	EventViolationLogged = "violation_logged"
	EventSessionFinished = "session_finished"
	EventLockClient      = "lock_client"
	EventTimeSync        = "time_sync"
	EventForceFinish     = "force_finish"
	EventTimeExtended    = "time_extended"
	EventChatMessage     = "chat_message"
	EventPanicMode       = "panic_mode"
	EventForceLogout     = "force_logout"
	EventAnswerBatch     = "answer_batch"
)

type Message struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type AnswerSavedPayload struct {
	SessionID  uint `json:"session_id"`
	UserID     uint `json:"user_id"`
	QuestionID uint `json:"question_id"`
	Answered   int  `json:"answered"`
	Total      int  `json:"total"`
}

type ViolationPayload struct {
	SessionID     uint   `json:"session_id"`
	UserID        uint   `json:"user_id"`
	ViolationType string `json:"violation_type"`
	Count         int    `json:"violation_count"`
}

type SessionFinishedPayload struct {
	SessionID uint    `json:"session_id"`
	UserID    uint    `json:"user_id"`
	Score     float64 `json:"score"`
	MaxScore  float64 `json:"max_score"`
}

type LockClientPayload struct {
	TargetUserID uint   `json:"target_user_id"`
	Message      string `json:"message"`
}

type TimeSyncPayload struct {
	ServerTime string `json:"server_time"`
}

type ForceFinishPayload struct {
	SessionID uint   `json:"session_id"`
	UserID    uint   `json:"user_id"`
	Message   string `json:"message"`
}

type TimeExtendedPayload struct {
	SessionID  uint   `json:"session_id"`
	UserID     uint   `json:"user_id"`
	AddMinutes int    `json:"add_minutes"`
	NewEndTime string `json:"new_end_time"` // ISO8601
}

type ChatMessagePayload struct {
	SenderName string `json:"sender_name"`
	Message    string `json:"message"`
}

type PanicModePayload struct {
	Active  bool   `json:"active"`
	Message string `json:"message"`
}

type ForceLogoutPayload struct {
	UserID  uint   `json:"user_id"`
	Message string `json:"message"`
}
