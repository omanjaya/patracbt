import client from './client'

export interface SessionRow {
  session_id: number
  user_id: number
  user_name: string
  username: string
  score: number
  max_score: number
  percent: number
  status: string
  answered_count: number
  total_questions: number
  violation_count: number
  duration_seconds: number
  start_time: string | null
  finished_at: string | null
}

export interface ReportStats {
  total: number
  finished: number
  mean: number
  median: number
  std_dev: number
  highest: number
  lowest: number
}

export interface ScheduleReport {
  schedule_id: number
  schedule_name: string
  sessions: SessionRow[]
  stats: ReportStats
}

export interface AnswerDetail {
  question_id: number
  question_type: string
  body: string
  score: number
  options: any
  correct_answer: any
  user_answer: any
  earned_score: number
  is_flagged: boolean
  is_correct: boolean
}

export interface PersonalReport {
  session: any
  answers: AnswerDetail[]
}

export interface QuestionAnalysis {
  question_id: number
  body: string
  question_type: string
  difficulty_index: number
  discrimination_index: number
  quality: string
}

export interface ExamAnalysis {
  schedule_id: number
  questions: QuestionAnalysis[]
  stats: ReportStats
}

export interface KeyChange {
  question_id: number
  question_number: number
  old_answer: any
  new_answer: any
}

export interface KeyChangesResponse {
  count: number
  changes: KeyChange[]
}

export interface RegradeResult {
  regraded: number
  total: number
  changes: number
  score_changes: { user_name: string; old_score: number; new_score: number }[]
}

export interface RegradeScoreChangeEntry {
  session_id: number
  old_score: number
  new_score: number
}

export interface RegradeLogEntry {
  id: number
  exam_schedule_id: number
  requested_by: number
  requested_name: string
  sessions_count: number
  score_changes: RegradeScoreChangeEntry[] | null
  created_at: string
}

export interface AIBatchResult {
  question_id: number
  score: number
  reason: string
  error?: string
}

export const reportApi = {
  getScheduleReport: (scheduleId: number) =>
    client.get<{ success: boolean; data: ScheduleReport }>(`/reports/${scheduleId}`),

  getPersonalReport: (scheduleId: number, sessionId: number) =>
    client.get<{ success: boolean; data: PersonalReport }>(`/reports/${scheduleId}/sessions/${sessionId}`),

  /** Student-facing: get own report by session ID (checks ownership + allow_see_result on backend) */
  getMyReport: (sessionId: number) =>
    client.get<{ success: boolean; data: PersonalReport }>(`/exam/sessions/${sessionId}/report`),

  getExamAnalysis: (scheduleId: number) =>
    client.get<{ success: boolean; data: ExamAnalysis }>(`/reports/${scheduleId}/analysis`),

  regrade: (scheduleId: number) =>
    client.post<{ success: boolean; data: RegradeResult }>(`/reports/${scheduleId}/regrade`),

  getKeyChanges: (scheduleId: number) =>
    client.get<{ success: boolean; data: KeyChangesResponse }>(`/reports/${scheduleId}/key-changes`),

  getRegradeLogs: (scheduleId: number) =>
    client.get<{ success: boolean; data: RegradeLogEntry[] }>(`/reports/${scheduleId}/regrade-logs`),

  exportLedger: (scheduleId: number, multiSheet = false) => {
    const params = new URLSearchParams()
    if (multiSheet) params.set('multi_sheet', 'true')
    const query = params.toString() ? `?${params.toString()}` : ''
    return `/api/v1/reports/${scheduleId}/export${query}`
  },

  exportUnfinished: (scheduleId: number) => {
    return `/api/v1/reports/${scheduleId}/unfinished/export`
  },

  gradeEssay: (sessionId: number, questionId: number, score: number) =>
    client.post(`/exam-sessions/${sessionId}/grade-essay`, { question_id: questionId, score }),

  aiGradeEssay: (sessionId: number, questionId: number, answer: string) =>
    client.post(`/exam-sessions/${sessionId}/ai-grade`, { question_id: questionId, answer }),

  aiGradeBatchEssay: (sessionId: number) =>
    client.post<{ success: boolean; data: AIBatchResult[] }>(`/exam-sessions/${sessionId}/ai-grade-batch`),
}

export const QUALITY_COLORS: Record<string, string> = {
  'Baik Sekali': '#15803d',
  'Baik': '#2563eb',
  'Cukup': '#d97706',
  'Revisi': '#dc2626',
  'Buang': '#7f1d1d',
}
