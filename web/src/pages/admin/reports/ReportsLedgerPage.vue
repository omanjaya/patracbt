<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { useToastStore } from '../../../stores/toast.store'
import client from '../../../api/client'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

interface Question {
  id: number
  type: string
  body: string
  no: number
}

interface AnswerCell {
  question_id: number
  response: string
}

interface LedgerRow {
  session_id: number
  student_name: string
  nis: string
  rombel: string
  final_score: number
  violation: number
  answers: AnswerCell[]
}

interface LedgerMeta {
  total: number
  total_pages: number
  current_page: number
  per_page: number
  first_item: number
  last_item: number
}

interface RombelOption {
  id: number
  name: string
}

const toast = useToastStore()

// ─── State ─────────────────────────────────────────────────────
const schedules = ref<ExamSchedule[]>([])
const selectedId = ref<number | null>(null)
const rows = ref<LedgerRow[]>([])
const questions = ref<Question[]>([])
const meta = ref<LedgerMeta | null>(null)
const availableRombels = ref<RombelOption[]>([])
const loading = ref(false)

const search = ref('')
const rombelId = ref<number | ''>('')
const perPage = ref(50)
const page = ref(1)

const showExportModal = ref(false)

// ─── Computed ──────────────────────────────────────────────────
const selectedSchedule = computed(() => schedules.value.find((s) => s.id === selectedId.value))
const totalPages = computed(() => meta.value?.total_pages ?? 1)
const total = computed(() => meta.value?.total ?? 0)

// ─── API ──────────────────────────────────────────────────────
async function loadSchedules() {
  try {
    const res = await examApi.listSchedules({ per_page: 100 })
    schedules.value = res.data.data ?? []
  } catch {}
}

async function fetchLedger() {
  if (!selectedId.value) return
  loading.value = true
  try {
    const res = await client.get(`/reports/${selectedId.value}/ledger`, {
      params: {
        search: search.value || undefined,
        rombel_id: rombelId.value || undefined,
        per_page: perPage.value,
        page: page.value,
      },
    })
    const data = res.data.data
    rows.value = data.rows ?? []
    questions.value = data.questions ?? []
    meta.value = data.meta ?? null
    availableRombels.value = data.available_rombels ?? []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat ledger')
  } finally {
    loading.value = false
  }
}

function selectSchedule(id: number) {
  selectedId.value = id
  search.value = ''
  rombelId.value = ''
  page.value = 1
  fetchLedger()
}

function applyFilter() {
  page.value = 1
  fetchLedger()
}

function resetFilter() {
  search.value = ''
  rombelId.value = ''
  perPage.value = 50
  page.value = 1
  fetchLedger()
}

function exportExcel(format: 'single' | 'sheets') {
  if (!selectedId.value) return
  const params = new URLSearchParams()
  params.set('format', format)
  if (rombelId.value) params.set('rombel_id', String(rombelId.value))
  window.open(`/api/reports/${selectedId.value}/ledger/export?${params.toString()}`, '_blank')
  showExportModal.value = false
}

onMounted(loadSchedules)
</script>

<template>
  <!-- Header -->
  <BasePageHeader
    title="Ledger Nilai"
    :subtitle="selectedSchedule ? selectedSchedule.name : 'Pilih jadwal ujian untuk melihat ledger'"
    :breadcrumbs="[{ label: 'Laporan', to: '/admin/reports' }, { label: 'Ledger Nilai' }]"
  >
    <template #actions>
      <template v-if="selectedId">
        <button class="btn btn-success btn-sm" @click="showExportModal = true">
          <i class="ti ti-file-spreadsheet me-1"></i>
          <span class="d-none d-sm-inline">Excel</span>
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

  <!-- Empty state -->
  <div v-if="!selectedId" class="card">
    <div class="empty">
      <div class="empty-icon">
        <i class="ti ti-table-off" style="font-size: 3rem;"></i>
      </div>
      <p class="empty-title">Belum ada data nilai</p>
      <p class="empty-subtitle text-muted">Pilih jadwal ujian di atas untuk melihat ledger nilai peserta.</p>
    </div>
  </div>

  <template v-else>
    <!-- Filter Toolbar -->
    <div class="card mb-3 d-print-none">
      <div class="card-body py-2">
        <div class="row g-2 align-items-end">
          <div class="col-12 col-sm-6 col-lg-4">
            <label class="form-label mb-1 small">Cari Peserta</label>
            <div class="input-group input-group-sm">
              <span class="input-group-text"><i class="ti ti-search"></i></span>
              <input
                v-model="search"
                type="text"
                class="form-control form-control-sm"
                placeholder="Cari nama siswa atau NIS..."
                @keyup.enter="applyFilter"
              />
            </div>
          </div>
          <div class="col-6 col-sm-3 col-lg-3">
            <label class="form-label mb-1 small">Kelas</label>
            <select v-model="rombelId" class="form-select form-select-sm">
              <option value="">Semua Kelas</option>
              <option v-for="r in availableRombels" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </div>
          <div class="col-6 col-sm-3 col-lg-2">
            <label class="form-label mb-1 small">Per Halaman</label>
            <select v-model="perPage" class="form-select form-select-sm">
              <option :value="25">25 / hal</option>
              <option :value="50">50 / hal</option>
              <option :value="100">100 / hal</option>
            </select>
          </div>
          <div class="col-12 col-lg-3 d-flex gap-2">
            <button class="btn btn-primary btn-sm flex-fill" @click="applyFilter">
              <i class="ti ti-filter me-1"></i>Filter
            </button>
            <button
              v-if="search || rombelId || perPage !== 50"
              class="btn btn-ghost-secondary btn-sm"
              @click="resetFilter"
            >
              Reset
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Summary bar -->
    <div class="d-flex align-items-center gap-2 mb-2 px-1">
      <span class="text-muted small">
        <template v-if="total > 0">
          Menampilkan <strong>{{ meta?.first_item }}–{{ meta?.last_item }}</strong>
          dari <strong>{{ total }}</strong> siswa
        </template>
        <template v-else>Tidak ada data</template>
      </span>
      <span v-if="rombelId" class="badge bg-blue-lt">
        {{ availableRombels.find(r => r.id === rombelId)?.name ?? rombelId }}
      </span>
      <span v-if="search" class="badge bg-yellow-lt">Pencarian: "{{ search }}"</span>
      <span class="badge bg-muted-lt ms-auto small">{{ questions.length }} soal</span>
    </div>

    <!-- Ledger Table Card -->
    <div class="card">
      <div v-if="loading" class="card-body text-center text-muted py-5">
        <span class="spinner-border spinner-border-sm me-2"></span>Memuat ledger...
      </div>

      <div v-else-if="rows.length === 0">
        <div class="empty">
          <div class="empty-icon">
            <i class="ti ti-database-off" style="font-size: 3rem;"></i>
          </div>
          <p class="empty-title">Belum ada data nilai</p>
          <p class="empty-subtitle text-muted">
            <template v-if="search">Tidak ditemukan siswa dengan kata kunci "{{ search }}".</template>
            <template v-else>Belum ada siswa yang menyelesaikan ujian ini.</template>
          </p>
          <div v-if="search || rombelId" class="empty-action">
            <button class="btn btn-sm btn-outline-primary" @click="resetFilter">
              <i class="ti ti-filter-off me-1"></i>Reset Filter
            </button>
          </div>
        </div>
      </div>

      <template v-else>
        <div class="table-responsive">
          <table class="table table-bordered table-vcenter mb-0" style="border-collapse:separate; border-spacing:0;">
            <thead>
              <tr>
                <th class="text-center ledger-sticky-col ledger-sticky-col-1" rowspan="2" style="min-width:42px; width:42px; white-space:nowrap;">No</th>
                <th rowspan="2" class="ledger-sticky-col ledger-sticky-col-2" style="min-width:180px; white-space:nowrap;">Nama Siswa</th>
                <th rowspan="2" style="min-width:90px; white-space:nowrap;">NIS</th>
                <th rowspan="2" class="text-center" style="min-width:70px; white-space:nowrap;">Kelas</th>
                <th rowspan="2" class="text-center" style="min-width:65px; white-space:nowrap;">Nilai</th>
                <th rowspan="2" class="text-center text-danger" style="min-width:40px; white-space:nowrap;" title="Pelanggaran">
                  <i class="ti ti-alert-triangle"></i>
                </th>
                <th
                  v-if="questions.length > 0"
                  class="text-center bg-light"
                  :colspan="questions.length"
                  style="font-weight:600; white-space:nowrap;"
                >
                  Jawaban per Soal
                </th>
              </tr>
              <tr v-if="questions.length > 0">
                <th
                  v-for="q in questions"
                  :key="q.id"
                  class="text-center bg-light"
                  style="min-width:48px; white-space:nowrap;"
                  :title="`Soal ${q.no} (${q.type})`"
                >
                  {{ q.no }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in rows" :key="row.session_id">
                <td class="text-center text-muted ledger-sticky-col ledger-sticky-col-1" style="font-size:0.8125rem;">
                  {{ ((page - 1) * perPage) + index + 1 }}
                </td>
                <td class="ledger-sticky-col ledger-sticky-col-2" style="font-size:0.8125rem; white-space:nowrap;">
                  <strong>{{ row.student_name }}</strong>
                </td>
                <td class="text-muted" style="font-size:0.8125rem; white-space:nowrap;">{{ row.nis }}</td>
                <td class="text-center">
                  <span class="badge bg-azure-lt" style="font-size:0.7rem;">{{ row.rombel }}</span>
                </td>
                <td class="text-center fw-bold" style="font-size:0.9375rem;">{{ row.final_score }}</td>
                <td
                  class="text-center"
                  :class="row.violation > 0 ? 'text-danger fw-bold' : 'text-muted'"
                  style="font-size:0.8125rem;"
                >
                  {{ row.violation > 0 ? row.violation : '-' }}
                </td>
                <td
                  v-for="ans in row.answers"
                  :key="ans.question_id"
                  class="text-center"
                  style="min-width:48px; font-size:0.8125rem;"
                >
                  {{ ans.response }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <BasePagination
          v-if="totalPages > 1"
          :page="page"
          :total-pages="totalPages"
          :total="total"
          :per-page="perPage"
          @change="p => { page = p; fetchLedger() }"
        />
      </template>
    </div>
  </template>

  <!-- Export Options Modal -->
  <BaseModal
    v-if="showExportModal"
    title="Pilih Format Export"
    size="sm"
    @close="showExportModal = false"
  >
    <p class="text-muted mb-3">
      <template v-if="!rombelId">Anda akan mengekspor <strong>Semua Rombel</strong>. Pilih format:</template>
      <template v-else>Export untuk rombel yang dipilih.</template>
    </p>
    <div class="d-grid gap-2">
      <button class="btn btn-primary" @click="exportExcel('single')">
        <i class="ti ti-file-spreadsheet me-1"></i>
        Download File Tunggal
      </button>
      <small class="text-muted text-center">Semua data dalam 1 sheet</small>
      <button class="btn btn-outline-primary" @click="exportExcel('sheets')">
        <i class="ti ti-copy me-1"></i>
        Download Multi-Sheet
      </button>
      <small class="text-muted text-center">Tiap kelas jadi sheet terpisah</small>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="showExportModal = false">Batal</button>
    </template>
  </BaseModal>
</template>

<style scoped>
/* Sticky columns for ledger table */
.ledger-sticky-col {
  position: sticky;
  z-index: 2;
  background: #ffffff;
}

.ledger-sticky-col-1 {
  left: 0;
  border-right: none;
}

.ledger-sticky-col-2 {
  left: 42px;
  border-right: 1px solid #e6e7e9 !important;
  box-shadow: 2px 0 4px -1px rgba(0, 0, 0, 0.08);
}

/* Keep sticky columns above regular cells */
thead .ledger-sticky-col {
  z-index: 3;
  background: #f8f9fa;
}

/* Dark mode support */
[data-bs-theme="dark"] .ledger-sticky-col {
  background: #1e2433;
}

[data-bs-theme="dark"] thead .ledger-sticky-col {
  background: #232837;
}

[data-bs-theme="dark"] .ledger-sticky-col-2 {
  border-right-color: #2c3145 !important;
  box-shadow: 2px 0 4px -1px rgba(0, 0, 0, 0.25);
}
</style>
