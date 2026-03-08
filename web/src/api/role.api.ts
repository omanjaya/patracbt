import client from './client'

export interface Role {
  id: number
  name: string
  guard_name: string
  created_at: string
}

export const roleApi = {
  list: (params?: { page?: number; per_page?: number; search?: string }) =>
    client.get('/admin/roles', { params }),

  create: (data: { name: string; guard_name: string }) =>
    client.post('/admin/roles', data),

  update: (id: number, data: { name: string; guard_name: string }) =>
    client.put(`/admin/roles/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/roles/${id}`),

  getPermissions: (id: number) =>
    client.get(`/admin/roles/${id}/permissions`),

  assignPermissions: (id: number, permission_ids: number[]) =>
    client.post(`/admin/roles/${id}/permissions`, { permission_ids }),
}
