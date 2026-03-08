<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { subjectApi, type Subject } from '../../../api/subject.api'
import { useToastStore } from '@/stores/toast.store'
import { useCrudTable } from '@/composables/useCrudTable'
import { useCrudModal } from '@/composables/useCrudModal'
import { useConfirmModal } from '@/composables/useConfirmModal'

const toast = useToastStore()
const confirm = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Mata Pelajaran' },
  { key: 'code', label: 'Kode' },
  { key: 'actions', label: 'Aksi', width: '120px' },
]

const { list, searchRaw, page, total, totalPages, loading, fetchList } = useCrudTable<Subject>({
  fetchFn: (params) => subjectApi.list(params),
  errorMessage: 'Gagal memuat data mata pelajaran',
})

const formErrors = reactive({ name: '' })

const { showModal, isEdit, saving, form, openCreate: _openCreate, openEdit: _openEdit, handleSave: _handleSave } = useCrudModal<{ name: string; code: string }>({
  createFn: (data) => subjectApi.create({ name: data.name, code: data.code || undefined }),
  updateFn: (id, data) => subjectApi.update(id, { name: data.name, code: data.code || undefined }),
  afterSave: fetchList,
  resetForm: () => ({ name: '', code: '' }),
  successCreate: 'Mata pelajaran berhasil ditambahkan',
  successUpdate: 'Mata pelajaran berhasil diperbarui',
  errorMessage: 'Gagal menyimpan mata pelajaran',
})

function openCreate() {
  formErrors.name = ''
  _openCreate()
}

function openEdit(item: Subject) {
  formErrors.name = ''
  _openEdit({ id: item.id, name: item.name, code: item.code ?? '' })
}

function validateForm(): boolean {
  let valid = true
  formErrors.name = ''
  if (!form.name || form.name.trim().length < 2) {
    formErrors.name = 'Nama mata pelajaran minimal 2 karakter'
    valid = false
  }
  return valid
}

async function handleSave() {
  if (!validateForm()) return
  await _handleSave()
}

function askDelete(id: number) {
  confirm.ask(
    'Konfirmasi Hapus',
    'Yakin ingin menghapus mata pelajaran ini?',
    async () => {
      try {
        await subjectApi.delete(id)
        toast.success('Mata pelajaran berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus mata pelajaran')
      }
    },
  )
}

onMounted(fetchList)
</script>

<template>
  <BasePageHeader
    title="Mata Pelajaran"
    subtitle="Kelola daftar mata pelajaran"
    :breadcrumbs="[{ label: 'Master Data', to: '/admin' }, { label: 'Mata Pelajaran' }]"
  >
    <template #actions>
      <button class="btn btn-primary" @click="openCreate">
        <i class="ti ti-plus me-1"></i>Tambah Mapel
      </button>
    </template>
  </BasePageHeader>

  <div class="card">
    <div class="card-header d-flex align-items-center gap-2 flex-wrap">
      <div class="input-group">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari mata pelajaran..." aria-label="Cari mata pelajaran" />
      </div>
    </div>

    <BaseTable :columns="columns" :loading="loading" empty="Belum ada mata pelajaran">
      <tr v-for="item in list" :key="item.id">
        <td>
          <div class="d-flex align-items-center gap-2">
            <i class="ti ti-book text-muted"></i>
            <span class="fw-medium">{{ item.name }}</span>
          </div>
        </td>
        <td>
          <code v-if="item.code" class="text-muted">{{ item.code }}</code>
          <span v-else class="text-muted">–</span>
        </td>
        <td>
          <div class="d-flex gap-1">
            <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit mata pelajaran" @click="openEdit(item)">
              <i class="ti ti-pencil"></i>
            </button>
            <button type="button" class="btn btn-sm btn-ghost-danger" aria-label="Hapus mata pelajaran" @click="askDelete(item.id)">
              <i class="ti ti-trash"></i>
            </button>
          </div>
        </td>
      </tr>
    </BaseTable>

    <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="(p: number) => { page = p; fetchList() }" />
  </div>

  <BaseModal v-if="showModal" :title="isEdit ? 'Edit Mata Pelajaran' : 'Tambah Mata Pelajaran'" @close="showModal = false">
    <form @submit.prevent="handleSave">
      <fieldset :disabled="saving">
        <BaseInput
          v-model="form.name"
          label="Nama Mata Pelajaran *"
          :error="formErrors.name"
          type="text"
          placeholder="Contoh: Matematika"
          required
        />
        <BaseInput
          v-model="form.code"
          label="Kode"
          type="text"
          placeholder="Contoh: MTK"
        />
      </fieldset>
    </form>
    <template #footer>
      <BaseButton variant="secondary" @click="showModal = false">Batal</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="handleSave">Simpan</BaseButton>
    </template>
  </BaseModal>

  <BaseConfirmModal
    v-if="confirm.show.value"
    :title="confirm.title.value"
    :message="confirm.message.value"
    :loading="confirm.loading.value"
    @confirm="confirm.confirm"
    @close="confirm.close"
  />
</template>
