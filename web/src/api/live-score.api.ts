import client from './client'

export interface LiveScoreRow {
  session_id: number
  user_id: number
  nis: string
  name: string
  rombel: string
  total_questions: number
  answered: number
  correct: number
  wrong: number
  unanswered: number
  score: number
  max_score: number
  percent: number
  status: string
  violation_count: number
  updated_at: string
}

export interface RombelInfo {
  id: number
  name: string
}

export interface LiveSummary {
  total_participants: number
  ongoing: number
  finished: number
  not_started: number
  average_score: number
  highest_score: number
}

export interface LiveScoreData {
  schedule_id: number
  schedule_name: string
  subject_name: string
  start_time: string
  end_time: string
  rombels: RombelInfo[]
  students: LiveScoreRow[]
  summary: LiveSummary
  timestamp: string
}

export const liveScoreApi = {
  /**
   * Get full live score data for a schedule
   */
  getLiveData: (scheduleId: number, rombelIds?: number[]) => {
    const params: Record<string, string> = {}
    if (rombelIds && rombelIds.length > 0) {
      params.rombel_ids = rombelIds.join(',')
    }
    return client.get<{ success: boolean; data: LiveScoreData }>(
      `/admin/live-score/${scheduleId}`,
      { params }
    )
  },

  /**
   * Get incremental update since a timestamp
   */
  getUpdate: (scheduleId: number, since: string, rombelIds?: number[]) => {
    const params: Record<string, string> = { since }
    if (rombelIds && rombelIds.length > 0) {
      params.rombel_ids = rombelIds.join(',')
    }
    return client.get<{ success: boolean; data: LiveScoreData }>(
      `/admin/live-score/${scheduleId}/update`,
      { params }
    )
  },
}
