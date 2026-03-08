import type { AxiosRequestConfig } from 'axios'
import client from './client'

export interface ExamSchedule {
  id: number
  name: string
  token: string
  start_time: string
  end_time: string
  duration_minutes: number
  status: 'draft' | 'published' | 'active' | 'finished'
  allow_see_result: boolean
  max_violations: number
  randomize_questions: boolean
  randomize_options: boolean
  next_exam_schedule_id?: number | null
  late_policy: 'allow_full_time' | 'deduct_time'
  min_working_time: number
  detect_cheating: boolean
  cheating_limit: number
  show_score_after: 'immediately' | 'after_end_time' | 'manual'
  created_by: number
  question_banks?: ExamScheduleBank[]
  rombels?: { exam_schedule_id: number; rombel_id: number; rombel: { id: number; name: string } }[]
  tags?: { exam_schedule_id: number; tag_id: number; tag: { id: number; name: string; color: string } }[]
  users?: ExamScheduleUser[]
}

export interface ExamScheduleBank {
  id: number
  exam_schedule_id: number
  question_bank_id: number
  question_count: number
  weight: number
  question_bank: { id: number; name: string; question_count: number }
}

export interface ExamScheduleUser {
  id: number
  exam_schedule_id: number
  user_id: number
  type: 'include' | 'exclude'
  user?: { id: number; name: string; username: string }
}

export interface ExamSession {
  id: number
  exam_schedule_id: number
  user_id: number
  status: 'not_started' | 'ongoing' | 'finished' | 'terminated'
  start_time: string | null
  end_time: string | null
  finished_at: string | null
  score: number
  max_score: number
  violation_count: number
  option_order?: Record<number, string[]>
  min_working_time?: number
  exam_schedule?: ExamSchedule
}

export interface SafeQuestion {
  id: number
  question_bank_id: number
  stimulus_id: number | null
  question_type: string
  body: string
  score: number
  difficulty: string
  options: unknown
  order_index: number
}

export interface ExamAnswer {
  id: number
  exam_session_id: number
  question_id: number
  answer: unknown
  is_flagged: boolean
  answered_at: string | null
}

// Fix #1: Tambah interface Stimulus agar wacana bisa di-render
export interface Stimulus {
  id: number
  question_bank_id: number
  title: string
  content: string // HTML content dari wacana
}

export interface StartExamResult {
  session: ExamSession
  questions: SafeQuestion[]
  answers: ExamAnswer[]
}

export const examApi = {
  // Schedules (admin/guru)
  listSchedules: (params?: { page?: number; per_page?: number; search?: string; status?: string }, config?: AxiosRequestConfig) =>
    client.get('/exam-schedules', { params, ...config }),

  listTrashedSchedules: (params?: { page?: number; per_page?: number; search?: string }, config?: AxiosRequestConfig) =>
    client.get('/exam-schedules/trashed', { params, ...config }),

  getSchedule: (id: number) =>
    client.get(`/exam-schedules/${id}`),

  previewSchedule: (id: number) =>
    client.get(`/exam-schedules/${id}/preview`),

  createSchedule: (data: CreateSchedulePayload) =>
    client.post('/exam-schedules', data),

  updateSchedule: (id: number, data: CreateSchedulePayload) =>
    client.put(`/exam-schedules/${id}`, data),

  updateStatus: (id: number, status: string) =>
    client.patch(`/exam-schedules/${id}/status`, { status }),

  deleteSchedule: (id: number) =>
    client.delete(`/exam-schedules/${id}`),

  restoreSchedule: (id: number) =>
    client.post(`/exam-schedules/${id}/restore`),

  forceDeleteSchedule: (id: number) =>
    client.delete(`/exam-schedules/${id}/force`),

  cloneSchedule: (id: number) =>
    client.post(`/exam-schedules/${id}/clone`),

  listOngoingSessions: (scheduleId: number) =>
    client.get(`/exam-schedules/${scheduleId}/sessions/ongoing`),

  listNotStartedSessions: (scheduleId: number) =>
    client.get(`/exam-schedules/${scheduleId}/sessions/not-started`),

  finishAllOngoing: (scheduleId: number) =>
    client.post(`/reports/${scheduleId}/finish-all`),

  getRoomTokens: (scheduleId: number) =>
    client.get(`/supervision/tokens/${scheduleId}`),

  saveRoomTokens: (scheduleId: number, data: { global_token: string; rooms: { room_id: number; token: string }[] }) =>
    client.post(`/supervision/tokens/${scheduleId}`, data),

  // Session (peserta)
  getAvailable: () =>
    client.get('/exam/available'),

  /** Get student's finished exam sessions (with schedule info) */
  getMyHistory: () =>
    client.get('/exam/history'),

  startExam: (data: { exam_schedule_id: number; token: string }) =>
    client.post('/exam/start', data),

  loadSession: (id: number) =>
    client.get(`/exam/sessions/${id}`),

  saveAnswer: (id: number, data: { question_id: number; answer: unknown; is_flagged?: boolean }) =>
    client.post(`/exam/sessions/${id}/answers`, data),

  batchSaveAnswers: (id: number, answers: { question_id: number; answer: unknown; is_flagged?: boolean }[]) =>
    client.post(`/exam/sessions/${id}/answers/batch`, answers),

  finishExam: (id: number) =>
    client.post(`/exam/sessions/${id}/finish`),

  logViolation: (id: number, data: { violation_type: string; description?: string }) =>
    client.post(`/exam/sessions/${id}/violations`, data),

  toggleFlag: (sessionId: number, questionId: number, isFlagged: boolean) =>
    client.post(`/exam/sessions/${sessionId}/questions/${questionId}/flag`, { is_flagged: isFlagged }),

  checkLockStatus: (sessionId: number) =>
    client.get<{ success: boolean; data: { status: string } }>(`/exam/sessions/${sessionId}/lock-status`),

  beaconSync: (sessionId: number, answers: { question_id: number; answer: unknown; is_flagged?: boolean }[]) =>
    client.post(`/exam/sessions/${sessionId}/beacon-sync`, { answers }),

  getTransition: (id: number) =>
    client.get<{ success: boolean; data: ExamSchedule }>(`/exam/sessions/${id}/transition`),

  startSection: (id: number) =>
    client.post<{ success: boolean; data: StartExamResult }>(`/exam/sessions/${id}/start-section`),

  // Fix #1: Fetch stimulus/wacana berdasarkan ID
  getStimulus: (stimulusId: number, config?: { timeout?: number }) =>
    client.get<{ success: boolean; data: Stimulus }>(`/stimuli/${stimulusId}`, config),
}

export interface CreateSchedulePayload {
  name: string
  start_time: string
  end_time: string
  duration_minutes: number
  allow_see_result?: boolean
  max_violations?: number
  randomize_questions?: boolean
  randomize_options?: boolean
  next_exam_schedule_id?: number | null
  late_policy?: string
  min_working_time?: number
  detect_cheating?: boolean
  cheating_limit?: number
  show_score_after?: string
  question_banks: { question_bank_id: number; question_count: number; weight: number }[]
  rombel_ids: number[]
  tag_ids: number[]
  include_users?: number[]
  exclude_users?: number[]
}

export const STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  published: 'Dipublikasi',
  active: 'Aktif',
  finished: 'Selesai',
}

export const STATUS_COLORS: Record<string, 'default' | 'info' | 'success' | 'warning' | 'danger'> = {
  draft: 'default',
  published: 'info',
  active: 'success',
  finished: 'warning',
}
