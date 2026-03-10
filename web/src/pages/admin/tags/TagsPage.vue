<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'

import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { tagApi, type Tag as TagType } from '../../../api/tag.api'
import { useToastStore } from '@/stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useCrudModal } from '@/composables/useCrudModal'
import { useDebounce } from '@/composables/useDebounce'

const toast = useToastStore()
const confirmModal = useConfirmModal()

// --- Table state (manual fetch) ---
const list = ref<TagType[]>([])
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
    const res = await tagApi.list({
      page: page.value,
      per_page: perPage.value,
      search: search.value,
    })
    list.value = res.data?.data ?? []
    total.value = res.data?.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data tag')
  } finally {
    loading.value = false
  }
}

watch(search, () => { page.value = 1; fetchList() })
watch(perPage, () => { page.value = 1; fetchList() })

// --- CRUD Modal ---
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

// --- Helper: check if tag is in use ---
function isTagInUse(item: TagType): boolean {
  return item.users_count > 0 || item.exam_schedules_count > 0
}

function tagInUseTooltip(item: TagType): string | undefined {
  if (!isTagInUse(item)) return undefined
  const parts: string[] = []
  if (item.users_count > 0) parts.push(`${item.users_count} peserta`)
  if (item.exam_schedules_count > 0) parts.push(`${item.exam_schedules_count} jadwal`)
  return `Tag digunakan: ${parts.join(', ')}`
}

// --- Single delete ---
function handleDelete(item: TagType) {
  confirmModal.ask(
    'Hapus Tag',
    `Hapus tag "${item.name}"?`,
    async () => {
      try {
        await tagApi.delete(item.id)
        toast.success('Tag berhasil dihapus')
        selectedIds.value = selectedIds.value.filter(id => id !== item.id)
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus tag')
      }
    },
  )
}

// --- Bulk selection & delete ---
const selectedIds = ref<number[]>([])

const selectableItems = computed(() => list.value.filter(item => !isTagInUse(item)))

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
    'Hapus Tag',
    `Hapus ${count} tag yang dipilih?`,
    async () => {
      try {
        await tagApi.bulkDelete([...selectedIds.value])
        toast.success(`${count} tag berhasil dihapus`)
        selectedIds.value = []
        await fetchList()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menghapus tag')
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
      <button class="btn btn-primary" @click="openCreate">
        <i class="ti ti-plus me-1"></i>Tambah Tag
      </button>
    </template>
  </BasePageHeader>

  <div class="card">
    <div class="card-header d-flex align-items-center gap-2 flex-wrap">
      <div class="input-group" style="max-width: 280px;">
        <span class="input-group-text"><i class="ti ti-search"></i></span>
        <input v-model="searchRaw" class="form-control" placeholder="Cari tag..." aria-label="Cari tag" />
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
      <i class="ti ti-tag-off fs-1 mb-2 d-block opacity-50"></i>
      <p class="text-muted">Belum ada tag</p>
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
            <th>Nama Tag</th>
            <th>Warna</th>
            <th style="width: 120px;">Peserta</th>
            <th style="width: 120px;">Jadwal</th>
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
                  :disabled="isTagInUse(item)"
                  :title="tagInUseTooltip(item)"
                  @change="toggleSelect(item.id)"
                />
              </td>
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
                <span class="badge bg-blue-lt">{{ item.users_count }} peserta</span>
              </td>
              <td>
                <span class="badge bg-cyan-lt">{{ item.exam_schedules_count }} jadwal</span>
              </td>
              <td>
                <div class="d-flex gap-1">
                  <button type="button" class="btn btn-sm btn-ghost-secondary" aria-label="Edit tag" @click="openEdit(item)">
                    <i class="ti ti-pencil"></i>
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost-danger"
                    :aria-label="isTagInUse(item) ? tagInUseTooltip(item) : 'Hapus tag'"
                    :disabled="isTagInUse(item)"
                    :title="tagInUseTooltip(item)"
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
    v-if="confirmModal.show.value"
    :title="confirmModal.title.value"
    :message="confirmModal.message.value"
    :loading="confirmModal.loading.value"
    @confirm="confirmModal.confirm"
    @close="confirmModal.close"
  />
</template>
