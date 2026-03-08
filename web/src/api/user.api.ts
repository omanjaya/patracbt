import type { AxiosRequestConfig } from 'axios'
import client from './client'

export interface UserItem {
  id: number
  name: string
  username: string
  email: string | null
  role: string
  avatar_path: string | null
  is_active: boolean
  last_login_at: string | null
  created_at: string
  profile?: {
    nis: string | null
    class: string | null
    major: string | null
  }
}

export const userApi = {
  list: (params?: { page?: number; per_page?: number; search?: string; role?: string; rombel_id?: number; room_id?: number; tag_id?: number }, config?: AxiosRequestConfig) =>
    client.get('/admin/users', { params, ...config }),

  listTrashed: (params?: { page?: number; per_page?: number; search?: string; role?: string }, config?: AxiosRequestConfig) =>
    client.get('/admin/users/trashed', { params, ...config }),

  create: (data: {
    name: string
    username: string
    password: string
    role: string
    email?: string
    is_active?: boolean
    rombel_ids?: number[]
    profile?: { nis?: string; class?: string; major?: string; year?: number }
  }) => client.post('/admin/users', data),

  update: (id: number, data: {
    name: string
    role: string
    email?: string
    password?: string
    is_active?: boolean
    rombel_ids?: number[]
    profile?: { nis?: string; class?: string; major?: string; year?: number }
  }) => client.put(`/admin/users/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/users/${id}`),

  restore: (id: number) =>
    client.post(`/admin/users/${id}/restore`),

  forceDelete: (id: number) =>
    client.delete(`/admin/users/${id}/force`),

  bulkAction: (action: 'delete' | 'restore' | 'force_delete', ids: number[]) =>
    client.post('/admin/users/bulk-action', { action, ids }),
}
