<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { examApi, type ExamSchedule, type ExamSession } from '../../../api/exam.api'
import { useAuthStore } from '../../../stores/auth.store'
import { useToastStore } from '../../../stores/toast.store'
import { getIllustration } from '../../../utils/avatar'

const router = useRouter()
const authStore = useAuthStore()
const toast = useToastStore()

const available = ref<ExamSchedule[]>([])
const history = ref<ExamSession[]>([])
const loading = ref(true)
const now = ref(Date.now())
const currentTime = ref(new Date())

// Countdown timer interval
let countdownInterval: ReturnType<typeof setInterval> | null = null

function startCountdown() {
  countdownInterval = setInterval(() => {
    now.value = Date.now()
    currentTime.value = new Date()
  }, 1000)
}

async function fetchAvailable() {
  loading.value = true
  try {
    const [availRes, histRes] = await Promise.all([
      examApi.getAvailable(),
      examApi.getMyHistory(),
    ])
    available.value = availRes.data.data ?? []
    history.value = histRes.data.data ?? []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat daftar ujian')
  } finally {
    loading.value = false
  }
}

/** Find session for a finished schedule (to link to report) */
function getSessionForSchedule(scheduleId: number): ExamSession | undefined {
  return history.value.find(s => s.exam_schedule_id === scheduleId)
}

/** Whether a finished schedule allows report viewing */
function canViewReport(s: ExamSchedule): boolean {
  return s.allow_see_result === true && !!getSessionForSchedule(s.id)
}

function goToExam(schedule: ExamSchedule) {
  router.push(`/peserta/confirm/${schedule.id}`)
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('id-ID', {
    day: 'numeric', month: 'short',
    year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

function formatDateShort(d: string) {
  return new Date(d).toLocaleString('id-ID', {
    day: 'numeric', month: 'short', year: 'numeric',
  })
}

function isActive(s: ExamSchedule) {
  return s.status === 'active' ||
    (s.status === 'published' && now.value >= new Date(s.start_time).getTime() && now.value < new Date(s.end_time).getTime())
}

function isUpcoming(s: ExamSchedule) {
  return s.status === 'published' && now.value < new Date(s.start_time).getTime()
}

function statusLabel(s: ExamSchedule) {
  if (s.status === 'finished') return 'Selesai'
  if (isActive(s)) return 'Berlangsung'
  if (isUpcoming(s)) return 'Akan Datang'
  return s.status
}

function statusClass(s: ExamSchedule) {
  if (s.status === 'finished') return 'bg-success-lt text-success'
  if (isActive(s)) return 'bg-green text-green-fg'
  if (isUpcoming(s)) return 'bg-yellow-lt text-yellow'
  return 'bg-secondary-lt text-secondary'
}

/** Get countdown string for upcoming exams */
function getCountdown(s: ExamSchedule): string {
  const diff = new Date(s.start_time).getTime() - now.value
  if (diff <= 0) return 'Segera dimulai'
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)
  if (days > 0) return `${days} hari ${hours} jam ${minutes} menit`
  if (hours > 0) return `${hours} jam ${minutes} menit ${seconds} detik`
  return `${minutes} menit ${seconds} detik`
}

/** Check if a session was terminated */
function isTerminated(s: ExamSession): boolean {
  return s.status === 'terminated'
}

/** Get user initials for avatar fallback */
const userInitials = computed(() => {
  const name = authStore.user?.name ?? ''
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return ((parts[0]?.[0] ?? '') + (parts[1]?.[0] ?? '')).toUpperCase()
  return name.substring(0, 2).toUpperCase()
})

/** Formatted current date */
const formattedCurrentDate = computed(() => {
  return currentTime.value.toLocaleDateString('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
})

const formattedCurrentTime = computed(() => {
  return currentTime.value.toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
})

// Stats
const activeExams = computed(() => available.value.filter(s => isActive(s)))

const totalExams = computed(() => available.value.length + history.value.length)

const completedExams = computed(() => {
  const finishedSchedules = available.value.filter(s => s.status === 'finished').length
  const finishedSessions = history.value.filter(s => s.status === 'finished' || s.status === 'terminated').length
  return Math.max(finishedSchedules, finishedSessions)
})

const ongoingExams = computed(() => activeExams.value.length)

onMounted(() => {
  fetchAvailable()
  startCountdown()
})

onUnmounted(() => {
  if (countdownInterval) {
    clearInterval(countdownInterval)
    countdownInterval = null
  }
})
</script>

<template>
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <h2 class="page-title">Halo, {{ authStore.user?.name }}</h2>
        <p class="text-muted mb-0">Daftar ujian yang tersedia untuk Anda</p>
      </div>
      <div class="col-auto">
        <button class="btn btn-outline-secondary btn-sm" @click="fetchAvailable" :disabled="loading">
          <i class="ti ti-refresh me-1" :class="loading ? 'animate-spin' : ''"></i>Refresh
        </button>
      </div>
    </div>
  </div>

  <!-- Mobile Stats Bar (visible only on small screens) -->
  <div class="d-lg-none mb-3">
    <div class="row g-2">
      <div class="col-4">
        <div class="card card-sm">
          <div class="card-body text-center py-2">
            <div class="text-muted small">Total Ujian</div>
            <div class="h2 mb-0">{{ totalExams }}</div>
          </div>
        </div>
      </div>
      <div class="col-4">
        <div class="card card-sm">
          <div class="card-body text-center py-2">
            <div class="text-muted small">Selesai</div>
            <div class="h2 mb-0 text-success">{{ completedExams }}</div>
          </div>
        </div>
      </div>
      <div class="col-4">
        <div class="card card-sm">
          <div class="card-body text-center py-2">
            <div class="text-muted small">Berlangsung</div>
            <div class="h2 mb-0 text-primary">{{ ongoingExams }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div class="row g-3">
    <!-- Left Sidebar (desktop only) -->
    <div class="col-lg-3 d-none d-lg-block">
      <div class="sidebar-sticky">
        <!-- User Profile Card -->
        <div class="card mb-3">
          <div class="card-body text-center">
            <div class="mb-3">
              <span class="avatar avatar-xl rounded-circle bg-primary-lt text-primary fs-2">
                {{ userInitials }}
              </span>
            </div>
            <h3 class="mb-1">{{ authStore.user?.name }}</h3>
            <p class="text-muted mb-0">
              <span class="badge bg-blue-lt">
                <i class="ti ti-user me-1"></i>Peserta
              </span>
            </p>
          </div>
        </div>

        <!-- Date/Time Card -->
        <div class="card mb-3">
          <div class="card-body">
            <div class="d-flex align-items-center mb-2">
              <i class="ti ti-calendar-event me-2 text-primary"></i>
              <span class="text-muted small">{{ formattedCurrentDate }}</span>
            </div>
            <div class="d-flex align-items-center">
              <i class="ti ti-clock me-2 text-primary"></i>
              <span class="fw-bold">{{ formattedCurrentTime }}</span>
            </div>
          </div>
        </div>

        <!-- Stats Cards -->
        <div class="card mb-3">
          <div class="card-header">
            <h3 class="card-title">
              <i class="ti ti-chart-bar me-2 text-primary"></i>Statistik
            </h3>
          </div>
          <div class="card-body">
            <div class="mb-3">
              <div class="d-flex align-items-center mb-1">
                <i class="ti ti-clipboard-list me-2 text-blue"></i>
                <span class="text-muted">Total Ujian</span>
                <span class="ms-auto fw-bold">{{ totalExams }}</span>
              </div>
              <div class="progress progress-sm">
                <div class="progress-bar bg-blue" :style="{ width: totalExams > 0 ? '100%' : '0%' }"></div>
              </div>
            </div>
            <div class="mb-3">
              <div class="d-flex align-items-center mb-1">
                <i class="ti ti-circle-check me-2 text-success"></i>
                <span class="text-muted">Sudah Selesai</span>
                <span class="ms-auto fw-bold text-success">{{ completedExams }}</span>
              </div>
              <div class="progress progress-sm">
                <div class="progress-bar bg-success" :style="{ width: totalExams > 0 ? (completedExams / totalExams * 100) + '%' : '0%' }"></div>
              </div>
            </div>
            <div>
              <div class="d-flex align-items-center mb-1">
                <i class="ti ti-broadcast me-2 text-primary"></i>
                <span class="text-muted">Sedang Berlangsung</span>
                <span class="ms-auto fw-bold text-primary">{{ ongoingExams }}</span>
              </div>
              <div class="progress progress-sm">
                <div class="progress-bar bg-primary" :style="{ width: totalExams > 0 ? (ongoingExams / totalExams * 100) + '%' : '0%' }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Right Content -->
    <div class="col-lg-9">
      <!-- Quick resume button for active exams -->
      <div v-if="!loading && activeExams.length" class="alert alert-success d-flex align-items-center mb-3">
        <i class="ti ti-broadcast me-2 fs-3"></i>
        <div class="me-auto">
          <strong>Ujian sedang berlangsung!</strong>
          <div class="small">{{ activeExams[0]!.name }}</div>
        </div>
        <button class="btn btn-success ms-3" @click="goToExam(activeExams[0]!)">
          <i class="ti ti-player-play me-1"></i>Lanjutkan Ujian
        </button>
      </div>

      <div v-if="loading" class="p-5 text-center text-muted">
        <span class="spinner-border spinner-border-sm me-2"></span>Memuat ujian...
      </div>

      <template v-else>
        <!-- Active Exams - Highlighted -->
        <div v-if="activeExams.length" class="mb-3">
          <h3 class="mb-2 text-green fw-medium"><i class="ti ti-broadcast me-2"></i>Sedang Berlangsung</h3>
          <div class="row g-3">
            <div v-for="s in activeExams" :key="`active-${s.id}`" class="col-md-6 col-lg-4">
              <div class="card border-success shadow-sm card-link card-link-pop" @click="goToExam(s)">
                <div class="card-body">
                  <div class="d-flex align-items-center gap-2 mb-2">
                    <span class="badge bg-green text-green-fg">
                      <i class="ti ti-broadcast me-1"></i>Sedang Berlangsung
                    </span>
                  </div>
                  <h4 class="card-title mb-2">{{ s.name }}</h4>
                  <div class="d-flex flex-column gap-1 text-muted small mb-3">
                    <div><i class="ti ti-calendar me-1"></i>{{ formatDate(s.start_time) }}</div>
                    <div><i class="ti ti-clock me-1"></i>Durasi: {{ s.duration_minutes }} menit</div>
                    <div><i class="ti ti-clock-stop me-1"></i>Berakhir: {{ formatDate(s.end_time) }}</div>
                  </div>
                  <button class="btn btn-success w-100">
                    <i class="ti ti-pencil me-1"></i>Mulai Ujian Sekarang
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- All Exams Table -->
        <div class="card">
          <div class="card-header">
            <h3 class="card-title">
              <i class="ti ti-list-check me-2 text-primary"></i>Jadwal Ujian Saya
            </h3>
          </div>

          <div v-if="!available.length" class="card-body text-center py-5">
            <img :src="getIllustration('calendar')" class="img-fluid mb-3 opacity-75" style="max-height:120px" alt="">
            <h3 class="text-muted fw-normal">Tidak ada ujian saat ini</h3>
            <p class="text-muted small mb-0">Ujian akan muncul di sini ketika sudah dipublikasi oleh pengajar.</p>
          </div>

          <div v-else class="table-responsive">
            <table class="table table-vcenter table-hover">
              <thead>
                <tr>
                  <th>Nama Ujian</th>
                  <th>Tanggal Mulai</th>
                  <th>Durasi</th>
                  <th>Status</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in available" :key="s.id">
                  <td>
                    <div class="fw-medium">{{ s.name }}</div>
                    <!-- Countdown for upcoming exams -->
                    <div v-if="isUpcoming(s)" class="text-yellow small mt-1">
                      <i class="ti ti-hourglass-low me-1"></i>Dimulai dalam {{ getCountdown(s) }}
                    </div>
                  </td>
                  <td class="text-muted">
                    <i class="ti ti-calendar me-1"></i>{{ formatDateShort(s.start_time) }}
                  </td>
                  <td class="text-muted">
                    <i class="ti ti-clock me-1"></i>{{ s.duration_minutes }} menit
                  </td>
                  <td>
                    <span class="badge" :class="statusClass(s)">
                      <i class="ti ti-broadcast me-1" v-if="isActive(s)"></i>
                      {{ statusLabel(s) }}
                    </span>
                  </td>
                  <td>
                    <div class="d-flex gap-1">
                      <button
                        v-if="isActive(s)"
                        class="btn btn-sm btn-success"
                        @click="goToExam(s)"
                      >
                        <i class="ti ti-pencil me-1"></i>Mulai Ujian
                      </button>
                      <button
                        v-else-if="isUpcoming(s)"
                        class="btn btn-sm btn-outline-secondary"
                        disabled
                      >
                        <i class="ti ti-clock me-1"></i>Belum Mulai
                      </button>
                      <template v-else>
                        <button class="btn btn-sm btn-ghost-secondary" disabled>
                          <i class="ti ti-check me-1"></i>Selesai
                        </button>
                        <button
                          v-if="canViewReport(s)"
                          class="btn btn-sm btn-outline-primary"
                          @click="router.push(`/peserta/report/${getSessionForSchedule(s.id)!.id}`)"
                        >
                          <i class="ti ti-file-analytics me-1"></i>Pembahasan
                        </button>
                      </template>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Exam History (finished sessions) -->
        <div v-if="history.length" class="mt-3">
          <h3 class="mb-2">
            <i class="ti ti-history me-2 text-green"></i>Riwayat Ujian
          </h3>
          <div class="row g-3">
            <div v-for="s in history" :key="s.id" class="col-12">
              <!-- Terminated / Locked exam card -->
              <div
                v-if="isTerminated(s)"
                class="card border-danger"
              >
                <div class="card-body">
                  <div class="row align-items-center">
                    <div class="col-auto">
                      <span class="avatar avatar-md bg-danger-lt text-danger rounded">
                        <i class="ti ti-lock fs-2"></i>
                      </span>
                    </div>
                    <div class="col">
                      <div class="d-flex align-items-center gap-2 mb-1">
                        <h4 class="card-title mb-0">{{ s.exam_schedule?.name ?? `Sesi #${s.id}` }}</h4>
                        <span class="badge bg-danger">
                          <i class="ti ti-lock me-1"></i>Terminated
                        </span>
                      </div>
                      <div class="text-muted small">
                        <i class="ti ti-calendar-check me-1"></i>
                        {{ s.finished_at ? formatDate(s.finished_at) : '-' }}
                      </div>
                      <div class="text-danger small mt-1">
                        <i class="ti ti-alert-triangle me-1"></i>
                        Ujian dihentikan karena pelanggaran ({{ s.violation_count }} pelanggaran tercatat)
                      </div>
                    </div>
                    <div class="col-auto">
                      <div class="text-end">
                        <div class="mb-1">
                          <span class="fw-bold">{{ s.score.toFixed(1) }}</span>
                          <span class="text-muted"> / {{ s.max_score.toFixed(1) }}</span>
                        </div>
                        <button
                          v-if="s.exam_schedule?.allow_see_result"
                          class="btn btn-sm btn-outline-primary"
                          @click="router.push(`/peserta/report/${s.id}`)"
                        >
                          <i class="ti ti-file-analytics me-1"></i>Pembahasan
                        </button>
                        <span v-else class="badge bg-secondary-lt text-muted">
                          <i class="ti ti-lock me-1"></i>Terkunci
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Normal finished exam card -->
              <div v-else class="card">
                <div class="card-body">
                  <div class="row align-items-center">
                    <div class="col-auto">
                      <span class="avatar avatar-md bg-success-lt text-success rounded">
                        <i class="ti ti-circle-check fs-2"></i>
                      </span>
                    </div>
                    <div class="col">
                      <h4 class="card-title mb-1">{{ s.exam_schedule?.name ?? `Sesi #${s.id}` }}</h4>
                      <div class="text-muted small">
                        <i class="ti ti-calendar-check me-1"></i>
                        {{ s.finished_at ? formatDate(s.finished_at) : '-' }}
                      </div>
                    </div>
                    <div class="col-auto">
                      <div class="text-end">
                        <div class="mb-1">
                          <span class="fw-bold fs-3">{{ s.score.toFixed(1) }}</span>
                          <span class="text-muted"> / {{ s.max_score.toFixed(1) }}</span>
                        </div>
                        <button
                          v-if="s.exam_schedule?.allow_see_result"
                          class="btn btn-sm btn-outline-primary"
                          @click="router.push(`/peserta/report/${s.id}`)"
                        >
                          <i class="ti ti-file-analytics me-1"></i>Lihat Pembahasan
                        </button>
                        <span v-else class="badge bg-secondary-lt text-muted">
                          <i class="ti ti-lock me-1"></i>Pembahasan belum tersedia
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.sidebar-sticky {
  position: sticky;
  top: 1rem;
}

/* Terminated card highlight */
.card.border-danger {
  border-width: 2px !important;
  border-left-width: 4px !important;
}

@media (max-width: 767px) {
  .page-header .row {
    flex-direction: column;
    gap: 0.5rem;
  }
  .page-header .col-auto {
    align-self: flex-start;
  }
  .page-header .page-title {
    font-size: 1.25rem;
  }
  /* Active exam cards full width */
  .col-md-6.col-lg-4 {
    flex: 0 0 100%;
    max-width: 100%;
  }
  .card-body .btn.w-100 {
    min-height: 48px;
    font-size: 0.95rem;
  }
  /* Alert responsive */
  .alert.alert-success {
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .alert.alert-success .btn {
    width: 100%;
    min-height: 44px;
    margin-left: 0 !important;
  }
  /* Table: horizontal scroll is handled by .table-responsive already */
  .table th,
  .table td {
    white-space: nowrap;
    font-size: 0.8rem;
  }
  .table .d-flex.gap-1 {
    flex-wrap: nowrap;
  }
  .table .btn-sm {
    font-size: 0.7rem;
    padding: 0.25rem 0.5rem;
    white-space: nowrap;
  }
  /* History cards responsive */
  .card-body .row.align-items-center {
    flex-direction: column;
    text-align: center;
    gap: 0.75rem;
  }
  .card-body .row.align-items-center .col-auto .text-end {
    text-align: center;
  }
}

@media (min-width: 768px) and (max-width: 1024px) {
  .card-body .btn.w-100 {
    min-height: 44px;
  }
}
</style>
