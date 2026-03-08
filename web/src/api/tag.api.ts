import client from './client'

export interface Tag {
  id: number
  name: string
  color: string
  created_at: string
}

export const tagApi = {
  list: (params?: { page?: number; per_page?: number; search?: string }) =>
    client.get('/admin/tags', { params }),

  listAll: () => client.get('/admin/tags/all'),

  create: (data: { name: string; color?: string }) =>
    client.post('/admin/tags', data),

  update: (id: number, data: { name: string; color?: string }) =>
    client.put(`/admin/tags/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/tags/${id}`),

  assignUsers: (id: number, userIds: number[]) =>
    client.post(`/admin/tags/${id}/assign-users`, { user_ids: userIds }),

  removeUser: (id: number, userId: number) =>
    client.post(`/admin/tags/${id}/remove-users`, { user_ids: [userId] }),

  importUsers: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return client.post('/admin/tags/import-users', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  exportTemplate: () =>
    client.get('/admin/tags/export-template', { responseType: 'blob' }),
}
