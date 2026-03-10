<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { useToastStore } from '../../../stores/toast.store'
import client from '../../../api/client'

interface Permission {
  id: number
  name: string
  group_name: string
  description: string
}

interface PermissionGroup {
  id: number
  name: string
  group_name: string
}

interface UserRow {
  id: number
  name: string
  username: string
  nis: string | null
  rombel: string | null
  status: string
  tags: { id: number; name: string; group_name: string }[]
}

const toast = useToastStore()

const columns = [
  { key: 'checkbox', label: '', width: '40px' },
  { key: 'user', label: 'Nama / Username' },
  { key: 'nis', label: 'NIS / Rombel' },
  { key: 'tags', label: 'Grup (Tag) Saat Ini' },
  { key: 'status', label: 'Status' },
]

// ─── State ─────────────────────────────────────────────────────
const list = ref<UserRow[]>([])
const loading = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)
const perPage = ref(20)

const search = ref('')
const filterPermissionId = ref<number | ''>('')
const filterNoPermissionId = ref<number | ''>('')

const allPermissions = ref<Permission[]>([])
const permissionGroups = ref<PermissionGroup[]>([])

const selectedIds = ref<Set<number>>(new Set())
const selectAll = ref(false)

// Assign/Remove modal
const showAssignModal = ref(false)
const showRemoveModal = ref(false)
const assignForm = reactive({ permission_id: '' as number | '', action: 'assign' as 'assign' | 'remove' })
const saving = ref(false)

// Import modal
const showImportModal = ref(false)
const importFile = ref<File | null>(null)
const importing = ref(false)

// ─── Computed ─────────────────────────────────────────────────
const hasSelection = computed(() => selectedIds.value.size > 0)
const selectedCount = computed(() => selectedIds.value.size)

// ─── Watchers ─────────────────────────────────────────────────
watch(selectAll, (val) => {
  if (val) {
    list.value.forEach(u => selectedIds.value.add(u.id))
  } else {
    selectedIds.value.clear()
  }
})

// ─── API ──────────────────────────────────────────────────────
async function loadAllPermissions() {
  try {
    const res = await client.get('/admin/permissions/all')
    allPermissions.value = res.data.data ?? []
    permissionGroups.value = res.data.data ?? []
  } catch (e) {
    console.warn('Failed to load permissions:', e)
  }
}

async function fetchList() {
  loading.value = true
  try {
    const res = await client.get('/admin/user-permissions', {
      params: {
        page: page.value,
        per_page: perPage.value,
        search: search.value || undefined,
        permission_id: filterPermissionId.value || undefined,
        no_permission_id: filterNoPermissionId.value || undefined,
      },
    })
    list.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data')
  } finally {
    loading.value = false
  }
}

function applyFilter() {
  page.value = 1
  selectedIds.value.clear()
  selectAll.value = false
  fetchList()
}

function toggleSelect(id: number) {
  if (selectedIds.value.has(id)) {
    selectedIds.value.delete(id)
  } else {
    selectedIds.value.add(id)
  }
  selectAll.value = selectedIds.value.size === list.value.length
}

function openAssignModal() {
  assignForm.permission_id = ''
  assignForm.action = 'assign'
  showAssignModal.value = true
}

function openRemoveModal() {
  assignForm.permission_id = ''
  assignForm.action = 'remove'
  showRemoveModal.value = true
}

async function handleAssign() {
  if (!assignForm.permission_id) return
  saving.value = true
  try {
    await client.post('/admin/user-permissions/assign', {
      user_ids: Array.from(selectedIds.value),
      permission_id: assignForm.permission_id,
    })
    toast.success(`Tag berhasil ditetapkan ke ${selectedCount.value} peserta`)
    showAssignModal.value = false
    selectedIds.value.clear()
    selectAll.value = false
    await fetchList()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menetapkan tag')
  } finally {
    saving.value = false
  }
}

async function handleRemove() {
  if (!assignForm.permission_id) return
  saving.value = true
  try {
    await client.post('/admin/user-permissions/remove', {
      user_ids: Array.from(selectedIds.value),
      permission_id: assignForm.permission_id,
    })
    toast.success(`Tag berhasil dihapus dari ${selectedCount.value} peserta`)
    showRemoveModal.value = false
    selectedIds.value.clear()
    selectAll.value = false
    await fetchList()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menghapus tag')
  } finally {
    saving.value = false
  }
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  importFile.value = input.files?.[0] ?? null
}

async function handleImport() {
  if (!importFile.value) return
  importing.value = true
  try {
    const formData = new FormData()
    formData.append('file', importFile.value)
    await client.post('/admin/user-permissions/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    toast.success('Import berhasil')
    showImportModal.value = false
    importFile.value = null
    await fetchList()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Import gagal')
  } finally {
    importing.value = false
  }
}

function downloadTemplate() {
  window.open('/api/admin/user-permissions/export-template', '_blank')
}

onMounted(() => {
  loadAllPermissions()
  fetchList()
})
</script>

<template>
  <!-- Header -->
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <h2 class="page-title">Pengaitan Grup Peserta</h2>
        <p class="text-muted mb-0">Tetapkan "Tag" (Grup Ujian / Status) ke banyak peserta sekaligus.</p>
      </div>
    </div>
  </div>

  <!-- Filter Card -->
  <div class="card card-body mb-3">
    <h3 class="card-title">Filter Peserta</h3>
    <div class="row g-3">
      <div class="col-md-3">
        <label class="form-label">Cari Peserta</label>
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input
            v-model="search"
            type="text"
            class="form-control"
            placeholder="Cari NIS, nama, username..."
            @keyup.enter="applyFilter"
          />
        </div>
      </div>
      <div class="col-md-3">
        <label class="form-label">Filter: Punya Tag...</label>
        <select v-model="filterPermissionId" class="form-select">
          <option value="">Semua Peserta</option>
          <option v-for="p in allPermissions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
      <div class="col-md-3">
        <label class="form-label">Filter: Tidak Punya Tag...</label>
        <select v-model="filterNoPermissionId" class="form-select">
          <option value="">Semua Peserta</option>
          <option v-for="p in allPermissions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
      <div class="col-md-2">
        <label class="form-label">Per Halaman</label>
        <select v-model="perPage" class="form-select">
          <option :value="10">10 data</option>
          <option :value="20">20 data</option>
          <option :value="50">50 data</option>
          <option :value="100">100 data</option>
        </select>
      </div>
      <div class="col-md-1 d-flex align-items-end">
        <button class="btn btn-primary w-100" @click="applyFilter">
          <i class="ti ti-filter"></i>
        </button>
      </div>
    </div>
  </div>

  <!-- Table Card -->
  <div class="card">
    <!-- Mass Action Bar -->
    <div v-if="hasSelection" class="card-header bg-primary-lt">
      <div class="d-flex align-items-center gap-2 flex-wrap">
        <span class="fw-bold">
          <span>{{ selectedCount }}</span> Peserta terpilih
        </span>
        <button class="btn btn-primary btn-sm" @click="openAssignModal">
          <i class="ti ti-tag me-1"></i>Tetapkan Grup (Tag)
        </button>
        <button class="btn btn-outline-danger btn-sm" @click="openRemoveModal">
          <i class="ti ti-tag-off me-1"></i>Hapus Grup (Tag)
        </button>
      </div>
    </div>

    <!-- Card Header -->
    <div class="card-header">
      <h3 class="card-title">Daftar Peserta</h3>
      <div class="ms-auto">
        <button class="btn btn-primary" @click="showImportModal = true">
          <i class="ti ti-upload me-1"></i>Import Pengaitan
        </button>
      </div>
    </div>

    <BaseTable :columns="columns" :loading="loading" empty="Belum ada data peserta">
      <template v-if="!loading">
        <tr v-if="list.length === 0">
          <td :colspan="columns.length" class="text-center text-muted py-5">
            <i class="ti ti-users-off fs-1 mb-2 d-block opacity-50"></i>
            Belum ada data peserta
          </td>
        </tr>
        <tr v-for="user in list" :key="user.id">
          <td>
            <input
              class="form-check-input"
              type="checkbox"
              :checked="selectedIds.has(user.id)"
              @change="toggleSelect(user.id)"
            />
          </td>
          <td>
            <div class="d-flex flex-column">
              <span class="fw-medium">{{ user.name }}</span>
              <span class="text-muted small">@{{ user.username }}</span>
            </div>
          </td>
          <td>
            <div class="d-flex flex-column">
              <span class="text-muted small">{{ user.nis ?? '—' }}</span>
              <span v-if="user.rombel" class="badge bg-azure-lt mt-1" style="font-size:0.7rem; width:fit-content;">
                {{ user.rombel }}
              </span>
            </div>
          </td>
          <td>
            <div class="d-flex flex-wrap gap-1">
              <span
                v-for="tag in user.tags"
                :key="tag.id"
                class="badge bg-blue-lt text-blue"
              >
                {{ tag.name }}
              </span>
              <span v-if="user.tags.length === 0" class="text-muted small">—</span>
            </div>
          </td>
          <td>
            <span
              class="badge"
              :class="user.status === 'active' ? 'bg-green-lt text-green' : 'bg-secondary-lt text-secondary'"
            >
              {{ user.status === 'active' ? 'Aktif' : 'Nonaktif' }}
            </span>
          </td>
        </tr>
      </template>
    </BaseTable>

    <!-- Select all row (above pagination) -->
    <div v-if="list.length > 0 && !loading" class="px-3 py-2 border-top d-flex align-items-center gap-2">
      <input
        class="form-check-input"
        type="checkbox"
        id="select-all-checkbox"
        v-model="selectAll"
      />
      <label class="form-check-label small text-muted" for="select-all-checkbox">
        Pilih semua {{ list.length }} data di halaman ini
      </label>
    </div>

    <BasePagination
      v-if="totalPages > 1"
      :page="page"
      :total-pages="totalPages"
      :total="total"
      :per-page="perPage"
      @change="p => { page = p; fetchList() }"
    />
  </div>

  <!-- Assign Modal -->
  <BaseModal v-if="showAssignModal" title="Tetapkan Grup (Tag)" @close="showAssignModal = false">
    <p class="text-muted mb-3">Tetapkan tag ke <strong>{{ selectedCount }}</strong> peserta terpilih.</p>
    <div class="mb-3">
      <label class="form-label required">Pilih Tag</label>
      <select v-model="assignForm.permission_id" class="form-select">
        <option value="" disabled>-- Pilih Tag --</option>
        <optgroup v-for="grp in [...new Set(allPermissions.map(p => p.group_name))]" :key="grp" :label="grp">
          <option
            v-for="p in allPermissions.filter(p => p.group_name === grp)"
            :key="p.id"
            :value="p.id"
          >
            {{ p.name }}
          </option>
        </optgroup>
      </select>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="showAssignModal = false">Batal</button>
      <button
        class="btn btn-primary"
        :disabled="saving || !assignForm.permission_id"
        @click="handleAssign"
      >
        <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
        Tetapkan
      </button>
    </template>
  </BaseModal>

  <!-- Remove Modal -->
  <BaseModal v-if="showRemoveModal" title="Hapus Grup (Tag)" @close="showRemoveModal = false">
    <p class="text-muted mb-3">Hapus tag dari <strong>{{ selectedCount }}</strong> peserta terpilih.</p>
    <div class="mb-3">
      <label class="form-label required">Pilih Tag yang akan dihapus</label>
      <select v-model="assignForm.permission_id" class="form-select">
        <option value="" disabled>-- Pilih Tag --</option>
        <optgroup v-for="grp in [...new Set(allPermissions.map(p => p.group_name))]" :key="grp" :label="grp">
          <option
            v-for="p in allPermissions.filter(p => p.group_name === grp)"
            :key="p.id"
            :value="p.id"
          >
            {{ p.name }}
          </option>
        </optgroup>
      </select>
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="showRemoveModal = false">Batal</button>
      <button
        class="btn btn-danger"
        :disabled="saving || !assignForm.permission_id"
        @click="handleRemove"
      >
        <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
        Hapus Tag
      </button>
    </template>
  </BaseModal>

  <!-- Import Modal -->
  <BaseModal v-if="showImportModal" title="Import Pengaitan Grup" @close="showImportModal = false">
    <p>Upload file Excel (.xlsx) untuk menetapkan atau menghapus grup (tag) dari peserta secara massal.</p>
    <p class="form-text">
      Template pengaitan group bisa diunduh
      <a href="#" @click.prevent="downloadTemplate">di sini</a>.
    </p>
    <div class="mb-3 mt-3">
      <label class="form-label required" for="import-file">Pilih File Excel</label>
      <input
        id="import-file"
        type="file"
        class="form-control"
        accept=".xlsx,.xls,.csv"
        @change="onFileChange"
      />
    </div>
    <template #footer>
      <button class="btn btn-secondary" @click="showImportModal = false">Batal</button>
      <button
        class="btn btn-primary"
        :disabled="importing || !importFile"
        @click="handleImport"
      >
        <span v-if="importing" class="spinner-border spinner-border-sm me-1"></span>
        Mulai Import
      </button>
    </template>
  </BaseModal>
</template>
