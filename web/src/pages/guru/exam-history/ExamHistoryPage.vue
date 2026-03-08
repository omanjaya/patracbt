<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { reportApi } from '../../../api/report.api'
import { useToastStore } from '../../../stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const router = useRouter()
const toast = useToastStore()

const loading = ref(true)
const schedules = ref<ExamSchedule[]>([])
const searchQuery = ref('')
const filterStatus = ref<string>('all')
const filterDateFrom = ref('')
const filterDateTo = ref('')

// Report cache: schedule_id -> { mean, total, finished }
const reportCache = ref<Record<number, { mean: number; total: number; finished: number; highest: number; lowest: number }>>({})
const loadingReports = ref<Set<number>>(new Set())

async function loadSchedules() {
  loading.value = true
  try {
    const res = await examApi.listSchedules({ per_page: 200 })
    schedules.value = res.data.data ?? []
    // Preload summary stats for finished schedules
    const finishedSchedules = schedules.value.filter(s => s.status === 'finished')
    for (const s of finishedSchedules.slice(0, 10)) {
      loadReportSummary(s.id)
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data jadwal')
  } finally {
    loading.value = false
  }
}

async function loadReportSummary(scheduleId: number) {
  if (reportCache.value[scheduleId] || loadingReports.value.has(scheduleId)) return
  loadingReports.value.add(scheduleId)
  try {
    const res = await reportApi.getScheduleReport(scheduleId)
    const data = res.data.data
    reportCache.value[scheduleId] = {
      mean: data.stats.mean,
      total: data.stats.total,
      finished: data.stats.finished,
      highest: data.stats.highest,
      lowest: data.stats.lowest,
    }
  } catch {
    // silently fail
  } finally {
    loadingReports.value.delete(scheduleId)
  }
}

const filteredSchedules = computed(() => {
  let result = [...schedules.value]

  // Status filter
  if (filterStatus.value !== 'all') {
    result = result.filter(s => s.status === filterStatus.value)
  }

  // Date range filter
  if (filterDateFrom.value) {
    const from = new Date(filterDateFrom.value)
    result = result.filter(s => new Date(s.start_time) >= from)
  }
  if (filterDateTo.value) {
    const to = new Date(filterDateTo.value)
    to.setHours(23, 59, 59, 999)
    result = result.filter(s => new Date(s.start_time) <= to)
  }

  // Search
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(s => s.name.toLowerCase().includes(q))
  }

  // Sort: most recent first
  result.sort((a, b) => new Date(b.start_time).getTime() - new Date(a.start_time).getTime())

  return result
})

const statusCounts = computed(() => {
  const counts: Record<string, number> = { all: schedules.value.length }
  for (const s of schedules.value) {
    counts[s.status] = (counts[s.status] || 0) + 1
  }
  return counts
})

function formatDate(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

function statusClass(status: string) {
  const map: Record<string, string> = {
    draft: 'bg-secondary-lt text-secondary',
    published: 'bg-blue-lt text-blue',
    active: 'bg-green-lt text-green',
    finished: 'bg-orange-lt text-orange',
  }
  return map[status] ?? 'bg-secondary-lt text-secondary'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    draft: 'Draft',
    published: 'Dipublikasi',
    active: 'Aktif',
    finished: 'Selesai',
  }
  return map[status] ?? status
}

function clearFilters() {
  searchQuery.value = ''
  filterStatus.value = 'all'
  filterDateFrom.value = ''
  filterDateTo.value = ''
}

onMounted(loadSchedules)
</script>

<template>
  <BasePageHeader
    title="Riwayat Ujian"
    subtitle="Ujian yang telah selesai dan ringkasan hasilnya"
    :breadcrumbs="[{ label: 'Dashboard', to: '/guru' }, { label: 'Riwayat Ujian' }]"
  >
    <template #actions>
      <button class="btn btn-ghost-primary" @click="router.push('/guru/reports')">
        <i class="ti ti-chart-bar me-1"></i>Laporan Lengkap
      </button>
    </template>
  </BasePageHeader>

  <!-- Filters -->
  <div class="card mb-3">
    <div class="card-body">
      <div class="row g-3 align-items-end">
        <div class="col-md-4">
          <label class="form-label small">Cari Ujian</label>
          <div class="input-icon">
            <span class="input-icon-addon"><i class="ti ti-search"></i></span>
            <input v-model="searchQuery" type="text" class="form-control" placeholder="Nama ujian...">
          </div>
        </div>
        <div class="col-md-2">
          <label class="form-label small">Status</label>
          <select v-model="filterStatus" class="form-select">
            <option value="all">Semua ({{ statusCounts.all || 0 }})</option>
            <option value="finished">Selesai ({{ statusCounts.finished || 0 }})</option>
            <option value="active">Aktif ({{ statusCounts.active || 0 }})</option>
            <option value="published">Dipublikasi ({{ statusCounts.published || 0 }})</option>
            <option value="draft">Draft ({{ statusCounts.draft || 0 }})</option>
          </select>
        </div>
        <div class="col-md-2">
          <label class="form-label small">Dari Tanggal</label>
          <input v-model="filterDateFrom" type="date" class="form-control">
        </div>
        <div class="col-md-2">
          <label class="form-label small">Sampai Tanggal</label>
          <input v-model="filterDateTo" type="date" class="form-control">
        </div>
        <div class="col-md-2">
          <button class="btn btn-ghost-secondary w-100" @click="clearFilters">
            <i class="ti ti-filter-off me-1"></i>Reset
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Loading -->
  <div v-if="loading" class="card">
    <div class="card-body p-5 text-center text-muted">
      <span class="spinner-border spinner-border-sm me-2"></span>Memuat data...
    </div>
  </div>

  <!-- Empty state -->
  <div v-else-if="!filteredSchedules.length" class="card">
    <div class="empty">
      <div class="empty-icon">
        <i class="ti ti-history-off" style="font-size: 3rem;"></i>
      </div>
      <p class="empty-title">Belum ada riwayat ujian</p>
      <p class="empty-subtitle text-muted">
        <template v-if="searchQuery || filterStatus !== 'all' || filterDateFrom || filterDateTo">Tidak ada ujian yang sesuai filter. Coba ubah kriteria pencarian.</template>
        <template v-else>Riwayat ujian yang telah selesai akan muncul di sini.</template>
      </p>
      <div v-if="searchQuery || filterStatus !== 'all' || filterDateFrom || filterDateTo" class="empty-action">
        <button class="btn btn-outline-secondary btn-sm" @click="clearFilters">
          <i class="ti ti-filter-off me-1"></i>Reset Filter
        </button>
      </div>
    </div>
  </div>

  <!-- Schedule cards -->
  <div v-else class="d-flex flex-column gap-3">
    <div v-for="schedule in filteredSchedules" :key="schedule.id" class="card">
      <div class="card-body">
        <div class="row align-items-center g-3">
          <!-- Left: info -->
          <div class="col-md-5">
            <div class="d-flex align-items-start gap-3">
              <span class="avatar avatar-md rounded" :class="schedule.status === 'finished' ? 'bg-orange-lt' : schedule.status === 'active' ? 'bg-green-lt' : 'bg-blue-lt'">
                <i class="ti fs-3" :class="schedule.status === 'finished' ? 'ti-check text-orange' : schedule.status === 'active' ? 'ti-player-play text-green' : 'ti-calendar text-blue'"></i>
              </span>
              <div>
                <h3 class="mb-1">{{ schedule.name }}</h3>
                <div class="d-flex align-items-center gap-2 flex-wrap">
                  <span class="badge" :class="statusClass(schedule.status)">{{ statusLabel(schedule.status) }}</span>
                  <span class="text-muted small">
                    <i class="ti ti-clock me-1"></i>{{ formatDate(schedule.start_time) }}
                  </span>
                  <span class="text-muted small">
                    <i class="ti ti-hourglass me-1"></i>{{ schedule.duration_minutes }} menit
                  </span>
                </div>
                <div v-if="schedule.tags?.length" class="mt-1 d-flex gap-1 flex-wrap">
                  <span v-for="t in schedule.tags" :key="t.tag_id" class="badge bg-secondary-lt text-secondary" style="font-size:0.65rem">
                    {{ t.tag?.name }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Middle: stats summary -->
          <div class="col-md-4">
            <template v-if="reportCache[schedule.id]">
              <div class="row g-2 text-center">
                <div class="col-3">
                  <div class="fw-bold">{{ reportCache[schedule.id]!.total }}</div>
                  <div class="text-muted" style="font-size:0.65rem">Peserta</div>
                </div>
                <div class="col-3">
                  <div class="fw-bold text-blue">{{ reportCache[schedule.id]!.mean.toFixed(1) }}</div>
                  <div class="text-muted" style="font-size:0.65rem">Rata-rata</div>
                </div>
                <div class="col-3">
                  <div class="fw-bold text-green">{{ reportCache[schedule.id]!.highest.toFixed(1) }}</div>
                  <div class="text-muted" style="font-size:0.65rem">Tertinggi</div>
                </div>
                <div class="col-3">
                  <div class="fw-bold text-red">{{ reportCache[schedule.id]!.lowest.toFixed(1) }}</div>
                  <div class="text-muted" style="font-size:0.65rem">Terendah</div>
                </div>
              </div>
            </template>
            <template v-else-if="loadingReports.has(schedule.id)">
              <div class="text-muted small text-center">
                <span class="spinner-border spinner-border-sm me-1"></span>Memuat ringkasan...
              </div>
            </template>
            <template v-else>
              <button
                v-if="schedule.status === 'finished' || schedule.status === 'active'"
                class="btn btn-sm btn-ghost-primary"
                @click="loadReportSummary(schedule.id)"
              >
                <i class="ti ti-chart-dots me-1"></i>Muat Ringkasan
              </button>
              <div v-else class="text-muted small text-center">-</div>
            </template>
          </div>

          <!-- Right: actions -->
          <div class="col-md-3">
            <div class="d-flex gap-2 justify-content-end flex-wrap">
              <button class="btn btn-sm btn-outline-primary" @click="router.push('/guru/reports')" title="Lihat laporan lengkap">
                <i class="ti ti-chart-bar me-1"></i>Laporan
              </button>
              <button
                v-if="schedule.status === 'finished' || schedule.status === 'active'"
                class="btn btn-sm btn-outline-orange"
                @click="router.push(`/guru/grading/${schedule.id}`)"
                title="Koreksi esai"
              >
                <i class="ti ti-pencil me-1"></i>Koreksi
              </button>
              <button
                class="btn btn-sm btn-ghost-secondary"
                @click="router.push(`/guru/exam-schedules/${schedule.id}/preview`)"
                title="Preview soal"
              >
                <i class="ti ti-eye"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Results count -->
  <div v-if="!loading && filteredSchedules.length" class="text-muted small mt-3 text-center">
    Menampilkan {{ filteredSchedules.length }} dari {{ schedules.length }} jadwal
  </div>
</template>
