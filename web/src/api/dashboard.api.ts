import client from './client'

export interface AdminDashboardStats {
  total_peserta: number
  total_guru: number
  total_rombel: number
  total_question_banks: number
  active_schedules: number
  total_sessions: number
  finished_sessions: number
  ongoing_sessions: number
}

export interface GuruDashboardStats {
  total_banks: number
  total_questions: number
  active_schedules: number
  total_schedules: number
}

export interface GuruEssayStats {
  ungraded_essays: number
  total_essays: number
  schedules_with_essays: number
}

export interface GuruOngoingExam {
  schedule_id: number
  schedule_name: string
  status: string
  total_students: number
  ongoing_count: number
  finished_count: number
  start_time: string
  end_time: string
}

export interface GuruAlert {
  type: 'ungraded_essay' | 'expiring_schedule'
  message: string
  schedule_id: number
  count: number
}

export interface PengawasDashboardStats {
  active_schedules: number
  ongoing_sessions: number
  finished_today: number
}

export interface PengawasMonitoringSummary {
  online_students: number
  violations_today: number
  active_sessions: number
  active_schedules: number
}

export interface PengawasViolation {
  id: number
  exam_session_id: number
  violation_type: string
  description: string
  created_at: string
  student_name: string
  schedule_name: string
  schedule_id: number
}

export interface PengawasActiveRoom {
  schedule_id: number
  schedule_name: string
  online_students: number
  total_students: number
  violation_count: number
  end_time: string
  duration_minutes: number
  status: string
}

export interface AdminAlert {
  type: string
  title: string
  message: string
  count: number
  link: string
  icon: string
  color: string
}

export interface ServerStats {
  cpu_percent: number
  ram_percent: number
  ram_used: string
  ram_total: string
  disk_percent: number
  disk_used: string
  disk_total: string
}

export interface OngoingExam {
  schedule_id: number
  schedule_name: string
  subject_name: string
  start_time: string
  end_time: string
  ongoing_count: number
  finished_count: number
  total_students: number
}

export const dashboardApi = {
  getAdminStats: () =>
    client.get<{ success: boolean; data: AdminDashboardStats }>('/admin/dashboard/stats'),
  getGuruStats: () =>
    client.get<{ success: boolean; data: GuruDashboardStats }>('/guru/dashboard/stats'),
  getPengawasStats: () =>
    client.get<{ success: boolean; data: PengawasDashboardStats }>('/pengawas/dashboard/stats'),
  getUpcomingExams: () => client.get('/admin/dashboard/upcoming-exams'),
  getRecentActivity: () => client.get('/admin/dashboard/recent-activity'),
  getGuruUpcomingExams: () => client.get('/guru/dashboard/upcoming-exams'),
  getGuruRecentActivity: () => client.get('/guru/dashboard/recent-activity'),
  getGuruEssayStats: () =>
    client.get<{ success: boolean; data: GuruEssayStats }>('/guru/dashboard/essay-stats'),
  getGuruOngoingExams: () =>
    client.get<{ success: boolean; data: GuruOngoingExam[] }>('/guru/dashboard/ongoing-exams'),
  getGuruAlerts: () =>
    client.get<{ success: boolean; data: GuruAlert[] }>('/guru/dashboard/alerts'),
  getPengawasMonitoringSummary: () =>
    client.get<{ success: boolean; data: PengawasMonitoringSummary }>('/pengawas/dashboard/monitoring-summary'),
  getPengawasRecentViolations: () =>
    client.get<{ success: boolean; data: PengawasViolation[] }>('/pengawas/dashboard/recent-violations'),
  getPengawasActiveRooms: () =>
    client.get<{ success: boolean; data: PengawasActiveRoom[] }>('/pengawas/dashboard/active-rooms'),
  getPengawasAllViolations: (params?: { schedule_id?: string; severity?: string; date?: string }) =>
    client.get<{ success: boolean; data: PengawasViolation[] }>('/pengawas/dashboard/all-violations', { params }),
  getServerStats: () =>
    client.get<{ success: boolean; data: ServerStats }>('/admin/dashboard/server-stats'),
  getOngoingExams: () =>
    client.get<{ success: boolean; data: OngoingExam[] }>('/admin/dashboard/ongoing-exams'),
  getAdminAlerts: () =>
    client.get<{ success: boolean; data: AdminAlert[] }>('/admin/dashboard/alerts'),
}
