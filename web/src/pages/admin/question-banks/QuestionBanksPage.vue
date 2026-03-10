<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import { useRouter } from 'vue-router'

import { questionBankApi, type QuestionBank } from '../../../api/question_bank.api'
import { subjectApi } from '../../../api/subject.api'
import { useToastStore } from '../../../stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useTableFilters } from '@/composables/useTableFilters'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import client from '../../../api/client'

const router = useRouter()
const toast = useToastStore()
const confirmModal = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Bank Soal' },
  { key: 'subject', label: 'Mata Pelajaran' },
  { key: 'count', label: 'Jumlah Soal' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Aksi', width: '180px' },
]

const list = ref<QuestionBank[]>([])
const { searchRaw, search, page, total, totalPages, loading } = useTableFilters(fetchList)
const subjectFilter = ref('')
const statusFilter = ref('')

watch(subjectFilter, () => { page.value = 1; fetchList() })
watch(statusFilter, () => { page.value = 1; fetchList() })
const subjects = ref<{ id: number; name: string; code: string }[]>([])

const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)
const form = reactive({ name: '', subject_id: '' as string | number, description: '' })

let searchController: AbortController | null = null

async function fetchList() {
  searchController?.abort()
  searchController = new AbortController()
  loading.value = true
  try {
    const res = await questionBankApi.list({
      page: page.value, per_page: 20,
      search: search.value,
      subject_id: subjectFilter.value ? Number(subjectFilter.value) : undefined,
      status: statusFilter.value || undefined,
    }, { signal: searchController.signal })
    list.value = res.data.data ?? []
    total.value = res.data.meta?.total ?? 0
  } catch (e: unknown) {
    if ((e as any)?.code === 'ERR_CANCELED') return
    throw e
  } finally {
    loading.value = false
  }
}

async function fetchSubjects() {
  try {
    const res = await subjectApi.listAll()
    subjects.value = res.data.data ?? []
  } catch (e) {
    console.warn('Failed to load subjects:', e)
  }
}

function openCreate() {
  isEdit.value = false; editId.value = null
  Object.assign(form, { name: '', subject_id: '', description: '' })
  showModal.value = true
}

function openEdit(item: QuestionBank) {
  isEdit.value = true; editId.value = item.id
  Object.assign(form, { name: item.name, subject_id: item.subject_id ?? '', description: item.description })
  showModal.value = true
}

async function handleSave() {
  saving.value = true
  try {
    const payload = {
      name: form.name,
      subject_id: form.subject_id ? Number(form.subject_id) : undefined,
      description: form.description,
    }
    if (isEdit.value && editId.value) await questionBankApi.update(editId.value, payload)
    else await questionBankApi.create(payload)
    showModal.value = false
    toast.success('Bank soal berhasil disimpan')
    await fetchList()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan bank soal')
  } finally {
    saving.value = false
  }
}

function handleDelete(item: QuestionBank) {
  confirmModal.ask(
    'Hapus Bank Soal',
    `Hapus bank soal "${item.name}"? Semua soal di dalamnya akan ikut terhapus.`,
    async () => {
      try {
        await questionBankApi.delete(item.id)
        toast.success('Bank soal berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus bank soal')
      }
    },
  )
}

function openDetail(item: QuestionBank) {
  router.push(`/admin/question-banks/${item.id}`)
}

const toggling = ref<number | null>(null)

async function handleToggleStatus(item: QuestionBank) {
  toggling.value = item.id
  try {
    await questionBankApi.toggleStatus(item.id)
    item.status = item.status === 'active' ? 'inactive' : 'active'
    toast.success(`Status bank soal berhasil diubah menjadi ${item.status === 'active' ? 'aktif' : 'nonaktif'}`)
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengubah status')
  } finally {
    toggling.value = null
  }
}

const cloning = ref<number | null>(null)

function handleClone(item: QuestionBank) {
  confirmModal.ask(
    'Duplikasi Bank Soal',
    `Duplikasi bank soal "${item.name}"? Semua soal akan ikut disalin.`,
    async () => {
      cloning.value = item.id
      try {
        await client.post(`/question-banks/${item.id}/clone`)
        toast.success('Bank soal berhasil diduplikasi')
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menduplikasi bank soal')
      } finally {
        cloning.value = null
      }
    },
  )
}

onMounted(() => { fetchList(); fetchSubjects() })
</script>

<template>
    <BasePageHeader
      title="Bank Soal"
      subtitle="Kelola kumpulan soal ujian berdasarkan mata pelajaran"
      :breadcrumbs="[{ label: 'Bank Soal' }]"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreate"><i class="ti ti-plus"></i>
          Buat Bank Soal</button>
      </template>
    </BasePageHeader>

    <div class="card">
      <div class="card-header d-flex align-items-center gap-2 flex-wrap">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchRaw" class="form-control" placeholder="Cari bank soal..." />
        </div>
        <select v-model="subjectFilter" class="form-select form-select-sm">
          <option value="">Semua Mapel</option>
          <option v-for="s in subjects" :key="s.id" :value="s.id">{{ s.name }}</option>
        </select>
        <select v-model="statusFilter" class="form-select form-select-sm">
          <option value="">Semua Status</option>
          <option value="active">Aktif</option>
          <option value="inactive">Nonaktif</option>
        </select>
      </div>

      <BaseTable :columns="columns" :loading="loading" :row-count="list.length" empty="Belum ada bank soal">
        <template #empty>
          <i class="ti ti-book-off fs-1 mb-2 d-block opacity-50"></i>
          <p class="mb-0">Belum ada bank soal</p>
        </template>
        <template #default>
          <tr v-for="item in list" :key="item.id">
            <td>
              <div class="d-flex align-items-start gap-2" style="cursor:pointer" @click="openDetail(item)">
                <i class="ti ti-books"></i>
                <div>
                  <p class="fw-medium d-flex align-items-center gap-1">
                    {{ item.name }}
                    <span v-if="item.is_locked" class="badge bg-warning-lt text-warning"><i class="ti ti-lock me-1"></i>Terkunci</span>
                  </p>
                  <p v-if="item.description" class="text-muted small mt-1 text-truncate" style="max-width:280px">{{ item.description }}</p>
                </div>
              </div>
            </td>
            <td>
              <span v-if="item.subject" class="badge bg-primary-lt text-primary fw-medium">{{ item.subject.name }}</span>
              <span v-else class="text-muted">–</span>
            </td>
            <td>
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-help-circle"></i>
                <span class="fw-medium">{{ item.question_count }}</span>
                <span class="text-muted text-sm">soal</span>
              </div>
            </td>
            <td>
              <label class="form-check form-switch mb-0" style="cursor:pointer" :title="item.status === 'active' ? 'Nonaktifkan' : 'Aktifkan'">
                <input
                  class="form-check-input"
                  type="checkbox"
                  :checked="item.status === 'active'"
                  :disabled="toggling === item.id"
                  @change="handleToggleStatus(item)"
                  style="cursor:pointer"
                >
                <span class="form-check-label" :class="item.status === 'active' ? 'text-success' : 'text-secondary'">
                  {{ item.status === 'active' ? 'Aktif' : 'Nonaktif' }}
                </span>
              </label>
            </td>
            <td>
              <div class="d-flex gap-1">
                <button class="btn btn-sm btn-ghost-primary" :aria-label="`Kelola soal ${item.name}`" @click="openDetail(item)">
                  <i class="ti ti-chevron-right"></i>
                </button>
                <button class="btn btn-sm btn-ghost-secondary" :aria-label="`Edit bank soal ${item.name}`" @click="openEdit(item)">
                  <i class="ti ti-pencil"></i>
                </button>
                <button
                  class="btn btn-sm btn-ghost-teal"
                  @click="handleClone(item)"
                  :disabled="cloning === item.id"
                  :aria-label="`Duplikasi bank soal ${item.name}`"
                >
                  <span v-if="cloning === item.id" class="spinner-border spinner-border-sm"></span>
                  <i v-else class="ti ti-copy"></i>
                </button>
                <button class="btn btn-sm btn-ghost-danger" :aria-label="`Hapus bank soal ${item.name}`" @click="handleDelete(item)">
                  <i class="ti ti-trash"></i>
                </button>
              </div>
            </td>
          </tr>
        </template>
      </BaseTable>

      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
    </div>

    <BaseModal v-if="showModal" :title="isEdit ? 'Edit Bank Soal' : 'Buat Bank Soal'" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <fieldset :disabled="saving">
        <div class="mb-3">
              <label class="form-label">Nama Bank Soal *</label>
              <input class="form-control" type="text" v-model="form.name" placeholder="Contoh: Matematika Kelas XII Semester 1" required />
            </div>
        <div class="mb-3">
          <label class="form-label">Mata Pelajaran</label>
          <select v-model="form.subject_id" class="form-select">
            <option value="">Tidak terkait mapel</option>
            <option v-for="s in subjects" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
        </div>
        <div class="mb-3">
              <label class="form-label">Deskripsi</label>
              <input class="form-control" type="text" v-model="form.description" placeholder="Keterangan singkat (opsional)" />
            </div>
        </fieldset>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="showModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSave"><span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan</button>
      </template>
    </BaseModal>

    <BaseConfirmModal
      v-if="confirmModal.show.value"
      :title="confirmModal.title.value"
      :message="confirmModal.message.value"
      @confirm="confirmModal.confirm"
      @close="confirmModal.close"
      :loading="confirmModal.loading.value"
    />
</template>

