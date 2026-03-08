import { ref } from 'vue'
import { useTableFilters } from './useTableFilters'
import { useToastStore } from '../stores/toast.store'

interface CrudTableOptions {
  fetchFn: (params: { page: number; per_page: number; search: string; [key: string]: any }) => Promise<any>
  defaultPerPage?: number
  errorMessage?: string
}

export function useCrudTable<T>(options: CrudTableOptions) {
  const toast = useToastStore()
  const list = ref<T[]>([])
  const perPage = options.defaultPerPage ?? 20

  async function fetchList() {
    loading.value = true
    try {
      const res = await options.fetchFn({
        page: page.value,
        per_page: perPage,
        search: search.value,
      })
      list.value = res.data?.data ?? []
      total.value = res.data?.meta?.total ?? 0
    } catch (e: any) {
      toast.error(e?.response?.data?.message ?? options.errorMessage ?? 'Gagal memuat data')
    } finally {
      loading.value = false
    }
  }

  const { searchRaw, search, page, total, totalPages, loading } = useTableFilters(fetchList, { perPage })

  return {
    list,
    searchRaw,
    search,
    page,
    total,
    totalPages,
    loading,
    fetchList,
  }
}
