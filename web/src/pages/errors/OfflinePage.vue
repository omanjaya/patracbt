<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useBrandingStore } from '../../stores/branding.store'

const branding = useBrandingStore()
const isRetrying = ref(false)

async function retry() {
  isRetrying.value = true
  try {
    // Check if online by trying to fetch
    if (navigator.onLine) {
      window.location.reload()
    } else {
      setTimeout(() => {
        isRetrying.value = false
      }, 1500)
    }
  } catch {
    isRetrying.value = false
  }
}

function handleOnline() {
  window.location.reload()
}

onMounted(() => {
  document.title = 'Tidak Ada Koneksi - ' + branding.appName
  window.addEventListener('online', handleOnline)
})

onUnmounted(() => {
  window.removeEventListener('online', handleOnline)
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: wifi off -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Wifi arcs -->
            <path d="M44 68c10-10 22-16 36-16s26 6 36 16" fill="none" stroke="#cbd5e1" stroke-width="4" stroke-linecap="round" />
            <path d="M54 80c7.5-7.5 16-12 26-12s18.5 4.5 26 12" fill="none" stroke="#cbd5e1" stroke-width="4" stroke-linecap="round" />
            <path d="M64 92c4.5-4.5 10-8 16-8s11.5 3.5 16 8" fill="none" stroke="#cbd5e1" stroke-width="4" stroke-linecap="round" />
            <!-- Wifi dot -->
            <circle cx="80" cy="104" r="5" fill="#cbd5e1" />
            <!-- Slash line (red) -->
            <line x1="48" y1="120" x2="112" y2="56" stroke="#d63939" stroke-width="4" stroke-linecap="round" />
            <!-- Red circle indicator -->
            <circle cx="112" cy="44" r="16" fill="#d63939" opacity="0.9" />
            <text x="112" y="50" text-anchor="middle" font-size="20" font-weight="bold" fill="white" font-family="sans-serif">!</text>
          </svg>
        </div>

        <p class="empty-title">Tidak Ada Koneksi Internet</p>
        <p class="empty-subtitle text-muted">
          Halaman ini belum tersimpan di memori aplikasi.
          Silakan hubungkan kembali internet Anda untuk melanjutkan.
        </p>
        <div class="empty-action">
          <button
            class="btn btn-primary"
            :disabled="isRetrying"
            @click="retry"
          >
            <i class="ti ti-refresh me-2" :class="{ 'ti-loader animate-spin': isRetrying }"></i>
            {{ isRetrying ? 'Mencoba...' : 'Coba Segarkan Halaman' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.animate-spin {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
