<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseBadge from '@/components/ui/BaseBadge.vue'
import BaseInput from '@/components/ui/BaseInput.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { userApi, type UserItem } from '../../../api/user.api'
import { authApi } from '../../../api/auth.api'
import { useAuthStore } from '../../../stores/auth.store'
import { useToastStore } from '../../../stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useTableFilters } from '@/composables/useTableFilters'
import client from '../../../api/client'

const authStore = useAuthStore()
const toast = useToastStore()
const confirmModal = useConfirmModal()

const columns = [
  { key: 'check', label: '', width: '40px' },
  { key: 'name', label: 'Nama' },
  { key: 'username', label: 'Username' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'last_login', label: 'Login Terakhir' },
  { key: 'actions', label: 'Aksi', width: '140px' },
]

const trashedColumns = [
  { key: 'check', label: '', width: '40px' },
  { key: 'name', label: 'Nama' },
  { key: 'username', label: 'Username' },
  { key: 'role', label: 'Role' },
  { key: 'actions', label: 'Aksi', width: '160px' },
]

const roleVariant: Record<string, 'info' | 'success' | 'warning' | 'danger' | 'default'> = {
  admin: 'danger', guru: 'info', pengawas: 'warning', peserta: 'success',
}

const activeTab = ref<'active' | 'trashed'>('active')
const list = ref<UserItem[]>([])
const { searchRaw, search, page, total, totalPages, loading } = useTableFilters(fetchList)
const roleFilter = ref('')

watch(roleFilter, () => { page.value = 1; fetchList() })
const selected = ref<number[]>([])

const isAllSelected = computed(() => list.value.length > 0 && selected.value.length === list.value.length)
const isSomeSelected = computed(() => selected.value.length > 0 && !isAllSelected.value)

function toggleSelectAll() {
  if (isAllSelected.value) {
    selected.value = []
  } else {
    selected.value = list.value.map(i => i.id)
  }
}

function toggleSelect(id: number) {
  const idx = selected.value.indexOf(id)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(id)
}

const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)
const form = reactive({
  name: '', username: '', password: '', role: 'peserta', email: '',
  nis: '', class: '', major: '', is_active: true,
})

let searchController: AbortController | null = null

async function fetchList() {
  searchController?.abort()
  searchController = new AbortController()
  loading.value = true
  selected.value = []
  try {
    if (activeTab.value === 'active') {
      const res = await userApi.list({
        page: page.value, per_page: 20,
        search: search.value, role: roleFilter.value || undefined,
      }, { signal: searchController.signal })
      list.value = res.data.data ?? []
      total.value = res.data.meta?.total ?? 0
    } else {
      const res = await userApi.listTrashed({
        page: page.value, per_page: 20,
        search: search.value, role: roleFilter.value || undefined,
      }, { signal: searchController.signal })
      list.value = res.data.data ?? []
      total.value = res.data.meta?.total ?? 0
    }
  } catch (e: unknown) {
    if ((e as any)?.code === 'ERR_CANCELED') return
    throw e
  } finally {
    loading.value = false
  }
}

function switchTab(tab: 'active' | 'trashed') {
  activeTab.value = tab
  page.value = 1
  search.value = ''
  fetchList()
}

// Validation
const formErrors = reactive<Record<string, string>>({})

function validateForm(): boolean {
  Object.keys(formErrors).forEach(k => delete formErrors[k])

  if (!form.name.trim()) formErrors.name = 'Nama wajib diisi'
  if (!form.username.trim()) {
    formErrors.username = 'Username wajib diisi'
  } else if (form.username.trim().length < 3) {
    formErrors.username = 'Username minimal 3 karakter'
  } else if (!/^[a-zA-Z0-9_]+$/.test(form.username.trim())) {
    formErrors.username = 'Username hanya boleh huruf, angka, dan underscore'
  }
  if (!isEdit.value && !form.password) {
    formErrors.password = 'Password wajib diisi'
  } else if (form.password && form.password.length < 6) {
    formErrors.password = 'Password minimal 6 karakter'
  }
  if (!form.role) formErrors.role = 'Role wajib dipilih'

  return Object.keys(formErrors).length === 0
}

function openCreate() {
  isEdit.value = false; editId.value = null
  Object.assign(form, { name: '', username: '', password: '', role: 'peserta', email: '', nis: '', class: '', major: '', is_active: true })
  Object.keys(formErrors).forEach(k => delete formErrors[k])
  showModal.value = true
}

function openEdit(item: UserItem) {
  isEdit.value = true; editId.value = item.id
  Object.assign(form, {
    name: item.name, username: item.username, password: '', role: item.role,
    email: item.email ?? '', nis: item.profile?.nis ?? '', class: item.profile?.class ?? '', major: item.profile?.major ?? '',
    is_active: item.is_active ?? true,
  })
  Object.keys(formErrors).forEach(k => delete formErrors[k])
  showModal.value = true
}

async function handleSave() {
  if (!validateForm()) return
  saving.value = true
  try {
    const profile = { nis: form.nis || undefined, class: form.class || undefined, major: form.major || undefined }
    if (isEdit.value && editId.value) {
      await userApi.update(editId.value, { name: form.name, role: form.role, email: form.email || undefined, password: form.password || undefined, is_active: form.is_active, profile })
    } else {
      await userApi.create({ name: form.name, username: form.username, password: form.password, role: form.role, email: form.email || undefined, is_active: form.is_active, profile })
    }
    showModal.value = false
    toast.success('Data user berhasil disimpan')
    await fetchList()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan data user')
  } finally {
    saving.value = false
  }
}

function handleDelete(item: UserItem) {
  confirmModal.ask(
    'Hapus User',
    `Hapus user "${item.name}"? User akan masuk tempat sampah.`,
    async () => {
      try {
        await userApi.delete(item.id)
        toast.success('User berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus user')
      }
    },
  )
}

function handleRestore(item: UserItem) {
  confirmModal.ask(
    'Pulihkan User',
    `Pulihkan user "${item.name}"?`,
    async () => {
      try {
        await userApi.restore(item.id)
        toast.success('User berhasil dipulihkan')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal memulihkan user')
      }
    },
  )
}

function handleForceDelete(item: UserItem) {
  confirmModal.ask(
    'Hapus Permanen',
    `Hapus permanen user "${item.name}"? Tindakan ini tidak dapat dibatalkan.`,
    async () => {
      try {
        await userApi.forceDelete(item.id)
        toast.success('User berhasil dihapus permanen')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus user')
      }
    },
  )
}

function handleBulkAction(action: 'delete' | 'restore' | 'force_delete') {
  if (selected.value.length === 0) return
  const labels = { delete: 'hapus', restore: 'pulihkan', force_delete: 'hapus permanen' }
  confirmModal.ask(
    'Konfirmasi Aksi Massal',
    `${labels[action]} ${selected.value.length} user terpilih?`,
    async () => {
      try {
        await userApi.bulkAction(action, selected.value)
        toast.success(`${selected.value.length} user berhasil di-${labels[action]}`)
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal melakukan aksi massal')
      }
    },
  )
}

function handlePreview(item: UserItem) {
  confirmModal.ask(
    'Preview sebagai Peserta',
    `Sistem akan log out dari akun saat ini dan masuk (login) sebagai Uji Coba Peserta "${item.name}". Anda harus login admin lagi nantinya. Lanjutkan?`,
    async () => {
      try {
        const res = await authApi.previewAsPeserta(item.id)
        const token = res.data.data?.preview_token
        if (token) {
          try {
            localStorage.setItem('access_token', token)
            localStorage.setItem('refresh_token', token)
          } catch {
            toast.error('Gagal menyimpan token. Periksa pengaturan browser Anda.')
            return
          }
          await authStore.fetchUser()
          window.location.href = '/peserta'
        }
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal masuk sebagai peserta')
      }
    },
  )
}

function formatDate(d: string | null) {
  if (!d) return '–'
  return new Date(d).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })
}

const downloadingTpl = ref(false)
async function downloadTemplate() {
  downloadingTpl.value = true
  try {
    const res = await client.get('/admin/users/import/template', { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', 'template-import-user.csv')
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal mengunduh template')
  } finally {
    downloadingTpl.value = false
  }
}

// --- Import Excel ---
const showImportModal = ref(false)
const importFile = ref<File | null>(null)
const importing = ref(false)
const importResult = ref<{ created?: number; skipped?: number; errors?: { row: number; column: string; message: string }[] } | null>(null)

function onImportFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  importFile.value = input.files?.[0] ?? null
  importResult.value = null
}

async function handleImport() {
  if (!importFile.value) return
  importing.value = true
  importResult.value = null
  try {
    const fd = new FormData()
    fd.append('file', importFile.value)
    const res = await client.post('/admin/users/import', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
    importResult.value = res.data.data
    if (res.data.data?.created > 0) {
      toast.success(`${res.data.data.created} user berhasil diimport`)
      fetchList()
    }
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal import user')
  } finally {
    importing.value = false
  }
}

onMounted(fetchList)
</script>

<template>
    <BasePageHeader
      title="Manajemen User"
      subtitle="Kelola akun pengguna sistem"
      :breadcrumbs="[{ label: 'Pengguna' }]"
    >
      <template #actions>
        <button class="btn btn-outline-secondary" v-if="activeTab === 'active'" @click="downloadTemplate" :disabled="downloadingTpl">
          <span v-if="downloadingTpl" class="spinner-border spinner-border-sm me-1"></span>
          <i v-else class="ti ti-download me-1"></i>Template CSV
        </button>
        <button class="btn btn-outline-primary" v-if="activeTab === 'active'" @click="showImportModal = true; importFile = null; importResult = null">
          <i class="ti ti-file-import me-1"></i>Import Excel
        </button>
        <button class="btn btn-primary" v-if="activeTab === 'active'" @click="openCreate"><i class="ti ti-plus"></i>
          Tambah User</button>
      </template>
    </BasePageHeader>

    <!-- Tabs -->
    <ul class="nav nav-tabs mb-3">
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'active' }" href="#" @click.prevent="switchTab('active')">User Aktif</a>
      </li>
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'trashed' }" href="#" @click.prevent="switchTab('trashed')">Sampah</a>
      </li>
    </ul>

    <div class="card">
      <div class="card-header d-flex align-items-center gap-2 flex-wrap">
        <input
          type="checkbox"
          class="form-check-input"
          :checked="isAllSelected"
          :indeterminate="isSomeSelected"
          aria-label="Pilih semua user"
          :disabled="list.length === 0"
          @change="toggleSelectAll"
        />
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchRaw" class="form-control" placeholder="Cari nama / username..." aria-label="Cari user" />
          <button v-if="searchRaw" class="btn btn-outline-secondary" type="button" aria-label="Hapus pencarian" @click="searchRaw = ''">
            <i class="ti ti-x"></i>
          </button>
        </div>
        <select v-model="roleFilter" class="form-select form-select-sm">
          <option value="">Semua Role</option>
          <option value="admin">Admin</option>
          <option value="guru">Guru</option>
          <option value="pengawas">Pengawas</option>
          <option value="peserta">Peserta</option>
        </select>

        <!-- Bulk actions -->
        <template v-if="selected.length > 0">
          <span class="badge bg-secondary-lt text-secondary">{{ selected.length }} dipilih</span>
          <template v-if="activeTab === 'active'">
            <button class="btn btn-sm btn-danger" @click="handleBulkAction('delete')"><i class="ti ti-trash"></i> Hapus</button>
          </template>
          <template v-else>
            <button class="btn btn-sm btn-secondary" @click="handleBulkAction('restore')"><i class="ti ti-rotate-clockwise"></i> Pulihkan</button>
            <button class="btn btn-sm btn-danger" @click="handleBulkAction('force_delete')"><i class="ti ti-eraser"></i> Hapus Permanen</button>
          </template>
        </template>
      </div>

      <!-- Active users table -->
      <BaseTable v-if="activeTab === 'active'" :columns="columns" :loading="loading" empty="Belum ada user">
        <template #default>
          <tr v-for="item in list" :key="item.id">
            <td>
              <input type="checkbox" class="form-check-input" :checked="selected.includes(item.id)" :aria-label="`Pilih user ${item.name}`" @change="toggleSelect(item.id)" />
            </td>
            <td>
              <div class="d-flex align-items-center gap-2">
                <span class="avatar avatar-sm rounded-circle">{{ item.name.split(' ').map(w => w[0]).slice(0, 2).join('').toUpperCase() }}</span>
                <div>
                  <p class="fw-medium">{{ item.name }}</p>
                  <p class="text-muted small">{{ item.email ?? '' }}</p>
                </div>
              </div>
            </td>
            <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
            <td><BaseBadge :variant="roleVariant[item.role] ?? 'default'">{{ item.role }}</BaseBadge></td>
            <td>
              <BaseBadge :variant="item.is_active ? 'success' : 'danger'">
                {{ item.is_active ? 'Aktif' : 'Nonaktif' }}
              </BaseBadge>
            </td>
            <td>{{ formatDate(item.last_login_at) }}</td>
            <td>
              <div class="d-flex gap-1">
                <button v-if="item.role === 'peserta'" class="btn btn-sm btn-ghost-primary" :aria-label="`Preview sebagai peserta ${item.name}`" @click="handlePreview(item)"><i class="ti ti-device-laptop"></i></button>
                <button class="btn btn-sm btn-ghost-secondary" :aria-label="`Edit user ${item.name}`" @click="openEdit(item)"><i class="ti ti-pencil"></i></button>
                <button class="btn btn-sm btn-ghost-danger" :aria-label="`Hapus user ${item.name}`" @click="handleDelete(item)"><i class="ti ti-trash"></i></button>
              </div>
            </td>
          </tr>
        </template>
      </BaseTable>

      <!-- Trashed users table -->
      <BaseTable v-else :columns="trashedColumns" :loading="loading" empty="Tidak ada user di sampah">
        <template #default>
          <tr v-for="item in list" :key="item.id">
            <td>
              <input type="checkbox" class="form-check-input" :checked="selected.includes(item.id)" :aria-label="`Pilih user ${item.name}`" @change="toggleSelect(item.id)" />
            </td>
            <td>
              <div class="d-flex align-items-center gap-2">
                <span class="avatar avatar-sm rounded-circle">{{ item.name.split(' ').map(w => w[0]).slice(0, 2).join('').toUpperCase() }}</span>
                <p class="fw-medium">{{ item.name }}</p>
              </div>
            </td>
            <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
            <td><BaseBadge :variant="roleVariant[item.role] ?? 'default'">{{ item.role }}</BaseBadge></td>
            <td>
              <div class="d-flex gap-1">
                <button class="btn btn-sm btn-ghost-secondary" :aria-label="`Pulihkan user ${item.name}`" @click="handleRestore(item)"><i class="ti ti-rotate-clockwise"></i></button>
                <button class="btn btn-sm btn-ghost-danger" :aria-label="`Hapus permanen user ${item.name}`" @click="handleForceDelete(item)"><i class="ti ti-eraser"></i></button>
              </div>
            </td>
          </tr>
        </template>
      </BaseTable>

      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
    </div>

    <BaseModal v-if="showModal" :title="isEdit ? 'Edit User' : 'Tambah User'" size="md" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <fieldset :disabled="saving">
        <div class="row g-3">
          <BaseInput
            v-model="form.name"
            label="Nama Lengkap *"
            type="text"
            placeholder="Nama lengkap"
            :error="formErrors.name"
          />
          <BaseInput
            v-model="form.username"
            label="Username *"
            type="text"
            placeholder="Username unik"
            :error="formErrors.username"
          />
          <BaseInput
            v-model="form.password"
            :label="isEdit ? 'Password Baru (kosongkan jika tidak diubah)' : 'Password *'"
            type="password"
            placeholder="Min 6 karakter"
            :error="formErrors.password"
          />
          <BaseInput
            v-model="form.email"
            label="Email"
            type="email"
            placeholder="email@domain.com"
          />
          <div class="mb-3">
            <label class="form-label">Role <span class="text-danger">*</span></label>
            <select v-model="form.role" class="form-select" :class="{ 'is-invalid': formErrors.role }">
              <option value="peserta">Peserta</option>
              <option value="guru">Guru</option>
              <option value="pengawas">Pengawas</option>
              <option value="admin">Admin</option>
            </select>
            <div v-if="formErrors.role" class="invalid-feedback">{{ formErrors.role }}</div>
          </div>
          <div class="mb-3">
            <label class="form-check form-switch">
              <input class="form-check-input" type="checkbox" v-model="form.is_active" />
              <span class="form-check-label">Status Aktif</span>
            </label>
          </div>
        </div>
        <div class="text-muted small fw-bold text-uppercase mt-3 mb-1">Profil (Opsional)</div>
        <div class="row g-3">
          <BaseInput v-model="form.nis" label="NIS/NIP" type="text" placeholder="Nomor induk" />
          <BaseInput v-model="form.class" label="Kelas" type="text" placeholder="Contoh: XII IPA 1" />
          <BaseInput v-model="form.major" label="Jurusan" type="text" placeholder="Contoh: IPA" />
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
      @confirm="confirmModal.confirm"
      @close="confirmModal.close"
      :loading="confirmModal.loading.value"
    />

    <!-- Import Excel Modal -->
    <BaseModal v-if="showImportModal" title="Import User dari Excel" @close="showImportModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="alert alert-info mb-0">
          <div class="d-flex gap-2">
            <i class="ti ti-info-circle flex-shrink-0 mt-1"></i>
            <div>
              <p class="mb-1">Format kolom: <strong>name, username, password, role, email, nis, nip, class, major, phone, rombel</strong></p>
              <p class="mb-0 small text-muted">Kolom <strong>rombel</strong> (K) bersifat opsional — jika diisi, siswa otomatis masuk ke rombel tersebut. Jika rombel belum ada, akan dibuat otomatis.</p>
            </div>
          </div>
        </div>
        <div>
          <label class="form-label">Pilih File Excel (.xlsx)</label>
          <input type="file" accept=".xlsx,.xls,.csv" class="form-control" @change="onImportFileChange" />
        </div>
        <!-- Import Result -->
        <div v-if="importResult" class="alert mb-0" :class="importResult.errors?.length ? 'alert-warning' : 'alert-success'">
          <div class="d-flex align-items-center gap-2 mb-2">
            <i :class="importResult.errors?.length ? 'ti ti-alert-triangle' : 'ti ti-check'"></i>
            <strong>{{ importResult.created ?? 0 }} berhasil, {{ importResult.skipped ?? 0 }} dilewati</strong>
          </div>
          <div v-if="importResult.errors?.length" class="mt-2" style="max-height: 200px; overflow-y: auto;">
            <div v-for="(err, i) in importResult.errors" :key="i" class="small text-muted">
              Baris {{ err.row }}<template v-if="err.column !== '-'"> ({{ err.column }})</template>: {{ err.message }}
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showImportModal = false">Tutup</button>
        <button class="btn btn-primary" :disabled="importing || !importFile" @click="handleImport">
          <span v-if="importing" class="spinner-border spinner-border-sm me-1"></span>
          <i v-else class="ti ti-upload me-1"></i>Import
        </button>
      </template>
    </BaseModal>
</template>

