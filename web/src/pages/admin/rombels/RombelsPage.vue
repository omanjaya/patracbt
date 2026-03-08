<script setup lang="ts">
import { onMounted } from 'vue'
import { getIllustration } from '../../../utils/avatar'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudTable } from '@/composables/useCrudTable'
import { useCrudModal } from '@/composables/useCrudModal'

const toast = useToastStore()
const confirmModal = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Rombel' },
  { key: 'grade_level', label: 'Tingkat' },
  { key: 'description', label: 'Deskripsi' },
  { key: 'actions', label: 'Aksi', width: '120px' },
]

const { list, searchRaw: search, page, total, totalPages, loading, fetchList } = useCrudTable<Rombel>({
  fetchFn: (params) => rombelApi.list(params),
  errorMessage: 'Gagal memuat data rombel',
})

const { showModal, isEdit, saving, form, openCreate, openEdit: _openEdit, handleSave } = useCrudModal<{ name: string; grade_level: string; description: string }>({
  createFn: (data) => rombelApi.create({ name: data.name, grade_level: data.grade_level || undefined, description: data.description || undefined }),
  updateFn: (id, data) => rombelApi.update(id, { name: data.name, grade_level: data.grade_level || undefined, description: data.description || undefined }),
  afterSave: fetchList,
  resetForm: () => ({ name: '', grade_level: '', description: '' }),
  successCreate: 'Rombel berhasil ditambahkan',
  successUpdate: 'Rombel berhasil diperbarui',
  errorMessage: 'Gagal menyimpan rombel',
})

function openEdit(item: Rombel) {
  _openEdit({ id: item.id, name: item.name, grade_level: item.grade_level ?? '', description: item.description ?? '' })
}

function handleDelete(item: Rombel) {
  confirmModal.ask(
    'Hapus Rombel',
    `Hapus rombel "${item.name}"?`,
    async () => {
      try {
        await rombelApi.delete(item.id)
        toast.success('Rombel berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus rombel')
      }
    },
  )
}

onMounted(fetchList)
</script>

<template>
  <BasePageHeader
    title="Rombel"
    subtitle="Kelola kelompok belajar / kelas"
    :breadcrumbs="[{ label: 'Master Data', to: '/admin' }, { label: 'Rombel' }]"
  >
    <template #actions>
      <button class="btn btn-primary" @click="openCreate">
        <i class="ti ti-plus me-1"></i>Tambah Rombel
      </button>
    </template>
  </BasePageHeader>

  <div class="card">
    <div class="card-header d-flex align-items-center gap-2 flex-wrap">
      <div class="input-group">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="search" @input="page = 1; fetchList()" class="form-control" placeholder="Cari rombel..." aria-label="Cari rombel" />
      </div>
    </div>

    <div v-if="!loading && list.length === 0" class="text-center py-5">
      <img :src="getIllustration('hybrid-work')" class="img-fluid mb-3 opacity-75" style="max-height:160px" alt="">
      <p class="text-muted">Belum ada rombel</p>
    </div>

    <BaseTable v-else :columns="columns" :loading="loading" empty="Belum ada rombel">
      <tr v-for="item in list" :key="item.id">
        <td>
          <div class="d-flex align-items-center gap-2">
            <i class="ti ti-users-group text-muted"></i>
            <span class="fw-medium">{{ item.name }}</span>
          </div>
        </td>
        <td>{{ item.grade_level ?? '–' }}</td>
        <td class="text-muted">{{ item.description ?? '–' }}</td>
        <td>
          <div class="d-flex gap-1">
            <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit rombel" @click="openEdit(item)">
              <i class="ti ti-pencil"></i>
            </button>
            <button type="button" class="btn btn-sm btn-ghost-danger" aria-label="Hapus rombel" @click="handleDelete(item)">
              <i class="ti ti-trash"></i>
            </button>
          </div>
        </td>
      </tr>
    </BaseTable>

    <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
  </div>

  <BaseModal v-if="showModal" :title="isEdit ? 'Edit Rombel' : 'Tambah Rombel'" @close="showModal = false">
    <form @submit.prevent="handleSave">
      <fieldset :disabled="saving">
        <BaseInput
          v-model="form.name"
          label="Nama Rombel *"
          type="text"
          placeholder="Contoh: XII IPA 1"
          required
        />
        <BaseInput
          v-model="form.grade_level"
          label="Tingkat"
          type="text"
          placeholder="Contoh: XII"
        />
        <BaseInput
          v-model="form.description"
          label="Deskripsi"
          type="text"
          placeholder="Opsional"
        />
      </fieldset>
    </form>
    <template #footer>
      <BaseButton variant="secondary" @click="showModal = false">Batal</BaseButton>
      <BaseButton variant="primary" :loading="saving" @click="handleSave">Simpan</BaseButton>
    </template>
  </BaseModal>

  <BaseConfirmModal
    v-if="confirmModal.show.value"
    :title="confirmModal.title.value"
    :message="confirmModal.message.value"
    :loading="confirmModal.loading.value"
    @confirm="confirmModal.confirm"
    @close="confirmModal.close"
  />
</template>
