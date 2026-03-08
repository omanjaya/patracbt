<script setup lang="ts">
import { ref, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import { tagApi, type Tag as TagType } from '../../../api/tag.api'
import { userApi, type UserItem } from '../../../api/user.api'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()

const columns = [
  { key: 'name', label: 'Nama' },
  { key: 'username', label: 'Username' },
  { key: 'actions', label: 'Aksi', width: '80px' },
]

const tags = ref<TagType[]>([])
const selectedTag = ref<TagType | null>(null)
const members = ref<UserItem[]>([])
const loadingTags = ref(false)
const loadingMembers = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

const showAddModal = ref(false)
const allUsers = ref<UserItem[]>([])
const selectedUserIds = ref<number[]>([])
const searchUser = ref('')
const adding = ref(false)

// Confirm modal state
const confirmModal = ref(false)
const confirmLoading = ref(false)
const pendingRemoveUser = ref<UserItem | null>(null)

// Import/Export state
const importing = ref(false)
const importResult = ref<{ total_rows: number; assigned: number; removed: number; skipped: number; errors: { row: number; column: string; message: string }[] } | null>(null)
const showImportResult = ref(false)

async function fetchTags() {
  loadingTags.value = true
  try {
    const res = await tagApi.listAll()
    tags.value = res.data.data ?? []
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat daftar tag')
  } finally {
    loadingTags.value = false
  }
}

async function selectTag(t: TagType) {
  selectedTag.value = t
  page.value = 1
  await fetchMembers()
}

async function fetchMembers() {
  if (!selectedTag.value) return
  loadingMembers.value = true
  try {
    const res = await userApi.list({ page: page.value, per_page: 20, tag_id: selectedTag.value.id })
    members.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat daftar anggota')
  } finally {
    loadingMembers.value = false
  }
}

async function openAddModal() {
  try {
    const res = await userApi.list({ per_page: 200 })
    allUsers.value = res.data.data ?? []
    selectedUserIds.value = []
    showAddModal.value = true
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat daftar user')
  }
}

async function handleAdd() {
  if (!selectedTag.value || !selectedUserIds.value.length) return
  adding.value = true
  try {
    await tagApi.assignUsers(selectedTag.value.id, selectedUserIds.value)
    toast.success('User berhasil ditambahkan ke tag')
    showAddModal.value = false
    await fetchMembers()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menambahkan user ke tag')
  } finally {
    adding.value = false
  }
}

function askRemove(user: UserItem) {
  pendingRemoveUser.value = user
  confirmModal.value = true
}

async function doRemove() {
  if (!selectedTag.value || !pendingRemoveUser.value) return
  confirmLoading.value = true
  try {
    await tagApi.removeUser(selectedTag.value.id, pendingRemoveUser.value.id)
    toast.success('User berhasil dihapus dari tag')
    await fetchMembers()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menghapus user dari tag')
  } finally {
    confirmLoading.value = false
    confirmModal.value = false
  }
}

const filteredUsers = () => allUsers.value.filter(u =>
  u.name.toLowerCase().includes(searchUser.value.toLowerCase()) ||
  u.username.toLowerCase().includes(searchUser.value.toLowerCase())
)

async function handleImportFile(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  importing.value = true
  importResult.value = null
  try {
    const res = await tagApi.importUsers(file)
    importResult.value = res.data.data
    showImportResult.value = true
    if (selectedTag.value) await fetchMembers()
  } catch (err: any) {
    toast.error(err?.response?.data?.error || 'Gagal mengimpor file')
  } finally {
    importing.value = false
    target.value = ''
  }
}

async function downloadTemplate() {
  try {
    const res = await tagApi.exportTemplate()
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', 'template-import-tag.xlsx')
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch {
    toast.error('Gagal mengunduh template')
  }
}

onMounted(fetchTags)
</script>

<template>
    <BasePageHeader
      title="Pengaitan Grup (Tag)"
      subtitle="Kelola pengelompokan user berdasarkan tag"
      :breadcrumbs="[{ label: 'Master Data', to: '/admin/user-tags' }, { label: 'Tag Peserta' }]"
    >
      <template #actions>
        <button class="btn btn-outline-primary" @click="downloadTemplate">
          <i class="ti ti-download"></i> Download Template
        </button>
        <label class="btn btn-primary mb-0" :class="{ disabled: importing }">
          <i class="ti ti-file-import"></i>
          <span v-if="importing">Mengimpor...</span>
          <span v-else>Import dari Excel</span>
          <input type="file" accept=".xlsx,.xls" class="d-none" @change="handleImportFile" :disabled="importing" />
        </label>
      </template>
    </BasePageHeader>

    <!-- Import Result Alert -->
    <div v-if="showImportResult && importResult" class="mb-3">
      <div class="card">
        <div class="card-body">
          <div class="d-flex align-items-center justify-content-between mb-2">
            <h4 class="card-title mb-0">Hasil Import</h4>
            <button class="btn-close" @click="showImportResult = false"></button>
          </div>
          <div class="d-flex gap-3 mb-2">
            <span class="badge bg-success-lt text-success">Ditambahkan: {{ importResult.assigned }}</span>
            <span class="badge bg-danger-lt text-danger">Dihapus: {{ importResult.removed }}</span>
            <span class="badge bg-warning-lt text-warning">Dilewati: {{ importResult.skipped }}</span>
            <span class="badge bg-secondary-lt text-secondary">Total Baris: {{ importResult.total_rows }}</span>
          </div>
          <div v-if="importResult.errors && importResult.errors.length" class="mt-2">
            <p class="text-danger small mb-1">Error ({{ importResult.errors.length }}):</p>
            <div class="border rounded overflow-auto" style="max-height: 10rem">
              <table class="table table-sm table-vcenter mb-0">
                <thead><tr><th>Baris</th><th>Kolom</th><th>Pesan</th></tr></thead>
                <tbody>
                  <tr v-for="(e, idx) in importResult.errors" :key="idx">
                    <td>{{ e.row }}</td><td>{{ e.column }}</td><td class="text-danger">{{ e.message }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="row g-3">
      <div class="col-md-3">
        <div class="card h-100">
          <div class="card-header fw-semibold small text-muted text-uppercase">Daftar Tag</div>
          <div v-if="loadingTags" class="p-3 text-center text-muted small">Memuat...</div>
          <div class="list-group list-group-flush">
            <a
              v-for="t in tags" :key="t.id"
              class="list-group-item list-group-item-action d-flex align-items-center gap-2"
              :class="{ active: selectedTag?.id === t.id }"
              href="#"
              @click.prevent="selectTag(t)"
            >
              <span class="d-inline-block rounded-circle flex-shrink-0" style="width:10px;height:10px" :style="{ background: t.color || '#6366f1' }" />
              {{ t.name }}
            </a>
          </div>
          <div v-if="!loadingTags && !tags.length" class="p-3 text-center text-muted small">Belum ada tag</div>
        </div>
      </div>

      <div class="col">
        <div class="card h-100">
          <div v-if="!selectedTag" class="d-flex flex-column align-items-center justify-content-center py-5 text-muted">
            <i class="ti ti-tag"></i>
            <p>Pilih tag untuk melihat anggota</p>
          </div>
          <template v-else>
            <div class="card-header d-flex align-items-center justify-content-between">
              <div class="d-flex align-items-center gap-2">
                <span class="d-inline-block rounded-circle flex-shrink-0" style="width:10px;height:10px" :style="{ background: selectedTag.color || '#6366f1' }" />
                <span class="card-title mb-0">{{ selectedTag.name }}</span>
              </div>
              <button class="btn btn-sm btn-primary" @click="openAddModal"><i class="ti ti-user-plus"></i>
                Tambah User</button>
            </div>
            <BaseTable :columns="columns" :loading="loadingMembers" empty="Belum ada anggota">
              <tr v-for="item in members" :key="item.id">
                <td class="fw-medium">{{ item.name }}</td>
                <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
                <td>
                  <a href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="askRemove(item)" title="Hapus tag">
                    <i class="ti ti-user-minus"></i>
                  </a>
                </td>
              </tr>
            </BaseTable>
            <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchMembers() }" />
          </template>
        </div>
      </div>
    </div>

    <BaseModal v-if="showAddModal" title="Tambah User ke Tag" size="md" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchUser" class="form-control" placeholder="Cari user..." />
        </div>
        <div class="border rounded overflow-auto mt-2" style="max-height:18rem">
          <label v-for="u in filteredUsers()" :key="u.id" class="d-flex align-items-center gap-2 px-3 py-2 cursor-pointer">
            <input type="checkbox" :value="u.id" v-model="selectedUserIds" />
            <span>{{ u.name }} <code class="text-muted small font-monospace">{{ u.username }}</code></span>
          </label>
          <p v-if="!filteredUsers().length" class="text-center text-muted small p-3">Tidak ada user ditemukan</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="adding" @click="handleAdd"><span v-if="adding" class="spinner-border spinner-border-sm me-1"></span>Tambahkan ({{ selectedUserIds.length }})</button>
      </template>
    </BaseModal>

    <BaseConfirmModal
      v-if="confirmModal"
      :message="`Hapus tag '${selectedTag?.name}' dari '${pendingRemoveUser?.name}'?`"
      :loading="confirmLoading"
      @confirm="doRemove"
      @close="confirmModal = false"
    />
</template>
