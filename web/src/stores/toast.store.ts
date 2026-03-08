import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: number
  type: ToastType
  message: string
  duration: number
  remaining: number
  startedAt: number
}

const MAX_VISIBLE = 5
let nextId = 0
const timers: Record<number, ReturnType<typeof setTimeout>> = {}

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])

  function add(type: ToastType, message: string, duration?: number) {
    // Deduplication: skip if identical message+type already visible
    const duplicate = toasts.value.find((t) => t.type === type && t.message === message)
    if (duplicate) return duplicate.id

    const dur = duration ?? (type === 'error' ? 7000 : 5000)
    const id = ++nextId
    toasts.value.push({ id, type, message, duration: dur, remaining: dur, startedAt: Date.now() })
    timers[id] = setTimeout(() => remove(id), dur)

    // Enforce max visible: remove oldest if exceeding limit
    while (toasts.value.length > MAX_VISIBLE) {
      const oldest = toasts.value[0]!
      remove(oldest.id)
    }

    return id
  }

  function remove(id: number) {
    clearTimeout(timers[id])
    delete timers[id]
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  function pause(id: number) {
    const t = toasts.value.find((t) => t.id === id)
    if (!t) return
    clearTimeout(timers[id])
    delete timers[id]
    t.remaining = t.remaining - (Date.now() - t.startedAt)
    if (t.remaining <= 0) t.remaining = 500
  }

  function resume(id: number) {
    const t = toasts.value.find((t) => t.id === id)
    if (!t) return
    t.startedAt = Date.now()
    timers[id] = setTimeout(() => remove(id), t.remaining)
  }

  const success = (msg: string, dur?: number) => add('success', msg, dur)
  const error = (msg: string, dur?: number) => add('error', msg, dur)
  const warning = (msg: string, dur?: number) => add('warning', msg, dur)
  const info = (msg: string, dur?: number) => add('info', msg, dur)

  return { toasts, add, remove, pause, resume, success, error, warning, info }
})
