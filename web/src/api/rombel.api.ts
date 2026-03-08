import type { AxiosRequestConfig } from 'axios'
import client from './client'

export interface Rombel {
  id: number
  name: string
  grade_level: string | null
  description: string | null
  created_at: string
}

export interface RombelListParams {
  page?: number
  per_page?: number
  search?: string
}

export const rombelApi = {
  list: (params?: RombelListParams, config?: AxiosRequestConfig) =>
    client.get('/admin/rombels', { params, ...config }),

  create: (data: { name: string; grade_level?: string; description?: string }) =>
    client.post('/admin/rombels', data),

  update: (id: number, data: { name: string; grade_level?: string; description?: string }) =>
    client.put(`/admin/rombels/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/rombels/${id}`),

  assignUsers: (id: number, userIds: number[]) =>
    client.post(`/admin/rombels/${id}/assign-users`, { user_ids: userIds }),

  removeUsers: (id: number, userIds: number[]) =>
    client.post(`/admin/rombels/${id}/remove-users`, { user_ids: userIds }),
}
