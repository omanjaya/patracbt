<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { getIllustration } from '../../../utils/avatar'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { roomApi, type Room } from '../../../api/room.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudModal } from '@/composables/useCrudModal'
import { useDebounce } from '@/composables/useDebounce'

const toast = useToastStore()
const confirmModal = useConfirmModal()

// --- Table state (manual fetch for extra param support) ---
const list = ref<Room[]>([])
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
    const res = await roomApi.list({
      page: page.value,
      per_page: perPage.value,
      search: search.value,
    })
    list.value = res.data?.data ?? []
    total.value = res.data?.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data ruangan')
  } finally {
    loading.value = false
  }
}

watch(search, () => { page.value = 1; fetchList() })
watch(perPage, () => { page.value = 1; fetchList() })

// --- CRUD Modal ---
const formErrors = reactive<Record<string, string>>({ name: '', capacity: '' })

const { showModal, isEdit, saving, form, openCreate: _openCreate, openEdit: _openEdit, handleSave: _handleSave } = useCrudModal<{ name: string; capacity: number; description: string }>({
  createFn: (data) => roomApi.create({ name: data.name, capacity: data.capacity, description: data.description || undefined }),
  updateFn: (id, data) => roomApi.update(id, { name: data.name, capacity: data.capacity, description: data.description || undefined }),
  afterSave: fetchList,
  resetForm: () => ({ name: '', capacity: 30, description: '' }),
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
  _openEdit({ id: item.id, name: item.name, capacity: item.capacity, description: item.description ?? '' })
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

// --- Single delete ---
function handleDelete(item: Room) {
  confirmModal.ask(
    'Hapus Ruangan',
    `Hapus ruangan "${item.name}"?`,
    async () => {
      try {
        await roomApi.delete(item.id)
        toast.success('Ruangan berhasil dihapus')
        selectedIds.value = selectedIds.value.filter(id => id !== item.id)
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus ruangan')
      }
    },
  )
}

// --- Bulk selection & delete ---
const selectedIds = ref<number[]>([])

const selectableItems = computed(() => list.value.filter(item => item.students_count === 0))

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
    'Hapus Ruangan',
    `Hapus ${count} ruangan yang dipilih?`,
    async () => {
      try {
        await roomApi.bulkDelete([...selectedIds.value])
        toast.success(`${count} ruangan berhasil dihapus`)
        selectedIds.value = []
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus ruangan')
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
      <div class="input-group" style="max-width: 280px;">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari ruangan..." aria-label="Cari ruangan" />
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
      <p class="text-muted">Belum ada ruangan</p>
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
            <th>Nama Ruangan</th>
            <th style="width: 140px;">Peserta</th>
            <th>Deskripsi</th>
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
                  :disabled="item.students_count > 0"
                  :title="item.students_count > 0 ? `Ruangan memiliki ${item.students_count} peserta` : undefined"
                  @change="toggleSelect(item.id)"
                />
              </td>
              <td>
                <div class="d-flex align-items-center gap-2">
                  <i class="ti ti-door text-muted"></i>
                  <span class="fw-medium">{{ item.name }}</span>
                </div>
              </td>
              <td>
                <span class="badge bg-blue-lt">{{ item.students_count }}/{{ item.capacity }} peserta</span>
              </td>
              <td class="text-muted">{{ item.description ?? '–' }}</td>
              <td>
                <div class="d-flex gap-1">
                  <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit ruangan" @click="openEdit(item)">
                    <i class="ti ti-pencil"></i>
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost-danger"
                    :aria-label="item.students_count > 0 ? `Ruangan memiliki ${item.students_count} peserta` : 'Hapus ruangan'"
                    :disabled="item.students_count > 0"
                    :title="item.students_count > 0 ? `Ruangan memiliki ${item.students_count} peserta` : undefined"
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
