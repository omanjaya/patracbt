export interface ApiErrorResponse {
  response?: {
    status?: number
    data?: { message?: string; code?: string; errors?: Record<string, string> }
  }
  message?: string
  code?: string
}

export function getErrorMessage(err: unknown, defaultMsg = 'Terjadi kesalahan'): string {
  if (!err || typeof err !== 'object') return defaultMsg
  const e = err as ApiErrorResponse

  // Network error (no response from server) - check axios error code first, then message fallback
  if (!e.response && (e.code === 'ERR_NETWORK' || e.message === 'Network Error')) {
    return 'Tidak dapat terhubung ke server. Periksa koneksi internet Anda.'
  }

  // Timeout error
  if (e.message?.includes('timeout')) {
    return 'Server tidak merespons. Silakan coba lagi.'
  }

  return e.response?.data?.message || e.message || defaultMsg
}

export function getFieldErrors(err: unknown): Record<string, string> {
  if (!err || typeof err !== 'object') return {}
  const e = err as ApiErrorResponse
  return e.response?.data?.errors ?? {}
}
