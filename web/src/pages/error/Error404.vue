<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth.store'
import { useBrandingStore } from '../../stores/branding.store'

const router = useRouter()
const authStore = useAuthStore()
const branding = useBrandingStore()

function goHome() {
  if (authStore.isAuthenticated && authStore.user) {
    const map: Record<string, string> = { admin: '/admin', guru: '/guru', pengawas: '/pengawas', peserta: '/peserta' }
    router.push(map[authStore.user.role] ?? '/login')
  } else {
    router.push('/login')
  }
}

function goBack() {
  router.back()
}

onMounted(() => {
  document.title = 'Halaman Tidak Ditemukan - ' + branding.appName
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: dokumen hilang -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Document body -->
            <rect x="48" y="30" width="64" height="84" rx="6" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" />
            <!-- Document fold -->
            <path d="M92 30v20h20" fill="#cbd5e1" />
            <path d="M92 30l20 20" stroke="#94a3b8" stroke-width="2" stroke-linecap="round" />
            <!-- Text lines -->
            <rect x="58" y="60" width="36" height="4" rx="2" fill="#94a3b8" />
            <rect x="58" y="70" width="28" height="4" rx="2" fill="#94a3b8" />
            <rect x="58" y="80" width="32" height="4" rx="2" fill="#94a3b8" />
            <rect x="58" y="90" width="20" height="4" rx="2" fill="#94a3b8" />
            <!-- Question mark -->
            <circle cx="110" cy="110" r="22" fill="#206bc4" opacity="0.9" />
            <text x="110" y="118" text-anchor="middle" font-size="28" font-weight="bold" fill="white" font-family="sans-serif">?</text>
          </svg>
        </div>

        <div class="empty-header">404</div>
        <p class="empty-title">Halaman Tidak Ditemukan</p>
        <p class="empty-subtitle text-muted">
          Maaf, halaman yang Anda cari tidak ditemukan atau sudah dipindahkan.
          Silakan periksa kembali URL atau kembali ke halaman utama.
        </p>
        <div class="empty-action d-flex gap-2 justify-content-center">
          <button class="btn btn-outline-secondary" @click="goBack">
            <i class="ti ti-arrow-left me-2"></i>
            Kembali
          </button>
          <button class="btn btn-primary" @click="goHome">
            <i class="ti ti-home me-2"></i>
            Kembali ke Beranda
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
