<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { examApi, type ExamSchedule } from '../../../api/exam.api'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const nextSchedule = ref<ExamSchedule | null>(null)
const loading = ref(true)
const starting = ref(false)
const error = ref('')
const confirmed = ref(false)
const waitSeconds = ref(0)
const timerDone = ref(false)
let countdown: ReturnType<typeof setInterval> | null = null

const timerFormatted = computed(() => {
  const m = Math.floor(waitSeconds.value / 60)
  const s = waitSeconds.value % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

const canStart = computed(() => timerDone.value && confirmed.value)

const totalWaitSeconds = ref(0) // for progress calculation
const progressPercent = computed(() => {
  if (totalWaitSeconds.value <= 0) return 100
  return Math.min(100, ((totalWaitSeconds.value - waitSeconds.value) / totalWaitSeconds.value) * 100)
})

async function load() {
  try {
    const res = await examApi.getTransition(sessionId)
    nextSchedule.value = res.data.data
    // Use break_duration_seconds from API response, default to 0
    const breakDuration = (res.data.data as any)?.break_duration_seconds ?? 0
    if (breakDuration > 0) {
      waitSeconds.value = breakDuration
      totalWaitSeconds.value = breakDuration
      timerDone.value = false
      startCountdown()
    } else {
      waitSeconds.value = 0
      totalWaitSeconds.value = 0
      timerDone.value = true
    }
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Tidak ada bagian berikutnya.'
  } finally {
    loading.value = false
  }
}

function startCountdown() {
  if (countdown) clearInterval(countdown)
  countdown = setInterval(() => {
    if (waitSeconds.value <= 1) {
      waitSeconds.value = 0
      timerDone.value = true
      if (countdown) {
        clearInterval(countdown)
        countdown = null
      }
    } else {
      waitSeconds.value--
    }
  }, 1000)
}

async function startSection() {
  if (!canStart.value) return
  starting.value = true
  try {
    const res = await examApi.startSection(sessionId)
    const newSessionId = res.data.data.session.id
    router.replace(`/peserta/exam/${newSessionId}`)
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memulai bagian berikutnya.'
    starting.value = false
  }
}

onMounted(load)
onUnmounted(() => { if (countdown) clearInterval(countdown) })
</script>

<template>
  <div class="d-flex justify-content-center align-items-center" style="min-height: calc(100vh - 8rem)">
    <div class="col-lg-5 col-md-7 col-sm-10">

      <div v-if="loading" class="card">
        <div class="card-body text-center py-5 text-muted">
          <span class="spinner-border me-2"></span>
          Memuat informasi bagian berikutnya...
        </div>
      </div>

      <div v-else-if="error" class="card border-danger">
        <div class="card-body text-center py-4">
          <i class="ti ti-alert-circle fs-4 text-danger mb-3 d-block"></i>
          <h3 class="text-danger">{{ error }}</h3>
          <div class="d-flex gap-2 justify-content-center mt-3">
            <button class="btn btn-outline-secondary" @click="error = ''; loading = true; load()">
              <i class="ti ti-refresh me-1"></i>Coba Lagi
            </button>
            <button class="btn btn-outline-danger" @click="router.replace(`/peserta/result/${sessionId}`)">
              <i class="ti ti-trophy me-1"></i>Lihat Hasil
            </button>
          </div>
        </div>
      </div>

      <div v-else class="card">
        <div class="card-body text-center py-4">
          <!-- Section complete indicator -->
          <span class="avatar avatar-xl bg-green-lt rounded-circle mb-3">
            <i class="ti ti-check text-green fs-2"></i>
          </span>
          <h2 class="fw-bold mb-1">Bagian Selesai!</h2>
          <p class="text-muted mb-3">Anda telah menyelesaikan bagian sebelumnya.</p>

          <hr />

          <!-- Next section info -->
          <div class="mb-3 mt-3">
            <p class="text-muted small mb-1">Bagian Selanjutnya:</p>
            <h3 class="fw-bold">{{ nextSchedule?.name ?? 'Bagian Berikutnya' }}</h3>
            <p v-if="nextSchedule?.duration_minutes" class="text-muted">
              Durasi: <strong>{{ nextSchedule.duration_minutes }} menit</strong>
            </p>
          </div>

          <!-- Countdown (if wait time > 0) -->
          <div v-if="!timerDone" class="alert alert-info mb-3">
            <div class="text-muted small mb-2">Waktu istirahat / persiapan:</div>
            <!-- Visual countdown circle -->
            <div class="d-flex justify-content-center mb-2">
              <div class="position-relative" style="width:100px;height:100px">
                <svg viewBox="0 0 100 100" width="100" height="100">
                  <circle cx="50" cy="50" r="44" fill="none" stroke="var(--tblr-border-color)" stroke-width="8" />
                  <circle
                    cx="50" cy="50" r="44" fill="none"
                    stroke="var(--tblr-info)"
                    stroke-width="8"
                    stroke-linecap="round"
                    :stroke-dasharray="`${progressPercent * 2.764} 276.4`"
                    transform="rotate(-90 50 50)"
                    style="transition: stroke-dasharray 1s linear"
                  />
                </svg>
                <div class="position-absolute top-50 start-50 translate-middle text-center">
                  <div class="h3 mb-0 fw-bold">{{ timerFormatted }}</div>
                </div>
              </div>
            </div>
            <!-- Progress bar -->
            <div class="progress mb-2" style="height:4px">
              <div class="progress-bar bg-info" :style="{ width: progressPercent + '%', transition: 'width 1s linear' }"></div>
            </div>
            <div class="text-muted small">Tombol akan aktif setelah waktu habis.</div>
          </div>

          <!-- Confirmation checkbox -->
          <div v-if="timerDone" class="mb-3">
            <label class="form-check">
              <input type="checkbox" v-model="confirmed" class="form-check-input" />
              <span class="form-check-label">
                Saya siap melanjutkan dan memahami bahwa <strong>tidak dapat kembali</strong> ke bagian sebelumnya.
              </span>
            </label>
          </div>

          <button
            class="btn btn-primary btn-lg w-100"
            :disabled="!canStart || starting"
            @click="startSection"
          >
            <template v-if="starting">
              <span class="spinner-border spinner-border-sm me-2"></span>Memulai...
            </template>
            <template v-else>
              Mulai Bagian Ini <i class="ti ti-arrow-right ms-1"></i>
            </template>
          </button>
        </div>
      </div>

    </div>
  </div>
</template>
