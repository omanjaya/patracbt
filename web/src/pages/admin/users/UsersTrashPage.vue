<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getAvatarUrl } from '../../../utils/avatar'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import { userApi, type UserItem } from '../../../api/user.api'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useDebounce } from '@/composables/useDebounce'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const router = useRouter()
const confirmDlg = useConfirmModal()

const columns = [
  { key: 'check', label: '', width: '40px' },
  { key: 'name', label: 'Nama / Email' },
  { key: 'role', label: 'Role' },
  { key: 'deleted_at', label: 'Tanggal Dihapus' },
  { key: 'actions', label: 'Aksi', width: '160px' },
]

const roleVariant: Record<string, string> = {
  admin: 'danger', guru: 'info', pengawas: 'warning', peserta: 'success',
}

function roleBadgeClass(role: string) {
  const c = roleVariant[role] ?? 'secondary'
  return `bg-${c}-lt text-${c}`
}

const list = ref<UserItem[]>([])
const loading = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)
const searchRaw = ref('')
const search = useDebounce(searchRaw, 500)
const roleFilter = ref('')
const perPage = ref(10)
const selected = ref<number[]>([])

watch(search, () => {
  page.value = 1
  fetchList()
})

function toggleSelect(id: number) {
  const idx = selected.value.indexOf(id)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(id)
}

async function fetchList() {
  loading.value = true
  selected.value = []
  try {
    const res = await userApi.listTrashed({
      page: page.value,
      per_page: perPage.value,
      search: search.value || undefined,
      role: roleFilter.value || undefined,
    })
    list.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } finally {
    loading.value = false
  }
}

function handleRestore(item: UserItem) {
  confirmDlg.ask(
    'Pulihkan User',
    `Pulihkan user "${item.name}"?`,
    async () => {
      await userApi.restore(item.id)
      await fetchList()
    },
  )
}

function handleForceDelete(item: UserItem) {
  confirmDlg.ask(
    'Hapus Permanen',
    `Hapus permanen user "${item.name}"? Tindakan ini tidak dapat dibatalkan.`,
    async () => {
      await userApi.forceDelete(item.id)
      await fetchList()
    },
  )
}

function handleBulkRestore() {
  if (selected.value.length === 0) return
  confirmDlg.ask(
    'Pulihkan User',
    `Pulihkan ${selected.value.length} user terpilih?`,
    async () => {
      await userApi.bulkAction('restore', selected.value)
      await fetchList()
    },
  )
}

function handleBulkForceDelete() {
  if (selected.value.length === 0) return
  confirmDlg.ask(
    'Hapus Permanen',
    `Hapus permanen ${selected.value.length} user terpilih? Tindakan ini tidak dapat dibatalkan.`,
    async () => {
      await userApi.bulkAction('force_delete', selected.value)
      await fetchList()
    },
  )
}

function formatDate(d: string | null) {
  if (!d) return '–'
  return new Date(d).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })
}

function onSearch() {
  page.value = 1
  fetchList()
}

onMounted(fetchList)
</script>

<template>
  <BasePageHeader
    title="Arsip User"
    subtitle="Daftar user yang telah dihapus (soft-delete)."
    :breadcrumbs="[{ label: 'Pengguna', to: '/admin/users' }, { label: 'Tempat Sampah' }]"
  >
    <template #actions>
      <button class="btn btn-outline-primary" @click="router.push('/admin/users')">
        <i class="ti ti-arrow-left me-1"></i>
        Kembali ke Daftar User
      </button>
    </template>
  </BasePageHeader>

  <!-- Filter -->
  <div class="card card-body mb-3">
    <h3 class="card-title mb-3">Filter Arsip</h3>
    <div class="row g-3">
      <div class="col-md-4">
        <label class="form-label">Cari User</label>
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input
            v-model="searchRaw"
            type="text"
            class="form-control"
            placeholder="Cari nama, username, atau email..."
            aria-label="Cari user di arsip"
          />
        </div>
      </div>
      <div class="col-md-4">
        <label class="form-label">Filter berdasarkan Role</label>
        <select v-model="roleFilter" @change="onSearch" class="form-select">
          <option value="">Semua Role</option>
          <option value="admin">Admin</option>
          <option value="guru">Guru</option>
          <option value="pengawas">Pengawas</option>
          <option value="peserta">Peserta</option>
        </select>
      </div>
      <div class="col-md-4">
        <label class="form-label">Per Halaman</label>
        <select v-model="perPage" @change="page = 1; fetchList()" class="form-select">
          <option :value="10">10 data</option>
          <option :value="20">20 data</option>
          <option :value="50">50 data</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Table Card -->
  <div class="card">
    <div class="card-header d-flex align-items-center justify-content-between">
      <h3 class="card-title mb-0">Daftar User di Arsip</h3>
      <template v-if="selected.length > 0">
        <div class="d-flex align-items-center gap-2">
          <span class="badge bg-secondary-lt text-secondary">{{ selected.length }} dipilih</span>
          <button class="btn btn-sm btn-success" aria-label="Pulihkan user terpilih" @click="handleBulkRestore">
            <i class="ti ti-rotate-clockwise me-1"></i>Pulihkan
          </button>
          <button class="btn btn-sm btn-danger" aria-label="Hapus permanen user terpilih" @click="handleBulkForceDelete">
            <i class="ti ti-trash me-1"></i>Hapus Permanen
          </button>
          <button class="btn btn-sm btn-ghost-secondary" aria-label="Batal pilih" @click="selected = []">
            <i class="ti ti-x"></i>
          </button>
        </div>
      </template>
    </div>

    <BaseTable :columns="columns" :loading="loading" empty="Tidak ada user di arsip">
      <template #default>
        <tr v-if="!loading && list.length === 0">
          <td :colspan="columns.length" class="text-center text-muted py-5">
            <i class="ti ti-archive-off fs-4 mb-2 d-block opacity-50"></i>
            Tidak ada user di arsip
          </td>
        </tr>
        <tr v-for="item in list" :key="item.id">
          <td>
            <input
              class="form-check-input m-0 align-middle"
              type="checkbox"
              :checked="selected.includes(item.id)"
              :aria-label="`Pilih user ${item.name}`"
              @change="toggleSelect(item.id)"
            />
          </td>
          <td>
            <div class="d-flex align-items-center gap-2">
              <span
                class="avatar avatar-sm rounded-circle"
                :style="`background-image:url(${getAvatarUrl(item.id)})`"
              ></span>
              <div>
                <p class="fw-medium mb-0">{{ item.name }}</p>
                <p class="text-muted small mb-0">{{ item.email ?? '' }}</p>
              </div>
            </div>
          </td>
          <td>
            <span class="badge" :class="roleBadgeClass(item.role)">{{ item.role }}</span>
          </td>
          <td class="text-muted small">
            {{ formatDate((item as any).deleted_at ?? null) }}
          </td>
          <td>
            <div class="d-flex gap-1">
              <button
                type="button"
                class="btn btn-sm btn-ghost-success"
                title="Pulihkan"
                aria-label="Pulihkan user"
                @click="handleRestore(item)"
              >
                <i class="ti ti-rotate-clockwise"></i>
              </button>
              <button
                type="button"
                class="btn btn-sm btn-ghost-danger"
                title="Hapus Permanen"
                aria-label="Hapus permanen user"
                @click="handleForceDelete(item)"
              >
                <i class="ti ti-trash"></i>
              </button>
            </div>
          </td>
        </tr>
      </template>
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

  <!-- Bulk Action Floating Bar -->
  <div
    v-if="selected.length > 0"
    class="d-flex align-items-center bg-dark text-white rounded-pill shadow-lg px-3 py-2 gap-3 border border-dark-subtle"
    style="position: fixed; bottom: 30px; left: 50%; transform: translateX(-50%); width: fit-content; z-index: 1000;"
  >
    <div class="d-flex align-items-center gap-2 border-end pe-3 border-secondary">
      <span class="fw-bold text-white small">{{ selected.length }} Dipilih</span>
      <button
        class="btn btn-sm btn-icon btn-dark shadow-none text-muted border-0"
        title="Batal Pilih"
        aria-label="Batal pilih semua"
        @click="selected = []"
      >
        <i class="ti ti-x" style="font-size: 14px;"></i>
      </button>
    </div>
    <div class="d-flex gap-2">
      <button class="btn btn-success btn-sm btn-pill" title="Pulihkan" aria-label="Pulihkan user terpilih" @click="handleBulkRestore">
        <i class="ti ti-rotate-clockwise me-1"></i>Pulihkan
      </button>
      <button class="btn btn-danger btn-sm btn-pill" title="Hapus Permanen" aria-label="Hapus permanen user terpilih" @click="handleBulkForceDelete">
        <i class="ti ti-trash me-1"></i>Hapus Permanen
      </button>
    </div>
  </div>

  <BaseConfirmModal
    v-if="confirmDlg.show.value"
    :title="confirmDlg.title.value"
    :message="confirmDlg.message.value"
    :loading="confirmDlg.loading.value"
    @confirm="confirmDlg.confirm"
    @close="confirmDlg.close"
  />
</template>
