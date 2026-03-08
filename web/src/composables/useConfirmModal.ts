import { ref } from 'vue'

export function useConfirmModal() {
  const show = ref(false)
  const loading = ref(false)
  const title = ref('Konfirmasi Hapus')
  const message = ref('Apakah Anda yakin ingin menghapus data ini? Tindakan ini tidak dapat dibatalkan.')
  const pendingAction = ref<(() => Promise<void>) | null>(null)

  function ask(t: string, msg: string, action: () => Promise<void>) {
    title.value = t
    message.value = msg
    pendingAction.value = action
    show.value = true
  }

  async function confirm() {
    if (!pendingAction.value) return
    loading.value = true
    try {
      await pendingAction.value()
    } finally {
      loading.value = false
      show.value = false
      pendingAction.value = null
    }
  }

  function close() {
    show.value = false
    pendingAction.value = null
    loading.value = false
  }

  return { show, loading, title, message, ask, confirm, close }
}
