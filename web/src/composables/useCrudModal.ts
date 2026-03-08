import { ref, reactive } from 'vue'
import { useToastStore } from '../stores/toast.store'

interface CrudModalOptions<T> {
  createFn: (data: T) => Promise<any>
  updateFn: (id: number, data: T) => Promise<any>
  afterSave: () => void | Promise<void>
  resetForm: () => T
  successCreate?: string
  successUpdate?: string
  errorMessage?: string
}

export function useCrudModal<T extends Record<string, any>>(options: CrudModalOptions<T>) {
  const toast = useToastStore()
  const showModal = ref(false)
  const isEdit = ref(false)
  const editId = ref<number | null>(null)
  const saving = ref(false)
  const form = reactive(options.resetForm()) as T

  function openCreate() {
    isEdit.value = false
    editId.value = null
    Object.assign(form, options.resetForm())
    showModal.value = true
  }

  function openEdit(item: any) {
    isEdit.value = true
    editId.value = item.id
    Object.assign(form, item)
    showModal.value = true
  }

  async function handleSave() {
    saving.value = true
    try {
      if (isEdit.value && editId.value) {
        await options.updateFn(editId.value, { ...form })
        toast.success(options.successUpdate ?? 'Data berhasil diperbarui')
      } else {
        await options.createFn({ ...form })
        toast.success(options.successCreate ?? 'Data berhasil ditambahkan')
      }
      showModal.value = false
      await options.afterSave()
    } catch (e: any) {
      toast.error(e?.response?.data?.message ?? options.errorMessage ?? 'Gagal menyimpan data')
    } finally {
      saving.value = false
    }
  }

  return {
    showModal,
    isEdit,
    editId,
    saving,
    form,
    openCreate,
    openEdit,
    handleSave,
  }
}
