<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { examApi, type ExamSchedule, STATUS_LABELS, STATUS_COLORS } from '../../../api/exam.api'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'

const router = useRouter()

const columns = [
  { key: 'name', label: 'Nama Ujian' },
  { key: 'time', label: 'Waktu Pelaksanaan' },
  { key: 'status', label: 'Status' },
  { key: 'deleted_at', label: 'Tanggal Diarsipkan' },
  { key: 'actions', label: 'Aksi', width: '160px' },
]

const list = ref<(ExamSchedule & { deleted_at?: string })[]>([])
const loading = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)
const search = ref('')

async function fetchList() {
  loading.value = true
  try {
    const res = await examApi.listTrashedSchedules({ page: page.value, per_page: 20, search: search.value })
    list.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } finally {
    loading.value = false
  }
}

async function handleRestore(item: ExamSchedule) {
  if (!confirm(`Pulihkan jadwal "${item.name}"? Jadwal akan kembali ke daftar aktif.`)) return
  await examApi.restoreSchedule(item.id)
  await fetchList()
}

async function handleForceDelete(item: ExamSchedule) {
  if (!confirm(`Hapus permanen jadwal "${item.name}"? Tindakan ini tidak dapat dibatalkan.`)) return
  await examApi.forceDeleteSchedule(item.id)
  await fetchList()
}

function formatDate(d: string | undefined) {
  if (!d) return '—'
  return new Date(d).toLocaleString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(() => fetchList())
</script>

<template>
  <div>
    <!-- Page Header -->
    <div class="page-header d-print-none mb-3">
      <div class="row align-items-center">
        <div class="col">
          <h2 class="page-title">Arsip Jadwal Ujian</h2>
          <p class="text-muted mb-0">Data jadwal ujian yang telah diarsipkan / dihapus sementara.</p>
        </div>
        <div class="col-auto ms-auto d-print-none">
          <button class="btn btn-outline-secondary" @click="router.push('/admin/exam-schedules')">
            <i class="ti ti-arrow-bar-left me-1"></i>
            Kembali ke Daftar Utama
          </button>
        </div>
      </div>
    </div>

    <!-- Card -->
    <div class="card">
      <!-- Search -->
      <div class="card-header">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input
            v-model="search"
            @input="page = 1; fetchList()"
            class="form-control"
            placeholder="Cari jadwal di arsip..."
          />
        </div>
      </div>

      <!-- Empty state -->
      <div v-if="!loading && list.length === 0" class="card-body">
        <div class="empty">
          <div class="empty-icon">
            <i class="ti ti-trash-off" style="font-size: 3rem;"></i>
          </div>
          <p class="empty-title">Tidak ada jadwal di tempat sampah</p>
          <p class="empty-subtitle text-muted">Semua jadwal ujian yang diarsipkan akan muncul di sini.</p>
          <div class="empty-action">
            <button class="btn btn-outline-secondary btn-sm" @click="router.push('/admin/exam-schedules')">
              <i class="ti ti-arrow-bar-left me-1"></i>Kembali ke Daftar
            </button>
          </div>
        </div>
      </div>

      <BaseTable v-else :columns="columns" :loading="loading" empty="Tidak ada data di arsip.">
        <tr v-for="item in list" :key="item.id">
          <!-- Nama Ujian -->
          <td>
            <div class="d-flex align-items-center gap-2">
              <i class="ti ti-calendar-off text-muted"></i>
              <div>
                <p class="fw-medium mb-0">{{ item.name }}</p>
                <p class="text-muted small mb-0">{{ item.duration_minutes }} menit</p>
              </div>
            </div>
          </td>

          <!-- Waktu Pelaksanaan -->
          <td>
            <p class="small mb-0">{{ formatDate(item.start_time) }}</p>
            <p class="text-muted small mb-0">s/d {{ formatDate(item.end_time) }}</p>
          </td>

          <!-- Status -->
          <td>
            <span
              class="badge"
              :class="`bg-${STATUS_COLORS[item.status as keyof typeof STATUS_COLORS] === 'default' ? 'secondary' : STATUS_COLORS[item.status as keyof typeof STATUS_COLORS]}-lt`"
            >
              {{ STATUS_LABELS[item.status] }}
            </span>
          </td>

          <!-- Tanggal Diarsipkan -->
          <td>
            <span class="text-muted small">{{ formatDate((item as ExamSchedule & { deleted_at?: string }).deleted_at) }}</span>
          </td>

          <!-- Aksi -->
          <td>
            <div class="d-flex gap-1">
              <button
                class="btn btn-sm btn-outline-success"
                title="Pulihkan"
                @click="handleRestore(item)"
              >
                <i class="ti ti-rotate-clockwise me-1"></i>Pulihkan
              </button>
              <button
                class="btn btn-sm btn-outline-danger"
                title="Hapus Permanen"
                @click="handleForceDelete(item)"
              >
                <i class="ti ti-eraser me-1"></i>Hapus
              </button>
            </div>
          </td>
        </tr>
      </BaseTable>

      <BasePagination
        v-if="totalPages > 1"
        :page="page"
        :total-pages="totalPages"
        :total="total"
        :per-page="20"
        @change="p => { page = p; fetchList() }"
      />
    </div>
  </div>
</template>
