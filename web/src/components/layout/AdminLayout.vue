<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth.store'
import { useBrandingStore } from '../../stores/branding.store'
import { getAvatarUrl } from '../../utils/avatar'

const authStore = useAuthStore()
const branding = useBrandingStore()
const route = useRoute()
const router = useRouter()

function goToProfile() {
  const role = authStore.user?.role
  const path = role === 'guru' ? '/guru/profile'
    : role === 'pengawas' ? '/pengawas/profile'
    : '/admin/profile'
  router.push(path)
}

interface MenuItem {
  label: string
  icon: string
  path?: string
  children?: { label: string; path: string; divider?: boolean }[]
  key?: string
}

const adminMenu: MenuItem[] = [
  { label: 'Dashboard', icon: 'ti-home', path: '/admin' },
  {
    label: 'Master Data', icon: 'ti-database', key: 'master',
    children: [
      { label: 'Rombel', path: '/admin/rombels' },
      { label: 'Ruangan', path: '/admin/rooms' },
      { label: 'Tag', path: '/admin/tags' },
      { label: 'Mata Pelajaran', path: '/admin/subjects' },
      { label: 'Hak dan Izin', path: '/admin/roles' },
    ],
  },
  {
    label: 'Manajemen User', icon: 'ti-users', key: 'users',
    children: [
      { label: 'Data User', path: '/admin/users' },
      { label: 'Pengaitan Rombel', path: '/admin/rombel-management' },
      { label: 'Pengaitan Ruangan', path: '/admin/room-management' },
      { label: 'Pengaitan Grup (Tag)', path: '/admin/user-tags' },
    ],
  },
  {
    label: 'Manajemen Ujian', icon: 'ti-book', key: 'exam',
    children: [
      { label: 'Bank Soal', path: '/admin/question-banks' },
      { label: 'Penjadwalan Ujian', path: '/admin/exam-schedules' },
      { label: 'Cetak Kartu Peserta', path: '/admin/print-cards' },
      { label: 'Ruang Pengawasan', path: '/admin/supervision' },
      { label: 'Laporan & Analisis', path: '/admin/reports', divider: true },
      { label: 'Live Score (TV Mode)', path: '/admin/live-score' },
    ],
  },
  { label: 'Pengaturan', icon: 'ti-settings', path: '/admin/settings' },
]

const guruMenu: MenuItem[] = [
  { label: 'Dashboard', icon: 'ti-home', path: '/guru' },
  {
    label: 'Manajemen Ujian', icon: 'ti-book', key: 'exam',
    children: [
      { label: 'Bank Soal', path: '/guru/question-banks' },
      { label: 'Penjadwalan Ujian', path: '/guru/exam-schedules' },
      { label: 'Ruang Pengawasan', path: '/guru/supervision' },
      { label: 'Laporan & Analisis', path: '/guru/reports' },
      { label: 'Riwayat Ujian', path: '/guru/exam-history' },
      { label: 'Live Score (TV Mode)', path: '/guru/live-score' },
    ],
  },
]

const pengawasMenu: MenuItem[] = [
  { label: 'Dashboard', icon: 'ti-home', path: '/pengawas' },
  { label: 'Hub Pengawasan', icon: 'ti-shield-check', path: '/pengawas/supervision' },
  { label: 'Log Pelanggaran', icon: 'ti-alert-octagon', path: '/pengawas/violations' },
  { label: 'Laporan & Analisis', icon: 'ti-chart-bar', path: '/pengawas/reports' },
  { label: 'Live Score (TV Mode)', icon: 'ti-device-tv', path: '/pengawas/live-score' },
]

const menuItems = computed<MenuItem[]>(() => {
  const role = authStore.user?.role
  if (role === 'guru') return guruMenu
  if (role === 'pengawas') return pengawasMenu
  return adminMenu
})

const mobileMenuOpen = ref(false)

const dashboardPath = computed(() => {
  const role = authStore.user?.role
  if (role === 'guru') return '/guru'
  if (role === 'pengawas') return '/pengawas'
  return '/admin'
})

const ROOT_PATHS = ['/admin', '/guru', '/pengawas']

function isActive(path: string) {
  if (ROOT_PATHS.includes(path)) return route.path === path
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: MenuItem) {
  return item.children?.some(c => isActive(c.path!)) ?? false
}

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

watch(() => route.path, () => {
  // Trigger global click to let Bootstrap close any open dropdowns
  document.body.click()
  // Close mobile navbar on navigation
  mobileMenuOpen.value = false
})
</script>

<template>
  <div class="page">

    <!-- TOP HEADER -->
    <header class="navbar navbar-expand-md d-print-none sticky-top" :style="branding.settings.app_header_bg ? { background: branding.settings.app_header_bg } : {}">
      <div class="container-xl">
        <button
          class="navbar-toggler"
          type="button"
          :aria-expanded="mobileMenuOpen"
          aria-controls="navbar-menu"
          aria-label="Buka/tutup menu navigasi"
          @click="toggleMobileMenu"
        >
          <span class="navbar-toggler-icon"></span>
        </button>

        <h1 class="navbar-brand navbar-brand-autodark d-none-navbar-horizontal pe-0 pe-md-3">
          <router-link :to="dashboardPath" class="d-flex align-items-center gap-2">
            <img v-if="branding.settings.app_logo" :src="branding.settings.app_logo" class="navbar-brand-image" alt="" style="height: 32px;">
            <div v-else class="avatar avatar-sm bg-primary">
              <i class="ti ti-book text-white"></i>
            </div>
            <span class="fw-bold">{{ branding.appName }}</span>
          </router-link>
        </h1>

        <div class="navbar-nav flex-row order-md-last">
          <div class="nav-item dropdown">
            <a href="#" class="nav-link d-flex lh-1 text-reset p-0" data-bs-toggle="dropdown" aria-label="Buka menu pengguna" aria-expanded="false" role="button">
              <span
                class="avatar avatar-sm"
                :style="authStore.user?.id ? `background-image:url(${getAvatarUrl(authStore.user.id)})` : ''"
              >
                <span v-if="!authStore.user?.id">{{ authStore.user?.name?.charAt(0).toUpperCase() }}</span>
              </span>
              <div class="d-none d-lg-block ps-2">
                <div>{{ authStore.user?.name }}</div>
                <div class="mt-1 small text-secondary">{{ authStore.user?.role }}</div>
              </div>
            </a>
            <div class="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
              <a href="#" class="dropdown-item" @click.prevent="goToProfile">Profil</a>
              <div class="dropdown-divider"></div>
              <a href="#" class="dropdown-item" @click.prevent="authStore.logout">Keluar</a>
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- NAVIGATION MENU -->
    <nav class="navbar-expand-md" aria-label="Menu utama">
      <div class="collapse navbar-collapse" :class="{ show: mobileMenuOpen }" id="navbar-menu" style="transition: height 0.3s ease, opacity 0.3s ease;">
        <div class="navbar">
          <div class="container-xl">
            <ul class="navbar-nav">
              <template v-for="item in menuItems" :key="item.path ?? item.key">

                <!-- Single menu item -->
                <li v-if="item.path" class="nav-item" :class="{ active: isActive(item.path) }">
                  <router-link class="nav-link" :to="item.path" :aria-label="item.label">
                    <span class="nav-link-icon d-md-none d-lg-inline-block">
                      <i :class="`ti ${item.icon}`"></i>
                    </span>
                    <span class="nav-link-title">{{ item.label }}</span>
                  </router-link>
                </li>

                <!-- Dropdown menu item -->
                <li v-else class="nav-item dropdown" :class="{ active: isGroupActive(item) }">
                  <a class="nav-link dropdown-toggle" href="#" data-bs-toggle="dropdown"
                    data-bs-auto-close="outside" role="button" aria-expanded="false"
                    :aria-label="`Menu ${item.label}`">
                    <span class="nav-link-icon d-md-none d-lg-inline-block">
                      <i :class="`ti ${item.icon}`"></i>
                    </span>
                    <span class="nav-link-title">{{ item.label }}</span>
                  </a>
                  <div class="dropdown-menu">
                    <template v-for="child in item.children" :key="child.path">
                      <div v-if="child.divider" class="dropdown-divider"></div>
                      <router-link
                        class="dropdown-item"
                        :class="{ active: isActive(child.path!) }"
                        :to="child.path!"
                      >
                        {{ child.label }}
                      </router-link>
                    </template>
                  </div>
                </li>

              </template>
            </ul>
          </div>
        </div>
      </div>
    </nav>

    <!-- CONTENT -->
    <div class="page-wrapper">
      <div class="page-body">
        <div class="container-xl">
          <router-view />
        </div>
      </div>
      <footer class="footer footer-transparent d-print-none">
        <div class="container-xl">
          <div class="row text-center align-items-center flex-row-reverse">
            <div class="col-lg-auto ms-lg-auto">
              <ul class="list-inline list-inline-dots mb-0">
                <li class="list-inline-item">{{ branding.footerText }}</li>
              </ul>
            </div>
            <div class="col-12 col-lg-auto mt-3 mt-lg-0">
              <ul class="list-inline list-inline-dots mb-0">
                <li class="list-inline-item">Copyright &copy; {{ new Date().getFullYear() }}</li>
              </ul>
            </div>
          </div>
        </div>
      </footer>
    </div>

  </div>
</template>
