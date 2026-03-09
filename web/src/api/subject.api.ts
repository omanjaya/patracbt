import client from './client'

export interface Subject {
  id: number
  name: string
  code: string | null
  question_banks_count: number
  created_at: string
}

export const subjectApi = {
  list: (params?: { page?: number; per_page?: number; search?: string }) =>
    client.get('/admin/subjects', { params }),

  listAll: () => client.get('/admin/subjects/all'),

  create: (data: { name: string; code?: string }) =>
    client.post('/admin/subjects', data),

  update: (id: number, data: { name: string; code?: string }) =>
    client.put(`/admin/subjects/${id}`, data),

  delete: (id: number) =>
    client.delete(`/admin/subjects/${id}`),

  bulkDelete: (ids: number[]) =>
    client.post('/admin/subjects/bulk-delete', { ids }),
}
