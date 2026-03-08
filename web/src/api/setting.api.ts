import client from './client'

export const settingApi = {
  getAll: () => client.get('/admin/settings'),

  update: (settings: Record<string, string>) =>
    client.post('/admin/settings', { settings }),

  createBackup: () => client.post('/settings/backup'),

  downloadBackupUrl: (filename: string) => `/api/v1/settings/backup/download/${filename}`,

  restoreBackup: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return client.post('/settings/restore', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  testAI: () => client.post('/settings/ai/test'),

  // System
  clearCache: () => client.post('/settings/system/clear-cache'),

  // Panic Mode
  panicModeStatus: () => client.get('/settings/panic-mode/status'),
  activatePanicMode: () => client.post('/settings/panic-mode/activate'),
  deactivatePanicMode: () => client.post('/settings/panic-mode/deactivate'),

  // MinIO Settings & Backup
  saveMinioSettings: (data: {
    minio_endpoint: string
    minio_bucket: string
    minio_access_key: string
    minio_secret_key: string
    minio_use_ssl: string
  }) => client.post('/settings/minio', data),

  testMinioConnection: () => client.get('/settings/minio/test'),

  backupToMinio: () => client.post('/settings/backup/minio'),

  listMinioBackups: () => client.get('/settings/backup/minio/list'),

  restoreFromMinio: (filename: string) =>
    client.post('/settings/restore/minio', { filename }),

  // Branding Upload
  uploadBranding: (field: string, file: File) => {
    const form = new FormData()
    form.append('field', field)
    form.append('file', file)
    return client.post('/admin/settings/upload-branding', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },

  // Database Management
  exportDatabase: () =>
    client.post('/admin/settings/database/export', null, {
      responseType: 'blob',
      timeout: 300000, // 5 min for large DBs
    }),

  exportAndSave: () => client.post('/admin/settings/database/export-save'),

  importDatabase: (file: File) => {
    const form = new FormData()
    form.append('backup', file)
    return client.post('/admin/settings/database/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 600000, // 10 min
    })
  },

  listDatabaseBackups: () => client.get('/admin/settings/database/backups'),

  deleteDatabaseBackup: (filename: string) =>
    client.delete(`/admin/settings/database/backups/${filename}`),

  // Chunked Restore
  uploadRestoreChunk: (batchId: string, chunk: Blob, chunkIndex: number) => {
    const form = new FormData()
    form.append('batch_id', batchId)
    form.append('chunk_index', String(chunkIndex))
    form.append('chunk', chunk)
    return client.post('/admin/settings/restore/chunk', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000, // 5 min per chunk
    })
  },

  processRestore: (batchId: string) =>
    client.post('/admin/settings/restore/process', { batch_id: batchId }),

  restoreProgress: (restoreId: string) =>
    client.get(`/admin/settings/restore/progress/${restoreId}`),
}
