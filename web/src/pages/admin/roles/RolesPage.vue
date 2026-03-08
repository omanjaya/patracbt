<script setup lang="ts">
import { ref, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import { roleApi, type Role } from '../../../api/role.api'
import client from '../../../api/client'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { useToastStore } from '../../../stores/toast.store'
import { useCrudTable } from '@/composables/useCrudTable'
import { useCrudModal } from '@/composables/useCrudModal'

interface Permission {
  id: number
  name: string
  group_name: string
}

const toast = useToastStore()

const columns = [
  { key: 'name', label: 'Nama Hak Akses' },
  { key: 'guard_name', label: 'Guard' },
  { key: 'actions', label: 'Aksi', width: '120px' },
]

const { list, searchRaw: search, page, total, totalPages, loading, fetchList } = useCrudTable<Role>({
  fetchFn: (params) => roleApi.list(params),
  errorMessage: 'Gagal memuat data role',
})

const { showModal, isEdit, saving, form, openCreate, openEdit: _openEdit, handleSave } = useCrudModal<{ name: string; guard_name: string }>({
  createFn: (data) => roleApi.create(data),
  updateFn: (id, data) => roleApi.update(id, data),
  afterSave: fetchList,
  resetForm: () => ({ name: '', guard_name: 'web' }),
  successCreate: 'Role berhasil ditambahkan',
  successUpdate: 'Role berhasil diperbarui',
  errorMessage: 'Gagal menyimpan role',
})

function openEdit(item: Role) {
  _openEdit({ id: item.id, name: item.name, guard_name: item.guard_name })
}

// Permissions modal state
const showPermModal = ref(false)
const allPermissions = ref<Permission[]>([])
const selectedRole = ref<Role | null>(null)
const selectedPermIds = ref<Set<number>>(new Set())

// Confirm modal state
const confirmModal = ref(false)
const confirmLoading = ref(false)
const pendingDeleteId = ref<number | null>(null)

async function loadAllPermissions() {
  try {
    const res = await client.get('/admin/permissions/all')
    allPermissions.value = res.data.data ?? []
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal memuat daftar izin')
  }
}

function askDelete(id: number) {
  pendingDeleteId.value = id
  confirmModal.value = true
}

async function doDelete() {
  if (!pendingDeleteId.value) return
  confirmLoading.value = true
  try {
    await roleApi.delete(pendingDeleteId.value)
    toast.success('Role berhasil dihapus')
    await fetchList()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menghapus role')
  } finally {
    confirmLoading.value = false
    confirmModal.value = false
  }
}

async function openPermissions(item: Role) {
  selectedRole.value = item
  selectedPermIds.value.clear()
  showPermModal.value = true

  try {
    const res = await roleApi.getPermissions(item.id)
    const perms: Permission[] = res.data.data ?? []
    perms.forEach(p => selectedPermIds.value.add(p.id))
  } catch (e: any) {
    toast.error('Gagal memuat izin role')
  }
}

function togglePerm(id: number) {
  if (selectedPermIds.value.has(id)) selectedPermIds.value.delete(id)
  else selectedPermIds.value.add(id)
}

async function handleSavePermissions() {
  if (!selectedRole.value) return
  saving.value = true
  try {
    await roleApi.assignPermissions(selectedRole.value.id, Array.from(selectedPermIds.value))
    toast.success('Izin berhasil diperbarui')
    showPermModal.value = false
  } catch (e: any) {
    toast.error('Gagal menyimpan izin')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchList()
  loadAllPermissions()
})
</script>

<template>
    <BasePageHeader
      title="Hak & Izin"
      subtitle="Kelola peran dan hak akses sistem"
      :breadcrumbs="[{ label: 'Master Data', to: '/admin' }, { label: 'Hak & Izin' }]"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreate"><i class="ti ti-plus"></i>
          Tambah Role</button>
      </template>
    </BasePageHeader>

    <div class="card">
      <div class="card-header d-flex align-items-center gap-2 flex-wrap">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="search" @input="page = 1; fetchList()" class="form-control" placeholder="Cari role..." />
        </div>
      </div>

      <BaseTable :columns="columns" :loading="loading" empty="Belum ada role">
        <tr v-for="item in list" :key="item.id">
          <td>
            <div class="d-flex align-items-center gap-2">
              <i class="ti ti-shield-check"></i>
              <span class="fw-medium">{{ item.name }}</span>
            </div>
          </td>
          <td><code class="text-muted small font-monospace">{{ item.guard_name }}</code></td>
          <td>
            <div class="d-flex gap-1">
              <button class="btn btn-sm btn-ghost-primary" @click="openPermissions(item)" title="Atur Izin (Permissions)">
                <i class="ti ti-shield-lock"></i>
              </button>
              <a href="#" class="btn btn-sm btn-ghost-secondary" @click.prevent="openEdit(item)"><i class="ti ti-pencil"></i></a>
              <a href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="askDelete(item.id)"><i class="ti ti-trash"></i></a>
            </div>
          </td>
        </tr>
      </BaseTable>

      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
    </div>

    <BaseModal v-if="showModal" :title="isEdit ? 'Edit Role' : 'Tambah Role'" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <div class="mb-3">
              <label class="form-label">Nama Role *</label>
              <input class="form-control" type="text" v-model="form.name" placeholder="Contoh: kepala_sekolah" required />
            </div>
        <div class="mb-3">
              <label class="form-label">Guard Name</label>
              <input class="form-control" type="text" v-model="form.guard_name" placeholder="web" />
            </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="showModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSave"><span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan</button>
      </template>
    </BaseModal>

    <BaseModal v-if="showPermModal" :title="`Izin Role: ${selectedRole?.name}`" @close="showPermModal = false">
      <p class="text-muted mb-3">Atur izin akses (permissions) untuk role ini. Izin ini dapat digunakan di middleware backend maupun frontend guard.</p>
      
      <div v-if="allPermissions.length === 0" class="alert alert-info">
        Belum ada master izin / permission. Silakan buat di menu Master Grup terlebih dahulu.
      </div>
      
      <div v-else style="max-height: 400px; overflow-y: auto" class="border rounded p-3">
        <div v-for="grp in [...new Set(allPermissions.map(p => p.group_name))]" :key="grp" class="mb-3">
          <div class="fw-bold mb-2 text-primary">{{ grp }}</div>
          <div class="row g-2">
            <div 
              v-for="p in allPermissions.filter(x => x.group_name === grp)" 
              :key="p.id" 
              class="col-md-6"
            >
              <label class="form-check m-0">
                <input 
                  class="form-check-input" 
                  type="checkbox" 
                  :checked="selectedPermIds.has(p.id)"
                  @change="togglePerm(p.id)"
                />
                <span class="form-check-label">{{ p.name }}</span>
              </label>
            </div>
          </div>
        </div>
      </div>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showPermModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSavePermissions"><span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan Izin</button>
      </template>
    </BaseModal>

    <BaseConfirmModal
      v-if="confirmModal"
      message="Yakin ingin menghapus role ini?"
      :loading="confirmLoading"
      @confirm="doDelete"
      @close="confirmModal = false"
    />
</template>
