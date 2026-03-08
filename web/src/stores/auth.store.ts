import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, LoginRequest } from '@/types'
import { authApi } from '@/api/auth.api'
import { getErrorMessage } from '@/utils/apiError'
import router from '@/router'

let _initPromise: Promise<void> | null = null

export const useAuthStore = defineStore('auth', () => {
    const user = ref<User | null>(null)
    const isLoading = ref(false)
    const error = ref<string | null>(null)

    const isAuthenticated = computed(() => !!user.value)
    const userRole = computed(() => user.value?.role ?? null)

    async function login(data: LoginRequest) {
        isLoading.value = true
        error.value = null
        try {
            const res = await authApi.login(data)
            if (res.data.success && res.data.data) {
                const { access_token, refresh_token, user: userData } = res.data.data
                localStorage.setItem('access_token', access_token)
                localStorage.setItem('refresh_token', refresh_token)
                user.value = userData
                redirectByRole(userData.role)
            }
        } catch (err: unknown) {
            error.value = getErrorMessage(err, 'Login gagal')
        } finally {
            isLoading.value = false
        }
    }

    async function fetchUser() {
        try {
            const res = await authApi.me()
            if (res.data.success && res.data.data) {
                user.value = res.data.data
            }
        } catch {
            logout()
        }
    }

    function logout() {
        error.value = null
        authApi.logout().catch((e) => { console.warn('Logout API call failed:', e) })
        user.value = null
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        // Full page reload clears all Pinia store state reliably
        window.location.href = '/login'
    }

    function redirectByRole(role: string) {
        switch (role) {
            case 'admin':
                router.push('/admin')
                break
            case 'guru':
                router.push('/guru')
                break
            case 'pengawas':
                router.push('/pengawas')
                break
            case 'peserta':
                router.push('/peserta')
                break
            default:
                router.push('/')
        }
    }

    async function init() {
        const token = localStorage.getItem('access_token')
        if (!token) return
        if (_initPromise) return _initPromise
        _initPromise = fetchUser().finally(() => { _initPromise = null })
        return _initPromise
    }

    return { user, isLoading, error, isAuthenticated, userRole, login, logout, fetchUser, init }
})
