<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../../stores/auth.store'
import { useBrandingStore } from '../../stores/branding.store'
import { useToastStore } from '../../stores/toast.store'

const authStore = useAuthStore()
const branding = useBrandingStore()
const toast = useToastStore()
const route = useRoute()

// ── User initials ──
const userInitials = computed(() => {
  const name = authStore.user?.name ?? ''
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return ((parts[0]?.[0] ?? '') + (parts[1]?.[0] ?? '')).toUpperCase()
  return name.substring(0, 2).toUpperCase()
})

// ── User dropdown ──
const showDropdown = ref(false)

function toggleDropdown() {
  showDropdown.value = !showDropdown.value
}

function closeDropdown() {
  showDropdown.value = false
}

function handleClickOutside(e: MouseEvent) {
  const dropdown = document.getElementById('peserta-user-dropdown')
  if (dropdown && !dropdown.contains(e.target as Node)) {
    showDropdown.value = false
  }
}

// ── Preview Mode ──
const isPreviewMode = computed(() => {
  return (authStore.user as any)?.is_preview === true || localStorage.getItem('preview_mode') === 'true'
})

function exitPreview() {
  if (typeof (authStore as any).exitPreview === 'function') {
    (authStore as any).exitPreview()
  } else {
    localStorage.removeItem('preview_mode')
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    window.location.href = '/admin'
  }
}

// ── Idle Timer ──
const IDLE_WARNING_MS = 5 * 60 * 1000   // 5 minutes
const IDLE_LOGOUT_MS = 6 * 60 * 1000    // 6 minutes
let lastActivity = Date.now()
let idleInterval: ReturnType<typeof setInterval> | null = null
let warningSent = false

const isInExam = computed(() => {
  return route.path.startsWith('/peserta/exam/')
})

function resetActivity() {
  lastActivity = Date.now()
  if (warningSent) {
    warningSent = false
  }
}

const activityEvents = ['mousemove', 'keydown', 'touchstart', 'click', 'scroll'] as const

function startIdleTimer() {
  activityEvents.forEach((evt) => {
    document.addEventListener(evt, resetActivity, { passive: true })
  })

  idleInterval = setInterval(() => {
    // Don't run idle timer during exam
    if (isInExam.value) {
      lastActivity = Date.now()
      warningSent = false
      return
    }

    const elapsed = Date.now() - lastActivity

    if (elapsed >= IDLE_LOGOUT_MS) {
      stopIdleTimer()
      toast.warning('Sesi Anda telah berakhir karena tidak ada aktivitas.', 5000)
      setTimeout(() => authStore.logout(), 500)
      return
    }

    if (elapsed >= IDLE_WARNING_MS && !warningSent) {
      warningSent = true
      toast.warning('Sesi Anda akan berakhir dalam 60 detik karena tidak ada aktivitas.', 15000)
    }
  }, 5000) // check every 5 seconds
}

function stopIdleTimer() {
  if (idleInterval) {
    clearInterval(idleInterval)
    idleInterval = null
  }
  activityEvents.forEach((evt) => {
    document.removeEventListener(evt, resetActivity)
  })
}

// ── Lifecycle ──
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  startIdleTimer()
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  stopIdleTimer()
})
</script>

<template>
  <div class="page">
    <!-- Preview Mode Banner -->
    <div v-if="isPreviewMode" class="preview-banner">
      <div class="container-xl d-flex align-items-center justify-content-between py-2">
        <div class="d-flex align-items-center gap-2">
          <i class="ti ti-eye fs-4"></i>
          <span class="fw-medium">Anda sedang dalam mode preview sebagai peserta.</span>
        </div>
        <button class="btn btn-sm btn-warning" @click="exitPreview">
          <i class="ti ti-arrow-back-up me-1"></i>
          Kembali ke Admin
        </button>
      </div>
    </div>

    <!-- Top navbar -->
    <header class="navbar navbar-expand-md d-print-none">
      <div class="container-xl">
        <router-link to="/peserta" class="navbar-brand d-flex align-items-center gap-2">
          <img v-if="branding.settings.app_logo" :src="branding.settings.app_logo" class="navbar-brand-image" alt="" style="height: 32px;">
          <div v-else class="avatar avatar-sm bg-primary">
            <i class="ti ti-book text-white" style="font-size:1rem"></i>
          </div>
          <span class="fw-bold">{{ branding.appName }}</span>
        </router-link>

        <nav class="navbar-nav ms-auto d-flex flex-row align-items-center gap-3" aria-label="Menu pengguna peserta">
          <router-link to="/peserta/dashboard" class="btn btn-ghost-primary btn-sm d-flex align-items-center gap-1" aria-label="Beranda peserta">
            <i class="ti ti-home"></i>
            <span class="d-none d-sm-inline">Beranda</span>
          </router-link>

          <!-- User Dropdown -->
          <div id="peserta-user-dropdown" class="nav-item dropdown" :class="{ show: showDropdown }">
            <a
              class="nav-link d-flex align-items-center gap-2 lh-1 text-reset p-0"
              href="#"
              role="button"
              @click.prevent="toggleDropdown"
              aria-haspopup="true"
              :aria-expanded="showDropdown"
            >
              <span class="avatar avatar-sm rounded-circle bg-primary-lt">
                <span class="fw-bold small">{{ userInitials }}</span>
              </span>
              <div class="d-none d-sm-block ps-1">
                <div class="small fw-medium lh-sm">{{ authStore.user?.name }}</div>
              </div>
            </a>
            <div class="dropdown-menu dropdown-menu-end" :class="{ show: showDropdown }">
              <!-- Mobile: show name -->
              <div class="d-sm-none dropdown-header">
                <span class="fw-medium">{{ authStore.user?.name }}</span>
              </div>
              <router-link to="/peserta/profil" class="dropdown-item" @click="closeDropdown">
                <i class="ti ti-user me-2"></i>
                Profil
              </router-link>
              <div class="dropdown-divider"></div>
              <a
                class="dropdown-item text-danger"
                href="#"
                @click.prevent="authStore.logout"
              >
                <i class="ti ti-logout me-2"></i>
                Keluar
              </a>
            </div>
          </div>
        </nav>
      </div>
    </header>

    <!-- Content -->
    <div class="page-wrapper">
      <div class="page-body">
        <div class="container-xl">
          <router-view />
        </div>
      </div>
      <footer class="footer footer-transparent d-print-none">
        <div class="container-xl">
          <div class="row text-center align-items-center">
            <div class="col-12">
              <span class="text-muted small">Copyright &copy; {{ new Date().getFullYear() }} {{ branding.footerText }}</span>
            </div>
          </div>
        </div>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.preview-banner {
  background-color: #f59f00;
  color: #1a1a2e;
  font-size: 0.875rem;
  position: relative;
  z-index: 1050;
}

.preview-banner .btn-warning {
  background-color: #1a1a2e;
  border-color: #1a1a2e;
  color: #f59f00;
  font-weight: 600;
}

.preview-banner .btn-warning:hover {
  background-color: #2d2d4a;
  border-color: #2d2d4a;
}

.dropdown-menu.show {
  display: block;
}

.nav-link .avatar {
  cursor: pointer;
}
</style>
