<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import ToastContainer from './components/ui/ToastContainer.vue'
import { useBrandingStore } from '@/stores/branding.store'

const brandingStore = useBrandingStore()
brandingStore.fetch()

const isOffline = ref(!navigator.onLine)

// Route progress bar
const routeLoading = ref(false)
const progressWidth = ref(0)
let progressTimer: ReturnType<typeof setInterval> | null = null

const router = useRouter()

function startProgress() {
  routeLoading.value = true
  progressWidth.value = 30
  progressTimer = setInterval(() => {
    if (progressWidth.value < 90) {
      progressWidth.value += Math.random() * 10
    }
  }, 200)
}

function stopProgress() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
  progressWidth.value = 100
  setTimeout(() => {
    routeLoading.value = false
    progressWidth.value = 0
  }, 300)
}

router.beforeEach(() => {
  startProgress()
})

router.afterEach(() => {
  stopProgress()
})

function updateOnlineStatus() {
  isOffline.value = !navigator.onLine
}

onMounted(() => {
  window.addEventListener('online', updateOnlineStatus)
  window.addEventListener('offline', updateOnlineStatus)
})

onUnmounted(() => {
  window.removeEventListener('online', updateOnlineStatus)
  window.removeEventListener('offline', updateOnlineStatus)
})
</script>

<template>
  <div v-if="routeLoading" class="route-progress" :style="{ width: progressWidth + '%' }"></div>

  <div v-if="isOffline" class="page page-center" style="position: fixed; top: 0; left: 0; z-index: 9999; width: 100vw; height: 100vh; background-color: var(--tblr-body-bg);">
    <div class="container-tight py-4">
      <div class="empty">
        <div class="empty-header"><i class="ti ti-wifi-off text-danger"></i></div>
        <p class="empty-title">Koneksi Terputus</p>
        <p class="empty-subtitle text-muted">
          Periksa jaringan kabel atau WiFi Anda. Sistem mendeteksi bahwa komputer Anda kehilangan koneksi. 
        </p>
      </div>
    </div>
  </div>
  
  <router-view v-else />
  <ToastContainer />
</template>
