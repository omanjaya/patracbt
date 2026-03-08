<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { roomApi, type Room } from '../../../api/room.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudTable } from '@/composables/useCrudTable'
import { useCrudModal } from '@/composables/useCrudModal'

const toast = useToastStore()
const confirm = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Ruangan' },
  { key: 'capacity', label: 'Kapasitas' },
  { key: 'actions', label: 'Aksi', width: '120px' },
]

const { list, searchRaw, page, total, totalPages, loading, fetchList } = useCrudTable<Room>({
  fetchFn: (params) => roomApi.list(params),
  errorMessage: 'Gagal memuat data ruangan',
})

const formErrors = reactive({ name: '', capacity: '' })

const { showModal, isEdit, saving, form, openCreate: _openCreate, openEdit: _openEdit, handleSave: _handleSave } = useCrudModal<{ name: string; capacity: number }>({
  createFn: (data) => roomApi.create(data),
  updateFn: (id, data) => roomApi.update(id, data),
  afterSave: fetchList,
  resetForm: () => ({ name: '', capacity: 30 }),
  successCreate: 'Ruangan berhasil ditambahkan',
  successUpdate: 'Ruangan berhasil diperbarui',
  errorMessage: 'Gagal menyimpan ruangan',
})

function openCreate() {
  formErrors.name = ''; formErrors.capacity = ''
  _openCreate()
}

function openEdit(item: Room) {
  formErrors.name = ''; formErrors.capacity = ''
  _openEdit({ id: item.id, name: item.name, capacity: item.capacity })
}

function validateForm(): boolean {
  let valid = true
  formErrors.name = ''; formErrors.capacity = ''
  if (!form.name || form.name.trim().length < 1) {
    formErrors.name = 'Nama ruangan wajib diisi'
    valid = false
  }
  if (!form.capacity || form.capacity <= 0) {
    formErrors.capacity = 'Kapasitas harus lebih dari 0'
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
    'Yakin ingin menghapus ruangan ini?',
    async () => {
      try {
        await roomApi.delete(id)
        toast.success('Ruangan berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus ruangan')
      }
    },
  )
}

onMounted(fetchList)
</script>

<template>
  <BasePageHeader
    title="Ruangan"
    subtitle="Kelola ruang ujian"
    :breadcrumbs="[{ label: 'Master Data', to: '/admin' }, { label: 'Ruangan' }]"
  >
    <template #actions>
      <button class="btn btn-primary" @click="openCreate">
        <i class="ti ti-plus me-1"></i>Tambah Ruangan
      </button>
    </template>
  </BasePageHeader>

  <div class="card">
    <div class="card-header d-flex align-items-center gap-2 flex-wrap">
      <div class="input-group">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari ruangan..." aria-label="Cari ruangan" />
      </div>
    </div>

    <BaseTable :columns="columns" :loading="loading" empty="Belum ada ruangan">
      <tr v-for="item in list" :key="item.id">
        <td>
          <div class="d-flex align-items-center gap-2">
            <i class="ti ti-door text-muted"></i>
            <span class="fw-medium">{{ item.name }}</span>
          </div>
        </td>
        <td>{{ item.capacity }} peserta</td>
        <td>
          <div class="d-flex gap-1">
            <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit ruangan" @click="openEdit(item)">
              <i class="ti ti-pencil"></i>
            </button>
            <button type="button" class="btn btn-sm btn-ghost-danger" aria-label="Hapus ruangan" @click="askDelete(item.id)">
              <i class="ti ti-trash"></i>
            </button>
          </div>
        </td>
      </tr>
    </BaseTable>

    <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
  </div>

  <BaseModal v-if="showModal" :title="isEdit ? 'Edit Ruangan' : 'Tambah Ruangan'" @close="showModal = false">
    <form @submit.prevent="handleSave">
      <fieldset :disabled="saving">
        <BaseInput
          v-model="form.name"
          label="Nama Ruangan *"
          :error="formErrors.name"
          type="text"
          placeholder="Contoh: Lab Komputer A"
          required
        />
        <BaseInput
          v-model="form.capacity"
          label="Kapasitas *"
          :error="formErrors.capacity"
          type="number"
          placeholder="30"
          min="1"
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
