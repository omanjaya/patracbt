import { ref, computed } from 'vue'

export function usePagination(initialPerPage = 15) {
  const currentPage = ref(1)
  const perPage = ref(initialPerPage)
  const total = ref(0)

  const totalPages = computed(() => Math.ceil(total.value / perPage.value))
  const offset = computed(() => (currentPage.value - 1) * perPage.value)

  function setPage(page: number) {
    currentPage.value = Math.max(1, Math.min(page, totalPages.value || 1))
  }

  function setTotal(t: number) {
    total.value = t
    if (currentPage.value > totalPages.value && totalPages.value > 0) {
      currentPage.value = totalPages.value
    }
  }

  function reset() {
    currentPage.value = 1
    total.value = 0
  }

  return { currentPage, perPage, total, totalPages, offset, setPage, setTotal, reset }
}
