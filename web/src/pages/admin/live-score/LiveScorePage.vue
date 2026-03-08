<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { liveScoreApi, type LiveScoreData } from '../../../api/live-score.api'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()

// ── State ────────────────────────────────────────────────────
const schedules = ref<ExamSchedule[]>([])
const selectedId = ref<number | null>(null)
const liveData = ref<LiveScoreData | null>(null)
const loading = ref(false)
const fullscreen = ref(false)
const darkMode = ref(true)
const viewMode = ref<'table' | 'card'>('table')
const filterRombelId = ref<number | null>(null)
const scrollSpeed = ref(1.5)
const isPaused = ref(false)
const lastTimestamp = ref('')

// Polling
let pollTimer: ReturnType<typeof setInterval> | null = null
const POLL_INTERVAL = 5000

// Auto-scroll
let scrollAnimationId: number | null = null
let scrollPos = 0
const scrollContainerRef = ref<HTMLElement | null>(null)
const scrollContentRef = ref<HTMLElement | null>(null)
const isAutoScrolling = ref(true)

// Clock
const clock = ref('--:--:--')
let clockTimer: ReturnType<typeof setInterval> | null = null

// ── Computed ─────────────────────────────────────────────────
const students = computed(() => liveData.value?.students ?? [])

const filteredStudents = computed(() => {
  if (!filterRombelId.value) return students.value
  const rombelName = liveData.value?.rombels.find(r => r.id === filterRombelId.value)?.name
  if (!rombelName) return students.value
  return students.value.filter(s => s.rombel.includes(rombelName))
})

const summary = computed(() => liveData.value?.summary ?? {
  total_participants: 0, ongoing: 0, finished: 0, not_started: 0, average_score: 0, highest_score: 0,
})

const rombels = computed(() => liveData.value?.rombels ?? [])

const scheduleName = computed(() => liveData.value?.schedule_name ?? '')
const subjectName = computed(() => liveData.value?.subject_name ?? '')

// ── Schedule Loading ─────────────────────────────────────────
async function loadSchedules() {
  try {
    const res = await examApi.listSchedules({ status: 'active', per_page: 50 })
    schedules.value = res.data.data ?? []
    // Also load published/finished for broader selection
    const res2 = await examApi.listSchedules({ status: 'published', per_page: 50 })
    const extra = res2.data.data ?? []
    const ids = new Set(schedules.value.map((s: ExamSchedule) => s.id))
    for (const s of extra) {
      if (!ids.has(s.id)) schedules.value.push(s)
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat daftar jadwal')
  }
}

// ── Data Fetching ────────────────────────────────────────────
async function fetchFullData() {
  if (!selectedId.value) return
  loading.value = true
  try {
    const rombelFilter = filterRombelId.value ? [filterRombelId.value] : undefined
    const res = await liveScoreApi.getLiveData(selectedId.value, rombelFilter)
    liveData.value = res.data.data
    lastTimestamp.value = res.data.data.timestamp
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data live score')
  } finally {
    loading.value = false
  }
}

async function fetchUpdate() {
  if (!selectedId.value || !lastTimestamp.value) {
    await fetchFullData()
    return
  }
  try {
    const rombelFilter = filterRombelId.value ? [filterRombelId.value] : undefined
    const res = await liveScoreApi.getUpdate(selectedId.value, lastTimestamp.value, rombelFilter)
    const update = res.data.data

    if (update.students.length > 0 && liveData.value) {
      // Merge updated students into existing data
      const updateMap = new Map(update.students.map(s => [s.session_id, s]))
      const merged = liveData.value.students.map(s => updateMap.get(s.session_id) ?? s)
      // Add any new students
      for (const s of update.students) {
        if (!merged.find(m => m.session_id === s.session_id)) {
          merged.push(s)
        }
      }
      // Re-sort by percent descending
      merged.sort((a, b) => b.percent - a.percent)
      liveData.value.students = merged
      liveData.value.summary = update.summary
    }
    lastTimestamp.value = update.timestamp
  } catch {
    // Fallback to full fetch on error
    await fetchFullData()
  }
}

// ── Polling ──────────────────────────────────────────────────
function startPolling() {
  stopPolling()
  pollTimer = setInterval(fetchUpdate, POLL_INTERVAL)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// ── Schedule Selection ───────────────────────────────────────
async function selectSchedule(id: number) {
  selectedId.value = id
  liveData.value = null
  lastTimestamp.value = ''
  await fetchFullData()
  startPolling()
}

// ── Fullscreen ───────────────────────────────────────────────
function toggleFullscreen() {
  fullscreen.value = !fullscreen.value
  if (fullscreen.value) {
    document.documentElement.requestFullscreen?.()
    nextTick(() => {
      if (isAutoScrolling.value) startAutoScroll()
    })
  } else {
    document.exitFullscreen?.()
    stopAutoScroll()
  }
}

function exitFullscreen() {
  fullscreen.value = false
  document.exitFullscreen?.()
  stopAutoScroll()
}

// ── Auto Scroll (TV Mode) ────────────────────────────────────
function startAutoScroll() {
  stopAutoScroll()
  scrollPos = 0
  const container = scrollContainerRef.value
  if (!container) return

  function step() {
    if (isPaused.value || !fullscreen.value) {
      scrollAnimationId = requestAnimationFrame(step)
      return
    }
    const content = scrollContentRef.value
    if (!container || !content) return

    scrollPos += scrollSpeed.value
    const maxScroll = content.scrollHeight - container.clientHeight
    if (maxScroll > 0 && scrollPos >= maxScroll) {
      scrollPos = 0
    }
    container.scrollTop = scrollPos
    scrollAnimationId = requestAnimationFrame(step)
  }
  scrollAnimationId = requestAnimationFrame(step)
}

function stopAutoScroll() {
  if (scrollAnimationId !== null) {
    cancelAnimationFrame(scrollAnimationId)
    scrollAnimationId = null
  }
}

function toggleAutoScroll() {
  isPaused.value = !isPaused.value
}

// ── Clock ────────────────────────────────────────────────────
function updateClock() {
  clock.value = new Date().toLocaleTimeString('id-ID', { hour12: false })
}

// ── Helpers ──────────────────────────────────────────────────
function gradeColor(pct: number): string {
  if (pct >= 90) return '#10b981'
  if (pct >= 75) return '#3b82f6'
  if (pct >= 60) return '#f59e0b'
  return '#ef4444'
}

function gradeBarClass(pct: number): string {
  if (pct >= 90) return 'bg-success'
  if (pct >= 75) return 'bg-primary'
  if (pct >= 60) return 'bg-warning'
  return 'bg-danger'
}

function statusBadge(status: string): { text: string; cls: string } {
  switch (status) {
    case 'ongoing': return { text: 'Mengerjakan', cls: 'bg-blue-lt text-blue' }
    case 'finished': return { text: 'Selesai', cls: 'bg-green-lt text-green' }
    case 'terminated': return { text: 'Dihentikan', cls: 'bg-red-lt text-red' }
    case 'not_started': return { text: 'Belum Mulai', cls: 'bg-muted-lt text-muted' }
    default: return { text: status, cls: 'bg-muted-lt' }
  }
}

function rankIcon(rank: number): string {
  if (rank === 1) return 'ti ti-crown-filled'
  if (rank === 2) return 'ti ti-crown'
  if (rank === 3) return 'ti ti-crown'
  return ''
}

function rankColor(rank: number): string {
  if (rank === 1) return '#eab308'
  if (rank === 2) return '#94a3b8'
  if (rank === 3) return '#a97142'
  return '#475569'
}

// ── Lifecycle ────────────────────────────────────────────────
onMounted(() => {
  loadSchedules()
  updateClock()
  clockTimer = setInterval(updateClock, 1000)
})

onUnmounted(() => {
  stopPolling()
  stopAutoScroll()
  if (clockTimer) clearInterval(clockTimer)
})

watch(filterRombelId, () => {
  if (selectedId.value) {
    liveData.value = null
    lastTimestamp.value = ''
    fetchFullData()
  }
})
</script>

<template>
  <!-- ═══════════════════════════════════════════════════════════
       NORMAL MODE (Non-Fullscreen)
       ═══════════════════════════════════════════════════════════ -->
  <div v-if="!fullscreen">
    <BasePageHeader
      title="Live Score (TV Mode)"
      subtitle="Papan skor langsung untuk ditampilkan di layar besar"
      :breadcrumbs="[{ label: 'Monitoring', to: '/admin/live-score' }, { label: 'Live Score' }]"
    >
      <template #actions>
        <button class="btn btn-ghost-secondary" @click="fetchFullData" :disabled="loading">
          <i class="ti ti-refresh me-1"></i>Refresh
        </button>
        <button v-if="selectedId" class="btn btn-primary" @click="toggleFullscreen">
          <i class="ti ti-device-tv me-1"></i>Layar Penuh
        </button>
      </template>
    </BasePageHeader>

    <!-- Controls -->
    <div class="row g-3 mb-3">
      <div class="col-md-6">
        <label class="form-label">Jadwal Ujian</label>
        <select
          class="form-select"
          :value="selectedId ?? ''"
          @change="selectSchedule(Number(($event.target as HTMLSelectElement).value))"
        >
          <option value="" disabled>Pilih Jadwal Ujian...</option>
          <option v-for="s in schedules" :key="s.id" :value="s.id">{{ s.name }}</option>
        </select>
      </div>
      <div class="col-md-3">
        <label class="form-label">Filter Rombel</label>
        <select class="form-select" v-model="filterRombelId">
          <option :value="null">Semua Rombel</option>
          <option v-for="r in rombels" :key="r.id" :value="r.id">{{ r.name }}</option>
        </select>
      </div>
      <div class="col-md-3">
        <label class="form-label">Tampilan</label>
        <div class="btn-group w-100">
          <button class="btn" :class="viewMode === 'table' ? 'btn-primary' : 'btn-outline-primary'" @click="viewMode = 'table'">
            <i class="ti ti-list me-1"></i>Tabel
          </button>
          <button class="btn" :class="viewMode === 'card' ? 'btn-primary' : 'btn-outline-primary'" @click="viewMode = 'card'">
            <i class="ti ti-layout-grid me-1"></i>Kartu
          </button>
        </div>
      </div>
    </div>

    <!-- Summary cards -->
    <div v-if="selectedId && liveData" class="row g-3 mb-3">
      <div class="col-6 col-md-3">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center">
              <span class="avatar bg-blue-lt me-3"><i class="ti ti-users text-blue"></i></span>
              <div>
                <div class="fw-bold fs-3">{{ summary.total_participants }}</div>
                <div class="text-muted small">Total Peserta</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center">
              <span class="avatar bg-green-lt me-3"><i class="ti ti-circle-check text-green"></i></span>
              <div>
                <div class="fw-bold fs-3">{{ summary.finished }}</div>
                <div class="text-muted small">Selesai</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center">
              <span class="avatar bg-azure-lt me-3"><i class="ti ti-pencil text-azure"></i></span>
              <div>
                <div class="fw-bold fs-3">{{ summary.ongoing }}</div>
                <div class="text-muted small">Mengerjakan</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center">
              <span class="avatar bg-yellow-lt me-3"><i class="ti ti-trophy text-yellow"></i></span>
              <div>
                <div class="fw-bold fs-3">{{ summary.average_score.toFixed(1) }}%</div>
                <div class="text-muted small">Rata-rata</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!selectedId" class="d-flex flex-column align-items-center justify-content-center py-5 text-muted">
      <i class="ti ti-device-tv mb-3" style="font-size: 4rem; opacity: 0.3"></i>
      <p>Pilih jadwal ujian untuk menampilkan scoreboard</p>
    </div>

    <!-- Loading -->
    <div v-else-if="loading && !liveData" class="text-center py-5 text-muted">
      <span class="spinner-border spinner-border-sm me-2"></span>Memuat data live score...
    </div>

    <!-- TABLE VIEW -->
    <div v-else-if="liveData && viewMode === 'table'" class="card">
      <div class="table-responsive">
        <table class="table table-vcenter card-table">
          <thead>
            <tr>
              <th style="width: 60px" class="text-center">#</th>
              <th>Nama</th>
              <th class="text-center">Rombel</th>
              <th class="text-center">Progress</th>
              <th class="text-center">B / S</th>
              <th>Skor</th>
              <th class="text-center">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in filteredStudents" :key="row.session_id" :class="i < 3 ? 'fw-bold' : ''">
              <td class="text-center">
                <span v-if="i < 3" class="avatar avatar-sm" :style="{ background: rankColor(i + 1) }">
                  <i :class="rankIcon(i + 1)" class="text-white"></i>
                </span>
                <span v-else class="text-muted">{{ i + 1 }}</span>
              </td>
              <td>
                <div class="fw-medium">{{ row.name }}</div>
                <div class="small text-muted">{{ row.nis }}</div>
              </td>
              <td class="text-center">
                <span class="badge bg-muted-lt">{{ row.rombel }}</span>
              </td>
              <td class="text-center" style="min-width: 100px">
                <div class="small text-muted">{{ row.answered }} / {{ row.total_questions }}</div>
                <div class="progress progress-sm mt-1" style="height: 4px">
                  <div class="progress-bar bg-azure" :style="{ width: (row.total_questions ? Math.round((row.answered / row.total_questions) * 100) : 0) + '%' }"></div>
                </div>
              </td>
              <td class="text-center">
                <span class="badge bg-green-lt text-green me-1"><i class="ti ti-check me-1"></i>{{ row.correct }}</span>
                <span class="badge bg-red-lt text-red"><i class="ti ti-x me-1"></i>{{ row.wrong }}</span>
              </td>
              <td style="min-width: 180px">
                <div class="d-flex align-items-center gap-2">
                  <div class="progress flex-grow-1" style="height: 8px">
                    <div class="progress-bar" :class="gradeBarClass(row.percent)" :style="{ width: row.percent + '%' }"></div>
                  </div>
                  <span class="fw-bold" :style="{ color: gradeColor(row.percent), minWidth: '55px', textAlign: 'right' }">
                    {{ row.percent.toFixed(1) }}%
                  </span>
                </div>
              </td>
              <td class="text-center">
                <span class="badge" :class="statusBadge(row.status).cls">{{ statusBadge(row.status).text }}</span>
              </td>
            </tr>
            <tr v-if="filteredStudents.length === 0">
              <td colspan="7" class="text-center text-muted py-4">Belum ada data peserta</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- CARD VIEW -->
    <div v-else-if="liveData && viewMode === 'card'" class="row g-3">
      <div v-for="(row, i) in filteredStudents" :key="row.session_id" class="col-md-6 col-lg-4">
        <div class="card" :class="i < 3 ? 'border-warning' : ''">
          <div class="card-body">
            <div class="d-flex align-items-center mb-3">
              <span class="avatar avatar-lg me-3" :style="{ background: rankColor(i + 1) }">
                <template v-if="i < 3"><i :class="rankIcon(i + 1)" class="text-white fs-2"></i></template>
                <template v-else><span class="text-white fw-bold">{{ i + 1 }}</span></template>
              </span>
              <div class="flex-grow-1">
                <div class="fw-bold">{{ row.name }}</div>
                <div class="small text-muted">{{ row.nis }} &bull; {{ row.rombel }}</div>
              </div>
              <span class="badge" :class="statusBadge(row.status).cls">{{ statusBadge(row.status).text }}</span>
            </div>
            <div class="d-flex justify-content-between small text-muted mb-1">
              <span>Dijawab: {{ row.answered }}/{{ row.total_questions }}</span>
              <span>B: {{ row.correct }} &bull; S: {{ row.wrong }}</span>
            </div>
            <div class="progress mb-2" style="height: 10px">
              <div class="progress-bar" :class="gradeBarClass(row.percent)" :style="{ width: row.percent + '%' }"></div>
            </div>
            <div class="text-end fw-bold fs-3" :style="{ color: gradeColor(row.percent) }">
              {{ row.percent.toFixed(1) }}%
            </div>
          </div>
        </div>
      </div>
      <div v-if="filteredStudents.length === 0" class="col-12 text-center text-muted py-5">
        Belum ada data peserta
      </div>
    </div>
  </div>

  <!-- ═══════════════════════════════════════════════════════════
       TV / FULLSCREEN MODE
       ═══════════════════════════════════════════════════════════ -->
  <Teleport to="body">
    <div v-if="fullscreen" class="live-score-tv" :class="darkMode ? 'tv-dark' : 'tv-light'">
      <!-- Header -->
      <div class="tv-header">
        <div class="tv-header-left">
          <div class="tv-live-badge">
            <span class="tv-live-dot"></span>
            LIVE
          </div>
          <div class="tv-schedule-info">
            <h1>{{ scheduleName }}</h1>
            <div class="tv-subtitle">{{ subjectName }}</div>
          </div>
        </div>

        <div class="tv-header-right">
          <!-- Filter rombel -->
          <div class="tv-control-panel">
            <i class="ti ti-filter"></i>
            <select v-model="filterRombelId" class="tv-select">
              <option :value="null">Semua Rombel</option>
              <option v-for="r in rombels" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </div>

          <!-- Scroll speed -->
          <div class="tv-control-panel">
            <button class="tv-btn-icon" @click="toggleAutoScroll" :title="isPaused ? 'Lanjutkan scroll' : 'Jeda scroll'">
              <i :class="isPaused ? 'ti ti-player-play' : 'ti ti-player-pause'"></i>
            </button>
            <input type="range" class="tv-range" v-model.number="scrollSpeed" min="0.5" max="5" step="0.5">
          </div>

          <!-- Theme toggle -->
          <button class="tv-btn-icon" @click="darkMode = !darkMode" :title="darkMode ? 'Mode terang' : 'Mode gelap'">
            <i :class="darkMode ? 'ti ti-sun' : 'ti ti-moon'"></i>
          </button>

          <!-- Stats box -->
          <div class="tv-stats-box">
            <div class="tv-clock">{{ clock }}</div>
            <div class="tv-participant-count">{{ summary.total_participants }} Peserta</div>
          </div>

          <!-- Exit -->
          <button class="tv-btn-icon" @click="exitFullscreen" title="Keluar layar penuh">
            <i class="ti ti-x"></i>
          </button>
        </div>
      </div>

      <!-- Summary bar -->
      <div class="tv-summary-bar">
        <div class="tv-stat-item">
          <i class="ti ti-pencil text-azure"></i>
          <span>{{ summary.ongoing }} Mengerjakan</span>
        </div>
        <div class="tv-stat-item">
          <i class="ti ti-circle-check text-green"></i>
          <span>{{ summary.finished }} Selesai</span>
        </div>
        <div class="tv-stat-item">
          <i class="ti ti-chart-bar text-yellow"></i>
          <span>Rata-rata: {{ summary.average_score.toFixed(1) }}%</span>
        </div>
        <div class="tv-stat-item">
          <i class="ti ti-trophy text-yellow"></i>
          <span>Tertinggi: {{ summary.highest_score.toFixed(1) }}%</span>
        </div>
      </div>

      <!-- Scroll container -->
      <div class="tv-scroll-container" ref="scrollContainerRef">
        <div class="tv-scroll-content" ref="scrollContentRef">
          <!-- Loading -->
          <div v-if="loading && !liveData" class="tv-loading">
            <div class="spinner-border text-primary" role="status"></div>
            <div class="mt-3">Menghubungkan ke Server...</div>
          </div>

          <!-- Student rows -->
          <template v-else>
            <div
              v-for="(row, i) in filteredStudents"
              :key="row.session_id"
              class="tv-student-row"
              :class="{
                'tv-rank-1': i === 0,
                'tv-rank-2': i === 1,
                'tv-rank-3': i === 2,
              }"
            >
              <!-- Rank circle -->
              <div class="tv-rank-wrap">
                <div class="tv-rank-circle">
                  <template v-if="i < 3">
                    <i :class="rankIcon(i + 1)" :style="{ fontSize: '1.6rem' }"></i>
                  </template>
                  <template v-else>
                    {{ i + 1 }}
                  </template>
                </div>
              </div>

              <!-- Student info -->
              <div class="tv-student-info">
                <div class="tv-student-name">
                  {{ row.name }}
                  <span v-if="row.status === 'finished' || row.status === 'terminated'" class="tv-badge-done">
                    {{ row.status === 'finished' ? 'Selesai' : 'Dihentikan' }}
                  </span>
                </div>
                <div class="tv-student-meta">
                  <span class="tv-pill">{{ row.rombel }}</span>
                  <span class="tv-pill">
                    <i class="ti ti-clipboard-check"></i>
                    {{ row.answered }} / {{ row.total_questions }}
                  </span>
                  <span class="tv-pill tv-pill-correct">
                    <i class="ti ti-check"></i> {{ row.correct }}
                  </span>
                  <span class="tv-pill tv-pill-wrong">
                    <i class="ti ti-x"></i> {{ row.wrong }}
                  </span>
                  <span v-if="row.violation_count > 0" class="tv-pill tv-pill-violation">
                    <i class="ti ti-alert-triangle"></i> {{ row.violation_count }}
                  </span>
                </div>
              </div>

              <!-- Score display -->
              <div class="tv-score-display">
                <div class="tv-score-value" :style="{ color: gradeColor(row.percent) }">
                  {{ row.percent.toFixed(1) }}
                </div>
                <div class="tv-score-label">Skor</div>
              </div>
            </div>

            <!-- Empty state -->
            <div v-if="filteredStudents.length === 0" class="tv-loading">
              <i class="ti ti-mood-empty" style="font-size: 3rem; opacity: 0.3"></i>
              <div class="mt-3">Belum ada peserta</div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* ═══════════════════════════════════════════════════════════
   TV MODE STYLES
   ═══════════════════════════════════════════════════════════ */

.live-score-tv {
  position: fixed;
  inset: 0;
  z-index: 99999;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-family: 'Inter', 'Outfit', sans-serif;
}

/* ── Dark Theme ─────────────────────────────────────────── */
.tv-dark {
  background: radial-gradient(circle at top right, #1e293b, #0f172a);
  color: #f8fafc;
}
.tv-dark::before {
  content: '';
  position: absolute;
  inset: -50%;
  width: 200%;
  height: 200%;
  background:
    radial-gradient(circle, rgba(59, 130, 246, 0.04) 0%, transparent 60%),
    radial-gradient(circle at 80% 20%, rgba(236, 72, 153, 0.03) 0%, transparent 40%);
  z-index: -1;
  pointer-events: none;
}

/* ── Light Theme ────────────────────────────────────────── */
.tv-light {
  background: #f1f5f9;
  color: #1e293b;
}

/* ── Header ─────────────────────────────────────────────── */
.tv-header {
  height: 80px;
  padding: 0 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  z-index: 10;
}
.tv-dark .tv-header {
  background: rgba(15, 23, 42, 0.9);
  backdrop-filter: blur(12px);
}
.tv-light .tv-header {
  background: #fff;
  border-bottom-color: #e2e8f0;
}

.tv-header-left {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.tv-header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

/* Live badge */
.tv-live-badge {
  display: flex;
  align-items: center;
  font-weight: 800;
  font-size: 0.85rem;
  padding: 0.4rem 0.8rem;
  border-radius: 8px;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  white-space: nowrap;
}
.tv-dark .tv-live-badge {
  background: rgba(220, 38, 38, 0.2);
  border: 1px solid rgba(220, 38, 38, 0.5);
  color: #fff;
}
.tv-light .tv-live-badge {
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #dc2626;
}

.tv-live-dot {
  width: 10px;
  height: 10px;
  background: #ef4444;
  border-radius: 50%;
  margin-right: 8px;
  box-shadow: 0 0 8px #ef4444;
  animation: pulse-red 1.5s infinite;
}

@keyframes pulse-red {
  0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.6); }
  70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(239, 68, 68, 0); }
  100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(239, 68, 68, 0); }
}

/* Schedule info */
.tv-schedule-info h1 {
  font-size: 1.3rem;
  margin: 0;
  font-weight: 700;
  line-height: 1.2;
}
.tv-dark .tv-schedule-info h1 {
  background: linear-gradient(90deg, #fff, #cbd5e1);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
.tv-subtitle {
  font-size: 0.85rem;
  margin-top: 2px;
}
.tv-dark .tv-subtitle { color: #94a3b8; }
.tv-light .tv-subtitle { color: #64748b; }

/* Controls */
.tv-control-panel {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  border-radius: 8px;
}
.tv-dark .tv-control-panel {
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #94a3b8;
}
.tv-light .tv-control-panel {
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  color: #64748b;
}

.tv-select {
  background: transparent;
  border: none;
  color: inherit;
  font-size: 0.85rem;
  cursor: pointer;
  outline: none;
  max-width: 150px;
}
.tv-select option {
  color: #1e293b;
  background: #fff;
}

.tv-btn-icon {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  font-size: 1.1rem;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tv-btn-icon:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
}
.tv-light .tv-btn-icon { color: #64748b; }
.tv-light .tv-btn-icon:hover { color: #1e293b; background: #e2e8f0; }

.tv-range {
  width: 80px;
  accent-color: #3b82f6;
}

/* Stats box */
.tv-stats-box {
  text-align: right;
}
.tv-clock {
  font-size: 1.5rem;
  font-weight: 700;
  font-family: 'SF Mono', 'Monaco', monospace;
  line-height: 1;
}
.tv-dark .tv-clock { color: #fff; }
.tv-light .tv-clock { color: #1e293b; }
.tv-participant-count {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-top: 3px;
}
.tv-dark .tv-participant-count { color: #64748b; }
.tv-light .tv-participant-count { color: #94a3b8; }

/* ── Summary bar ────────────────────────────────────────── */
.tv-summary-bar {
  display: flex;
  justify-content: center;
  gap: 2rem;
  padding: 0.6rem 2rem;
  flex-shrink: 0;
}
.tv-dark .tv-summary-bar {
  background: rgba(30, 41, 59, 0.5);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.tv-light .tv-summary-bar {
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
}

.tv-stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  font-weight: 500;
}
.tv-dark .tv-stat-item { color: #cbd5e1; }
.tv-light .tv-stat-item { color: #475569; }

/* ── Scroll container ───────────────────────────────────── */
.tv-scroll-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.tv-scroll-content {
  padding: 1.5rem 12%;
  min-height: 100%;
}

/* ── Student row ────────────────────────────────────────── */
.tv-student-row {
  margin-bottom: 0.8rem;
  border-radius: 14px;
  padding: 1rem 1.5rem;
  display: flex;
  align-items: center;
  transition: transform 0.3s, background 0.3s, opacity 0.5s;
  position: relative;
  overflow: hidden;
}
.tv-student-row::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 5px;
}

.tv-dark .tv-student-row {
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.04);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
.tv-dark .tv-student-row::before { background: #475569; }
.tv-light .tv-student-row {
  background: #fff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}
.tv-light .tv-student-row::before { background: #cbd5e1; }

/* Rank 1 */
.tv-dark .tv-rank-1 {
  background: linear-gradient(90deg, rgba(234, 179, 8, 0.08), rgba(30, 41, 59, 0.7));
  border-color: rgba(234, 179, 8, 0.25);
}
.tv-dark .tv-rank-1::before { background: #eab308; box-shadow: 0 0 12px #eab308; }
.tv-light .tv-rank-1 { border-color: #fde68a; background: linear-gradient(90deg, #fffbeb, #fff); }
.tv-light .tv-rank-1::before { background: #eab308; }

/* Rank 2 */
.tv-dark .tv-rank-2 {
  background: linear-gradient(90deg, rgba(148, 163, 184, 0.08), rgba(30, 41, 59, 0.7));
  border-color: rgba(148, 163, 184, 0.25);
}
.tv-dark .tv-rank-2::before { background: #94a3b8; }
.tv-light .tv-rank-2 { border-color: #cbd5e1; }
.tv-light .tv-rank-2::before { background: #94a3b8; }

/* Rank 3 */
.tv-dark .tv-rank-3 {
  background: linear-gradient(90deg, rgba(169, 113, 66, 0.08), rgba(30, 41, 59, 0.7));
  border-color: rgba(169, 113, 66, 0.25);
}
.tv-dark .tv-rank-3::before { background: #a97142; }
.tv-light .tv-rank-3 { border-color: #d6bcab; }
.tv-light .tv-rank-3::before { background: #a97142; }

/* Rank circle */
.tv-rank-wrap {
  margin-right: 1.2rem;
}

.tv-rank-circle {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 1.3rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}
.tv-dark .tv-rank-circle {
  background: #1e293b;
  color: #94a3b8;
  border: 2px solid #334155;
}
.tv-light .tv-rank-circle {
  background: #f1f5f9;
  color: #64748b;
  border: 2px solid #e2e8f0;
}

.tv-rank-1 .tv-rank-circle {
  background: linear-gradient(135deg, #eab308, #a16207) !important;
  color: #fff !important;
  border: none !important;
  box-shadow: 0 0 16px rgba(234, 179, 8, 0.3);
}
.tv-rank-2 .tv-rank-circle {
  background: linear-gradient(135deg, #94a3b8, #475569) !important;
  color: #fff !important;
  border: none !important;
}
.tv-rank-3 .tv-rank-circle {
  background: linear-gradient(135deg, #ca8a04, #713f12) !important;
  color: #fff !important;
  border: none !important;
}

/* Student info */
.tv-student-info {
  flex: 1;
  min-width: 0;
}

.tv-student-name {
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.3px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.tv-badge-done {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.tv-student-meta {
  display: flex;
  gap: 8px;
  margin-top: 6px;
  flex-wrap: wrap;
}

.tv-pill {
  font-size: 0.8rem;
  padding: 3px 8px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.tv-dark .tv-pill {
  background: rgba(255, 255, 255, 0.05);
  color: #94a3b8;
}
.tv-light .tv-pill {
  background: #f1f5f9;
  color: #64748b;
}

.tv-pill-correct { color: #10b981 !important; }
.tv-pill-wrong { color: #ef4444 !important; }
.tv-pill-violation {
  color: #f59e0b !important;
}
.tv-dark .tv-pill-violation { background: rgba(245, 158, 11, 0.1) !important; }
.tv-light .tv-pill-violation { background: #fffbeb !important; }

/* Score display */
.tv-score-display {
  text-align: right;
  margin-left: auto;
  padding-left: 1rem;
}

.tv-score-value {
  font-size: 2.2rem;
  font-weight: 800;
  line-height: 1;
}

.tv-rank-1 .tv-score-value {
  text-shadow: 0 0 16px rgba(250, 204, 21, 0.3);
}

.tv-score-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-top: 2px;
}
.tv-dark .tv-score-label { color: #64748b; }
.tv-light .tv-score-label { color: #94a3b8; }

/* Loading */
.tv-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 50vh;
  color: #64748b;
}

/* ── Scrollbar ──────────────────────────────────────────── */
.tv-scroll-container::-webkit-scrollbar {
  width: 6px;
}
.tv-dark .tv-scroll-container::-webkit-scrollbar-track { background: transparent; }
.tv-dark .tv-scroll-container::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); border-radius: 3px; }
.tv-light .tv-scroll-container::-webkit-scrollbar-track { background: transparent; }
.tv-light .tv-scroll-container::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.1); border-radius: 3px; }
</style>
