<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { dashboardApi, type PengawasViolation, type PengawasActiveRoom } from '../../../api/dashboard.api'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()
const loading = ref(true)
const violations = ref<PengawasViolation[]>([])
const schedules = ref<PengawasActiveRoom[]>([])

// Filters
const filterSchedule = ref('')
const filterSeverity = ref('')
const filterDate = ref('')

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

function violationSeverity(type: string): 'high' | 'medium' | 'low' {
  if (['fullscreen_exit', 'multi_tab', 'alt_tab', 'external_paste'].includes(type)) return 'high'
  if (['tab_switch', 'blur_extended', 'window_resize', 'focus_lost'].includes(type)) return 'medium'
  return 'low'
}

function severityLabel(type: string): string {
  const sev = violationSeverity(type)
  return sev === 'high' ? 'Tinggi' : sev === 'medium' ? 'Sedang' : 'Rendah'
}

function severityBadgeClass(type: string): string {
  const sev = violationSeverity(type)
  return sev === 'high' ? 'bg-danger text-danger-fg' : sev === 'medium' ? 'bg-warning text-warning-fg' : 'bg-secondary text-secondary-fg'
}

function severityIconClass(type: string): string {
  const sev = violationSeverity(type)
  return sev === 'high' ? 'text-danger' : sev === 'medium' ? 'text-warning' : 'text-secondary'
}

function formatDateTime(d: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', {
    day: 'numeric', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

// Summary stats
const summaryHigh = computed(() => violations.value.filter(v => violationSeverity(v.violation_type) === 'high').length)
const summaryMedium = computed(() => violations.value.filter(v => violationSeverity(v.violation_type) === 'medium').length)
const summaryLow = computed(() => violations.value.filter(v => violationSeverity(v.violation_type) === 'low').length)

async function loadViolations() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (filterSchedule.value) params.schedule_id = filterSchedule.value
    if (filterSeverity.value) params.severity = filterSeverity.value
    if (filterDate.value) params.date = filterDate.value

    const res = await dashboardApi.getPengawasAllViolations(params)
    violations.value = res.data.data ?? []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data pelanggaran')
  } finally {
    loading.value = false
  }
}

async function loadSchedules() {
  try {
    const res = await dashboardApi.getPengawasActiveRooms()
    schedules.value = res.data.data ?? []
  } catch {
    // Silently fail, schedules are for filter only
  }
}

function resetFilters() {
  filterSchedule.value = ''
  filterSeverity.value = ''
  filterDate.value = ''
  loadViolations()
}

onMounted(() => {
  Promise.all([loadViolations(), loadSchedules()])
})
</script>

<template>
  <BasePageHeader
    title="Log Pelanggaran"
    subtitle="Semua pelanggaran peserta ujian"
    :breadcrumbs="[{ label: 'Log Pelanggaran' }]"
  >
    <template #actions>
      <button class="btn btn-ghost-secondary" @click="loadViolations">
        <i class="ti ti-refresh"></i> Refresh
      </button>
    </template>
  </BasePageHeader>

  <!-- Summary Cards -->
  <div class="row g-3 mb-3">
    <div class="col-sm-4">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="avatar bg-danger-lt">
                <i class="ti ti-alert-octagon fs-4 text-danger"></i>
              </span>
            </div>
            <div class="col">
              <div class="fw-medium h3 mb-0">{{ summaryHigh }}</div>
              <div class="text-muted small">Tingkat Tinggi</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-sm-4">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="avatar bg-warning-lt">
                <i class="ti ti-alert-triangle fs-4 text-warning"></i>
              </span>
            </div>
            <div class="col">
              <div class="fw-medium h3 mb-0">{{ summaryMedium }}</div>
              <div class="text-muted small">Tingkat Sedang</div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-sm-4">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="avatar bg-secondary-lt">
                <i class="ti ti-info-circle fs-4 text-secondary"></i>
              </span>
            </div>
            <div class="col">
              <div class="fw-medium h3 mb-0">{{ summaryLow }}</div>
              <div class="text-muted small">Tingkat Rendah</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Filters -->
  <div class="card mb-3">
    <div class="card-body">
      <div class="row g-3 align-items-end">
        <div class="col-md-3">
          <label class="form-label">Jadwal Ujian</label>
          <select v-model="filterSchedule" class="form-select" @change="loadViolations">
            <option value="">Semua Jadwal</option>
            <option v-for="s in schedules" :key="s.schedule_id" :value="String(s.schedule_id)">
              {{ s.schedule_name }}
            </option>
          </select>
        </div>
        <div class="col-md-3">
          <label class="form-label">Tingkat Keparahan</label>
          <select v-model="filterSeverity" class="form-select" @change="loadViolations">
            <option value="">Semua Tingkat</option>
            <option value="high">Tinggi</option>
            <option value="medium">Sedang</option>
            <option value="low">Rendah</option>
          </select>
        </div>
        <div class="col-md-3">
          <label class="form-label">Tanggal</label>
          <input type="date" v-model="filterDate" class="form-control" @change="loadViolations" />
        </div>
        <div class="col-md-3">
          <button class="btn btn-outline-secondary w-100" @click="resetFilters">
            <i class="ti ti-filter-off me-1"></i>Reset Filter
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Table -->
  <div class="card">
    <div class="card-header">
      <h3 class="card-title">
        <i class="ti ti-list me-2"></i>Daftar Pelanggaran
        <span class="badge bg-secondary-lt text-secondary ms-2">{{ violations.length }}</span>
      </h3>
    </div>

    <div v-if="loading" class="card-body">
      <div class="placeholder-glow">
        <div v-for="n in 5" :key="n" class="placeholder col-12 mb-2" style="height:44px"></div>
      </div>
    </div>

    <div v-else-if="!violations.length" class="card-body text-center py-5">
      <i class="ti ti-shield-check fs-1 text-success d-block mb-2"></i>
      <h3 class="text-muted">Tidak Ada Pelanggaran</h3>
      <p class="text-muted">Tidak ditemukan pelanggaran dengan filter yang dipilih.</p>
    </div>

    <div v-else class="table-responsive">
      <table class="table table-vcenter card-table table-hover">
        <thead>
          <tr>
            <th>Waktu</th>
            <th>Peserta</th>
            <th>Ujian</th>
            <th>Jenis Pelanggaran</th>
            <th>Tingkat</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in violations" :key="v.id">
            <td class="text-nowrap">
              <span class="text-muted small">{{ formatDateTime(v.created_at) }}</span>
            </td>
            <td>
              <div class="fw-medium">{{ v.student_name }}</div>
            </td>
            <td>
              <span class="text-muted">{{ v.schedule_name }}</span>
            </td>
            <td>
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-alert-triangle" :class="severityIconClass(v.violation_type)"></i>
                <span>{{ violationLabels[v.violation_type] ?? v.violation_type }}</span>
              </div>
              <div v-if="v.description" class="text-muted small mt-1">{{ v.description }}</div>
            </td>
            <td>
              <span class="badge" :class="severityBadgeClass(v.violation_type)">
                {{ severityLabel(v.violation_type) }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
