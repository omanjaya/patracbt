<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../../stores/auth.store'
import { useToastStore } from '../../../stores/toast.store'
import {
  dashboardApi,
  type GuruDashboardStats,
  type GuruEssayStats,
  type GuruOngoingExam,
  type GuruAlert,
} from '../../../api/dashboard.api'

import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const authStore = useAuthStore()
const router = useRouter()
const toast = useToastStore()
const loading = ref(true)
const stats = ref<GuruDashboardStats | null>(null)
const essayStats = ref<GuruEssayStats | null>(null)
const ongoingExams = ref<GuruOngoingExam[]>([])
const alerts = ref<GuruAlert[]>([])
const upcomingExams = ref<any[]>([])
const recentActivity = ref<any[]>([])

function formatDate(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
}

function formatTime(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 11) return 'Selamat Pagi'
  if (hour < 15) return 'Selamat Siang'
  if (hour < 18) return 'Selamat Sore'
  return 'Selamat Malam'
})

onMounted(async () => {
  try {
    const [statsRes, essayRes, ongoingRes, alertsRes, upcomingRes, activityRes] = await Promise.all([
      dashboardApi.getGuruStats(),
      dashboardApi.getGuruEssayStats(),
      dashboardApi.getGuruOngoingExams(),
      dashboardApi.getGuruAlerts(),
      dashboardApi.getGuruUpcomingExams(),
      dashboardApi.getGuruRecentActivity(),
    ])
    stats.value = statsRes.data.data
    essayStats.value = essayRes.data.data
    ongoingExams.value = ongoingRes.data.data ?? []
    alerts.value = alertsRes.data.data ?? []
    upcomingExams.value = upcomingRes.data.data ?? []
    recentActivity.value = activityRes.data.data ?? []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data dashboard')
  } finally {
    loading.value = false
  }
})

const statCards = computed(() => [
  { label: 'Bank Soal Saya', value: stats.value?.total_banks ?? 0, icon: 'ti-books', color: 'purple', path: '/guru/question-banks' },
  { label: 'Total Soal', value: stats.value?.total_questions ?? 0, icon: 'ti-file-text', color: 'green', path: '/guru/question-banks' },
  { label: 'Jadwal Aktif', value: stats.value?.active_schedules ?? 0, icon: 'ti-calendar-event', color: 'red', path: '/guru/exam-schedules' },
  { label: 'Total Jadwal', value: stats.value?.total_schedules ?? 0, icon: 'ti-circle-check', color: 'primary', path: '/guru/exam-schedules' },
  { label: 'Esai Belum Dinilai', value: essayStats.value?.ungraded_essays ?? 0, icon: 'ti-writing', color: 'orange', path: '/guru/reports' },
])

const quickLinks = [
  { label: 'Bank Soal', path: '/guru/question-banks', icon: 'ti-books', desc: 'Kelola bank soal' },
  { label: 'Penjadwalan Ujian', path: '/guru/exam-schedules', icon: 'ti-calendar-event', desc: 'Buat & atur jadwal' },
  { label: 'Laporan & Analisis', path: '/guru/reports', icon: 'ti-chart-bar', desc: 'Rekap nilai & analisis' },
  { label: 'Riwayat Ujian', path: '/guru/exam-history', icon: 'ti-history', desc: 'Ujian yang telah selesai' },
  { label: 'Ruang Pengawasan', path: '/guru/supervision', icon: 'ti-device-desktop-analytics', desc: 'Pantau ujian realtime' },
  { label: 'Live Score (TV Mode)', path: '/guru/live-score', icon: 'ti-device-tv', desc: 'Tampilkan skor langsung' },
]

function statusBadge(status: string) {
  return status === 'ongoing' ? 'bg-danger-lt text-danger'
    : status === 'finished' ? 'bg-success-lt text-success'
    : 'bg-secondary-lt text-secondary'
}

function alertIcon(type: string) {
  return type === 'ungraded_essay' ? 'ti-writing' : 'ti-clock-exclamation'
}

function alertColor(type: string) {
  return type === 'ungraded_essay' ? 'orange' : 'red'
}
</script>

<template>
  <BasePageHeader
    title="Dashboard Guru"
    :subtitle="`${greeting}, ${authStore.user?.name}`"
    :breadcrumbs="[{ label: 'Dashboard' }]"
  />

  <!-- Welcome Header -->
  <div class="card mb-3 bg-primary-lt border-0">
    <div class="card-body">
      <div class="row align-items-center">
        <div class="col-auto">
          <span class="avatar avatar-xl rounded-circle" :style="authStore.user?.avatar_url ? { backgroundImage: `url(${authStore.user.avatar_url})` } : {}">{{ !authStore.user?.avatar_url ? (authStore.user?.name ?? '').split(' ').map((w: string) => w[0]).slice(0, 2).join('').toUpperCase() : '' }}</span>
        </div>
        <div class="col">
          <h2 class="mb-1">{{ greeting }}, {{ authStore.user?.name }}!</h2>
          <p class="text-muted mb-0">
            <i class="ti ti-school me-1"></i>Dashboard Guru
            <span v-if="stats" class="mx-2">|</span>
            <span v-if="stats" class="text-primary fw-medium">
              {{ stats.active_schedules }} ujian aktif
            </span>
            <span v-if="essayStats && essayStats.ungraded_essays > 0" class="mx-2">|</span>
            <span v-if="essayStats && essayStats.ungraded_essays > 0" class="text-orange fw-medium">
              <i class="ti ti-alert-circle me-1"></i>{{ essayStats.ungraded_essays }} esai perlu dinilai
            </span>
          </p>
        </div>
        <div class="col-auto d-none d-md-block">
          <i class="ti ti-school fs-1 opacity-50"></i>
        </div>
      </div>
    </div>
  </div>

  <!-- Stats Cards -->
  <div v-if="loading" class="row g-3 mb-3">
    <div v-for="n in 5" :key="n" class="col-sm-6 col-lg">
      <div class="card placeholder-glow">
        <div class="card-body">
          <div class="placeholder col-6 mb-2"></div>
          <div class="placeholder col-4"></div>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="row g-3 mb-3">
    <div v-for="card in statCards" :key="card.label" class="col-sm-6 col-lg">
      <div class="card card-sm card-link cursor-pointer" @click="router.push(card.path)" style="cursor:pointer">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span :class="`bg-${card.color}-lt`" class="avatar">
                <i :class="['ti fs-4', card.icon, `text-${card.color}`]"></i>
              </span>
            </div>
            <div class="col">
              <div class="fw-medium h3 mb-0">{{ card.value }}</div>
              <div class="text-muted small">{{ card.label }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Alerts / Notifications -->
  <div v-if="!loading && alerts.length" class="mb-3">
    <div v-for="(alert, i) in alerts" :key="i"
      class="alert d-flex align-items-center gap-3 mb-2 cursor-pointer"
      :class="alert.type === 'ungraded_essay' ? 'alert-warning' : 'alert-danger'"
      style="cursor:pointer"
      @click="alert.type === 'ungraded_essay' ? router.push(`/guru/grading/${alert.schedule_id}`) : router.push('/guru/exam-schedules')"
    >
      <span class="avatar avatar-sm" :class="`bg-${alertColor(alert.type)}-lt`">
        <i :class="['ti', alertIcon(alert.type), `text-${alertColor(alert.type)}`]"></i>
      </span>
      <div class="flex-fill">
        <div class="fw-medium">{{ alert.message }}</div>
        <div class="text-muted small">
          {{ alert.type === 'ungraded_essay' ? 'Klik untuk mulai menilai' : 'Klik untuk melihat jadwal' }}
        </div>
      </div>
      <i class="ti ti-chevron-right text-muted"></i>
    </div>
  </div>

  <!-- Ongoing Exams Section -->
  <div v-if="!loading && ongoingExams.length" class="card mb-3">
    <div class="card-header">
      <h3 class="card-title">
        <i class="ti ti-live-photo me-2 text-red"></i>Ujian Berlangsung
      </h3>
      <div class="card-actions">
        <button class="btn btn-sm btn-ghost-primary" @click="router.push('/guru/supervision')">
          <i class="ti ti-device-desktop-analytics me-1"></i>Pantau Semua
        </button>
      </div>
    </div>
    <div class="table-responsive">
      <table class="table table-vcenter table-hover card-table">
        <thead>
          <tr>
            <th>Nama Ujian</th>
            <th>Status</th>
            <th>Waktu</th>
            <th>Peserta</th>
            <th>Progres</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="exam in ongoingExams" :key="exam.schedule_id">
            <td>
              <div class="fw-medium">{{ exam.schedule_name }}</div>
            </td>
            <td>
              <span class="badge" :class="exam.status === 'active' ? 'bg-green-lt text-green' : 'bg-blue-lt text-blue'">
                {{ exam.status === 'active' ? 'Aktif' : 'Dipublikasi' }}
              </span>
            </td>
            <td class="text-muted small">
              <i class="ti ti-clock me-1"></i>{{ formatTime(exam.start_time) }} - {{ formatTime(exam.end_time) }}
            </td>
            <td>
              <div class="d-flex align-items-center gap-2">
                <span class="badge bg-blue-lt text-blue">{{ exam.ongoing_count }} mengerjakan</span>
                <span class="badge bg-green-lt text-green">{{ exam.finished_count }} selesai</span>
              </div>
            </td>
            <td style="min-width:120px">
              <div class="d-flex align-items-center gap-2">
                <div class="progress progress-sm flex-fill">
                  <div class="progress-bar bg-green" :style="{ width: (exam.total_students ? (exam.finished_count / exam.total_students * 100) : 0) + '%' }"></div>
                  <div class="progress-bar bg-blue" :style="{ width: (exam.total_students ? (exam.ongoing_count / exam.total_students * 100) : 0) + '%' }"></div>
                </div>
                <span class="small text-muted">{{ exam.total_students }}</span>
              </div>
            </td>
            <td>
              <button class="btn btn-sm btn-ghost-primary" @click="router.push('/guru/supervision')">
                <i class="ti ti-eye me-1"></i>Pantau
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <!-- Quick Links -->
  <div class="card mb-3">
    <div class="card-header">
      <h3 class="card-title">
        <i class="ti ti-bolt me-2 text-primary"></i>Akses Cepat
      </h3>
    </div>
    <div class="card-body">
      <div class="row g-3">
        <div v-for="link in quickLinks" :key="link.path" class="col-sm-6 col-md-4 col-lg-2">
          <div
            class="card card-sm border cursor-pointer h-100 text-center"
            style="cursor:pointer"
            @click="router.push(link.path)"
          >
            <div class="card-body py-3 px-2">
              <span class="avatar avatar-md bg-primary-lt mb-2">
                <i :class="`ti ${link.icon} fs-3 text-primary`"></i>
              </span>
              <div class="fw-medium small">{{ link.label }}</div>
              <div class="text-muted" style="font-size:0.7rem">{{ link.desc }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Upcoming + Activity -->
  <div v-if="loading" class="row g-3 mb-3">
    <div v-for="n in 2" :key="n" class="col-md-6">
      <div class="card placeholder-glow">
        <div class="card-header"><div class="placeholder col-4"></div></div>
        <div class="card-body">
          <div class="placeholder col-8 mb-2"></div>
          <div class="placeholder col-6 mb-2"></div>
          <div class="placeholder col-7"></div>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="row g-3">
    <div class="col-md-6">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-calendar-clock me-2 text-primary"></i>Ujian Mendatang
          </h3>
        </div>
        <div v-if="!upcomingExams.length" class="card-body text-center text-muted py-4">
          <i class="ti ti-calendar-off fs-1 mb-2 d-block opacity-50"></i>
          <p class="mb-0">Tidak ada ujian mendatang</p>
        </div>
        <ul v-else class="list-group list-group-flush">
          <li v-for="exam in upcomingExams" :key="exam.id" class="list-group-item">
            <div class="d-flex align-items-center gap-3">
              <span class="avatar avatar-sm bg-primary-lt">
                <i class="ti ti-calendar text-primary"></i>
              </span>
              <div class="flex-fill">
                <div class="fw-medium">{{ exam.title || exam.name }}</div>
                <div class="text-muted small">
                  <i class="ti ti-clock me-1"></i>{{ formatDate(exam.start_time) }}
                </div>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>

    <div class="col-md-6">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-activity me-2 text-primary"></i>Aktivitas Terbaru
          </h3>
        </div>
        <div v-if="!recentActivity.length" class="card-body text-center text-muted py-4">
          <i class="ti ti-activity fs-1 mb-2 d-block opacity-50"></i>
          <p class="mb-0">Belum ada aktivitas</p>
        </div>
        <ul v-else class="list-group list-group-flush">
          <li v-for="session in recentActivity" :key="session.id" class="list-group-item">
            <div class="d-flex align-items-center gap-3">
              <span class="avatar avatar-sm" :class="session.status === 'finished' ? 'bg-green-lt' : 'bg-blue-lt'">
                <i class="ti" :class="session.status === 'finished' ? 'ti-check text-green' : 'ti-player-play text-blue'"></i>
              </span>
              <div class="flex-fill">
                <div class="fw-medium">{{ session.user?.name ?? 'Peserta' }}</div>
                <div class="d-flex align-items-center gap-2 mt-1">
                  <span class="badge" :class="statusBadge(session.status)">{{ session.status }}</span>
                  <span class="text-muted small">{{ session.exam_schedule?.title ?? session.exam_schedule?.name ?? '' }}</span>
                </div>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
