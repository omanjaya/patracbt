<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { reportApi, type ScheduleReport, type SessionRow } from '../../../api/report.api'
import { examApi, type ExamSchedule } from '../../../api/exam.api'

import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()
const scheduleId = Number(route.params.scheduleId)

const schedule = ref<ExamSchedule | null>(null)
const report = ref<ScheduleReport | null>(null)
const loading = ref(true)
const error = ref('')

async function load() {
  try {
    const [scheduleRes, reportRes] = await Promise.all([
      examApi.getSchedule(scheduleId),
      reportApi.getScheduleReport(scheduleId),
    ])
    schedule.value = scheduleRes.data.data
    report.value = reportRes.data.data
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat data.'
  } finally {
    loading.value = false
  }
}

function essayProgress(row: SessionRow) {
  // answered_count approximates essay answers; total_questions includes all types
  return `${row.answered_count}/${row.total_questions}`
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    finished: 'Selesai',
    ongoing: 'Mengerjakan',
    not_started: 'Belum Mulai',
    terminated: 'Terblokir',
  }
  return map[status] ?? status
}

function statusClass(status: string) {
  const map: Record<string, string> = {
    finished: 'bg-green-lt text-green',
    ongoing: 'bg-blue-lt text-blue',
    not_started: 'bg-secondary-lt text-secondary',
    terminated: 'bg-red-lt text-red',
  }
  return map[status] ?? 'bg-secondary-lt text-secondary'
}

onMounted(load)
</script>

<template>
  <BasePageHeader
    title="Koreksi Esai"
    :subtitle="schedule?.name"
    :breadcrumbs="[{ label: 'Penilaian', to: '/guru/reports' }, { label: 'Detail' }]"
  >
    <template #actions>
      <button class="btn btn-ghost-secondary" @click="router.back()">
        <i class="ti ti-arrow-left me-1"></i>Kembali ke Laporan
      </button>
    </template>
  </BasePageHeader>

  <div v-if="loading" class="p-5 text-center text-muted">
    <span class="spinner-border spinner-border-sm me-2"></span>Memuat data...
  </div>

  <div v-else-if="error" class="alert alert-danger">{{ error }}</div>

  <div v-else class="card">
    <div class="card-header">
      <h3 class="card-title">
        <i class="ti ti-users me-2"></i>Daftar Siswa
      </h3>
    </div>
    <div class="table-responsive">
      <table class="table table-vcenter table-hover">
        <thead>
          <tr>
            <th>Nama Siswa</th>
            <th>Username</th>
            <th>Status Ujian</th>
            <th>Soal Dijawab</th>
            <th>Skor</th>
            <th>Koreksi</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!report?.sessions?.length">
            <td colspan="7" class="text-center py-5">
              <i class="ti ti-certificate-off fs-1 mb-2 d-block opacity-50"></i>
              <div class="text-muted">Belum ada siswa yang mengerjakan ujian ini.</div>
            </td>
          </tr>
          <tr v-for="row in report?.sessions" :key="row.session_id">
            <td class="fw-medium">{{ row.user_name }}</td>
            <td class="text-muted small">{{ row.username }}</td>
            <td>
              <span class="badge" :class="statusClass(row.status)" :aria-label="`Status ujian: ${statusLabel(row.status)}`">{{ statusLabel(row.status) }}</span>
            </td>
            <td>{{ essayProgress(row) }}</td>
            <td>{{ row.score }} / {{ row.max_score }}</td>
            <td>
              <span v-if="row.status === 'not_started'" class="badge bg-secondary-lt text-secondary">
                <i class="ti ti-minus me-1"></i>Belum
              </span>
              <span v-else-if="row.score > 0 && row.percent >= 0" class="badge bg-green-lt text-green">
                <i class="ti ti-check me-1"></i>Sudah Dinilai
              </span>
              <span v-else class="badge bg-yellow-lt text-yellow">
                <i class="ti ti-clock me-1"></i>Belum Dinilai
              </span>
            </td>
            <td>
              <button
                class="btn btn-sm btn-primary"
                :disabled="row.status === 'not_started'"
                :title="row.status === 'not_started' ? 'Siswa belum memulai ujian' : 'Koreksi jawaban esai siswa'"
                @click="router.push(`/guru/grading/${scheduleId}/${row.user_id}/${row.session_id}`)"
              >
                Koreksi <i class="ti ti-chevron-right ms-1"></i>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
