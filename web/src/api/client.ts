import axios from 'axios'
import type { ApiResponse } from '../types'
import router from '@/router'

const api = axios.create({
    baseURL: '/api/v1',
    headers: { 'Content-Type': 'application/json' },
    timeout: 30000,
})

// Request interceptor: inject token
api.interceptors.request.use((config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

function redirectToLogin() {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    router.push({ path: '/login', query: { expired: '1' } })
}

// Shared refresh state to prevent concurrent refresh attempts
let isRefreshing = false
let refreshPromise: Promise<string | null> | null = null

function doRefresh(): Promise<string | null> {
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) return Promise.resolve(null)
    return axios.post<ApiResponse<{ access_token: string }>>('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
    }).then((res) => {
        if (res.data.success && res.data.data) {
            localStorage.setItem('access_token', res.data.data.access_token)
            return res.data.data.access_token
        }
        console.error('[api/client] Token refresh returned unsuccessful response')
        return null
    }).catch((refreshError) => {
        console.error('[api/client] Token refresh failed:', refreshError)
        return null
    })
}

// Response interceptor: auto refresh token
api.interceptors.response.use(
    (response) => response,
    async (error) => {
        const original = error.config
        const status = error.response?.status

        // Skip token refresh for auth endpoints (login, refresh)
        const isAuthEndpoint = original.url?.startsWith('/auth/')

        // Don't try to refresh if the refresh endpoint itself returned 401
        if (status === 401 && original.url?.includes('/auth/refresh')) {
            redirectToLogin()
            return Promise.reject(error)
        }

        if (status === 401 && !original._retry && !isAuthEndpoint) {
            original._retry = true
            if (!isRefreshing) {
                isRefreshing = true
                refreshPromise = doRefresh().finally(() => {
                    isRefreshing = false
                    refreshPromise = null
                })
            }
            const newToken = await refreshPromise
            if (newToken) {
                original.headers.Authorization = `Bearer ${newToken}`
                return api(original)
            }
            redirectToLogin()
        }

        if (status === 403) {
            router.push('/403')
        }

        if (status === 500) {
            error.response.data = {
                ...error.response.data,
                message: error.response.data?.message || 'Terjadi kesalahan pada server',
            }
        }

        return Promise.reject(error)
    }
)

export default api
