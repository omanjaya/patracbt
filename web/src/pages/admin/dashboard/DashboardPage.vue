<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../../stores/auth.store'
import { dashboardApi, type AdminDashboardStats, type ServerStats, type OngoingExam, type AdminAlert } from '../../../api/dashboard.api'
import { useToastStore } from '@/stores/toast.store'

const authStore = useAuthStore()
const router = useRouter()
const toast = useToastStore()

const loading = ref(true)
const stats = ref<AdminDashboardStats | null>(null)
const upcomingExams = ref<any[]>([])
const recentActivity = ref<any[]>([])
const serverStats = ref<ServerStats | null>(null)
const ongoingExams = ref<OngoingExam[]>([])
const adminAlerts = ref<AdminAlert[]>([])
const now = ref(Date.now())

let countdownInterval: ReturnType<typeof setInterval> | null = null

function formatDate(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })
}

function formatTime(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { timeStyle: 'short' })
}

function formatCountdown(startTime: string): string {
  const diff = new Date(startTime).getTime() - now.value
  if (diff <= 0) return 'Segera dimulai'
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  if (days > 0) return `${days}h ${hours}j`
  if (hours > 0) return `${hours}j ${minutes}m`
  return `${minutes}m`
}

function progressColor(percent: number): string {
  if (percent > 80) return 'bg-danger'
  if (percent > 50) return 'bg-warning'
  return 'bg-primary'
}

const serverHasWarning = computed(() => {
  if (!serverStats.value) return false
  return serverStats.value.cpu_percent > 50 || serverStats.value.ram_percent > 50 || serverStats.value.disk_percent > 80
})

onMounted(async () => {
  try {
    const [statsRes, upcomingRes, activityRes, serverRes, ongoingRes, alertsRes] = await Promise.allSettled([
      dashboardApi.getAdminStats(),
      dashboardApi.getUpcomingExams(),
      dashboardApi.getRecentActivity(),
      dashboardApi.getServerStats(),
      dashboardApi.getOngoingExams(),
      dashboardApi.getAdminAlerts(),
    ])
    if (statsRes.status === 'fulfilled') stats.value = statsRes.value.data.data
    if (upcomingRes.status === 'fulfilled') upcomingExams.value = upcomingRes.value.data.data ?? []
    if (activityRes.status === 'fulfilled') recentActivity.value = activityRes.value.data.data ?? []
    if (serverRes.status === 'fulfilled') serverStats.value = serverRes.value.data.data
    if (ongoingRes.status === 'fulfilled') ongoingExams.value = ongoingRes.value.data.data ?? []
    if (alertsRes.status === 'fulfilled') adminAlerts.value = alertsRes.value.data.data ?? []

    const failed = [statsRes, upcomingRes, activityRes].filter(r => r.status === 'rejected')
    if (failed.length > 0) toast.error('Sebagian data dashboard gagal dimuat')
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data dashboard')
  } finally {
    loading.value = false
  }
  countdownInterval = setInterval(() => { now.value = Date.now() }, 60_000)
})

onUnmounted(() => {
  if (countdownInterval) { clearInterval(countdownInterval); countdownInterval = null }
})

const statCards = [
  { label: 'Total Peserta', key: 'total_peserta' as const, icon: 'ti-users', color: 'primary' },
  { label: 'Total Guru', key: 'total_guru' as const, icon: 'ti-school', color: 'purple' },
  { label: 'Total Rombel', key: 'total_rombel' as const, icon: 'ti-users-group', color: 'yellow' },
  { label: 'Bank Soal', key: 'total_question_banks' as const, icon: 'ti-books', color: 'green' },
]

const sessionCards = [
  { label: 'Jadwal Aktif', key: 'active_schedules' as const, icon: 'ti-calendar-event', color: 'cyan' },
  { label: 'Berlangsung', key: 'ongoing_sessions' as const, icon: 'ti-device-desktop-analytics', color: 'red' },
  { label: 'Selesai', key: 'finished_sessions' as const, icon: 'ti-circle-check', color: 'green' },
  { label: 'Total Sesi', key: 'total_sessions' as const, icon: 'ti-calendar-stats', color: 'secondary' },
]

const isAdmin = computed(() => authStore.user?.role === 'admin')

const quickLinks = computed(() => {
  const links = [
    { label: 'Kelola User', path: '/admin/users', icon: 'ti-users', color: 'primary' },
    { label: 'Rombel', path: '/admin/rombels', icon: 'ti-users-group', color: 'azure' },
    { label: 'Bank Soal', path: '/admin/question-banks', icon: 'ti-books', color: 'green' },
    { label: 'Jadwal Ujian', path: '/admin/exam-schedules', icon: 'ti-calendar-event', color: 'cyan' },
    { label: 'Pengawasan', path: '/admin/supervision', icon: 'ti-device-desktop-analytics', color: 'red' },
    { label: 'Laporan', path: '/admin/reports', icon: 'ti-chart-bar', color: 'purple' },
    { label: 'Pengaturan', path: '/admin/settings', icon: 'ti-settings', color: 'secondary' },
  ]
  if (isAdmin.value) {
    links.push({ label: 'Backup', path: '/admin/backup', icon: 'ti-database-export', color: 'orange' })
  }
  return links
})

function statusBadge(status: string) {
  return status === 'ongoing' ? 'bg-danger-lt text-danger'
    : status === 'finished' ? 'bg-success-lt text-success'
    : 'bg-secondary-lt text-secondary'
}
</script>

<template>
  <!-- Page header with greeting + alerts badge -->
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <div class="page-pretitle">Overview</div>
        <h2 class="page-title">
          Selamat datang, {{ authStore.user?.name ?? '' }}
        </h2>
      </div>
      <div class="col-auto d-flex gap-2">
        <button v-if="adminAlerts.length" class="btn btn-warning btn-sm position-relative" @click="router.push(adminAlerts[0]?.link ?? '#')">
          <i class="ti ti-alert-triangle me-1"></i>{{ adminAlerts.length }} Perhatian
        </button>
        <button class="btn btn-primary btn-sm d-none d-sm-inline-block" @click="router.push('/admin/exam-schedules')">
          <i class="ti ti-plus me-1"></i>Jadwal Baru
        </button>
        <button class="btn btn-white btn-sm d-none d-sm-inline-block" @click="router.push('/admin/users')">
          <i class="ti ti-user-plus me-1"></i>Tambah User
        </button>
      </div>
    </div>
  </div>

  <!-- Server stats bar (only shows when warning or always slim) -->
  <div v-if="serverStats" class="card card-sm mb-3" :class="{ 'border-warning': serverHasWarning }">
    <div class="card-body py-2 px-3">
      <div class="d-flex align-items-center gap-3 flex-wrap">
        <span class="text-muted d-flex align-items-center" style="font-size:.78rem">
          <i class="ti ti-server me-1"></i>Server
        </span>
        <div class="d-flex align-items-center gap-2 flex-fill">
          <span class="text-muted" style="font-size:.72rem;width:40px">CPU</span>
          <div class="progress flex-fill" style="height:5px">
            <div class="progress-bar" :class="progressColor(serverStats.cpu_percent)" :style="{ width: serverStats.cpu_percent + '%' }"></div>
          </div>
          <span class="text-muted" style="font-size:.72rem;width:32px">{{ serverStats.cpu_percent }}%</span>
        </div>
        <div class="d-flex align-items-center gap-2 flex-fill">
          <span class="text-muted" style="font-size:.72rem;width:40px">RAM</span>
          <div class="progress flex-fill" style="height:5px">
            <div class="progress-bar" :class="progressColor(serverStats.ram_percent)" :style="{ width: serverStats.ram_percent + '%' }"></div>
          </div>
          <span class="text-muted" style="font-size:.72rem;width:32px">{{ serverStats.ram_percent }}%</span>
        </div>
        <div class="d-flex align-items-center gap-2 flex-fill">
          <span class="text-muted" style="font-size:.72rem;width:40px">Disk</span>
          <div class="progress flex-fill" style="height:5px">
            <div class="progress-bar" :class="progressColor(serverStats.disk_percent)" :style="{ width: serverStats.disk_percent + '%' }"></div>
          </div>
          <span class="text-muted" style="font-size:.72rem;width:32px">{{ serverStats.disk_percent }}%</span>
        </div>
      </div>
    </div>
  </div>

  <!-- Stats row -->
  <div v-if="loading" class="row g-2 mb-3">
    <div v-for="n in 4" :key="n" class="col-6 col-lg-3">
      <div class="card placeholder-glow"><div class="card-body py-3"><div class="placeholder col-6 mb-1"></div><div class="placeholder col-4"></div></div></div>
    </div>
  </div>
  <div v-else class="row g-2 mb-3">
    <div v-for="card in statCards" :key="card.key" class="col-6 col-lg-3">
      <div class="card">
        <div class="card-body py-3">
          <div class="d-flex align-items-center">
            <div class="subheader">{{ card.label }}</div>
            <div class="ms-auto lh-1">
              <span class="avatar avatar-sm rounded" :class="`bg-${card.color}-lt`">
                <i class="ti" :class="[card.icon, `text-${card.color}`]"></i>
              </span>
            </div>
          </div>
          <div class="h1 mb-0 mt-1">{{ stats?.[card.key] ?? 0 }}</div>
        </div>
      </div>
    </div>
  </div>

  <!-- Main layout: 9 cols + 3 cols sidebar -->
  <div class="row row-cards">
    <!-- LEFT: Main content -->
    <div class="col-lg-9">

      <!-- Ongoing exams (LIVE) -->
      <div v-if="ongoingExams.length > 0" class="card mb-3">
        <div class="card-header">
          <h3 class="card-title">Ujian Sedang Berjalan</h3>
          <div class="card-actions">
            <span class="badge bg-success" style="animation:pulse 2s infinite">Live</span>
          </div>
        </div>
        <div class="list-group list-group-flush">
          <div v-for="exam in ongoingExams" :key="exam.schedule_id" class="list-group-item">
            <div class="row align-items-center">
              <div class="col-auto">
                <span class="avatar bg-green-lt">
                  {{ exam.subject_name?.[0] ?? '?' }}
                </span>
              </div>
              <div class="col text-truncate">
                <a href="#" class="text-reset d-block">{{ exam.schedule_name }}</a>
                <div class="d-block text-muted text-truncate mt-n1">
                  {{ formatTime(exam.start_time) }} - {{ formatTime(exam.end_time) }}
                  &bull; {{ exam.subject_name }}
                </div>
              </div>
              <div class="col-auto text-muted small text-nowrap">
                <i class="ti ti-users me-1"></i>
                <span class="text-success fw-medium">{{ exam.ongoing_count }}</span> /
                <span class="text-primary">{{ exam.finished_count }}</span> /
                {{ exam.total_students }}
              </div>
              <div class="col-auto">
                <button class="btn btn-success btn-sm" @click="router.push('/admin/supervision')">
                  Pantau
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Quick links grid (like ExamPatra) -->
      <div class="card mb-3">
        <div class="card-header">
          <h3 class="card-title">Akses Cepat</h3>
        </div>
        <div class="card-body">
          <div class="row g-3 text-center">
            <div v-for="link in quickLinks" :key="link.path" class="col-6 col-sm-3">
              <a
                href="#"
                class="btn btn-outline-secondary w-100 h-100 d-flex flex-column align-items-center justify-content-center py-3 border-0 shadow-sm"
                @click.prevent="router.push(link.path)"
              >
                <i class="ti mb-2" :class="[link.icon, `text-${link.color}`]" style="font-size:1.5rem"></i>
                <span style="font-size:.82rem">{{ link.label }}</span>
              </a>
            </div>
          </div>
        </div>
      </div>

      <!-- Session stats (inline mini cards) -->
      <div class="row g-2 mb-3">
        <div v-for="card in sessionCards" :key="card.key" class="col-6 col-lg-3">
          <div class="card card-sm">
            <div class="card-body py-2 px-3">
              <div class="d-flex align-items-center">
                <i class="ti me-2" :class="[card.icon, `text-${card.color}`]" style="font-size:1.1rem"></i>
                <div>
                  <div class="fw-bold" style="line-height:1.2">{{ stats?.[card.key] ?? 0 }}</div>
                  <div class="text-muted" style="font-size:.7rem">{{ card.label }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Upcoming exams -->
      <div class="card mb-3">
        <div class="card-header">
          <h3 class="card-title">Jadwal Akan Datang</h3>
          <div class="card-actions">
            <span class="badge bg-blue-lt">{{ upcomingExams.length }}</span>
          </div>
        </div>
        <div v-if="!upcomingExams.length" class="card-body text-center text-muted py-4">
          <i class="ti ti-calendar-off d-block opacity-50 mb-1" style="font-size:1.5rem"></i>
          Tidak ada ujian mendatang
        </div>
        <div v-else class="list-group list-group-flush">
          <div v-for="exam in upcomingExams" :key="exam.id" class="list-group-item">
            <div class="d-flex justify-content-between align-items-center">
              <div>
                <div class="fw-bold">{{ exam.title }}</div>
                <div class="text-muted small">
                  <i class="ti ti-clock me-1"></i>{{ formatDate(exam.start_time) }}
                </div>
              </div>
              <div class="text-end">
                <span class="badge bg-cyan-lt text-cyan">
                  {{ formatCountdown(exam.start_time) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Alerts detail (if any) -->
      <div v-if="adminAlerts.length > 0" class="card mb-3">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-alert-triangle me-1 text-warning"></i>Perlu Perhatian
          </h3>
        </div>
        <div class="list-group list-group-flush">
          <a
            v-for="alert in adminAlerts"
            :key="alert.type"
            href="#"
            class="list-group-item list-group-item-action d-flex align-items-center"
            @click.prevent="router.push(alert.link)"
          >
            <span class="avatar avatar-sm me-3 rounded" :class="`bg-${alert.color}-lt`">
              <i class="ti" :class="[alert.icon, `text-${alert.color}`]"></i>
            </span>
            <div class="flex-fill">
              <div class="fw-medium">{{ alert.title }}</div>
              <div class="text-muted small">{{ alert.message }}</div>
            </div>
            <span class="badge ms-2" :class="`bg-${alert.color}`">{{ alert.count }}</span>
          </a>
        </div>
      </div>

    </div>

    <!-- RIGHT: Sticky sidebar -->
    <div class="col-lg-3">
      <div style="position:sticky;top:1rem">

        <!-- Recent activity -->
        <div class="card">
          <div class="card-header">
            <h3 class="card-title">
              Aktivitas Terbaru
            </h3>
          </div>
          <div v-if="loading" class="card-body placeholder-glow">
            <div v-for="n in 5" :key="n" class="mb-3">
              <div class="placeholder col-8 mb-1"></div>
              <div class="placeholder col-5"></div>
            </div>
          </div>
          <div v-else-if="!recentActivity.length" class="card-body text-center text-muted py-4">
            <i class="ti ti-activity d-block opacity-50 mb-1" style="font-size:1.5rem"></i>
            Belum ada aktivitas
          </div>
          <div v-else class="list-group list-group-flush">
            <div v-for="session in recentActivity" :key="session.id" class="list-group-item">
              <div class="row align-items-center">
                <div class="col-auto">
                  <span class="avatar avatar-sm rounded-circle">
                    {{ (session.user?.name ?? 'P').split(' ').map((w: string) => w[0]).slice(0, 2).join('').toUpperCase() }}
                  </span>
                </div>
                <div class="col text-truncate">
                  <div class="text-reset d-block fw-bold">{{ session.user?.name ?? 'Peserta' }}</div>
                  <div class="d-block text-muted text-truncate mt-n1 small">
                    <span class="badge me-1" :class="statusBadge(session.status)" style="font-size:.6rem">{{ session.status }}</span>
                    {{ session.exam_schedule?.title ?? '' }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
.btn-outline-secondary:hover {
  background-color: rgba(var(--tblr-primary-rgb), 0.04);
  border-color: var(--tblr-primary);
  color: var(--tblr-primary);
}
</style>
