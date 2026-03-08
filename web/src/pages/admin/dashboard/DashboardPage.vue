<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../../stores/auth.store'
import { dashboardApi, type AdminDashboardStats, type ServerStats, type OngoingExam } from '../../../api/dashboard.api'
import { getIllustration } from '../../../utils/avatar'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const authStore = useAuthStore()
const router = useRouter()
const toast = useToastStore()

const loading = ref(true)
const stats = ref<AdminDashboardStats | null>(null)
const upcomingExams = ref<any[]>([])
const recentActivity = ref<any[]>([])
const serverStats = ref<ServerStats | null>(null)
const ongoingExams = ref<OngoingExam[]>([])
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

  if (days > 0) return `${days} hari ${hours} jam ${minutes} menit`
  if (hours > 0) return `${hours} jam ${minutes} menit`
  return `${minutes} menit`
}

function progressColor(percent: number): string {
  if (percent > 80) return 'bg-danger'
  if (percent > 50) return 'bg-warning'
  return 'bg-primary'
}

onMounted(async () => {
  try {
    const [statsRes, upcomingRes, activityRes, serverRes, ongoingRes] = await Promise.allSettled([
      dashboardApi.getAdminStats(),
      dashboardApi.getUpcomingExams(),
      dashboardApi.getRecentActivity(),
      dashboardApi.getServerStats(),
      dashboardApi.getOngoingExams(),
    ])
    if (statsRes.status === 'fulfilled') stats.value = statsRes.value.data.data
    if (upcomingRes.status === 'fulfilled') upcomingExams.value = upcomingRes.value.data.data ?? []
    if (activityRes.status === 'fulfilled') recentActivity.value = activityRes.value.data.data ?? []
    if (serverRes.status === 'fulfilled') serverStats.value = serverRes.value.data.data
    if (ongoingRes.status === 'fulfilled') ongoingExams.value = ongoingRes.value.data.data ?? []

    const failed = [statsRes, upcomingRes, activityRes].filter(r => r.status === 'rejected')
    if (failed.length > 0) {
      toast.error('Sebagian data dashboard gagal dimuat')
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data dashboard')
  } finally {
    loading.value = false
  }

  // Update countdown every 60 seconds
  countdownInterval = setInterval(() => {
    now.value = Date.now()
  }, 60_000)
})

onUnmounted(() => {
  if (countdownInterval) {
    clearInterval(countdownInterval)
    countdownInterval = null
  }
})

const statCards = [
  { label: 'Total Peserta', key: 'total_peserta' as const, icon: 'ti-users', color: 'primary' },
  { label: 'Total Guru', key: 'total_guru' as const, icon: 'ti-school', color: 'purple' },
  { label: 'Total Rombel', key: 'total_rombel' as const, icon: 'ti-users-group', color: 'yellow' },
  { label: 'Bank Soal', key: 'total_question_banks' as const, icon: 'ti-books', color: 'green' },
  { label: 'Jadwal Aktif', key: 'active_schedules' as const, icon: 'ti-calendar-event', color: 'cyan' },
  { label: 'Sesi Berlangsung', key: 'ongoing_sessions' as const, icon: 'ti-device-desktop-analytics', color: 'red' },
  { label: 'Sesi Selesai', key: 'finished_sessions' as const, icon: 'ti-circle-check', color: 'green' },
  { label: 'Total Sesi', key: 'total_sessions' as const, icon: 'ti-calendar-stats', color: 'secondary' },
]

const isAdmin = computed(() => authStore.user?.role === 'admin')

const quickLinks = computed(() => {
  const links = [
    { label: 'Kelola User', path: '/admin/users', icon: 'ti-users' },
    { label: 'Bank Soal', path: '/admin/question-banks', icon: 'ti-books' },
    { label: 'Penjadwalan Ujian', path: '/admin/exam-schedules', icon: 'ti-calendar-event' },
    { label: 'Ruang Pengawasan', path: '/admin/supervision', icon: 'ti-device-desktop-analytics' },
    { label: 'Laporan & Analisis', path: '/admin/reports', icon: 'ti-chart-bar' },
    { label: 'Pengaturan', path: '/admin/settings', icon: 'ti-settings' },
  ]
  if (isAdmin.value) {
    links.push({ label: 'Backup', path: '/admin/backup', icon: 'ti-database-export' })
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
  <BasePageHeader
    title="Dashboard Admin"
    :subtitle="`Selamat datang kembali, ${authStore.user?.name ?? ''}`"
    :breadcrumbs="[{ label: 'Dashboard' }]"
  />

  <div>
    <!-- Stats Cards -->
    <div v-if="loading" class="row g-3 mb-3">
      <div v-for="n in 8" :key="n" class="col-sm-6 col-lg-3">
        <div class="card placeholder-glow">
          <div class="card-body">
            <div class="placeholder col-6 mb-2"></div>
            <div class="placeholder col-4"></div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="row g-3 mb-3">
      <div v-for="card in statCards" :key="card.key" class="col-sm-6 col-lg-3">
        <div class="card card-sm">
          <div class="card-body">
            <div class="row align-items-center">
              <div class="col-auto">
                <span :class="`bg-${card.color}-lt`" class="avatar">
                  <i class="ti fs-4" :class="[card.icon, `text-${card.color}`]"></i>
                </span>
              </div>
              <div class="col">
                <div class="fw-medium h3 mb-0">{{ stats?.[card.key] ?? 0 }}</div>
                <div class="text-muted small">{{ card.label }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Server Stats -->
    <div class="card mb-3">
      <div class="card-header">
        <h3 class="card-title">
          <i class="ti ti-server me-2 text-primary"></i>Status Server
        </h3>
      </div>
      <div class="card-body">
        <div v-if="serverStats" class="row g-3">
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium">
                <i class="ti ti-cpu me-1"></i>CPU
              </span>
              <span class="text-muted">{{ serverStats.cpu_percent }}%</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div
                class="progress-bar"
                :class="progressColor(serverStats.cpu_percent)"
                role="progressbar"
                :style="{ width: serverStats.cpu_percent + '%' }"
              ></div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium">
                <i class="ti ti-device-desktop me-1"></i>RAM
              </span>
              <span class="text-muted">{{ serverStats.ram_used }} / {{ serverStats.ram_total }} ({{ serverStats.ram_percent }}%)</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div
                class="progress-bar"
                :class="progressColor(serverStats.ram_percent)"
                role="progressbar"
                :style="{ width: serverStats.ram_percent + '%' }"
              ></div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium">
                <i class="ti ti-database me-1"></i>Disk
              </span>
              <span class="text-muted">{{ serverStats.disk_used }} / {{ serverStats.disk_total }} ({{ serverStats.disk_percent }}%)</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div
                class="progress-bar"
                :class="progressColor(serverStats.disk_percent)"
                role="progressbar"
                :style="{ width: serverStats.disk_percent + '%' }"
              ></div>
            </div>
          </div>
        </div>
        <div v-else class="row g-3">
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium"><i class="ti ti-cpu me-1"></i>CPU</span>
              <span class="text-muted">--</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div class="progress-bar bg-secondary" role="progressbar" style="width: 0%"></div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium"><i class="ti ti-device-desktop me-1"></i>RAM</span>
              <span class="text-muted">--</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div class="progress-bar bg-secondary" role="progressbar" style="width: 0%"></div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="d-flex justify-content-between mb-1">
              <span class="fw-medium"><i class="ti ti-database me-1"></i>Disk</span>
              <span class="text-muted">--</span>
            </div>
            <div class="progress" style="height: 8px;">
              <div class="progress-bar bg-secondary" role="progressbar" style="width: 0%"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Ongoing Exams -->
    <div v-if="ongoingExams.length > 0" class="card mb-3">
      <div class="card-header">
        <h3 class="card-title d-flex align-items-center">
          <i class="ti ti-live-photo me-2 text-success"></i>Ujian Sedang Berjalan
          <span class="badge bg-success ms-2" style="animation: pulse 2s infinite;">
            LIVE
          </span>
        </h3>
      </div>
      <ul class="list-group list-group-flush">
        <li v-for="exam in ongoingExams" :key="exam.schedule_id" class="list-group-item">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="avatar bg-success-lt">
                <i class="ti ti-live-photo text-success fs-4"></i>
              </span>
            </div>
            <div class="col">
              <div class="fw-medium">{{ exam.schedule_name }}</div>
              <div class="text-muted small">
                <i class="ti ti-book me-1"></i>{{ exam.subject_name }}
                <span class="mx-1">|</span>
                <i class="ti ti-clock me-1"></i>{{ formatTime(exam.start_time) }} - {{ formatTime(exam.end_time) }}
              </div>
              <div class="text-muted small mt-1">
                <i class="ti ti-users me-1"></i>
                <span class="text-success fw-medium">{{ exam.ongoing_count }}</span> sedang mengerjakan,
                <span class="text-primary fw-medium">{{ exam.finished_count }}</span> selesai
                dari {{ exam.total_students }} peserta
              </div>
            </div>
            <div class="col-auto">
              <button class="btn btn-success btn-sm" @click="router.push('/admin/supervision')">
                <i class="ti ti-eye me-1"></i>Pantau
              </button>
            </div>
          </div>
        </li>
      </ul>
    </div>

    <!-- Quick links -->
    <div class="card mb-3">
      <div class="card-header">
        <h3 class="card-title">
          <i class="ti ti-bolt me-2 text-primary"></i>Akses Cepat
        </h3>
      </div>
      <div class="card-body">
        <div class="d-flex flex-wrap gap-2">
          <button
            v-for="link in quickLinks"
            :key="link.path"
            class="btn btn-outline-primary"
            @click="router.push(link.path)"
          >
            <i class="ti me-1" :class="link.icon"></i>
            {{ link.label }}
          </button>
        </div>
      </div>
    </div>

    <!-- Upcoming + Activity -->
    <div v-if="loading" class="row g-3">
      <div v-for="n in 2" :key="n" class="col-md-6">
        <div class="card h-100 placeholder-glow">
          <div class="card-header">
            <div class="placeholder col-5"></div>
          </div>
          <div class="card-body">
            <div v-for="m in 3" :key="m" class="mb-3">
              <div class="placeholder col-8 mb-1"></div>
              <div class="placeholder col-5"></div>
            </div>
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
            <img :src="getIllustration('calendar')" class="img-fluid mb-3 opacity-75" style="max-height:100px" alt="">
            <p class="mb-0">Tidak ada ujian mendatang</p>
          </div>
          <ul v-else class="list-group list-group-flush">
            <li v-for="exam in upcomingExams" :key="exam.id" class="list-group-item">
              <div class="row align-items-center">
                <div class="col">
                  <div class="fw-medium">{{ exam.title }}</div>
                  <div class="text-muted small">
                    <i class="ti ti-clock me-1"></i>{{ formatDate(exam.start_time) }}
                  </div>
                  <div class="mt-1">
                    <span class="badge bg-cyan-lt text-cyan">
                      <i class="ti ti-hourglass-low me-1"></i>{{ formatCountdown(exam.start_time) }}
                    </span>
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
            <img :src="getIllustration('hybrid-work')" class="img-fluid mb-3 opacity-75" style="max-height:100px" alt="">
            <p class="mb-0">Belum ada aktivitas</p>
          </div>
          <ul v-else class="list-group list-group-flush">
            <li v-for="session in recentActivity" :key="session.id" class="list-group-item">
              <div class="row align-items-center">
                <div class="col">
                  <div class="fw-medium">{{ session.user?.name ?? 'Peserta' }}</div>
                  <div class="d-flex align-items-center gap-2 mt-1">
                    <span class="badge" :class="statusBadge(session.status)">{{ session.status }}</span>
                    <span class="text-muted small">{{ session.exam_schedule?.title ?? '' }}</span>
                  </div>
                </div>
              </div>
            </li>
          </ul>
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
</style>
