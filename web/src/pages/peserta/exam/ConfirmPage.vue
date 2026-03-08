<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { getIllustration } from '../../../utils/avatar'

const route = useRoute()
const router = useRouter()
const scheduleId = Number(route.params.id)

const schedule = ref<ExamSchedule | null>(null)
const loading = ref(true)
const token = ref('')
const starting = ref(false)
const verifying = ref(false)
const error = ref('')

async function fetchSchedule() {
  try {
    const res = await examApi.getSchedule(scheduleId)
    schedule.value = res.data.data
  } catch {
    error.value = 'Jadwal ujian tidak ditemukan.'
  } finally {
    loading.value = false
  }
}

async function handleStart() {
  const trimmed = token.value.trim()
  if (!trimmed) { error.value = 'Masukkan token ujian.'; return }
  if (!/^[a-zA-Z0-9\-]+$/.test(trimmed)) { error.value = 'Format token tidak valid. Gunakan huruf, angka, atau tanda hubung.'; return }
  starting.value = true
  verifying.value = true
  error.value = ''
  try {
    const res = await examApi.startExam({ exam_schedule_id: scheduleId, token: token.value.trim().toUpperCase() })
    const sessionId = res.data.data.session.id
    router.replace(`/peserta/exam/${sessionId}`)
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Token tidak valid atau ujian belum tersedia.'
  } finally {
    starting.value = false
    verifying.value = false
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchSchedule)
</script>

<template>
  <div class="d-flex justify-content-center align-items-start py-4">
    <div class="col-lg-6 col-md-8 col-sm-11">

      <div v-if="loading" class="card">
        <div class="card-body py-4">
          <!-- Loading skeleton -->
          <div class="d-flex align-items-center gap-3 mb-4">
            <div class="placeholder-glow"><span class="placeholder rounded-circle" style="width:48px;height:48px;display:block"></span></div>
            <div class="flex-fill placeholder-glow">
              <span class="placeholder col-7 mb-1" style="height:1.2rem;display:block;border-radius:4px"></span>
              <span class="placeholder col-4" style="height:0.8rem;display:block;border-radius:4px"></span>
            </div>
          </div>
          <div class="placeholder-glow mb-3">
            <span class="placeholder col-12 mb-2" style="height:0.9rem;display:block;border-radius:4px"></span>
            <span class="placeholder col-8 mb-2" style="height:0.9rem;display:block;border-radius:4px"></span>
            <span class="placeholder col-10" style="height:0.9rem;display:block;border-radius:4px"></span>
          </div>
          <div class="text-center text-muted">
            <span class="spinner-border spinner-border-sm me-2"></span>Memuat informasi ujian...
          </div>
        </div>
      </div>

      <div v-else-if="!schedule" class="card border-danger">
        <div class="card-body text-center py-4">
          <i class="ti ti-alert-triangle fs-4 text-danger mb-2 d-block"></i>
          <p class="text-danger">{{ error || 'Ujian tidak ditemukan.' }}</p>
        </div>
      </div>

      <div v-else class="card">
        <!-- Header -->
        <div class="card-header">
          <div class="d-flex align-items-center gap-3">
            <span class="avatar avatar-lg bg-primary-lt">
              <i class="ti ti-books text-primary fs-3"></i>
            </span>
            <div>
              <h3 class="card-title mb-0">{{ schedule.name }}</h3>
              <p class="text-muted mb-0 small">Informasi Ujian</p>
            </div>
          </div>
        </div>

        <div class="card-body">
          <div class="text-center">
            <img :src="getIllustration('boy-and-laptop')" class="img-fluid mb-3" style="max-height:120px" alt="">
          </div>
          <!-- Info grid -->
          <div class="row g-2 mb-3">
            <div class="col-sm-6">
              <div class="d-flex align-items-start gap-2">
                <i class="ti ti-calendar-clock text-muted mt-1"></i>
                <div>
                  <div class="text-muted small">Waktu Mulai</div>
                  <div class="fw-medium small">{{ formatDate(schedule.start_time) }}</div>
                </div>
              </div>
            </div>
            <div class="col-sm-6">
              <div class="d-flex align-items-start gap-2">
                <i class="ti ti-calendar-clock text-muted mt-1"></i>
                <div>
                  <div class="text-muted small">Waktu Selesai</div>
                  <div class="fw-medium small">{{ formatDate(schedule.end_time) }}</div>
                </div>
              </div>
            </div>
            <div class="col-sm-6">
              <div class="d-flex align-items-start gap-2">
                <i class="ti ti-clock text-muted mt-1"></i>
                <div>
                  <div class="text-muted small">Durasi Pengerjaan</div>
                  <div class="fw-medium">{{ schedule.duration_minutes }} menit</div>
                </div>
              </div>
            </div>
            <div class="col-sm-6">
              <div class="d-flex align-items-start gap-2">
                <i class="ti ti-alert-triangle text-orange mt-1"></i>
                <div>
                  <div class="text-muted small">Maks. Pelanggaran</div>
                  <div class="fw-medium">{{ schedule.max_violations }}x pindah tab/window</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Rules -->
          <div class="alert alert-warning mb-3">
            <h4 class="alert-title">Peraturan Ujian</h4>
            <ul class="mb-0 ps-3 small">
              <li>Pastikan koneksi internet stabil sebelum memulai.</li>
              <li>Jangan pindah tab atau minimize browser selama ujian berlangsung.</li>
              <li>Jawaban disimpan otomatis saat Anda menjawab.</li>
              <li>Ujian akan otomatis berakhir saat waktu habis.</li>
              <li v-if="schedule.randomize_questions">Urutan soal diacak untuk setiap peserta.</li>
            </ul>
          </div>

          <!-- Token input -->
          <div class="mb-3">
            <label class="form-label">
              <i class="ti ti-key me-1"></i>Token Ujian
            </label>
            <input
              v-model="token"
              class="form-control form-control-lg text-center text-uppercase"
              placeholder="Contoh: ABC-123"
              @keyup.enter="handleStart"
              :disabled="starting"
              autocomplete="off"
            />
            <div v-if="error" class="invalid-feedback d-block">{{ error }}</div>
          </div>
        </div>

        <div class="card-footer d-flex gap-2">
          <button class="btn btn-ghost-secondary" @click="$router.back()">
            <i class="ti ti-arrow-left me-1"></i>Kembali
          </button>
          <button class="btn btn-primary flex-fill" @click="handleStart" :disabled="starting">
            <span v-if="starting" class="spinner-border spinner-border-sm me-1"></span>
            <i v-else class="ti ti-player-play me-1"></i>
            {{ verifying ? 'Memverifikasi token...' : starting ? 'Memulai...' : 'Mulai Ujian' }}
          </button>
        </div>
      </div>

    </div>
  </div>
</template>

<style scoped>
@media (max-width: 767px) {
  .card-body .row.g-2 .col-sm-6 {
    margin-bottom: 0.25rem;
  }
  .card-footer {
    flex-direction: column;
  }
  .card-footer .btn {
    width: 100%;
    min-height: 48px;
    font-size: 1rem;
  }
  .card-footer .btn-primary {
    order: -1;
  }
  .form-control-lg {
    font-size: 1.1rem;
    min-height: 52px;
  }
  .alert ul {
    font-size: 0.875rem;
    line-height: 1.7;
  }
}

@media (min-width: 768px) and (max-width: 1024px) {
  .card-footer .btn {
    min-height: 44px;
  }
  .form-control-lg {
    min-height: 48px;
  }
}
</style>
