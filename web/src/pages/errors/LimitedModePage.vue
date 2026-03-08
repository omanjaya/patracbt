<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth.store'
import { useBrandingStore } from '../../stores/branding.store'

const router = useRouter()
const authStore = useAuthStore()
const branding = useBrandingStore()

async function handleLogout() {
  await authStore.logout()
  router.push('/login')
}

function goHome() {
  if (authStore.isAuthenticated && authStore.user) {
    const map: Record<string, string> = { admin: '/admin', guru: '/guru', pengawas: '/pengawas', peserta: '/peserta' }
    router.push(map[authStore.user.role] ?? '/login')
  } else {
    router.push('/login')
  }
}

onMounted(() => {
  document.title = 'Mode Terbatas - ' + branding.appName
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: maintenance/limited -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Warning triangle -->
            <path d="M80 36L36 116h88L80 36z" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" stroke-linejoin="round" />
            <!-- Inner triangle highlight -->
            <path d="M80 52L50 108h60L80 52z" fill="white" />
            <!-- Exclamation mark -->
            <rect x="77" y="68" width="6" height="24" rx="3" fill="#f76707" />
            <circle cx="80" cy="100" r="4" fill="#f76707" />
            <!-- Gear icon (maintenance) -->
            <circle cx="118" cy="44" r="16" fill="#f76707" opacity="0.9" />
            <path d="M118 36v2m0 12v2m-6.93-14.93l1.41 1.41m9.03 9.03l1.41 1.41m-14.85 0l1.41-1.41m9.03-9.03l1.41-1.41M110 44h2m12 0h2" stroke="white" stroke-width="1.5" stroke-linecap="round" />
            <circle cx="118" cy="44" r="4" fill="none" stroke="white" stroke-width="2" />
          </svg>
        </div>

        <p class="empty-title">Sistem Sedang Dalam Mode Terbatas</p>
        <p class="empty-subtitle text-muted">
          Mohon maaf, sistem sedang dalam mode terbatas (maintenance).
          Beberapa fitur mungkin tidak tersedia untuk sementara waktu.
          Silakan hubungi administrator untuk informasi lebih lanjut.
        </p>

        <!-- Contact info -->
        <div class="card card-body text-start mb-4" style="max-width: 380px; margin: 0 auto;">
          <p class="fw-bold text-dark mb-2">
            <i class="ti ti-info-circle me-1"></i>
            Informasi:
          </p>
          <ul class="mb-0 ps-3 text-muted list-unstyled">
            <li class="mb-1">
              <i class="ti ti-shield-lock me-1 text-warning"></i>
              Sistem diaktifkan mode darurat oleh administrator
            </li>
            <li class="mb-1">
              <i class="ti ti-clock me-1 text-warning"></i>
              Akses akan dipulihkan setelah mode dicabut
            </li>
            <li>
              <i class="ti ti-user me-1 text-warning"></i>
              Hubungi admin jika Anda memerlukan akses segera
            </li>
          </ul>
        </div>

        <div class="empty-action d-flex gap-2 justify-content-center flex-wrap">
          <button class="btn btn-primary" @click="goHome">
            <i class="ti ti-home me-2"></i>
            Coba Kembali
          </button>
          <button class="btn btn-outline-danger" @click="handleLogout">
            <i class="ti ti-logout me-2"></i>
            Logout
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
