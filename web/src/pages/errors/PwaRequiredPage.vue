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

onMounted(() => {
  document.title = 'Gunakan Aplikasi PWA - ' + branding.appName
})
</script>

<template>
  <div class="page page-center">
    <div class="container-tight py-4">
      <div class="empty">
        <!-- SVG Illustration: phone/app -->
        <div class="empty-img mb-3">
          <svg xmlns="http://www.w3.org/2000/svg" width="160" height="160" viewBox="0 0 160 160" fill="none">
            <!-- Background circle -->
            <circle cx="80" cy="80" r="72" fill="#f1f5f9" />
            <!-- Phone body -->
            <rect x="56" y="28" width="48" height="88" rx="8" fill="#e2e8f0" stroke="#94a3b8" stroke-width="2" />
            <!-- Phone screen -->
            <rect x="62" y="40" width="36" height="60" rx="2" fill="white" />
            <!-- App icon on screen -->
            <rect x="70" y="52" width="20" height="20" rx="4" fill="#206bc4" opacity="0.9" />
            <!-- Download arrow on app icon -->
            <path d="M80 57v10m-4-3l4 4 4-4" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            <!-- Phone home button -->
            <circle cx="80" cy="108" r="4" fill="none" stroke="#94a3b8" stroke-width="1.5" />
            <!-- Notification badge -->
            <circle cx="108" cy="40" r="14" fill="#206bc4" opacity="0.9" />
            <path d="M103 40l4 4 6-8" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </div>

        <p class="empty-title">Gunakan Aplikasi</p>
        <p class="empty-subtitle text-muted">
          Ujian ini wajib dikerjakan melalui aplikasi
          <strong>{{ branding.appName }}</strong>
          yang sudah diinstall di perangkat Anda.
        </p>

        <!-- Installation steps -->
        <div class="card card-body text-start mb-4" style="max-width: 380px; margin: 0 auto;">
          <p class="fw-bold text-dark mb-2">
            <i class="ti ti-list-numbers me-1"></i>
            Langkah Install Aplikasi:
          </p>
          <ol class="mb-0 ps-3 text-muted">
            <li class="mb-1">Buka website ini di browser <strong>Chrome</strong></li>
            <li class="mb-1">Ketuk ikon <strong>menu (tiga titik)</strong> di pojok kanan atas</li>
            <li class="mb-1">Pilih <strong>"Install Aplikasi"</strong> atau <strong>"Tambahkan ke Layar Utama"</strong></li>
            <li class="mb-1">Buka aplikasi yang sudah terinstall</li>
            <li>Login dan kerjakan ujian dari dalam aplikasi</li>
          </ol>
        </div>

        <div class="empty-action d-flex gap-2 justify-content-center flex-wrap">
          <button class="btn btn-outline-danger" @click="handleLogout">
            <i class="ti ti-logout me-2"></i>
            Logout &amp; Kembali
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
