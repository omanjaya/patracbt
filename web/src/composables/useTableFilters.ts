import { ref, computed, watch } from 'vue'
import { useDebounce } from '@/composables/useDebounce'

export function useTableFilters(fetchFn: () => Promise<void>, options?: { perPage?: number }) {
  const searchRaw = ref('')
  const search = useDebounce(searchRaw, 500)
  const page = ref(1)
  const perPage = ref(options?.perPage ?? 20)
  const total = ref(0)
  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / perPage.value)))
  const loading = ref(false)

  watch(search, () => {
    page.value = 1
    fetchFn()
  })

  return { searchRaw, search, page, perPage, total, totalPages, loading }
}
