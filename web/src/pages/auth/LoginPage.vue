<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth.store'

import { getErrorMessage } from '@/utils/apiError'
import { authApi } from '@/api/auth.api'
import BaseModal from '@/components/ui/BaseModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import { useBrandingStore } from '@/stores/branding.store'

const branding = useBrandingStore()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// Show session expired message if redirected from interceptor
const sessionExpired = ref(route.query.expired === '1')

function redirectByRole(role: string) {
  const routes: Record<string, string> = { admin: '/admin', guru: '/guru', pengawas: '/pengawas', peserta: '/peserta' }
  router.push(routes[role] ?? '/')
}
const form = ref({ login: '', password: '' })

// Validation
const errors = reactive<Record<string, string>>({})
const touched = reactive<Record<string, boolean>>({})

function validateField(field: string) {
  delete errors[field]
  if (field === 'login' && !form.value.login.trim()) {
    errors.login = 'Username wajib diisi'
  }
  if (field === 'password' && !form.value.password) {
    errors.password = 'Password wajib diisi'
  }
}

function onBlur(field: string) {
  touched[field] = true
  validateField(field)
}

function validate(): boolean {
  Object.keys(errors).forEach(k => delete errors[k])
  if (!form.value.login.trim()) errors.login = 'Username wajib diisi'
  if (!form.value.password) errors.password = 'Password wajib diisi'
  return Object.keys(errors).length === 0
}

// Clear error when user starts typing
watch(() => [form.value.login, form.value.password], () => {
  if (authStore.error) authStore.error = null
})

// Force login modal state
const showForceModal = ref(false)
const forceModalTitle = ref('')
const forceModalMessage = ref('')
const forceLoading = ref(false)
const forceSessionInfo = ref<{ device?: string; ip?: string; last_active?: string } | null>(null)

async function handleLogin() {
  if (!validate()) return
  authStore.isLoading = true
  authStore.error = null
  try {
    const res = await authApi.login({ login: form.value.login, password: form.value.password })
    if (res.data.success && res.data.data) {
      const { access_token, refresh_token, user } = res.data.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      authStore.user = user
      redirectByRole(user.role)
    }
  } catch (err: unknown) {
    const axiosErr = err as { response?: { status?: number; data?: { message?: string; code?: string; session?: { device?: string; ip?: string; last_active?: string } | null } } }
    const status = axiosErr.response?.status
    const code = axiosErr.response?.data?.code
    const message = axiosErr.response?.data?.message
    const sessionData = axiosErr.response?.data?.session ?? null

    if (status === 409 && code === 'SESSION_EXISTS') {
      forceModalTitle.value = 'Login Ganda Terdeteksi'
      forceModalMessage.value = 'Sesi aktif ditemukan di perangkat lain. Paksa login dan logout sesi sebelumnya?'
      forceSessionInfo.value = sessionData
      showForceModal.value = true
    } else if (status === 409 && code === 'EXAM_IN_PROGRESS') {
      forceModalTitle.value = 'Ujian Sedang Berlangsung'
      forceModalMessage.value = 'Anda memiliki ujian yang sedang berlangsung. Paksa login akan mengakhiri ujian.'
      forceSessionInfo.value = sessionData
      showForceModal.value = true
    } else if (status === 403 && code === 'USER_INACTIVE') {
      authStore.error = 'Akun Anda tidak aktif'
    } else {
      authStore.error = message ?? 'Login gagal'
    }
  } finally {
    authStore.isLoading = false
  }
}

async function handleForceLogin() {
  forceLoading.value = true
  authStore.error = null
  try {
    const res = await authApi.login({ login: form.value.login, password: form.value.password, force_login: true })
    if (res.data.success && res.data.data) {
      const { access_token, refresh_token, user } = res.data.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      authStore.user = user
      showForceModal.value = false
      redirectByRole(user.role)
    }
  } catch (err: unknown) {
    authStore.error = getErrorMessage(err, 'Gagal memaksa login')
    showForceModal.value = false
  } finally {
    forceLoading.value = false
  }
}
</script>

<template>
  <div class="d-flex align-items-center min-vh-100 bg-primary-lt" :style="branding.settings.login_bg_image ? { backgroundImage: `url(${branding.settings.login_bg_image})`, backgroundSize: 'cover', backgroundPosition: 'center' } : {}">
    <div class="container py-4">
      <div class="row align-items-center justify-content-center">
        <!-- Illustration: left side, large screens only -->
        <div class="col-lg-6 d-none d-lg-flex justify-content-center align-items-center">
          <i class="ti ti-school text-primary" style="font-size:8rem"></i>
        </div>

        <!-- Login form: right side -->
        <div class="col-lg-6">
      <div class="text-center mb-4">
        <div class="d-inline-flex align-items-center gap-2 mb-3">
          <img v-if="branding.settings.app_logo" :src="branding.settings.app_logo" alt="" style="height: 48px;">
          <div v-else class="avatar bg-primary">
            <i class="ti ti-book text-white fs-4"></i>
          </div>
        </div>
        <h2 class="fw-bold mb-1">{{ branding.appName }}</h2>
        <p class="text-muted">{{ branding.loginSubtitle }}</p>
      </div>

      <div class="card card-md shadow-sm">
        <div class="card-body">
          <!-- Session expired -->
          <div v-if="sessionExpired" class="alert alert-warning alert-dismissible mb-3">
            <button class="btn-close" @click="sessionExpired = false"></button>
            <i class="ti ti-clock-off me-2"></i>
            Sesi Anda telah berakhir. Silakan login kembali.
          </div>

          <!-- Error -->
          <div v-if="authStore.error" class="alert alert-danger alert-dismissible mb-3">
            <i class="ti ti-alert-circle me-2"></i>
            {{ authStore.error }}
          </div>

          <form @submit.prevent="handleLogin" aria-label="Form login">
            <BaseInput
              v-model="form.login"
              label="Username"
              :error="errors.login"
              type="text"
              placeholder="Masukkan username"
              autocomplete="username"
              @blur="onBlur('login')"
              @input="errors.login = ''"
            />

            <BaseInput
              v-model="form.password"
              label="Password"
              :error="errors.password"
              type="password"
              placeholder="Masukkan password"
              autocomplete="current-password"
              @blur="onBlur('password')"
              @input="errors.password = ''"
            />

            <div class="d-flex justify-content-end mt-3">
              <button type="submit" class="btn btn-primary w-100" :disabled="authStore.isLoading">
                <span v-if="authStore.isLoading" class="spinner-border spinner-border-sm me-2"></span>
                <i v-else class="ti ti-login me-1"></i>
                Masuk
              </button>
            </div>
          </form>
        </div>
      </div>

      <div class="text-center text-muted mt-3 small">
        &copy; {{ new Date().getFullYear() }} {{ branding.appName }}
      </div>
        </div><!-- end col login -->
      </div><!-- end row -->
    </div>
  </div>

  <!-- Force Login Confirmation Modal -->
  <BaseModal v-if="showForceModal" :title="forceModalTitle" size="sm" @close="showForceModal = false">
    <div class="text-center">
      <div class="mb-3">
        <i class="ti ti-alert-triangle text-warning" style="font-size: 3rem;"></i>
      </div>
      <p class="text-muted">{{ forceModalMessage }}</p>
      <div v-if="forceSessionInfo" class="text-start bg-light rounded p-3 mt-2 small">
        <div v-if="forceSessionInfo.device" class="mb-1">
          <i class="ti ti-device-desktop me-1"></i><strong>Perangkat:</strong> {{ forceSessionInfo.device }}
        </div>
        <div v-if="forceSessionInfo.ip" class="mb-1">
          <i class="ti ti-world me-1"></i><strong>IP:</strong> {{ forceSessionInfo.ip }}
        </div>
        <div v-if="forceSessionInfo.last_active">
          <i class="ti ti-clock me-1"></i><strong>Terakhir aktif:</strong> {{ forceSessionInfo.last_active }}
        </div>
      </div>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="showForceModal = false" :disabled="forceLoading">Batal</button>
      <button class="btn btn-danger" @click="handleForceLogin" :disabled="forceLoading">
        <span v-if="forceLoading" class="spinner-border spinner-border-sm me-1"></span>
        <i v-else class="ti ti-login me-1"></i>
        Paksa Login
      </button>
    </template>
  </BaseModal>
</template>
