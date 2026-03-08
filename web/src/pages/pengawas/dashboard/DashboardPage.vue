<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../../stores/auth.store'
import {
  dashboardApi,
  type PengawasDashboardStats,
  type PengawasMonitoringSummary,
  type PengawasViolation,
  type PengawasActiveRoom,
} from '../../../api/dashboard.api'
import { getIllustration } from '../../../utils/avatar'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const authStore = useAuthStore()
const router = useRouter()
const loading = ref(true)
const stats = ref<PengawasDashboardStats | null>(null)
const monitoring = ref<PengawasMonitoringSummary | null>(null)
const recentViolations = ref<PengawasViolation[]>([])
const activeRooms = ref<PengawasActiveRoom[]>([])

let refreshTimer: ReturnType<typeof setInterval> | null = null

const violationLabels: Record<string, string> = {
  tab_switch: 'Pindah Tab',
  blur_extended: 'Tab Tidak Aktif Lama',
  multi_tab: 'Multi-Tab Terdeteksi',
  popup_detected: 'Buka Popup/Window',
  background_detected: 'Split-Screen / Background',
  external_paste: 'Paste Teks Eksternal',
  alt_tab: 'Alt+Tab / Pindah Window',
  fullscreen_exit: 'Keluar Fullscreen',
  window_resize: 'Split-Screen / Floating App',
  focus_lost: 'Window Kehilangan Fokus',
}


function formatTime(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

function timeRemaining(endTime: string): string {
  const diff = new Date(endTime).getTime() - Date.now()
  if (diff <= 0) return 'Selesai'
  const hours = Math.floor(diff / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (hours > 0) return `${hours}j ${mins}m`
  return `${mins} menit`
}

function violationSeverity(type: string): 'high' | 'medium' | 'low' {
  if (['fullscreen_exit', 'multi_tab', 'alt_tab', 'external_paste'].includes(type)) return 'high'
  if (['tab_switch', 'blur_extended', 'window_resize', 'focus_lost'].includes(type)) return 'medium'
  return 'low'
}

function severityBadgeClass(type: string) {
  const sev = violationSeverity(type)
  return sev === 'high' ? 'bg-danger-lt text-danger' : sev === 'medium' ? 'bg-warning-lt text-warning' : 'bg-secondary-lt text-secondary'
}

async function loadData() {
  try {
    const [statsRes, monRes, violRes, roomsRes] = await Promise.allSettled([
      dashboardApi.getPengawasStats(),
      dashboardApi.getPengawasMonitoringSummary(),
      dashboardApi.getPengawasRecentViolations(),
      dashboardApi.getPengawasActiveRooms(),
    ])
    if (statsRes.status === 'fulfilled') stats.value = statsRes.value.data.data
    if (monRes.status === 'fulfilled') monitoring.value = monRes.value.data.data
    if (violRes.status === 'fulfilled') recentViolations.value = violRes.value.data.data ?? []
    if (roomsRes.status === 'fulfilled') activeRooms.value = roomsRes.value.data.data ?? []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
  // Auto-refresh every 30 seconds
  refreshTimer = setInterval(() => {
    loadData()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

const statCards = [
  { label: 'Jadwal Aktif', key: 'active_schedules' as const, icon: 'ti-calendar-event', color: 'primary' },
  { label: 'Sesi Berlangsung', key: 'ongoing_sessions' as const, icon: 'ti-device-desktop-analytics', color: 'red' },
  { label: 'Selesai Hari Ini', key: 'finished_today' as const, icon: 'ti-circle-check', color: 'green' },
]
</script>

<template>
  <BasePageHeader
    title="Dashboard Pengawas"
    :subtitle="`Selamat datang, ${authStore.user?.name}`"
    :breadcrumbs="[{ label: 'Dashboard' }]"
  >
    <template #actions>
      <span class="badge bg-green-lt text-green" title="Auto-refresh setiap 30 detik">
        <i class="ti ti-refresh me-1"></i>Live
      </span>
    </template>
  </BasePageHeader>

  <!-- Stats Cards -->
  <div v-if="loading" class="row g-3 mb-3">
    <div v-for="n in 3" :key="n" class="col-sm-6 col-lg-4">
      <div class="card placeholder-glow">
        <div class="card-body">
          <div class="placeholder col-6 mb-2"></div>
          <div class="placeholder col-4"></div>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="row g-3 mb-3">
    <div v-for="card in statCards" :key="card.key" class="col-sm-6 col-lg-4">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span :class="`bg-${card.color}-lt`" class="avatar">
                <i :class="['ti fs-4', card.icon, `text-${card.color}`]"></i>
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

  <!-- Real-time Monitoring Summary -->
  <div v-if="!loading && monitoring" class="row g-3 mb-3">
    <div class="col-6 col-lg-3">
      <div class="card card-sm border-start border-primary border-3">
        <div class="card-body">
          <div class="d-flex align-items-center">
            <span class="avatar bg-primary-lt me-3">
              <i class="ti ti-users fs-4 text-primary"></i>
            </span>
            <div>
              <div class="h3 mb-0 fw-bold">{{ monitoring.online_students }}</div>
              <div class="text-muted small">Peserta Online</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-6 col-lg-3">
      <div class="card card-sm border-start border-danger border-3">
        <div class="card-body">
          <div class="d-flex align-items-center">
            <span class="avatar bg-danger-lt me-3">
              <i class="ti ti-alert-triangle fs-4 text-danger"></i>
            </span>
            <div>
              <div class="h3 mb-0 fw-bold">{{ monitoring.violations_today }}</div>
              <div class="text-muted small">Pelanggaran Hari Ini</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-6 col-lg-3">
      <div class="card card-sm border-start border-success border-3">
        <div class="card-body">
          <div class="d-flex align-items-center">
            <span class="avatar bg-success-lt me-3">
              <i class="ti ti-activity fs-4 text-success"></i>
            </span>
            <div>
              <div class="h3 mb-0 fw-bold">{{ monitoring.active_sessions }}</div>
              <div class="text-muted small">Sesi Aktif</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-6 col-lg-3">
      <div class="card card-sm border-start border-info border-3">
        <div class="card-body">
          <div class="d-flex align-items-center">
            <span class="avatar bg-info-lt me-3">
              <i class="ti ti-calendar-stats fs-4 text-info"></i>
            </span>
            <div>
              <div class="h3 mb-0 fw-bold">{{ monitoring.active_schedules }}</div>
              <div class="text-muted small">Jadwal Aktif</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Quick Action Cards -->
  <div class="row g-3 mb-3">
    <div class="col-sm-6 col-lg-4">
      <div class="card card-link card-link-pop cursor-pointer" @click="router.push('/pengawas/supervision')">
        <div class="card-body d-flex align-items-center">
          <span class="avatar avatar-lg bg-primary-lt me-3">
            <i class="ti ti-shield-check fs-2 text-primary"></i>
          </span>
          <div>
            <h3 class="mb-1">Mulai Pengawasan</h3>
            <p class="text-muted mb-0 small">Pantau ruang ujian secara real-time</p>
          </div>
          <i class="ti ti-chevron-right ms-auto text-muted"></i>
        </div>
      </div>
    </div>
    <div class="col-sm-6 col-lg-4">
      <div class="card card-link card-link-pop cursor-pointer" @click="router.push('/pengawas/live-score')">
        <div class="card-body d-flex align-items-center">
          <span class="avatar avatar-lg bg-green-lt me-3">
            <i class="ti ti-device-tv fs-2 text-green"></i>
          </span>
          <div>
            <h3 class="mb-1">Live Score TV Mode</h3>
            <p class="text-muted mb-0 small">Tampilkan skor langsung di layar</p>
          </div>
          <i class="ti ti-chevron-right ms-auto text-muted"></i>
        </div>
      </div>
    </div>
    <div class="col-sm-6 col-lg-4">
      <div class="card card-link card-link-pop cursor-pointer" @click="router.push('/pengawas/violations')">
        <div class="card-body d-flex align-items-center">
          <span class="avatar avatar-lg bg-red-lt me-3">
            <i class="ti ti-alert-octagon fs-2 text-red"></i>
          </span>
          <div>
            <h3 class="mb-1">Log Pelanggaran</h3>
            <p class="text-muted mb-0 small">Lihat semua pelanggaran peserta</p>
          </div>
          <i class="ti ti-chevron-right ms-auto text-muted"></i>
        </div>
      </div>
    </div>
  </div>

  <div class="row g-3 mb-3">
    <!-- Ruangan Saya / Active Exam Rooms -->
    <div class="col-lg-7">
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-building me-2 text-primary"></i>Ruang Ujian Aktif
          </h3>
          <div class="card-actions">
            <a href="#" class="btn btn-sm btn-outline-primary" @click.prevent="router.push('/pengawas/supervision')">
              <i class="ti ti-eye me-1"></i>Lihat Semua
            </a>
          </div>
        </div>

        <div v-if="loading" class="card-body">
          <div class="placeholder-glow">
            <div class="placeholder col-12 mb-2" style="height:60px"></div>
            <div class="placeholder col-12 mb-2" style="height:60px"></div>
          </div>
        </div>

        <div v-else-if="!activeRooms.length" class="card-body text-center py-5">
          <img :src="getIllustration('hybrid-work')" class="img-fluid mb-3 opacity-75" style="max-height:100px" alt="">
          <p class="text-muted mb-0">Tidak ada ujian aktif saat ini.</p>
        </div>

        <div v-else class="list-group list-group-flush">
          <div
            v-for="room in activeRooms"
            :key="room.schedule_id"
            class="list-group-item list-group-item-action cursor-pointer"
            @click="router.push(`/pengawas/supervision/${room.schedule_id}`)"
          >
            <div class="row align-items-center">
              <div class="col-auto">
                <span class="avatar" :class="room.status === 'active' ? 'bg-green-lt' : 'bg-secondary-lt'">
                  <i class="ti" :class="room.status === 'active' ? 'ti-broadcast text-green' : 'ti-calendar-clock text-secondary'"></i>
                </span>
              </div>
              <div class="col text-truncate">
                <div class="fw-medium d-flex align-items-center gap-2">
                  {{ room.schedule_name }}
                  <span v-if="room.status === 'active'" class="badge bg-green text-green-fg">LIVE</span>
                </div>
                <div class="d-flex gap-3 mt-1 text-muted small">
                  <span><i class="ti ti-users me-1"></i>{{ room.online_students }}/{{ room.total_students }} online</span>
                  <span v-if="room.violation_count > 0" class="text-danger">
                    <i class="ti ti-alert-triangle me-1"></i>{{ room.violation_count }} pelanggaran
                  </span>
                  <span><i class="ti ti-clock me-1"></i>{{ timeRemaining(room.end_time) }}</span>
                </div>
              </div>
              <div class="col-auto">
                <div class="progress progress-sm" style="width:60px">
                  <div
                    class="progress-bar"
                    :class="room.online_students > 0 ? 'bg-green' : 'bg-secondary'"
                    :style="{ width: (room.total_students ? (room.online_students / room.total_students) * 100 : 0) + '%' }"
                  ></div>
                </div>
              </div>
              <div class="col-auto">
                <i class="ti ti-chevron-right text-muted"></i>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pelanggaran Terbaru -->
    <div class="col-lg-5">
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-alert-triangle me-2 text-danger"></i>Pelanggaran Terbaru
          </h3>
          <div class="card-actions">
            <a href="#" class="btn btn-sm btn-outline-danger" @click.prevent="router.push('/pengawas/violations')">
              <i class="ti ti-list me-1"></i>Semua
            </a>
          </div>
        </div>

        <div v-if="loading" class="card-body">
          <div class="placeholder-glow">
            <div class="placeholder col-12 mb-2" style="height:40px"></div>
            <div class="placeholder col-12 mb-2" style="height:40px"></div>
            <div class="placeholder col-12" style="height:40px"></div>
          </div>
        </div>

        <div v-else-if="!recentViolations.length" class="card-body text-center py-4">
          <i class="ti ti-shield-check fs-1 text-success d-block mb-2"></i>
          <p class="text-muted mb-0">Tidak ada pelanggaran hari ini.</p>
        </div>

        <div v-else class="list-group list-group-flush">
          <div v-for="v in recentViolations.slice(0, 5)" :key="v.id" class="list-group-item">
            <div class="d-flex align-items-start gap-2">
              <span class="badge mt-1" :class="severityBadgeClass(v.violation_type)">
                <i class="ti ti-alert-triangle"></i>
              </span>
              <div class="flex-fill">
                <div class="fw-medium small">{{ v.student_name }}</div>
                <div class="text-muted small">
                  {{ violationLabels[v.violation_type] ?? v.violation_type }}
                  <span class="text-muted"> &mdash; {{ v.schedule_name }}</span>
                </div>
              </div>
              <span class="text-muted small text-nowrap">{{ formatTime(v.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cursor-pointer { cursor: pointer; }
</style>
