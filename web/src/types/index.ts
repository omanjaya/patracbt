export interface User {
    id: number
    name: string
    username: string
    role: 'admin' | 'guru' | 'pengawas' | 'peserta'
    avatar_url: string | null
    profile?: UserProfile
}

export interface UserProfile {
    nis: string | null
    class: string | null
    major: string | null
}

export interface LoginRequest {
    login: string
    password: string
    force_login?: boolean
}

export interface LoginResponse {
    access_token: string
    refresh_token: string
    expires_in: number
    user: User
}

export interface ApiResponse<T = unknown> {
    success: boolean
    message?: string
    code?: string
    data?: T
    errors?: Record<string, string>
    meta?: PaginationMeta
}

export interface PaginationMeta {
    page: number
    per_page: number
    total: number
    total_pages: number
}
