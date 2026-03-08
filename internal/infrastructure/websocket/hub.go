package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/omanjaya/patra/pkg/logger"
)

// RoomClient wraps a WebSocket client with room/role metadata.
type RoomClient struct {
	Client     *Client
	ScheduleID uint
	UserID     uint
	Role       string // "peserta" | "pengawas" | "admin"
	SessionID  uint
}

// pendingBroadcast accumulates answer events for batched delivery.
type pendingBroadcast struct {
	scheduleID uint
	answers    []AnswerSavedPayload
	timer      *time.Timer
}

// Hub manages rooms keyed by exam schedule ID.
type Hub struct {
	mu           sync.RWMutex
	rooms        map[uint]map[*Client]*RoomClient
	clientToRoom map[*Client]uint // reverse map: O(1) room lookup on unregister
	Register     chan *RoomClient
	Unregister   chan *Client
	broadcast    chan roomMsg
	done         chan struct{}

	supervisorBatchMu sync.Mutex
	supervisorBatch   map[uint]*pendingBroadcast
}

type roomMsg struct {
	scheduleID uint
	data       []byte
}

// NewHub creates a Hub with buffered channels.
//
// Buffer sizing rationale (targeting up to ~2000 concurrent exam participants):
//   - Register/Unregister (256): handles burst joins at exam start when many
//     students connect within seconds. 256 provides headroom for ~2000 users
//     joining over a few seconds while the Run loop drains the channel.
//   - broadcast (1024): answer saves and status updates can spike when many
//     students submit simultaneously. 1024 keeps the pipeline non-blocking
//     under normal load with comfortable margin.
func NewHub() *Hub {
	return &Hub{
		rooms:           make(map[uint]map[*Client]*RoomClient),
		clientToRoom:    make(map[*Client]uint),
		Register:        make(chan *RoomClient, 256),
		Unregister:      make(chan *Client, 256),
		broadcast:       make(chan roomMsg, 1024),
		done:            make(chan struct{}),
		supervisorBatch: make(map[uint]*pendingBroadcast),
	}
}

// Stop signals the Run loop to close all client connections and return.
func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for c := range h.clientToRoom {
				c.closeSend()
			}
			h.mu.Unlock()
			// Stop all pending supervisor batch timers to prevent goroutine leaks
			h.supervisorBatchMu.Lock()
			for _, pb := range h.supervisorBatch {
				if pb.timer != nil {
					pb.timer.Stop()
				}
			}
			h.supervisorBatchMu.Unlock()
			return
		case rc := <-h.Register:
			h.mu.Lock()
			if h.rooms[rc.ScheduleID] == nil {
				h.rooms[rc.ScheduleID] = make(map[*Client]*RoomClient)
			}
			if len(h.rooms[rc.ScheduleID]) >= 2000 {
				h.mu.Unlock()
				logger.Log.Warnf("hub: room %d full (2000 clients), rejecting connection for user %d", rc.ScheduleID, rc.UserID)
				rc.Client.closeSend()
				continue
			}
			h.rooms[rc.ScheduleID][rc.Client] = rc
			h.clientToRoom[rc.Client] = rc.ScheduleID
			h.mu.Unlock()

			// Post-register validation: send a time_sync message to verify
			// the client connection is still alive. If the send channel is
			// full or closed, unregister the client immediately.
			welcomeMsg, err := json.Marshal(Message{
				Event: EventTimeSync,
				Data:  TimeSyncPayload{ServerTime: time.Now().UTC().Format(time.RFC3339)},
			})
			if err != nil {
				logger.Log.Errorf("hub: failed to marshal welcome message: %v", err)
				continue
			}
			select {
			case rc.Client.send <- welcomeMsg:
				// Connection is responsive.
			default:
				// Send buffer full or closed — client is not viable.
				logger.Log.Warnf("hub: post-register send failed for user %d in schedule %d, unregistering", rc.UserID, rc.ScheduleID)
				h.mu.Lock()
				delete(h.clientToRoom, rc.Client)
				if room, exists := h.rooms[rc.ScheduleID]; exists {
					delete(room, rc.Client)
					if len(room) == 0 {
						delete(h.rooms, rc.ScheduleID)
					}
				}
				h.mu.Unlock()
				continue
			}

			if rc.Role == "peserta" {
				h.broadcastToSupervisors(rc.ScheduleID, Message{
					Event: EventStudentJoined,
					Data:  map[string]any{"user_id": rc.UserID, "session_id": rc.SessionID},
				})
			}

		case c := <-h.Unregister:
			h.mu.Lock()
			var leftRC *RoomClient
			if roomID, ok := h.clientToRoom[c]; ok {
				delete(h.clientToRoom, c)
				if room, exists := h.rooms[roomID]; exists {
					leftRC = room[c]
					delete(room, c)
					if len(room) == 0 {
						delete(h.rooms, roomID)
					}
				}
			}
			h.mu.Unlock()
			if leftRC != nil && leftRC.Role == "peserta" {
				h.broadcastToSupervisors(leftRC.ScheduleID, Message{
					Event: EventStudentLeft,
					Data:  map[string]any{"user_id": leftRC.UserID, "session_id": leftRC.SessionID},
				})
			}

		case rb := <-h.broadcast:
			h.mu.RLock()
			room := h.rooms[rb.scheduleID]
			clients := make([]*Client, 0, len(room))
			for c := range room {
				clients = append(clients, c)
			}
			h.mu.RUnlock()

			for _, c := range clients {
				select {
				case c.send <- rb.data:
				default:
					logger.Log.Debugf("hub: broadcast dropped message for client in schedule %d (send buffer full)", rb.scheduleID)
				}
			}
		}
	}
}

// Broadcast sends a message to all clients in a room.
// Uses a non-blocking send to prevent caller goroutines from stalling
// if the broadcast channel is temporarily full under extreme load.
func (h *Hub) Broadcast(scheduleID uint, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: Broadcast marshal failed for schedule %d (event: %s): %v", scheduleID, msg.Event, err)
		return
	}
	select {
	case h.broadcast <- roomMsg{scheduleID: scheduleID, data: data}:
	default:
		logger.Log.Warnf("hub: broadcast channel full, dropping message for schedule %d (event: %s)", scheduleID, msg.Event)
	}
}

// BroadcastToSupervisors sends to pengawas/admin only.
func (h *Hub) BroadcastToSupervisors(scheduleID uint, msg Message) {
	h.broadcastToSupervisors(scheduleID, msg)
}

func (h *Hub) broadcastToSupervisors(scheduleID uint, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: broadcastToSupervisors marshal failed for schedule %d: %v", scheduleID, err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c, rc := range h.rooms[scheduleID] {
		if rc.Role == "pengawas" || rc.Role == "admin" {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// SendToUser sends a message to a specific user.
func (h *Hub) SendToUser(scheduleID, targetUserID uint, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: SendToUser marshal failed for user %d in schedule %d: %v", targetUserID, scheduleID, err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c, rc := range h.rooms[scheduleID] {
		if rc.UserID == targetUserID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// BroadcastAll sends a message to all connected clients across all rooms.
func (h *Hub) BroadcastAll(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: BroadcastAll marshal failed: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.rooms {
		for c := range room {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// SendToUserGlobal sends a message to a specific user across all rooms.
func (h *Hub) SendToUserGlobal(targetUserID uint, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: SendToUserGlobal marshal failed for user %d: %v", targetUserID, err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.rooms {
		for c, rc := range room {
			if rc.UserID == targetUserID {
				select {
				case c.send <- data:
				default:
				}
			}
		}
	}
}

// GetRoomClients returns all connected clients in a room.
func (h *Hub) GetRoomClients(scheduleID uint) []*RoomClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	list := make([]*RoomClient, 0, len(h.rooms[scheduleID]))
	for _, rc := range h.rooms[scheduleID] {
		list = append(list, rc)
	}
	return list
}

// CountUserConnections returns the number of active connections for a given
// user across all rooms. Used to enforce per-user connection limits.
func (h *Hub) CountUserConnections(userID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, room := range h.rooms {
		for _, rc := range room {
			if rc.UserID == userID {
				count++
			}
		}
	}
	return count
}

// BroadcastAnswerToSupervisors accumulates answer events and flushes them
// as a single batched message after 200ms of inactivity. This reduces
// per-answer JSON marshal + lock overhead when many students submit
// answers simultaneously.
func (h *Hub) BroadcastAnswerToSupervisors(scheduleID uint, payload AnswerSavedPayload) {
	h.supervisorBatchMu.Lock()
	defer h.supervisorBatchMu.Unlock()

	pb, exists := h.supervisorBatch[scheduleID]
	if !exists {
		pb = &pendingBroadcast{
			scheduleID: scheduleID,
			answers:    make([]AnswerSavedPayload, 0, 32),
		}
		h.supervisorBatch[scheduleID] = pb
	}
	pb.answers = append(pb.answers, payload)

	if pb.timer != nil {
		pb.timer.Stop()
	}
	pb.timer = time.AfterFunc(200*time.Millisecond, func() {
		h.flushSupervisorBatch(scheduleID)
	})
}

// flushSupervisorBatch marshals all accumulated answer payloads into one
// message and sends it to supervisors/admins in the room.
func (h *Hub) flushSupervisorBatch(scheduleID uint) {
	// Copy batch data under lock, then release before broadcasting.
	h.supervisorBatchMu.Lock()
	pb, exists := h.supervisorBatch[scheduleID]
	if !exists || len(pb.answers) == 0 {
		h.supervisorBatchMu.Unlock()
		return
	}
	// Take ownership of the slice and reset immediately while still holding the lock.
	answers := pb.answers
	pb.answers = make([]AnswerSavedPayload, 0, 32)
	pb.timer = nil
	// Marshal while still holding supervisorBatchMu to prevent concurrent modification
	// of the answers slice between unlock and marshal.
	msg := Message{Event: EventAnswerBatch, Data: answers}
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Errorf("hub: flushSupervisorBatch marshal failed for schedule %d: %v", scheduleID, err)
		h.supervisorBatchMu.Unlock()
		return
	}
	h.supervisorBatchMu.Unlock()

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c, rc := range h.rooms[scheduleID] {
		if rc.Role == "pengawas" || rc.Role == "admin" {
			select {
			case c.send <- data:
			default:
				// channel full — supervisor will catch up on next batch
			}
		}
	}
}
