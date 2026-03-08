import client from './client'

export const supervisionApi = {
  forceFinish: (scheduleId: number, sessionId: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/force-finish`),

  extendTime: (scheduleId: number, sessionId: number, minutes: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/extend-time`, { minutes }),

  sendMessage: (scheduleId: number, sessionId: number, message: string, senderName?: string) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/send-message`, {
      message,
      sender_name: senderName ?? 'Pengawas',
    }),

  unlock: (scheduleId: number, sessionId: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/unlock`),

  reset: (scheduleId: number, sessionId: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/reset`),

  returnToExam: (scheduleId: number, sessionId: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/return-to-exam`),

  forceLogout: (scheduleId: number, sessionId: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/sessions/${sessionId}/force-logout`),

  getRoomTokens: (scheduleId: number) =>
    client.get(`/api/v1/supervision/tokens/${scheduleId}`),

  generateRoomTokens: (scheduleId: number, rooms: number[]) =>
    client.post(`/api/v1/supervision/tokens/${scheduleId}`, { rooms }),

  bulkAction: (scheduleId: number, action: 'force_finish' | 'extend_time' | 'unlock', sessionIds: number[], minutes?: number) =>
    client.post(`/api/v1/monitoring/${scheduleId}/bulk-action`, {
      action,
      session_ids: sessionIds,
      minutes,
    }),

  // ── Supervision Setup ──────────────────────────────────────────

  getSupervisionSetup: (scheduleId: number) =>
    client.get(`/api/v1/admin/supervision/${scheduleId}/setup`),

  generateTokens: (scheduleId: number) =>
    client.post(`/api/v1/admin/supervision/${scheduleId}/generate-tokens`),

  regenerateToken: (scheduleId: number, roomId: number) =>
    client.post(`/api/v1/admin/supervision/${scheduleId}/rooms/${roomId}/regenerate-token`),

  getGlobalStats: (scheduleId: number, roomId?: number | string) =>
    client.get(`/api/v1/admin/supervision/${scheduleId}/global-stats`, {
      params: roomId ? { room_id: roomId } : undefined,
    }),

  fetchStudentsByRoom: (scheduleId: number, roomId: number) =>
    client.get(`/api/v1/admin/supervision/${scheduleId}/rooms/${roomId}/students`),

  exitSupervisionSession: (scheduleId: number) =>
    client.post(`/api/v1/pengawas/supervision/${scheduleId}/exit`),
}
