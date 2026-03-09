<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { getIllustration } from '../../../utils/avatar'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudModal } from '@/composables/useCrudModal'
import { useDebounce } from '@/composables/useDebounce'

const toast = useToastStore()
const confirmModal = useConfirmModal()

// --- Grade level options ---
const gradeLevelGroups = [
  { label: 'TK', options: ['TK A', 'TK B'] },
  { label: 'SD', options: ['Kelas 1', 'Kelas 2', 'Kelas 3', 'Kelas 4', 'Kelas 5', 'Kelas 6'] },
  { label: 'SMP', options: ['Kelas 7', 'Kelas 8', 'Kelas 9'] },
  { label: 'SMA', options: ['Kelas 10', 'Kelas 11', 'Kelas 12'] },
  { label: 'Semester', options: ['Smt 1', 'Smt 2', 'Smt 3', 'Smt 4', 'Smt 5', 'Smt 6', 'Smt 7', 'Smt 8', 'Smt 9'] },
  { label: 'Lainnya', options: ['Lainnya'] },
]

// --- Table state (manual fetch for extra param support) ---
const list = ref<Rombel[]>([])
const searchRaw = ref('')
const search = useDebounce(searchRaw, 500)
const gradeLevel = ref('')
const page = ref(1)
const perPage = ref(20)
const total = ref(0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / perPage.value)))
const loading = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await rombelApi.list({
      page: page.value,
      per_page: perPage.value,
      search: search.value,
      grade_level: gradeLevel.value || undefined,
    })
    list.value = res.data?.data ?? []
    total.value = res.data?.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data rombel')
  } finally {
    loading.value = false
  }
}

watch(search, () => { page.value = 1; fetchList() })
watch(gradeLevel, () => { page.value = 1; fetchList() })
watch(perPage, () => { page.value = 1; fetchList() })

// --- CRUD Modal ---
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

// --- Single delete ---
function handleDelete(item: Rombel) {
  confirmModal.ask(
    'Hapus Rombel',
    `Hapus rombel "${item.name}"?`,
    async () => {
      try {
        await rombelApi.delete(item.id)
        toast.success('Rombel berhasil dihapus')
        selectedIds.value = selectedIds.value.filter(id => id !== item.id)
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus rombel')
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
    const selectableIds = new Set(selectableItems.value.map(i => i.id))
    selectedIds.value = selectedIds.value.filter(id => !selectableIds.has(id))
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
    'Hapus Rombel',
    `Hapus ${count} rombel yang dipilih?`,
    async () => {
      try {
        await rombelApi.bulkDelete([...selectedIds.value])
        toast.success(`${count} rombel berhasil dihapus`)
        selectedIds.value = []
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
      <div class="input-group" style="max-width: 280px;">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari rombel..." aria-label="Cari rombel" />
      </div>

      <select v-model="gradeLevel" class="form-select" style="max-width: 200px;">
        <option value="">Semua Tingkat</option>
        <optgroup v-for="group in gradeLevelGroups" :key="group.label" :label="group.label">
          <option v-for="opt in group.options" :key="opt" :value="opt">{{ opt }}</option>
        </optgroup>
      </select>

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
      <p class="text-muted">Belum ada rombel</p>
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
            <th>Nama Rombel</th>
            <th>Tingkat</th>
            <th style="width: 100px;">Siswa</th>
            <th>Deskripsi</th>
            <th style="width: 120px;">Aksi</th>
          </tr>
        </thead>
        <tbody>
          <template v-if="loading">
            <tr v-for="n in 5" :key="n">
              <td v-for="c in 6" :key="c">
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
                  :title="item.students_count > 0 ? `Rombel memiliki ${item.students_count} siswa` : undefined"
                  @change="toggleSelect(item.id)"
                />
              </td>
              <td>
                <div class="d-flex align-items-center gap-2">
                  <i class="ti ti-users-group text-muted"></i>
                  <span class="fw-medium">{{ item.name }}</span>
                </div>
              </td>
              <td>{{ item.grade_level ?? '–' }}</td>
              <td>
                <span class="badge bg-blue-lt">{{ item.students_count }} siswa</span>
              </td>
              <td class="text-muted">{{ item.description ?? '–' }}</td>
              <td>
                <div class="d-flex gap-1">
                  <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit rombel" @click="openEdit(item)">
                    <i class="ti ti-pencil"></i>
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost-danger"
                    :aria-label="item.students_count > 0 ? `Rombel memiliki ${item.students_count} siswa` : 'Hapus rombel'"
                    :disabled="item.students_count > 0"
                    :title="item.students_count > 0 ? `Rombel memiliki ${item.students_count} siswa` : undefined"
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
        <div class="mb-3">
          <label class="form-label">Tingkat</label>
          <select v-model="form.grade_level" class="form-select">
            <option value="">-- Pilih Tingkat --</option>
            <optgroup v-for="group in gradeLevelGroups" :key="group.label" :label="group.label">
              <option v-for="opt in group.options" :key="opt" :value="opt">{{ opt }}</option>
            </optgroup>
          </select>
        </div>
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
