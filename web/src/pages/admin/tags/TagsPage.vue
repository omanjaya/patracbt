<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { tagApi, type Tag as TagType } from '../../../api/tag.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudTable } from '@/composables/useCrudTable'
import { useCrudModal } from '@/composables/useCrudModal'

const toast = useToastStore()
const confirm = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Tag' },
  { key: 'color', label: 'Warna' },
  { key: 'actions', label: 'Aksi', width: '120px' },
]

const { list, searchRaw, page, total, totalPages, loading, fetchList } = useCrudTable<TagType>({
  fetchFn: (params) => tagApi.list(params),
  errorMessage: 'Gagal memuat data tag',
})

const formErrors = reactive({ name: '', color: '' })

const { showModal, isEdit, saving, form, openCreate: _openCreate, openEdit: _openEdit, handleSave: _handleSave } = useCrudModal<{ name: string; color: string }>({
  createFn: (data) => tagApi.create(data),
  updateFn: (id, data) => tagApi.update(id, data),
  afterSave: fetchList,
  resetForm: () => ({ name: '', color: '#6B7280' }),
  successCreate: 'Tag berhasil ditambahkan',
  successUpdate: 'Tag berhasil diperbarui',
  errorMessage: 'Gagal menyimpan tag',
})

function openCreate() {
  formErrors.name = ''; formErrors.color = ''
  _openCreate()
}

function openEdit(item: TagType) {
  formErrors.name = ''; formErrors.color = ''
  _openEdit({ id: item.id, name: item.name, color: item.color })
}

function validateForm(): boolean {
  let valid = true
  formErrors.name = ''; formErrors.color = ''
  if (!form.name || form.name.trim().length < 1) {
    formErrors.name = 'Nama tag wajib diisi'
    valid = false
  }
  if (!form.color) {
    formErrors.color = 'Warna wajib dipilih'
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
    'Yakin ingin menghapus tag ini?',
    async () => {
      try {
        await tagApi.delete(id)
        toast.success('Tag berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus tag')
      }
    },
  )
}

onMounted(fetchList)
</script>

<template>
    <BasePageHeader
      title="Tag"
      subtitle="Kelola label / grup peserta"
      :breadcrumbs="[{ label: 'Master Data', to: '/admin' }, { label: 'Tag' }]"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreate"><i class="ti ti-plus"></i>
          Tambah Tag</button>
      </template>
    </BasePageHeader>

    <div class="card">
      <div class="card-header d-flex align-items-center gap-2 flex-wrap">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchRaw" class="form-control" placeholder="Cari tag..." aria-label="Cari tag" />
        </div>
      </div>

      <BaseTable :columns="columns" :loading="loading" empty="Belum ada tag">
        <tr v-for="item in list" :key="item.id">
          <td>
            <div class="d-flex align-items-center gap-2">
              <i class="ti ti-tag"></i>
              <span class="fw-medium">{{ item.name }}</span>
            </div>
          </td>
          <td>
            <div class="d-inline-block rounded me-1" style="width:16px;height:16px;border:1px solid rgba(0,0,0,.15)" :style="`background:${item.color}`" />
            <span class="text-muted small font-monospace">{{ item.color }}</span>
          </td>
          <td>
            <div class="d-flex gap-1">
              <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit tag" @click="openEdit(item)"><i class="ti ti-pencil"></i></button>
              <button type="button" class="btn btn-sm btn-ghost-danger" aria-label="Hapus tag" @click="askDelete(item.id)"><i class="ti ti-trash"></i></button>
            </div>
          </td>
        </tr>
      </BaseTable>

      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="(p: number) => { page = p; fetchList() }" />
    </div>

    <BaseModal v-if="showModal" :title="isEdit ? 'Edit Tag' : 'Tambah Tag'" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <fieldset :disabled="saving">
          <BaseInput
            v-model="form.name"
            label="Nama Tag *"
            :error="formErrors.name"
            type="text"
            placeholder="Contoh: Remedial"
            required
          />
          <div class="mb-3">
            <label class="form-label">Warna *</label>
            <div class="d-flex align-items-center gap-2">
              <input type="color" v-model="form.color" class="form-control form-control-color p-1" :class="{ 'is-invalid': formErrors.color }" style="width:48px;height:36px" />
              <span class="font-monospace small text-muted">{{ form.color }}</span>
            </div>
            <div v-if="formErrors.color" class="invalid-feedback d-block">{{ formErrors.color }}</div>
          </div>
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

