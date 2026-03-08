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

function reload() {
  window.location.reload()
}

onMounted(() => {
  document.title = 'Kesalahan Server - ' + branding.appName
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: server error -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Server box top -->
            <rect x="40" y="38" width="80" height="28" rx="4" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" />
            <circle cx="56" cy="52" r="4" fill="#94a3b8" />
            <rect x="66" y="50" width="20" height="4" rx="2" fill="#94a3b8" />
            <circle cx="104" cy="52" r="4" fill="#22c55e" />
            <!-- Server box bottom -->
            <rect x="40" y="72" width="80" height="28" rx="4" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" />
            <circle cx="56" cy="86" r="4" fill="#94a3b8" />
            <rect x="66" y="84" width="20" height="4" rx="2" fill="#94a3b8" />
            <circle cx="104" cy="86" r="4" fill="#ef4444" />
            <!-- Warning triangle -->
            <path d="M80 108l18 30H62l18-30z" fill="#ef4444" opacity="0.9" />
            <text x="80" y="133" text-anchor="middle" font-size="20" font-weight="bold" fill="white" font-family="sans-serif">!</text>
          </svg>
        </div>

        <div class="empty-header">500</div>
        <p class="empty-title">Terjadi Kesalahan Server</p>
        <p class="empty-subtitle text-muted">
          Maaf, terjadi kesalahan pada server. Silakan coba lagi nanti.
        </p>
        <div class="empty-action d-flex gap-2 justify-content-center">
          <button class="btn btn-outline-secondary" @click="reload">
            <i class="ti ti-refresh me-2"></i>
            Coba Lagi
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
