<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  reportApi, type ScheduleReport, type PersonalReport,
  type ExamAnalysis, type SessionRow, type KeyChangesResponse, type RegradeResult,
  type RegradeLogEntry
} from '../../../api/report.api'
import { getIllustration } from '../../../utils/avatar'
import { sanitizeHtml } from '@/composables/useSafeHtml'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const QUALITY_CLASSES: Record<string, string> = {
  'Baik Sekali': 'bg-success-lt text-success',
  'Baik': 'bg-primary-lt text-primary',
  'Cukup': 'bg-warning-lt text-warning',
  'Revisi': 'bg-danger-lt text-danger',
  'Buang': 'bg-danger text-white',
}
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { useToastStore } from '../../../stores/toast.store'
import { useAuthStore } from '../../../stores/auth.store'

const router = useRouter()
const authStore = useAuthStore()

const toast = useToastStore()

// ─── State ─────────────────────────────────────────────────────
const schedules = ref<ExamSchedule[]>([])
const selectedId = ref<number | null>(null)
const activeTab = ref<'rekap' | 'analisis' | 'regrade'>('rekap')
const report = ref<ScheduleReport | null>(null)
const analysis = ref<ExamAnalysis | null>(null)
const personalReport = ref<PersonalReport | null>(null)
const selectedSession = ref<SessionRow | null>(null)
const loadingReport = ref(false)
const loadingAnalysis = ref(false)
const loadingPersonal = ref(false)
const regrading = ref(false)
const keyChanges = ref<KeyChangesResponse | null>(null)
const regradeResult = ref<RegradeResult | null>(null)
const regradeLogs = ref<RegradeLogEntry[]>([])
const loadingRegradeLogs = ref(false)
const expandedLogId = ref<number | null>(null)
const showRegradeModal = ref(false)
const multiSheetExport = ref(false)
const exportLoading = ref(false)
const exportUnfinishedLoading = ref(false)

// Confirm modal for regrade
const showRegradeConfirm = ref(false)

// ─── Computed ─────────────────────────────────────────────────
const selectedSchedule = computed(() => schedules.value.find((s) => s.id === selectedId.value))
const sortedSessions = computed(() =>
  [...(report.value?.sessions ?? [])].sort((a, b) => b.percent - a.percent)
)

// ─── API ──────────────────────────────────────────────────────
async function loadSchedules() {
  try {
    const res = await examApi.listSchedules({ per_page: 100 })
    schedules.value = res.data.data ?? []
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat daftar jadwal')
  }
}

async function selectSchedule(id: number) {
  selectedId.value = id
  personalReport.value = null
  selectedSession.value = null
  keyChanges.value = null
  regradeLogs.value = []
  await Promise.all([fetchReport(id), fetchAnalysis(id), fetchKeyChanges(id), fetchRegradeLogs(id)])
}

async function fetchKeyChanges(id: number) {
  try {
    const res = await reportApi.getKeyChanges(id)
    keyChanges.value = res.data.data
  } catch {
    keyChanges.value = null
  }
}

async function fetchRegradeLogs(id: number) {
  loadingRegradeLogs.value = true
  try {
    const res = await reportApi.getRegradeLogs(id)
    regradeLogs.value = res.data.data ?? []
  } catch {
    regradeLogs.value = []
  } finally {
    loadingRegradeLogs.value = false
  }
}

function toggleLogExpand(logId: number) {
  expandedLogId.value = expandedLogId.value === logId ? null : logId
}

function formatDateTime(iso: string) {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' }) +
    ' ' + d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

function countChanges(log: RegradeLogEntry): number {
  if (!log.score_changes || !Array.isArray(log.score_changes)) return 0
  return log.score_changes.filter(sc => sc.old_score !== sc.new_score).length
}

async function fetchReport(id: number) {
  loadingReport.value = true
  try {
    const res = await reportApi.getScheduleReport(id)
    report.value = res.data.data
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat laporan')
  } finally {
    loadingReport.value = false
  }
}

async function fetchAnalysis(id: number) {
  loadingAnalysis.value = true
  try {
    const res = await reportApi.getExamAnalysis(id)
    analysis.value = res.data.data
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat analisis butir soal')
  } finally {
    loadingAnalysis.value = false
  }
}

async function showPersonalReport(row: SessionRow) {
  if (!selectedId.value) return
  selectedSession.value = row
  loadingPersonal.value = true
  try {
    const res = await reportApi.getPersonalReport(selectedId.value, row.session_id)
    personalReport.value = res.data.data
  } catch (e: any) {
    toast.error('Gagal memuat laporan personal')
  } finally {
    loadingPersonal.value = false
  }
}

function backToReport() {
  personalReport.value = null
  selectedSession.value = null
}

function askRegrade() {
  showRegradeConfirm.value = true
}

async function doRegrade() {
  if (!selectedId.value) return
  showRegradeConfirm.value = false
  regrading.value = true
  try {
    const res = await reportApi.regrade(selectedId.value)
    regradeResult.value = res.data.data
    showRegradeModal.value = true
    await Promise.all([fetchReport(selectedId.value), fetchKeyChanges(selectedId.value), fetchRegradeLogs(selectedId.value)])
  } catch {
    toast.error('Regrade gagal')
  } finally {
    regrading.value = false
  }
}

async function exportReport() {
  if (!selectedId.value) return
  exportLoading.value = true
  try {
    const url = reportApi.exportLedger(selectedId.value, multiSheetExport.value)
    const token = localStorage.getItem('access_token')
    const a = document.createElement('a')
    a.href = url + (url.includes('?') ? '&' : '?') + `token=${token}`
    a.download = ''
    a.click()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal mengekspor laporan')
  } finally {
    setTimeout(() => { exportLoading.value = false }, 2000)
  }
}

async function exportUnfinished() {
  if (!selectedId.value) return
  exportUnfinishedLoading.value = true
  try {
    const url = reportApi.exportUnfinished(selectedId.value)
    const token = localStorage.getItem('access_token')
    const a = document.createElement('a')
    a.href = url + (url.includes('?') ? '&' : '?') + `token=${token}`
    a.download = ''
    a.click()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal mengekspor data belum selesai')
  } finally {
    setTimeout(() => { exportUnfinishedLoading.value = false }, 2000)
  }
}

function printReport() {
  window.print()
}

function formatDuration(secs: number) {
  if (!secs) return '–'
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}m ${s}s`
}


function rankBadge(i: number) {
  if (i === 0) return '🥇'
  if (i === 1) return '🥈'
  if (i === 2) return '🥉'
  return `${i + 1}`
}

function statusLabel(s: string) {
  return { finished: 'Selesai', ongoing: 'Berlangsung', terminated: 'Dihentikan', not_started: 'Belum mulai' }[s] ?? s
}

function gradeColorClass(pct: number) {
  if (pct >= 90) return 'text-success'
  if (pct >= 75) return 'text-primary'
  if (pct >= 60) return 'text-warning'
  return 'text-danger'
}

function gradeBgClass(pct: number) {
  if (pct >= 90) return 'bg-success'
  if (pct >= 75) return 'bg-primary'
  if (pct >= 60) return 'bg-warning'
  return 'bg-danger'
}

// Score distribution: 10 buckets with color gradient (red → green)
const scoreDistribution = computed(() => {
  const sessions = report.value?.sessions ?? []
  const buckets = [
    { label: '0-10',   min: 0,  max: 10,  count: 0, color: '#d63939' },
    { label: '11-20',  min: 11, max: 20,  count: 0, color: '#e25b2a' },
    { label: '21-30',  min: 21, max: 30,  count: 0, color: '#f17d1c' },
    { label: '31-40',  min: 31, max: 40,  count: 0, color: '#f59f00' },
    { label: '41-50',  min: 41, max: 50,  count: 0, color: '#f5c400' },
    { label: '51-60',  min: 51, max: 60,  count: 0, color: '#d5d926' },
    { label: '61-70',  min: 61, max: 70,  count: 0, color: '#a0c34e' },
    { label: '71-80',  min: 71, max: 80,  count: 0, color: '#6fad4e' },
    { label: '81-90',  min: 81, max: 90,  count: 0, color: '#4ea24e' },
    { label: '91-100', min: 91, max: 100, count: 0, color: '#2fb344' },
  ]
  for (const s of sessions) {
    const pct = Math.round(s.percent)
    for (const b of buckets) {
      if (pct >= b.min && pct <= b.max) { b.count++; break }
    }
  }
  const maxCount = Math.max(...buckets.map(b => b.count), 1)
  return buckets.map(b => ({
    ...b,
    heightPct: Math.round((b.count / maxCount) * 100)
  }))
})

const hoveredBucket = ref<string | null>(null)

onMounted(loadSchedules)
</script>

<template>
  <!-- Header -->
  <BasePageHeader
    title="Laporan & Analisis"
    subtitle="Rekap nilai, analisis butir soal, dan grading esai"
    :breadcrumbs="[{ label: 'Laporan & Analisis' }]"
  >
    <template #actions>
      <template v-if="selectedId && !personalReport">
        <button class="btn btn-ghost-secondary" @click="askRegrade" :disabled="regrading">
          <span v-if="regrading" class="spinner-border spinner-border-sm me-1"></span>
          <i v-else class="ti ti-refresh me-1"></i>Hitung Ulang Nilai
        </button>
        <button class="btn btn-ghost-secondary" @click="printReport">
          <i class="ti ti-printer me-1"></i>Cetak PDF
        </button>
        <div class="d-flex align-items-center gap-2">
          <label class="form-check form-check-inline mb-0">
            <input class="form-check-input" type="checkbox" v-model="multiSheetExport" />
            <span class="form-check-label small">Export per Kelas (Multi-sheet)</span>
          </label>
          <button class="btn btn-ghost-secondary" @click="exportReport" :disabled="exportLoading">
            <span v-if="exportLoading" class="spinner-border spinner-border-sm me-1"></span>
            <i v-else class="ti ti-file-spreadsheet me-1"></i>Export Excel
          </button>
        </div>
        <button class="btn btn-ghost-secondary" @click="exportUnfinished" :disabled="exportUnfinishedLoading">
          <span v-if="exportUnfinishedLoading" class="spinner-border spinner-border-sm me-1"></span>
          <i v-else class="ti ti-user-exclamation me-1"></i>Export Belum Selesai
        </button>
        <button
          v-if="authStore.user?.role === 'guru' || authStore.user?.role === 'admin'"
          class="btn btn-ghost-secondary"
          @click="router.push(`/guru/grading/${selectedId}`)"
        >
          <i class="ti ti-pencil me-1"></i>Koreksi Esai
        </button>
      </template>
    </template>
  </BasePageHeader>

  <!-- Schedule selector -->
  <div class="mb-3 d-print-none">
    <select
      class="form-select"
      :value="selectedId ?? ''"
      @change="selectSchedule(Number(($event.target as HTMLSelectElement).value))"
    >
      <option value="" disabled>Pilih Jadwal Ujian...</option>
      <option v-for="s in schedules" :key="s.id" :value="s.id">{{ s.name }}</option>
    </select>
  </div>

  <!-- Key Changes Alert -->
  <div v-if="keyChanges && keyChanges.count > 0" class="alert alert-warning d-print-none mb-3">
    <div class="d-flex align-items-start gap-2">
      <i class="ti ti-alert-triangle fs-3"></i>
      <div>
        <div class="fw-bold">{{ keyChanges.count }} soal telah diubah sejak penilaian terakhir. Pertimbangkan untuk melakukan regrade.</div>
        <div class="text-muted small mt-1">
          Soal yang berubah: <strong>{{ keyChanges.changes.map(c => '#' + c.question_number).join(', ') }}</strong>
        </div>
      </div>
    </div>
  </div>

  <!-- Regrade Summary Modal -->
  <teleport to="body">
    <div v-if="showRegradeModal && regradeResult" class="modal modal-blur d-block" tabindex="-1" style="background:rgba(0,0,0,.4)">
      <div class="modal-dialog modal-dialog-centered" :class="regradeResult.score_changes?.length ? 'modal-lg' : ''">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Hasil Regrade</h5>
            <button type="button" class="btn-close" @click="showRegradeModal = false"></button>
          </div>
          <div class="modal-body">
            <div class="alert alert-success mb-3">
              <div class="fw-bold">Regrade selesai. {{ regradeResult.total }} sesi diproses, {{ regradeResult.changes }} perubahan nilai.</div>
            </div>
            <div v-if="regradeResult.score_changes?.length" class="table-responsive">
              <table class="table table-vcenter table-sm">
                <thead>
                  <tr>
                    <th>Nama</th>
                    <th class="text-end">Nilai Lama</th>
                    <th class="text-end">Nilai Baru</th>
                    <th class="text-end">Selisih</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="sc in regradeResult.score_changes" :key="sc.user_name">
                    <td>{{ sc.user_name }}</td>
                    <td class="text-end">{{ sc.old_score.toFixed(1) }}</td>
                    <td class="text-end fw-bold" :class="sc.new_score > sc.old_score ? 'text-success' : sc.new_score < sc.old_score ? 'text-danger' : ''">{{ sc.new_score.toFixed(1) }}</td>
                    <td class="text-end">
                      <span :class="sc.new_score - sc.old_score > 0 ? 'text-success' : sc.new_score - sc.old_score < 0 ? 'text-danger' : 'text-muted'">
                        {{ sc.new_score - sc.old_score > 0 ? '+' : '' }}{{ (sc.new_score - sc.old_score).toFixed(1) }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-primary" @click="showRegradeModal = false">Tutup</button>
          </div>
        </div>
      </div>
    </div>
  </teleport>

  <!-- Empty state -->
  <div v-if="!selectedId" class="card">
    <div class="card-body text-center text-muted py-5">
      <img :src="getIllustration('chart')" class="img-fluid mb-3 opacity-75" style="max-height:120px" alt="">
      <p>Pilih jadwal ujian untuk melihat laporan</p>
    </div>
  </div>

  <!-- Personal report view -->
  <div v-else-if="personalReport && selectedSession">
    <button class="btn btn-ghost-secondary mb-3 d-print-none" @click="backToReport">
      <i class="ti ti-arrow-left me-1"></i>Kembali
    </button>

    <div class="d-none d-print-block fw-bold mb-2">
      Laporan Personal — {{ selectedSession.user_name }} | {{ selectedSchedule?.name }}
    </div>

    <!-- Personal summary card -->
    <div class="card mb-3">
      <div class="card-body">
        <div class="row g-3">
          <div class="col-sm-3 col-6">
            <div class="text-muted small">Peserta</div>
            <div class="fw-medium">{{ selectedSession.user_name }}</div>
          </div>
          <div class="col-sm-3 col-6">
            <div class="text-muted small">Nilai</div>
            <div class="fw-medium" :class="gradeColorClass(selectedSession.percent)">
              {{ selectedSession.score.toFixed(1) }} / {{ selectedSession.max_score.toFixed(1) }}
              ({{ selectedSession.percent.toFixed(1) }}%)
            </div>
          </div>
          <div class="col-sm-3 col-6">
            <div class="text-muted small">Durasi</div>
            <div class="fw-medium">{{ formatDuration(selectedSession.duration_seconds) }}</div>
          </div>
          <div class="col-sm-3 col-6">
            <div class="text-muted small">Pelanggaran</div>
            <div class="fw-medium">{{ selectedSession.violation_count }}x</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Answers -->
    <div v-if="loadingPersonal" class="p-4 text-center text-muted">
      <span class="spinner-border spinner-border-sm me-2"></span>Memuat...
    </div>
    <div v-else class="d-flex flex-column gap-2">
      <div v-for="a in personalReport.answers" :key="a.question_id" class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="avatar avatar-sm" :class="a.is_correct ? 'bg-green-lt' : 'bg-red-lt'">
                <i class="ti" :class="a.is_correct ? 'ti-check text-green' : 'ti-x text-red'"></i>
              </span>
            </div>
            <div class="col">
              <div class="badge bg-secondary-lt text-secondary mb-1">{{ a.question_type.replace('_', ' ').toUpperCase() }}</div>
              <div class="text-muted small" v-html="sanitizeHtml(a.body.substring(0, 200))"></div>
            </div>
            <div class="col-auto text-end">
              <span class="fw-bold" :class="gradeColorClass(a.is_correct ? 100 : 0)">{{ a.earned_score.toFixed(1) }}</span>
              <span class="text-muted"> / {{ a.score.toFixed(1) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Main report view -->
  <template v-else-if="selectedId">
    <!-- Stats row -->
    <div class="row g-2 mb-3" v-if="report">
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0">{{ report.stats.total }}</div>
            <div class="text-muted small">Total Peserta</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0">{{ report.stats.finished }}</div>
            <div class="text-muted small">Selesai</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0 text-blue">{{ report.stats.mean.toFixed(1) }}</div>
            <div class="text-muted small">Rata-rata</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0 text-green">{{ report.stats.highest.toFixed(1) }}</div>
            <div class="text-muted small">Tertinggi</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0 text-red">{{ report.stats.lowest.toFixed(1) }}</div>
            <div class="text-muted small">Terendah</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg-2">
        <div class="card card-sm">
          <div class="card-body text-center">
            <div class="h3 fw-bold mb-0">{{ report.stats.std_dev.toFixed(1) }}</div>
            <div class="text-muted small">Std. Dev</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Score distribution histogram -->
    <div v-if="report && sortedSessions.length" class="card mb-3">
      <div class="card-header">
        <h3 class="card-title"><i class="ti ti-chart-bar me-2"></i>Distribusi Nilai</h3>
      </div>
      <div class="card-body">
        <div class="d-flex" style="height: 12rem;">
          <!-- Y-axis labels -->
          <div class="d-flex flex-column justify-content-between align-items-end pe-2" style="width: 2.5rem; padding-bottom: 1.75rem;">
            <span class="text-muted" style="font-size: 0.6875rem; line-height: 1;">{{ Math.max(...scoreDistribution.map(b => b.count)) }}</span>
            <span class="text-muted" style="font-size: 0.6875rem; line-height: 1;">{{ Math.round(Math.max(...scoreDistribution.map(b => b.count)) / 2) }}</span>
            <span class="text-muted" style="font-size: 0.6875rem; line-height: 1;">0</span>
          </div>
          <!-- Bars area -->
          <div class="d-flex align-items-end flex-fill gap-1" style="border-left: 1px solid #e6e7e9; border-bottom: 1px solid #e6e7e9; padding-bottom: 0;">
            <div
              v-for="b in scoreDistribution"
              :key="b.label"
              class="d-flex flex-column align-items-center flex-fill position-relative"
              style="height: 100%;"
              @mouseenter="hoveredBucket = b.label"
              @mouseleave="hoveredBucket = null"
            >
              <!-- Count label on top -->
              <div class="d-flex flex-column justify-content-end flex-fill w-100" style="padding-bottom: 0;">
                <div class="text-center fw-semibold" style="font-size: 0.6875rem; margin-bottom: 2px;" :style="{ color: b.color }">
                  {{ b.count }}
                </div>
                <div
                  class="w-100 rounded-top position-relative"
                  :style="{
                    height: (b.heightPct || 1) + '%',
                    minHeight: '3px',
                    backgroundColor: b.color,
                    opacity: hoveredBucket === b.label ? 1 : 0.85,
                    transition: 'opacity 0.15s ease, transform 0.15s ease',
                    transform: hoveredBucket === b.label ? 'scaleY(1.03)' : 'scaleY(1)',
                    transformOrigin: 'bottom',
                    cursor: 'default',
                  }"
                ></div>
              </div>
              <!-- X-axis label -->
              <div class="text-muted text-center" style="font-size: 0.625rem; margin-top: 4px; white-space: nowrap; line-height: 1.2;">
                {{ b.label }}
              </div>
              <!-- Hover tooltip -->
              <div
                v-if="hoveredBucket === b.label"
                class="position-absolute bg-dark text-white rounded px-2 py-1 text-center"
                style="bottom: calc(100% + 4px); font-size: 0.6875rem; white-space: nowrap; z-index: 10; pointer-events: none;"
              >
                {{ b.label }}: <strong>{{ b.count }}</strong> peserta
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <ul class="nav nav-tabs mb-3 d-print-none">
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'rekap' }" href="#" @click.prevent="activeTab = 'rekap'">
          <i class="ti ti-award me-1"></i>Rekap Nilai
        </a>
      </li>
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'analisis' }" href="#" @click.prevent="activeTab = 'analisis'">
          <i class="ti ti-trending-up me-1"></i>Analisis Butir Soal
        </a>
      </li>
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'regrade' }" href="#" @click.prevent="activeTab = 'regrade'">
          <i class="ti ti-history me-1"></i>Riwayat Regrade
          <span v-if="regradeLogs.length" class="badge bg-blue-lt text-blue ms-1">{{ regradeLogs.length }}</span>
        </a>
      </li>
    </ul>

    <!-- Tab: Rekap Nilai -->
    <div v-if="activeTab === 'rekap'">
      <div class="d-none d-print-block fw-bold mb-2">Rekap Nilai — {{ selectedSchedule?.name }}</div>
      <div v-if="loadingReport" class="p-4 text-center text-muted">
        <span class="spinner-border spinner-border-sm me-2"></span>Memuat laporan...
      </div>
      <div v-else-if="sortedSessions.length" class="card">
        <div class="table-responsive">
          <table class="table table-vcenter table-hover">
            <thead>
              <tr>
                <th>No</th>
                <th>Peserta</th>
                <th>Nilai</th>
                <th>%</th>
                <th>Terjawab</th>
                <th>Durasi</th>
                <th>Pelanggaran</th>
                <th>Status</th>
                <th class="d-print-none">Detail</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in sortedSessions" :key="row.session_id">
                <td>{{ rankBadge(i) }}</td>
                <td>
                  <div class="d-flex align-items-center gap-2">
                    <i class="ti ti-user text-muted"></i>
                    <span>{{ row.user_name }}</span>
                    <span class="text-muted small">{{ row.username }}</span>
                  </div>
                </td>
                <td>
                  <span class="fw-semibold" :class="gradeColorClass(row.percent)">{{ row.score.toFixed(1) }}</span>
                  <span class="text-muted"> / {{ row.max_score.toFixed(1) }}</span>
                </td>
                <td class="w-10">
                  <div class="d-flex align-items-center gap-2">
                    <div class="progress progress-sm flex-fill">
                      <div class="progress-bar" :class="gradeBgClass(row.percent)" :style="{ width: row.percent + '%' }"></div>
                    </div>
                    <span class="small">{{ row.percent.toFixed(1) }}%</span>
                  </div>
                </td>
                <td>{{ row.answered_count }} / {{ row.total_questions }}</td>
                <td>{{ formatDuration(row.duration_seconds) }}</td>
                <td>
                  <span v-if="row.violation_count" class="badge bg-red-lt text-red">{{ row.violation_count }}x</span>
                  <span v-else class="badge bg-green-lt text-green">0</span>
                </td>
                <td>
                  <span class="badge"
                    :class="{
                      'bg-green-lt text-green': row.status === 'finished',
                      'bg-blue-lt text-blue': row.status === 'ongoing',
                      'bg-red-lt text-red': row.status === 'terminated',
                      'bg-secondary-lt text-secondary': row.status === 'not_started',
                    }"
                  >{{ statusLabel(row.status) }}</span>
                </td>
                <td class="d-print-none">
                  <button class="btn btn-sm btn-ghost-primary" @click="showPersonalReport(row)">
                    <i class="ti ti-chevron-right"></i>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else class="card">
        <div class="card-body text-center text-muted py-4">
          <img :src="getIllustration('search')" class="img-fluid mb-3 opacity-75" style="max-height:100px" alt="">
          <p class="mb-0">Belum ada data sesi</p>
        </div>
      </div>
    </div>

    <!-- Tab: Analisis Butir Soal -->
    <div v-if="activeTab === 'analisis'">
      <div class="d-none d-print-block fw-bold mb-2">Analisis Butir Soal — {{ selectedSchedule?.name }}</div>
      <div v-if="loadingAnalysis" class="p-4 text-center text-muted">
        <span class="spinner-border spinner-border-sm me-2"></span>Memuat analisis...
      </div>
      <div v-else-if="analysis?.questions?.length" class="card">
        <div class="table-responsive">
          <table class="table table-vcenter table-hover">
            <thead>
              <tr>
                <th>No</th>
                <th>Soal</th>
                <th>Tipe</th>
                <th>TK (p)</th>
                <th>DP (D)</th>
                <th>Kualitas</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(q, i) in analysis.questions" :key="q.question_id">
                <td>{{ i + 1 }}</td>
                <td class="text-muted small">{{ q.body.substring(0, 100) }}{{ q.body.length > 100 ? '...' : '' }}</td>
                <td><span class="badge bg-blue-lt text-blue">{{ q.question_type.replace('_', ' ') }}</span></td>
                <td style="min-width: 100px">
                  <div class="d-flex align-items-center gap-2">
                    <div class="progress progress-sm flex-fill">
                      <div class="progress-bar bg-primary" :style="{ width: (q.difficulty_index * 100) + '%' }"></div>
                    </div>
                    <span class="small">{{ (q.difficulty_index * 100).toFixed(1) }}%</span>
                  </div>
                </td>
                <td>
                  <span :class="q.discrimination_index >= 0.3 ? 'text-primary fw-semibold' : 'text-danger fw-semibold'">
                    {{ q.discrimination_index.toFixed(3) }}
                  </span>
                </td>
                <td>
                  <span class="badge" :class="QUALITY_CLASSES[q.quality] ?? 'bg-secondary-lt text-secondary'">
                    {{ q.quality }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div v-else class="card">
        <div class="card-body text-center text-muted py-4">
          <img :src="getIllustration('chart')" class="img-fluid mb-3 opacity-75" style="max-height:100px" alt="">
          <p class="mb-0">Belum cukup data untuk analisis (perlu minimal 6 sesi selesai)</p>
        </div>
      </div>
    </div>

    <!-- Tab: Riwayat Regrade -->
    <div v-if="activeTab === 'regrade'">
      <div v-if="loadingRegradeLogs" class="p-4 text-center text-muted">
        <span class="spinner-border spinner-border-sm me-2"></span>Memuat riwayat regrade...
      </div>
      <div v-else-if="regradeLogs.length">
        <!-- Summary stats -->
        <div class="row g-2 mb-3">
          <div class="col-6 col-sm-4 col-lg-3">
            <div class="card card-sm">
              <div class="card-body text-center">
                <div class="h3 fw-bold mb-0 text-blue">{{ regradeLogs.length }}</div>
                <div class="text-muted small">Total Regrade</div>
              </div>
            </div>
          </div>
          <div class="col-6 col-sm-4 col-lg-3">
            <div class="card card-sm">
              <div class="card-body text-center">
                <div class="h3 fw-bold mb-0 text-orange">{{ regradeLogs.reduce((sum, l) => sum + countChanges(l), 0) }}</div>
                <div class="text-muted small">Total Perubahan Nilai</div>
              </div>
            </div>
          </div>
          <div v-if="regradeLogs.length" class="col-6 col-sm-4 col-lg-3">
            <div class="card card-sm">
              <div class="card-body text-center">
                <div class="h3 fw-bold mb-0">{{ formatDateTime(regradeLogs[0]!.created_at) }}</div>
                <div class="text-muted small">Regrade Terakhir</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Regrade log entries -->
        <div class="d-flex flex-column gap-2">
          <div v-for="log in regradeLogs" :key="log.id" class="card">
            <div class="card-header cursor-pointer" @click="toggleLogExpand(log.id)" style="cursor:pointer">
              <div class="d-flex align-items-center w-100">
                <div class="d-flex align-items-center gap-3 flex-fill">
                  <span class="avatar avatar-sm bg-blue-lt">
                    <i class="ti ti-refresh text-blue"></i>
                  </span>
                  <div>
                    <div class="fw-medium">
                      Regrade oleh <strong>{{ log.requested_name }}</strong>
                    </div>
                    <div class="text-muted small">
                      <i class="ti ti-calendar me-1"></i>{{ formatDateTime(log.created_at) }}
                      <span class="mx-2">|</span>
                      <i class="ti ti-users me-1"></i>{{ log.sessions_count }} sesi diproses
                      <span v-if="countChanges(log) > 0" class="mx-2">|</span>
                      <span v-if="countChanges(log) > 0" class="text-orange fw-medium">
                        <i class="ti ti-arrow-right me-1"></i>{{ countChanges(log) }} perubahan nilai
                      </span>
                      <span v-else class="mx-2">|</span>
                      <span v-if="countChanges(log) === 0" class="text-muted">Tidak ada perubahan</span>
                    </div>
                  </div>
                </div>
                <i class="ti" :class="expandedLogId === log.id ? 'ti-chevron-up' : 'ti-chevron-down'"></i>
              </div>
            </div>

            <!-- Expanded: score changes detail -->
            <div v-if="expandedLogId === log.id && log.score_changes && log.score_changes.length" class="card-body pt-0">
              <div class="table-responsive">
                <table class="table table-vcenter table-sm table-hover mb-0">
                  <thead>
                    <tr>
                      <th>No</th>
                      <th>Session ID</th>
                      <th class="text-end">Skor Lama</th>
                      <th class="text-end">Skor Baru</th>
                      <th class="text-end">Selisih</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(sc, i) in log.score_changes" :key="i">
                      <td>{{ i + 1 }}</td>
                      <td class="text-muted">#{{ sc.session_id }}</td>
                      <td class="text-end">{{ sc.old_score.toFixed(1) }}</td>
                      <td class="text-end fw-bold" :class="sc.new_score > sc.old_score ? 'text-success' : sc.new_score < sc.old_score ? 'text-danger' : ''">
                        {{ sc.new_score.toFixed(1) }}
                      </td>
                      <td class="text-end">
                        <span v-if="sc.new_score !== sc.old_score"
                          :class="sc.new_score - sc.old_score > 0 ? 'text-success' : 'text-danger'"
                        >
                          {{ sc.new_score - sc.old_score > 0 ? '+' : '' }}{{ (sc.new_score - sc.old_score).toFixed(1) }}
                        </span>
                        <span v-else class="text-muted">0</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <div v-else-if="expandedLogId === log.id && (!log.score_changes || !log.score_changes.length)" class="card-body pt-0">
              <div class="text-muted small">Tidak ada detail perubahan nilai tercatat.</div>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="card">
        <div class="card-body text-center text-muted py-5">
          <i class="ti ti-history fs-1 d-block mb-2 opacity-50"></i>
          <p class="mb-1">Belum ada riwayat regrade</p>
          <p class="small mb-0">Gunakan tombol "Hitung Ulang Nilai" di atas untuk melakukan regrade.</p>
        </div>
      </div>
    </div>
  </template>

  <BaseConfirmModal
    v-if="showRegradeConfirm"
    title="Hitung Ulang Nilai"
    message="Hitung ulang nilai semua sesi? Nilai lama akan diganti."
    confirm-label="Ya, Hitung Ulang"
    confirm-variant="warning"
    @confirm="doRegrade"
    @close="showRegradeConfirm = false"
  />
</template>
