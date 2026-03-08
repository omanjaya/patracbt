import api from './client'
import type { ApiResponse, LoginRequest, LoginResponse, User } from '../types'

export const authApi = {
    login(data: LoginRequest) {
        return api.post<ApiResponse<LoginResponse>>('/auth/login', data)
    },

    logout() {
        return api.post<ApiResponse>('/auth/logout')
    },

    refresh(refreshToken: string) {
        return api.post<ApiResponse<{ access_token: string; expires_in: number }>>('/auth/refresh', {
            refresh_token: refreshToken,
        })
    },

    me() {
        return api.get<ApiResponse<User>>('/auth/me')
    },

    previewAsPeserta(userId: number) {
        return api.post<ApiResponse<{ preview_token: string, peserta_user_id: number, expires_in: number }>>('/admin/preview-as-peserta', {
            user_id: userId
        })
    },
}
