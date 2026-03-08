<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { examApi } from '../../../api/exam.api'
import { useWebSocket } from '../../../composables/useWebSocket'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const examName = ref('Ujian')
const initialLoading = ref(true)
const wsConnected = ref(false)
const wsFailedToConnect = ref(false)
const statusText = ref('Menghubungkan kembali...')
const showReloadBtn = ref(false)
const redirecting = ref(false)
const reconnecting = ref(true)

let ws: ReturnType<typeof useWebSocket> | null = null
let wsTimeoutId: ReturnType<typeof setTimeout> | null = null

async function checkAndRedirect() {
  if (redirecting.value) return
  try {
    const res = await examApi.loadSession(sessionId)
    const data = res.data.data
    if (data?.session?.status === 'ongoing') {
      redirecting.value = true
      router.replace(`/peserta/exam/${sessionId}`)
    }
  } catch {
    // ignore
  }
}

function showManualReload() {
  wsFailedToConnect.value = true
  showReloadBtn.value = true
  reconnecting.value = false
  statusText.value = 'Koneksi real-time tidak tersedia. Setelah pengawas membuka kunci, tekan tombol di bawah untuk memuat ulang.'
}

function initWS(scheduleId: number) {
  ws = useWebSocket(scheduleId, sessionId)

  ws.on('time_sync', () => {
    wsConnected.value = true
    reconnecting.value = false
    statusText.value = 'Halaman ini akan otomatis mengalihkan Anda ke ujian begitu pengawas membuka kunci. Jangan tutup halaman ini.'
    if (wsTimeoutId) {
      clearTimeout(wsTimeoutId)
      wsTimeoutId = null
    }
  })

  // unlock_client — server sends this when peserta is unlocked
  ws.on('lock_client', () => {
    // If we receive a lock_client with status ongoing it means unlock happened
    // Also check status via API
    checkAndRedirect()
  })

  // When server broadcasts that session is back to ongoing
  ws.on('force_finish', () => {
    checkAndRedirect()
  })

  ws.connect()

  // If not connected after 3 seconds, show manual reload
  wsTimeoutId = setTimeout(() => {
    if (!wsConnected.value) {
      showManualReload()
    }
  }, 3000)
}

async function loadSessionInfo() {
  try {
    const res = await examApi.loadSession(sessionId)
    const data = res.data.data
    if (data?.session) {
      examName.value = data.session.exam_schedule?.name ?? 'Ujian'

      // If session is already ongoing, redirect immediately
      if (data.session.status === 'ongoing') {
        redirecting.value = true
        router.replace(`/peserta/exam/${sessionId}`)
        return
      }

      // Connect WebSocket
      const scheduleId = data.session.exam_schedule_id
      if (scheduleId) {
        initWS(scheduleId)
      } else {
        showManualReload()
      }
    }
  } catch {
    showManualReload()
  } finally {
    initialLoading.value = false
  }
}

function manualReload() {
  window.location.reload()
}

onMounted(() => {
  loadSessionInfo()
})

onUnmounted(() => {
  if (wsTimeoutId) clearTimeout(wsTimeoutId)
  ws?.disconnect()
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">

      <!-- Initial Loading -->
      <div v-if="initialLoading" class="card card-md shadow-lg border-0 rounded-3">
        <div class="card-body text-center p-5">
          <span class="spinner-border text-yellow mb-3" style="width:3rem;height:3rem"></span>
          <p class="text-muted mb-0">Memeriksa status ujian...</p>
        </div>
      </div>

      <div v-else class="card card-md shadow-lg border-0 rounded-3">
        <div class="card-body text-center p-5">

          <!-- Lock Icon -->
          <div class="mb-4 text-yellow">
            <i class="ti ti-lock" style="font-size: 96px; line-height: 1;"></i>
          </div>

          <h1 class="mb-2">Ujian Dikunci</h1>
          <p class="text-muted mb-1">{{ examName }}</p>
          <p class="text-muted mb-4">Menunggu pengawas membuka kunci...</p>

          <!-- Reconnecting message (shown immediately) -->
          <div class="mb-3" v-if="reconnecting && !wsConnected && !showReloadBtn">
            <span class="badge bg-yellow-lt text-yellow">
              <span class="spinner-border spinner-border-sm me-1" style="width:12px;height:12px;border-width:2px"></span>
              Menghubungkan kembali...
            </span>
          </div>

          <!-- Pulse animation (shown while waiting for WS, after connecting) -->
          <div class="mb-4" v-if="!showReloadBtn && !reconnecting">
            <span class="spinner-grow spinner-grow-sm text-yellow me-1"></span>
            <span class="spinner-grow spinner-grow-sm text-yellow me-1" style="animation-delay: 0.15s;"></span>
            <span class="spinner-grow spinner-grow-sm text-yellow" style="animation-delay: 0.3s;"></span>
          </div>

          <!-- WS Connected badge -->
          <div class="mb-3" v-if="wsConnected && !showReloadBtn">
            <span class="badge bg-success-lt text-success">
              <i class="ti ti-wifi me-1"></i>Terhubung real-time
            </span>
          </div>

          <p class="text-muted small mb-4">{{ statusText }}</p>

          <!-- Manual reload button (shown if WS fails) -->
          <button
            v-if="showReloadBtn"
            class="btn btn-primary mb-3"
            @click="manualReload"
          >
            <i class="ti ti-refresh me-2"></i>
            Muat Ulang
          </button>

          <div>
            <router-link to="/peserta" class="btn btn-outline-secondary">
              <i class="ti ti-arrow-left me-2"></i>
              Kembali ke Dashboard
            </router-link>
          </div>

        </div>
      </div>
    </div>
  </div>
</template>
