<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { useToastStore } from '../../../stores/toast.store'
import client from '../../../api/client'

interface Permission {
  id: number
  name: string
  group_name: string
  description: string | null
  created_at: string
}

const toast = useToastStore()

const columns = [
  { key: 'name', label: 'Nama (Permission Name)' },
  { key: 'group_name', label: 'Grup' },
  { key: 'description', label: 'Deskripsi' },
  { key: 'actions', label: 'Aksi', width: '100px' },
]

// ─── State ─────────────────────────────────────────────────────
const list = ref<Permission[]>([])
const loading = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)
const perPage = ref(20)

const search = ref('')
const filterGroupName = ref('')
const availableGroups = ref<string[]>([])

const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)

const form = reactive({
  name: '',
  group_name: '',
  description: '',
})

const nameError = ref('')
const groupError = ref('')

// ─── Computed ─────────────────────────────────────────────────
const modalTitle = computed(() => isEdit.value ? 'Edit Grup (Tag)' : 'Tambah Grup (Tag)')

// ─── API ──────────────────────────────────────────────────────
async function fetchList() {
  loading.value = true
  try {
    const res = await client.get('/admin/permissions', {
      params: {
        page: page.value,
        per_page: perPage.value,
        search: search.value || undefined,
        group_name: filterGroupName.value || undefined,
      },
    })
    list.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
    // collect unique groups from all results for filter
    if (res.data.meta?.groups) {
      availableGroups.value = res.data.meta.groups
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data')
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    const res = await client.get('/admin/permissions/groups')
    availableGroups.value = res.data.data ?? []
  } catch {}
}

function applyFilter() {
  page.value = 1
  fetchList()
}

function openCreate() {
  isEdit.value = false
  editId.value = null
  Object.assign(form, { name: '', group_name: '', description: '' })
  nameError.value = ''
  groupError.value = ''
  showModal.value = true
}

function openEdit(item: Permission) {
  isEdit.value = true
  editId.value = item.id
  Object.assign(form, {
    name: item.name,
    group_name: item.group_name,
    description: item.description ?? '',
  })
  nameError.value = ''
  groupError.value = ''
  showModal.value = true
}

async function handleSave() {
  nameError.value = ''
  groupError.value = ''

  if (!form.name.trim()) { nameError.value = 'Nama wajib diisi'; return }
  if (!form.group_name.trim()) { groupError.value = 'Grup wajib diisi'; return }

  saving.value = true
  try {
    if (isEdit.value && editId.value) {
      await client.put(`/admin/permissions/${editId.value}`, form)
      toast.success('Grup berhasil diperbarui')
    } else {
      await client.post('/admin/permissions', form)
      toast.success('Grup berhasil ditambahkan')
    }
    showModal.value = false
    await fetchList()
    await loadGroups()
  } catch (e: any) {
    const errors = e?.response?.data?.errors
    if (errors?.name) nameError.value = errors.name[0]
    if (errors?.group_name) groupError.value = errors.group_name[0]
    if (!errors) toast.error(e?.response?.data?.message ?? 'Gagal menyimpan')
  } finally {
    saving.value = false
  }
}

async function handleDelete(item: Permission) {
  if (!confirm(`Hapus grup "${item.name}"?\n\nPeserta yang sudah memiliki tag ini akan kehilangan tag tersebut.`)) return
  try {
    await client.delete(`/admin/permissions/${item.id}`)
    toast.success(`Grup "${item.name}" berhasil dihapus`)
    await fetchList()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menghapus')
  }
}

onMounted(() => {
  loadGroups()
  fetchList()
})
</script>

<template>
  <!-- Header -->
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <h2 class="page-title">Master Grup (Tag)</h2>
        <p class="text-muted mb-0">Kelola semua grup dan status (permissions) untuk sistem.</p>
      </div>
    </div>
  </div>

  <!-- Filter Card -->
  <div class="card card-body mb-3">
    <h3 class="card-title">Filter Grup (Tag)</h3>
    <div class="row g-3">
      <div class="col-md-5">
        <label class="form-label">Cari</label>
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input
            v-model="search"
            type="text"
            class="form-control"
            placeholder="Cari nama, grup, atau deskripsi..."
            @keyup.enter="applyFilter"
          />
        </div>
      </div>
      <div class="col-md-4">
        <label class="form-label">Filter berdasarkan Grup</label>
        <select v-model="filterGroupName" class="form-select" @change="applyFilter">
          <option value="">Semua Grup</option>
          <option v-for="grp in availableGroups" :key="grp" :value="grp">{{ grp }}</option>
        </select>
      </div>
      <div class="col-md-3">
        <label class="form-label">Per Halaman</label>
        <select v-model="perPage" class="form-select" @change="applyFilter">
          <option :value="10">10 data</option>
          <option :value="20">20 data</option>
          <option :value="50">50 data</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Table Card -->
  <div class="card">
    <div class="card-header">
      <h3 class="card-title">Daftar Grup (Tag)</h3>
      <div class="ms-auto">
        <button class="btn btn-primary" @click="openCreate">
          <i class="ti ti-plus me-1"></i>Tambah Grup (Tag)
        </button>
      </div>
    </div>

    <BaseTable :columns="columns" :loading="loading" empty="Belum ada grup (tag)">
      <tr v-for="item in list" :key="item.id">
        <td>
          <div class="d-flex align-items-center gap-2">
            <span class="avatar avatar-sm bg-blue-lt text-blue">
              <i class="ti ti-tag"></i>
            </span>
            <div>
              <div class="fw-medium">{{ item.name }}</div>
            </div>
          </div>
        </td>
        <td>
          <span class="badge bg-purple-lt text-purple">{{ item.group_name }}</span>
        </td>
        <td class="text-muted small">{{ item.description ?? '—' }}</td>
        <td>
          <div class="d-flex gap-1">
            <button class="btn btn-sm btn-ghost-secondary" @click="openEdit(item)" title="Edit">
              <i class="ti ti-pencil"></i>
            </button>
            <button class="btn btn-sm btn-ghost-danger" @click="handleDelete(item)" title="Hapus">
              <i class="ti ti-trash"></i>
            </button>
          </div>
        </td>
      </tr>
    </BaseTable>

    <BasePagination
      v-if="totalPages > 1"
      :page="page"
      :total-pages="totalPages"
      :total="total"
      :per-page="perPage"
      @change="p => { page = p; fetchList() }"
    />
  </div>

  <!-- Create / Edit Modal -->
  <BaseModal v-if="showModal" :title="modalTitle" @close="showModal = false">
    <form @submit.prevent="handleSave">
      <div class="mb-3">
        <label class="form-label required">Nama (Permission Name)</label>
        <input
          v-model="form.name"
          type="text"
          class="form-control"
          :class="{ 'is-invalid': nameError }"
          placeholder="Contoh: group:olimpiade-bio"
          required
        />
        <div class="form-text">Gunakan awalan 'group:' atau 'status:'.</div>
        <div v-if="nameError" class="invalid-feedback">{{ nameError }}</div>
      </div>
      <div class="mb-3">
        <label class="form-label required">Grup</label>
        <input
          v-model="form.group_name"
          type="text"
          class="form-control"
          :class="{ 'is-invalid': groupError }"
          placeholder="Contoh: Grup Ujian"
          list="group-suggestions"
          required
        />
        <datalist id="group-suggestions">
          <option v-for="grp in availableGroups" :key="grp" :value="grp" />
        </datalist>
        <div v-if="groupError" class="invalid-feedback">{{ groupError }}</div>
      </div>
      <div class="mb-3">
        <label class="form-label">Deskripsi (Opsional)</label>
        <input
          v-model="form.description"
          type="text"
          class="form-control"
          placeholder="Contoh: Untuk peserta olimpiade biologi"
        />
      </div>
    </form>
    <template #footer>
      <button class="btn btn-secondary" @click="showModal = false">Batal</button>
      <button class="btn btn-primary" :disabled="saving" @click="handleSave">
        <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
        Simpan
      </button>
    </template>
  </BaseModal>
</template>
