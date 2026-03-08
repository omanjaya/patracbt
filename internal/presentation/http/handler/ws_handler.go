package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/omanjaya/patra/internal/infrastructure/websocket"
	"github.com/omanjaya/patra/pkg/response"
)

// BUG-12 fix: WSHandler menyimpan allowed origins untuk validasi CheckOrigin
type WSHandler struct {
	hub            *ws.Hub
	allowedOrigins []string
}

func NewWSHandler(hub *ws.Hub, allowedOrigins string) *WSHandler {
	origins := strings.Split(allowedOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return &WSHandler{hub: hub, allowedOrigins: origins}
}

func (h *WSHandler) upgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// BUG-12 fix: validasi origin dari config, bukan accept semua
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // same-origin request (non-browser)
			}
			for _, allowed := range h.allowedOrigins {
				if allowed == "*" || strings.EqualFold(origin, allowed) {
					return true
				}
			}
			return false
		},
	}
}

// GET /ws/exam/:scheduleId  (upgraded to WebSocket)
func (h *WSHandler) HandleExam(c *gin.Context) {
	userID := c.GetUint("user_id")
	role := c.GetString("role")

	scheduleID, err := strconv.ParseUint(c.Param("scheduleId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID")
		return
	}

	var sessionID uint64
	if sid := c.Query("session_id"); sid != "" {
		sessionID, _ = strconv.ParseUint(sid, 10, 64)
	}

	// Per-user connection limit: reject upgrade if user already has 3+ connections.
	if h.hub.CountUserConnections(userID) >= 3 {
		response.Error(c, http.StatusTooManyRequests, "TOO_MANY_CONNECTIONS", "Terlalu banyak koneksi WebSocket aktif")
		return
	}

	upgrader := h.upgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(h.hub, conn)
	rc := &ws.RoomClient{
		Client:     client,
		ScheduleID: uint(scheduleID),
		UserID:     userID,
		Role:       role,
		SessionID:  uint(sessionID),
	}

	h.hub.Register <- rc

	go client.WritePump()
	client.ReadPump(rc) // blocks until disconnect
}

// GET /api/v1/monitoring/:scheduleId/clients
func (h *WSHandler) GetRoomClients(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("scheduleId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID")
		return
	}

	clients := h.hub.GetRoomClients(uint(scheduleID))
	list := make([]gin.H, 0, len(clients))
	for _, rc := range clients {
		list = append(list, gin.H{
			"user_id":    rc.UserID,
			"role":       rc.Role,
			"session_id": rc.SessionID,
		})
	}
	response.Success(c, gin.H{"clients": list})
}

// POST /api/v1/monitoring/:scheduleId/lock
func (h *WSHandler) LockClient(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("scheduleId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID")
		return
	}

	var req struct {
		TargetUserID uint   `json:"target_user_id" binding:"required"`
		Message      string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	msg := req.Message
	if msg == "" {
		msg = "Akses dikunci oleh pengawas"
	}

	h.hub.SendToUser(uint(scheduleID), req.TargetUserID, ws.Message{
		Event: ws.EventLockClient,
		Data:  ws.LockClientPayload{TargetUserID: req.TargetUserID, Message: msg},
	})

	response.Success(c, gin.H{"locked": true})
}
