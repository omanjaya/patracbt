import { onUnmounted } from 'vue'

export function useDebounceFn<T extends (...args: any[]) => any>(fn: T, delay: number) {
  let timer: ReturnType<typeof setTimeout> | null = null

  function debounced(...args: Parameters<T>) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), delay)
  }

  function cancel() {
    if (timer) { clearTimeout(timer); timer = null }
  }

  function flush(...args: Parameters<T>) {
    cancel()
    fn(...args)
  }

  onUnmounted(() => {
    cancel()
  })

  return { debounced, cancel, flush }
}
