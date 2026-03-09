<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { getIllustration } from '../../../utils/avatar'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { subjectApi, type Subject } from '../../../api/subject.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudModal } from '@/composables/useCrudModal'
import { useDebounce } from '@/composables/useDebounce'

const toast = useToastStore()
const confirmModal = useConfirmModal()

// --- Table state (manual fetch) ---
const list = ref<Subject[]>([])
const searchRaw = ref('')
const search = useDebounce(searchRaw, 500)
const page = ref(1)
const perPage = ref(20)
const total = ref(0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / perPage.value)))
const loading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await subjectApi.list({
      page: page.value,
      per_page: perPage.value,
      search: search.value,
    })
    list.value = res.data?.data ?? []
    total.value = res.data?.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data mata pelajaran')
  } finally {
    loading.value = false
  }
}

watch(search, () => { page.value = 1; fetchList() })
watch(perPage, () => { page.value = 1; fetchList() })

// --- CRUD Modal ---
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

// --- Helper: check if subject is in use ---
function isSubjectInUse(item: Subject): boolean {
  return item.question_banks_count > 0
}

function subjectInUseTooltip(item: Subject): string | undefined {
  if (!isSubjectInUse(item)) return undefined
  return `Mapel memiliki ${item.question_banks_count} bank soal`
}

// --- Single delete ---
function handleDelete(item: Subject) {
  confirmModal.ask(
    'Hapus Mata Pelajaran',
    `Hapus mata pelajaran "${item.name}"?`,
    async () => {
      try {
        await subjectApi.delete(item.id)
        toast.success('Mata pelajaran berhasil dihapus')
        selectedIds.value = selectedIds.value.filter(id => id !== item.id)
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus mata pelajaran')
      }
    },
  )
}

// --- Bulk selection & delete ---
const selectedIds = ref<number[]>([])

const selectableItems = computed(() => list.value.filter(item => !isSubjectInUse(item)))

const allSelectableSelected = computed(() =>
  selectableItems.value.length > 0 && selectableItems.value.every(item => selectedIds.value.includes(item.id))
)

const someSelected = computed(() =>
  selectedIds.value.length > 0 && !allSelectableSelected.value
)

function toggleSelectAll() {
  if (allSelectableSelected.value) {
    const selectableIdSet = new Set(selectableItems.value.map(i => i.id))
    selectedIds.value = selectedIds.value.filter(id => !selectableIdSet.has(id))
  } else {
    const currentIds = new Set(selectedIds.value)
    for (const item of selectableItems.value) {
      currentIds.add(item.id)
    }
    selectedIds.value = Array.from(currentIds)
  }
}

function toggleSelect(id: number) {
  const idx = selectedIds.value.indexOf(id)
  if (idx === -1) {
    selectedIds.value.push(id)
  } else {
    selectedIds.value.splice(idx, 1)
  }
}

function handleBulkDelete() {
  const count = selectedIds.value.length
  confirmModal.ask(
    'Hapus Mata Pelajaran',
    `Hapus ${count} mata pelajaran yang dipilih?`,
    async () => {
      try {
        await subjectApi.bulkDelete([...selectedIds.value])
        toast.success(`${count} mata pelajaran berhasil dihapus`)
        selectedIds.value = []
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus mata pelajaran')
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
      <div class="input-group" style="max-width: 280px;">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari mata pelajaran..." aria-label="Cari mata pelajaran" />
      </div>

      <select v-model.number="perPage" class="form-select" style="max-width: 130px;">
        <option :value="10">10 / hal</option>
        <option :value="20">20 / hal</option>
        <option :value="50">50 / hal</option>
        <option :value="100">100 / hal</option>
      </select>
    </div>

    <!-- Floating action bar for bulk selection -->
    <div v-if="selectedIds.length > 0" class="card-body py-2 bg-blue-lt d-flex align-items-center gap-3">
      <span class="fw-medium">{{ selectedIds.length }} dipilih</span>
      <button class="btn btn-sm btn-danger" @click="handleBulkDelete">
        <i class="ti ti-trash me-1"></i>Hapus
      </button>
    </div>

    <div v-if="!loading && list.length === 0" class="text-center py-5">
      <img :src="getIllustration('hybrid-work')" class="img-fluid mb-3 opacity-75" style="max-height:160px" alt="">
      <p class="text-muted">Belum ada mata pelajaran</p>
    </div>

    <div v-else class="table-responsive">
      <table class="table table-vcenter card-table">
        <thead>
          <tr>
            <th style="width: 40px;">
              <input
                type="checkbox"
                class="form-check-input m-0 align-middle"
                :checked="allSelectableSelected && selectableItems.length > 0"
                :indeterminate="someSelected"
                :disabled="selectableItems.length === 0"
                aria-label="Pilih semua"
                @change="toggleSelectAll"
              />
            </th>
            <th>Nama Mata Pelajaran</th>
            <th>Kode</th>
            <th style="width: 140px;">Bank Soal</th>
            <th style="width: 120px;">Aksi</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="loading">
            <tr v-for="n in 5" :key="n">
              <td v-for="c in 5" :key="c">
                <div class="placeholder-glow"><span class="placeholder col-8" /></div>
              </td>
            </tr>
          </template>
          <template v-else>
            <tr v-for="item in list" :key="item.id">
              <td>
                <input
                  type="checkbox"
                  class="form-check-input m-0 align-middle"
                  :checked="selectedIds.includes(item.id)"
                  :disabled="isSubjectInUse(item)"
                  :title="subjectInUseTooltip(item)"
                  @change="toggleSelect(item.id)"
                />
              </td>
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
                <span class="badge bg-blue-lt">{{ item.question_banks_count }} bank soal</span>
              </td>
              <td>
                <div class="d-flex gap-1">
                  <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit mata pelajaran" @click="openEdit(item)">
                    <i class="ti ti-pencil"></i>
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost-danger"
                    :aria-label="isSubjectInUse(item) ? subjectInUseTooltip(item) : 'Hapus mata pelajaran'"
                    :disabled="isSubjectInUse(item)"
                    :title="subjectInUseTooltip(item)"
                    @click="handleDelete(item)"
                  >
                    <i class="ti ti-trash"></i>
                  </button>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="perPage" @change="p => { page = p; fetchList() }" />
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
    v-if="confirmModal.show.value"
    :title="confirmModal.title.value"
    :message="confirmModal.message.value"
    :loading="confirmModal.loading.value"
    @confirm="confirmModal.confirm"
    @close="confirmModal.close"
  />
</template>
