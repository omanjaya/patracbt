<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { examApi, type ExamSession } from '../../../api/exam.api'
import { getIllustration } from '../../../utils/avatar'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const session = ref<ExamSession | null>(null)
const loading = ref(true)
const rechecking = ref(false)
const error = ref('')

// Show-score logic
const showScoreAfter = computed(() => session.value?.exam_schedule?.show_score_after ?? 'immediately')

const canShowScore = computed(() => {
  const policy = showScoreAfter.value
  if (policy === 'immediately') return true
  if (policy === 'after_end_time') {
    const endTime = session.value?.exam_schedule?.end_time
    if (!endTime) return false
    return Date.now() >= new Date(endTime).getTime()
  }
  // 'manual' — score is only visible if backend already computed & released it
  // We show it if allow_see_result is true on the schedule
  if (policy === 'manual') {
    return session.value?.exam_schedule?.allow_see_result === true
  }
  return true
})

const pendingMessage = computed(() => {
  const policy = showScoreAfter.value
  if (policy === 'after_end_time') {
    const endTime = session.value?.exam_schedule?.end_time
    const endStr = endTime ? formatDate(endTime) : ''
    return `Nilai akan ditampilkan setelah waktu ujian berakhir${endStr ? ' (' + endStr + ')' : ''}.`
  }
  if (policy === 'manual') {
    return 'Nilai akan ditampilkan setelah disetujui oleh admin.'
  }
  return ''
})

const canShowReport = computed(() => {
  return session.value?.exam_schedule?.allow_see_result === true
})

const scorePercent = computed(() => {
  if (!session.value || !session.value.max_score) return 0
  return Math.round((session.value.score / session.value.max_score) * 100)
})

const gradeLabel = computed(() => {
  const p = scorePercent.value
  if (p >= 90) return { label: 'Sangat Baik', color: 'green' }
  if (p >= 75) return { label: 'Baik', color: 'blue' }
  if (p >= 60) return { label: 'Cukup', color: 'yellow' }
  return { label: 'Perlu Perbaikan', color: 'red' }
})

const gradeColorVar = computed(() => {
  const map: Record<string, string> = {
    green: 'var(--tblr-success)',
    blue: 'var(--tblr-primary)',
    yellow: 'var(--tblr-warning)',
    red: 'var(--tblr-danger)',
  }
  return map[gradeLabel.value.color] ?? 'var(--tblr-secondary)'
})

async function fetchResult() {
  loading.value = true
  error.value = ''
  try {
    const res = await examApi.loadSession(sessionId)
    session.value = res.data.data.session
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat hasil ujian.'
  } finally {
    loading.value = false
  }
}

async function recheckResult() {
  rechecking.value = true
  try {
    const res = await examApi.loadSession(sessionId)
    session.value = res.data.data.session
  } catch {
    // silent — keep existing data
  } finally {
    rechecking.value = false
  }
}

function formatDate(d: string | null) {
  if (!d) return '–'
  return new Date(d).toLocaleString('id-ID', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function duration() {
  if (!session.value?.start_time || !session.value?.finished_at) return '–'
  const diff = Math.floor((new Date(session.value.finished_at).getTime() - new Date(session.value.start_time).getTime()) / 1000)
  const m = Math.floor(diff / 60)
  const s = diff % 60
  return `${m} menit ${s} detik`
}

onMounted(fetchResult)
</script>

<template>
  <div v-if="loading" class="p-5 text-center text-muted">
    <span class="spinner-border spinner-border-sm me-2"></span>Memuat hasil ujian...
  </div>

  <div v-else-if="error" class="alert alert-danger d-flex align-items-center gap-2">
    <i class="ti ti-alert-circle"></i>
    <div>
      {{ error }}
      <div class="mt-2 d-flex gap-2">
        <button class="btn btn-sm btn-outline-danger" @click="fetchResult">
          <i class="ti ti-refresh me-1"></i>Coba Lagi
        </button>
        <button class="btn btn-sm btn-outline-secondary" @click="router.push('/peserta')">Kembali ke Dashboard</button>
      </div>
    </div>
  </div>

  <div v-else-if="session" class="row justify-content-center">
    <div class="col-lg-7">
      <div class="card">
        <div class="card-header">
          <div class="d-flex align-items-center gap-3">
            <span :class="`bg-${gradeLabel.color}-lt`" class="avatar avatar-lg rounded-circle">
              <i class="ti ti-trophy fs-3" :class="`text-${gradeLabel.color}`"></i>
            </span>
            <div>
              <h3 class="card-title mb-0">Ujian Selesai!</h3>
              <p class="text-muted mb-0 small">{{ session.exam_schedule?.name }}</p>
            </div>
          </div>
        </div>

        <div class="card-body text-center py-4">
          <img :src="getIllustration('graduation')" class="img-fluid mb-3" style="max-height:160px" alt="">

          <!-- Score visible -->
          <template v-if="canShowScore">
            <!-- Score circle -->
            <div class="d-flex justify-content-center mb-3">
              <div class="position-relative" style="width:140px;height:140px">
                <svg viewBox="0 0 120 120" width="140" height="140">
                  <circle cx="60" cy="60" r="54" fill="none" stroke="var(--tblr-border-color)" stroke-width="10" />
                  <circle
                    cx="60" cy="60" r="54" fill="none"
                    :stroke="gradeColorVar"
                    stroke-width="10"
                    stroke-linecap="round"
                    :stroke-dasharray="`${scorePercent * 3.393} 339.3`"
                    transform="rotate(-90 60 60)"
                  />
                </svg>
                <div class="position-absolute top-50 start-50 translate-middle text-center">
                  <div class="h2 mb-0 fw-bold">{{ scorePercent }}%</div>
                  <div class="text-muted small">{{ session.score.toFixed(1) }} / {{ session.max_score.toFixed(1) }}</div>
                </div>
              </div>
            </div>

            <span class="badge fs-6 mb-4" :class="`bg-${gradeLabel.color}-lt text-${gradeLabel.color}`">
              {{ gradeLabel.label }}
            </span>

            <!-- Stats -->
            <div class="row g-3 text-start mb-4">
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-secondary-lt"><i class="ti ti-clock text-secondary"></i></span>
                  <div>
                    <div class="text-muted small">Durasi Pengerjaan</div>
                    <div class="fw-medium">{{ duration() }}</div>
                  </div>
                </div>
              </div>
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-primary-lt"><i class="ti ti-star text-primary"></i></span>
                  <div>
                    <div class="text-muted small">Nilai</div>
                    <div class="fw-medium">{{ session.score.toFixed(1) }}</div>
                  </div>
                </div>
              </div>
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-red-lt"><i class="ti ti-alert-triangle text-red"></i></span>
                  <div>
                    <div class="text-muted small">Pelanggaran</div>
                    <div class="fw-medium">{{ session.violation_count }}x</div>
                  </div>
                </div>
              </div>
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-green-lt"><i class="ti ti-circle-check text-green"></i></span>
                  <div>
                    <div class="text-muted small">Selesai</div>
                    <div class="fw-medium">{{ formatDate(session.finished_at) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- Score hidden / pending -->
          <template v-else>
            <div class="mb-4">
              <span class="avatar avatar-xl bg-yellow-lt rounded-circle mb-3">
                <i class="ti ti-clock-pause fs-1 text-yellow"></i>
              </span>
              <h3 class="text-muted fw-normal">Ujian Telah Selesai</h3>
              <div class="alert alert-warning d-inline-flex align-items-center gap-2 mt-2">
                <i class="ti ti-info-circle"></i>
                <span>{{ pendingMessage }}</span>
              </div>

              <!-- Recheck button for manual release -->
              <div v-if="showScoreAfter === 'manual'" class="mt-3">
                <p class="text-muted small mb-2">Nilai akan tersedia setelah admin merilis hasil. Anda dapat mengecek secara berkala.</p>
                <button class="btn btn-outline-primary btn-sm" @click="recheckResult" :disabled="rechecking">
                  <span v-if="rechecking" class="spinner-border spinner-border-sm me-1"></span>
                  <i v-else class="ti ti-refresh me-1"></i>
                  {{ rechecking ? 'Memeriksa...' : 'Muat ulang' }}
                </button>
              </div>

              <!-- Info for after_end_time -->
              <div v-if="showScoreAfter === 'after_end_time'" class="mt-3">
                <p class="text-muted small mb-2">Halaman ini akan menampilkan nilai secara otomatis setelah waktu ujian berakhir. Anda dapat kembali nanti atau muat ulang halaman.</p>
                <button class="btn btn-outline-primary btn-sm" @click="recheckResult" :disabled="rechecking">
                  <span v-if="rechecking" class="spinner-border spinner-border-sm me-1"></span>
                  <i v-else class="ti ti-refresh me-1"></i>
                  {{ rechecking ? 'Memeriksa...' : 'Cek sekarang' }}
                </button>
              </div>
            </div>

            <!-- Minimal stats without score -->
            <div class="row g-3 text-start mb-4">
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-secondary-lt"><i class="ti ti-clock text-secondary"></i></span>
                  <div>
                    <div class="text-muted small">Durasi Pengerjaan</div>
                    <div class="fw-medium">{{ duration() }}</div>
                  </div>
                </div>
              </div>
              <div class="col-6">
                <div class="d-flex align-items-center gap-2">
                  <span class="avatar avatar-sm bg-green-lt"><i class="ti ti-circle-check text-green"></i></span>
                  <div>
                    <div class="text-muted small">Selesai</div>
                    <div class="fw-medium">{{ formatDate(session.finished_at) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <div class="d-flex flex-column gap-2">
            <button
              v-if="canShowReport"
              class="btn btn-outline-primary w-100"
              @click="router.push(`/peserta/report/${sessionId}`)"
            >
              <i class="ti ti-file-analytics me-1"></i>Lihat Pembahasan
            </button>
            <button class="btn btn-primary w-100" @click="router.push('/peserta')">
              <i class="ti ti-arrow-left me-1"></i>Kembali ke Dashboard
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@media (max-width: 767px) {
  .row.g-3.text-start .col-6 {
    flex: 0 0 100%;
    max-width: 100%;
  }
  .d-flex.flex-column.gap-2 .btn {
    min-height: 48px;
    font-size: 0.95rem;
  }
  .card-body.text-center {
    padding: 1.5rem 1rem !important;
  }
  .card-header .d-flex {
    flex-wrap: wrap;
  }
  .badge.fs-6 {
    font-size: 1rem !important;
  }
}

@media (min-width: 768px) and (max-width: 1024px) {
  .d-flex.flex-column.gap-2 .btn {
    min-height: 44px;
  }
}
</style>
