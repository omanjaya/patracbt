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
  document.title = 'Akses Ditolak - ' + branding.appName
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: shield/lock -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Shield -->
            <path d="M80 28L48 44v32c0 28 13.6 43.2 32 52 18.4-8.8 32-24 32-52V44L80 28z" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" stroke-linejoin="round" />
            <!-- Lock body -->
            <rect x="68" y="72" width="24" height="20" rx="4" fill="#206bc4" opacity="0.9" />
            <!-- Lock shackle -->
            <path d="M72 72V64a8 8 0 0116 0v8" fill="none" stroke="#206bc4" stroke-width="3" stroke-linecap="round" opacity="0.9" />
            <!-- Keyhole -->
            <circle cx="80" cy="80" r="3" fill="white" />
            <rect x="79" y="82" width="2" height="5" rx="1" fill="white" />
          </svg>
        </div>

        <div class="empty-header">403</div>
        <p class="empty-title">Akses Ditolak</p>
        <p class="empty-subtitle text-muted">
          Maaf, Anda tidak memiliki izin untuk mengakses halaman ini.
          Peran akun Anda (<strong>{{ authStore.user?.role ?? 'tidak diketahui' }}</strong>) tidak memiliki hak akses ke halaman yang diminta.
          Hubungi administrator jika Anda memerlukan akses.
        </p>
        <div class="empty-action d-flex gap-2 justify-content-center flex-wrap">
          <button class="btn btn-outline-secondary" @click="goBack">
            <i class="ti ti-arrow-left me-2"></i>
            Kembali
          </button>
          <button class="btn btn-primary" @click="goHome">
            <i class="ti ti-home me-2"></i>
            Kembali ke Dashboard
          </button>
          <button class="btn btn-outline-danger" @click="router.push('/login')">
            <i class="ti ti-login me-2"></i>
            Login Ulang
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
