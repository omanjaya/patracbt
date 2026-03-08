import type { AxiosRequestConfig } from 'axios'
import client from './client'

export interface Room {
  id: number
  name: string
  capacity: number
  created_at: string
}

export const roomApi = {
  list: (params?: { page?: number; per_page?: number; search?: string }, config?: AxiosRequestConfig) =>
    client.get('/admin/rooms', { params, ...config }),

  create: (data: { name: string; capacity?: number }) =>
    client.post('/admin/rooms', data),

  update: (id: number, data: { name: string; capacity?: number }) =>
    client.put(`/admin/rooms/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/rooms/${id}`),

  assignUsers: (id: number, userIds: number[]) =>
    client.post(`/admin/rooms/${id}/assign-users`, { user_ids: userIds }),

  removeUsers: (id: number, userIds: number[]) =>
    client.post(`/admin/rooms/${id}/remove-users`, { user_ids: userIds }),

  getUsers: (id: number, params?: { page?: number; per_page?: number }) =>
    client.get(`/admin/rooms/${id}/users`, { params }),
}
