<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { dashboardApi, type PengawasActiveRoom } from '../../../api/dashboard.api'

import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const router = useRouter()
const toast = useToastStore()
const loading = ref(true)
const loadError = ref('')
const rooms = ref<PengawasActiveRoom[]>([])
let refreshTimer: ReturnType<typeof setInterval> | null = null

function timeRemaining(endTime: string): string {
  const diff = new Date(endTime).getTime() - Date.now()
  if (diff <= 0) return 'Selesai'
  const hours = Math.floor(diff / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (hours > 0) return `${hours}j ${mins}m tersisa`
  return `${mins} menit tersisa`
}

function progressPct(room: PengawasActiveRoom): number {
  if (!room.total_students) return 0
  return Math.round((room.online_students / room.total_students) * 100)
}

function statusColor(room: PengawasActiveRoom): string {
  if (room.status === 'active' && room.online_students > 0) return 'green'
  if (room.status === 'active') return 'yellow'
  return 'secondary'
}

async function loadRooms() {
  loadError.value = ''
  try {
    const res = await dashboardApi.getPengawasActiveRooms()
    rooms.value = res.data.data ?? []
  } catch (e: any) {
    const msg = e?.response?.data?.message ?? 'Gagal memuat data ruangan'
    loadError.value = msg
    toast.error(msg)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRooms()
  refreshTimer = setInterval(loadRooms, 30000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <BasePageHeader
    title="Hub Pengawasan"
    subtitle="Pantau semua ruang ujian yang sedang berlangsung"
    :breadcrumbs="[{ label: 'Pengawasan' }]"
  >
    <template #actions>
      <span class="badge bg-green-lt text-green" title="Auto-refresh setiap 30 detik">
        <i class="ti ti-refresh me-1"></i>Auto-refresh 30s
      </span>
      <button class="btn btn-ghost-secondary" @click="loadRooms">
        <i class="ti ti-refresh"></i> Refresh
      </button>
    </template>
  </BasePageHeader>

  <!-- Loading -->
  <div v-if="loading" class="row g-3">
    <div v-for="n in 4" :key="n" class="col-md-6 col-xl-3">
      <div class="card placeholder-glow">
        <div class="card-body">
          <div class="placeholder col-8 mb-3"></div>
          <div class="placeholder col-6 mb-2"></div>
          <div class="placeholder col-12" style="height:8px"></div>
          <div class="placeholder col-4 mt-3"></div>
        </div>
      </div>
    </div>
  </div>

  <!-- Error state -->
  <div v-else-if="loadError" class="text-center py-5">
    <div class="alert alert-danger d-inline-flex align-items-center gap-2 mb-3">
      <i class="ti ti-alert-circle"></i>
      <span>{{ loadError }}</span>
    </div>
    <div>
      <button class="btn btn-outline-primary" @click="loading = true; loadRooms()">
        <i class="ti ti-refresh me-1"></i>Coba Lagi
      </button>
    </div>
  </div>

  <!-- Empty state -->
  <div v-else-if="!rooms.length" class="text-center py-5">
    <i class="ti ti-eye-off fs-1 mb-2 d-block opacity-50"></i>
    <h3 class="text-muted">Tidak Ada Ujian Aktif</h3>
    <p class="text-muted">Ruang ujian yang sedang berlangsung akan muncul di sini.</p>
    <button class="btn btn-outline-primary mt-2" @click="router.push('/pengawas')">
      <i class="ti ti-arrow-left me-1"></i>Kembali ke Dashboard
    </button>
  </div>

  <!-- Room Cards Grid -->
  <div v-else class="row g-3">
    <div v-for="room in rooms" :key="room.schedule_id" class="col-md-6 col-xl-4">
      <div
        class="card card-link card-link-pop h-100 cursor-pointer"
        :class="{ 'border-green': room.status === 'active' && room.online_students > 0 }"
        @click="router.push(`/pengawas/supervision/${room.schedule_id}`)"
      >
        <div class="card-body">
          <!-- Header -->
          <div class="d-flex align-items-start mb-3">
            <span class="avatar avatar-lg me-3" :class="`bg-${statusColor(room)}-lt`">
              <i class="ti ti-broadcast fs-2" :class="`text-${statusColor(room)}`"></i>
            </span>
            <div class="flex-fill">
              <h3 class="mb-1">{{ room.schedule_name }}</h3>
              <div class="d-flex align-items-center gap-2">
                <span v-if="room.status === 'active'" class="badge bg-green text-green-fg">
                  <i class="ti ti-broadcast me-1"></i>LIVE
                </span>
                <span v-else class="badge bg-secondary-lt text-secondary">
                  <i class="ti ti-clock me-1"></i>Published
                </span>
              </div>
            </div>
          </div>

          <!-- Stats row -->
          <div class="row g-2 mb-3">
            <div class="col-4 text-center">
              <div class="h2 mb-0 fw-bold text-primary">{{ room.online_students }}</div>
              <div class="text-muted small">Online</div>
            </div>
            <div class="col-4 text-center">
              <div class="h2 mb-0 fw-bold">{{ room.total_students }}</div>
              <div class="text-muted small">Total</div>
            </div>
            <div class="col-4 text-center">
              <div class="h2 mb-0 fw-bold" :class="room.violation_count > 0 ? 'text-danger' : 'text-muted'">{{ room.violation_count }}</div>
              <div class="text-muted small">Pelanggaran</div>
            </div>
          </div>

          <!-- Progress bar -->
          <div class="mb-2">
            <div class="d-flex justify-content-between small text-muted mb-1">
              <span>Kehadiran Online</span>
              <span>{{ progressPct(room) }}%</span>
            </div>
            <div class="progress progress-sm">
              <div
                class="progress-bar"
                :class="progressPct(room) >= 80 ? 'bg-green' : progressPct(room) >= 50 ? 'bg-yellow' : 'bg-red'"
                :style="{ width: progressPct(room) + '%' }"
              ></div>
            </div>
          </div>

          <!-- Footer info -->
          <div class="d-flex justify-content-between align-items-center mt-3 pt-2 border-top">
            <span class="text-muted small">
              <i class="ti ti-clock me-1"></i>{{ timeRemaining(room.end_time) }}
            </span>
            <span class="text-muted small">
              <i class="ti ti-hourglass me-1"></i>{{ room.duration_minutes }} menit
            </span>
          </div>
        </div>

        <!-- Action footer -->
        <div class="card-footer text-center">
          <span class="text-primary fw-medium">
            <i class="ti ti-eye me-1"></i>Masuk Ruang Pengawasan
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cursor-pointer { cursor: pointer; }
.border-green { border-color: var(--tblr-green) !important; border-width: 2px !important; }
</style>
