import { ref } from 'vue'
import { useToastStore } from '@/stores/toast.store'
import { getErrorMessage, getFieldErrors } from '@/utils/apiError'

export function useApi<T = unknown>() {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const fieldErrors = ref<Record<string, string>>({})

  async function execute(fn: () => Promise<T>, options?: { successMsg?: string; errorMsg?: string; silent?: boolean }) {
    loading.value = true
    error.value = null
    fieldErrors.value = {}
    try {
      const result = await fn()
      data.value = result as T
      if (options?.successMsg) {
        useToastStore().success(options.successMsg)
      }
      return result
    } catch (e: unknown) {
      const msg = getErrorMessage(e)
      error.value = msg
      fieldErrors.value = getFieldErrors(e)
      if (!options?.silent && options?.errorMsg !== '') {
        useToastStore().error(options?.errorMsg || msg)
      }
      throw e
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, fieldErrors, execute }
}
