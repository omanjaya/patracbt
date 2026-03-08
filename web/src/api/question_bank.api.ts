import type { AxiosRequestConfig } from 'axios'
import client from './client'

export interface QuestionBank {
  id: number
  name: string
  subject_id: number | null
  subject?: { id: number; name: string; code: string }
  description: string
  question_count: number
  is_locked?: boolean
  status?: string
  created_by: number
  created_at: string
}

export interface Stimulus {
  id: number
  question_bank_id: number
  content: string
  created_at: string
}

export interface Question {
  id: number
  question_bank_id: number
  stimulus_id: number | null
  question_type: string
  body: string
  score: number
  difficulty: string
  options: unknown
  correct_answer: unknown
  audio_path: string | null
  audio_limit: number
  order_index: number
  bloom_level: number  // 0-6 (0=unset)
  topic_code: string
  created_at: string
}

export const questionBankApi = {
  list: (params?: { page?: number; per_page?: number; search?: string; subject_id?: number; status?: string }, config?: AxiosRequestConfig) =>
    client.get('/question-banks', { params, ...config }),

  getById: (id: number) =>
    client.get(`/question-banks/${id}`),

  create: (data: { name: string; subject_id?: number; description?: string }) =>
    client.post('/question-banks', data),

  update: (id: number, data: { name: string; subject_id?: number; description?: string }) =>
    client.put(`/question-banks/${id}`, data),

  delete: (id: number) =>
    client.delete(`/question-banks/${id}`),

  bulkDelete: (ids: number[]) =>
    client.post('/question-banks/bulk-delete', { ids }),

  toggleStatus: (id: number) =>
    client.patch(`/question-banks/${id}/toggle-status`),

  // Questions
  listQuestions: (bankId: number, params?: { page?: number; per_page?: number }) =>
    client.get(`/question-banks/${bankId}/questions`, { params }),

  createQuestion: (bankId: number, data: CreateQuestionPayload, audioFile?: File) => {
    if (audioFile) {
      const fd = buildQuestionFormData(data, audioFile)
      return client.post(`/question-banks/${bankId}/questions`, fd, { headers: { 'Content-Type': 'multipart/form-data' } })
    }
    return client.post(`/question-banks/${bankId}/questions`, data)
  },

  updateQuestion: (id: number, data: Partial<CreateQuestionPayload> & { remove_audio?: boolean; audio_limit?: number }, audioFile?: File) => {
    if (audioFile || data.remove_audio) {
      const fd = buildQuestionFormData(data, audioFile)
      if (data.remove_audio) fd.append('remove_audio', 'true')
      return client.put(`/questions/${id}`, fd, { headers: { 'Content-Type': 'multipart/form-data' } })
    }
    return client.put(`/questions/${id}`, data)
  },

  deleteQuestion: (id: number) =>
    client.delete(`/questions/${id}`),

  bulkActionQuestions: (action: 'delete' | 'move', ids: number[], targetBankId?: number) =>
    client.post('/questions/bulk-action', { action, ids, target_bank_id: targetBankId }),

  importQuestions: (bankId: number, data: { content: string }) =>
    client.post(`/question-banks/${bankId}/import`, data),

  reorderQuestions: (bankId: number, items: { id: number; order_index: number }[]) =>
    client.patch(`/question-banks/${bankId}/questions/reorder`, { items }),

  cloneBank: (id: number) =>
    client.post(`/question-banks/${id}/clone`),

  // Print / Export / Import
  printQuestions: (bankId: number) =>
    client.get(`/question-banks/${bankId}/questions/print`),

  getQuestionIds: (bankId: number, search?: string) =>
    client.get(`/question-banks/${bankId}/questions/ids`, { params: search ? { q: search } : undefined }),

  exportQuestionsZIP: (bankId: number) =>
    client.get(`/question-banks/${bankId}/export-zip`, { responseType: 'blob' }),

  importQuestionsZIP: (bankId: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return client.post(`/question-banks/${bankId}/import-zip`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  // Stimuli
  listStimuli: (bankId: number) =>
    client.get(`/question-banks/${bankId}/stimuli`),

  createStimulus: (bankId: number, data: { content: string }) =>
    client.post(`/question-banks/${bankId}/stimuli`, data),

  updateStimulus: (id: number, data: { content: string }) =>
    client.put(`/stimuli/${id}`, data),

  deleteStimulus: (id: number) =>
    client.delete(`/stimuli/${id}`),

  // AI Question Generation
  generateAI: (data: {
    topic: string
    count?: number
    type?: string
    difficulty?: string
    language?: string
    prompt?: string
  }) =>
    client.post('/admin/questions/generate-ai', data, {
      timeout: 120000, // 2 min for AI generation
    }),

  // MathML Conversion
  convertMathML: (mathml: string) =>
    client.post('/admin/questions/convert-mathml', { mathml }),

  // Upload Image during Import
  uploadImportImage: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return client.post('/admin/questions/import/upload-image', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },
}

export interface CreateQuestionPayload {
  stimulus_id?: number
  question_type: string
  body: string
  score?: number
  difficulty?: string
  options?: unknown
  correct_answer?: unknown
  order_index?: number
  audio_limit?: number
  bloom_level?: number
  topic_code?: string
}

function buildQuestionFormData(data: Partial<CreateQuestionPayload> & { remove_audio?: boolean }, audioFile?: File): FormData {
  const fd = new FormData()
  if (data.question_type) fd.append('question_type', data.question_type)
  if (data.body) fd.append('body', data.body)
  if (data.score !== undefined) fd.append('score', String(data.score))
  if (data.difficulty) fd.append('difficulty', data.difficulty)
  if (data.order_index !== undefined) fd.append('order_index', String(data.order_index))
  if (data.stimulus_id !== undefined) fd.append('stimulus_id', String(data.stimulus_id))
  if (data.options !== undefined) fd.append('options', JSON.stringify(data.options))
  if (data.correct_answer !== undefined) fd.append('correct_answer', JSON.stringify(data.correct_answer))
  if (data.audio_limit !== undefined) fd.append('audio_limit', String(data.audio_limit))
  if (data.bloom_level !== undefined) fd.append('bloom_level', String(data.bloom_level))
  if (data.topic_code) fd.append('topic_code', data.topic_code)
  if (audioFile) fd.append('audio', audioFile)
  return fd
}

export const QUESTION_TYPES = [
  { value: 'pg', label: 'Pilihan Ganda (PG)' },
  { value: 'pgk', label: 'Pilihan Ganda Kompleks (PGK)' },
  { value: 'benar_salah', label: 'Benar / Salah' },
  { value: 'menjodohkan', label: 'Menjodohkan' },
  { value: 'isian_singkat', label: 'Isian Singkat' },
  { value: 'matrix', label: 'Matrix / Tabel' },
  { value: 'esai', label: 'Esai' },
]

export const DIFFICULTY_LABELS: Record<string, string> = {
  easy: 'Mudah',
  medium: 'Sedang',
  hard: 'Sulit',
}

export const BLOOM_LEVELS = [
  { value: 0, label: 'Tidak Ditentukan' },
  { value: 1, label: 'C1 - Mengingat' },
  { value: 2, label: 'C2 - Memahami' },
  { value: 3, label: 'C3 - Menerapkan' },
  { value: 4, label: 'C4 - Menganalisis' },
  { value: 5, label: 'C5 - Mengevaluasi' },
  { value: 6, label: 'C6 - Mencipta' },
]
