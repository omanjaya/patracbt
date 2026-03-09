<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BaseConfirmModal from '../../../components/ui/BaseConfirmModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { userApi, type UserItem } from '../../../api/user.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { useConfirmModal } from '@/composables/useConfirmModal'

const rombels = ref<Rombel[]>([])
const selectedRombel = ref<Rombel | null>(null)
const members = ref<UserItem[]>([])
const loadingRombels = ref(false)
const loadingMembers = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

// Search & filter
const searchMember = ref('')
const statusFilter = ref<'' | 'true' | 'false'>('')

// Bulk selection
const selectedMemberIds = ref<number[]>([])

// Confirm modal
const confirmModal = useConfirmModal()

// Add modal
const showAddModal = ref(false)
const allUsers = ref<UserItem[]>([])
const selectedUserIds = ref<number[]>([])
const searchUser = ref('')
const adding = ref(false)

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout> | null = null
function onSearchChange() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    page.value = 1
    fetchMembers()
  }, 300)
}

function onStatusFilterChange() {
  page.value = 1
  fetchMembers()
}

// Select all logic
const allSelected = computed(() =>
  members.value.length > 0 && members.value.every(m => selectedMemberIds.value.includes(m.id))
)
const someSelected = computed(() =>
  !allSelected.value && members.value.some(m => selectedMemberIds.value.includes(m.id))
)

function toggleSelectAll() {
  if (allSelected.value) {
    selectedMemberIds.value = []
  } else {
    selectedMemberIds.value = members.value.map(m => m.id)
  }
}

function toggleMember(id: number) {
  const idx = selectedMemberIds.value.indexOf(id)
  if (idx >= 0) {
    selectedMemberIds.value.splice(idx, 1)
  } else {
    selectedMemberIds.value.push(id)
  }
}

async function fetchRombels() {
  loadingRombels.value = true
  try {
    const res = await rombelApi.list({ per_page: 100 })
    rombels.value = res.data.data ?? []
  } finally {
    loadingRombels.value = false
  }
}

async function selectRombel(r: Rombel) {
  selectedRombel.value = r
  page.value = 1
  searchMember.value = ''
  statusFilter.value = ''
  selectedMemberIds.value = []
  await fetchMembers()
}

async function fetchMembers() {
  if (!selectedRombel.value) return
  loadingMembers.value = true
  selectedMemberIds.value = []
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      per_page: 20,
      rombel_id: selectedRombel.value.id,
    }
    if (searchMember.value.trim()) {
      params.search = searchMember.value.trim()
    }
    if (statusFilter.value !== '') {
      params.is_active = statusFilter.value === 'true'
    }
    const res = await userApi.list(params as Parameters<typeof userApi.list>[0])
    members.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } finally {
    loadingMembers.value = false
  }
}

async function openAddModal() {
  const res = await userApi.list({ per_page: 200, role: 'peserta' })
  allUsers.value = res.data.data ?? []
  selectedUserIds.value = []
  searchUser.value = ''
  showAddModal.value = true
}

async function handleAdd() {
  if (!selectedRombel.value || !selectedUserIds.value.length) return
  adding.value = true
  try {
    await rombelApi.assignUsers(selectedRombel.value.id, selectedUserIds.value)
    showAddModal.value = false
    await fetchMembers()
    await fetchRombels()
  } finally {
    adding.value = false
  }
}

function handleRemove(user: UserItem) {
  if (!selectedRombel.value) return
  confirmModal.ask(
    'Keluarkan Anggota',
    `Keluarkan "${user.name}" dari rombel ini?`,
    async () => {
      await rombelApi.removeUsers(selectedRombel.value!.id, [user.id])
      await fetchMembers()
      await fetchRombels()
    }
  )
}

function handleBulkRemove() {
  if (!selectedRombel.value || !selectedMemberIds.value.length) return
  const count = selectedMemberIds.value.length
  confirmModal.ask(
    'Keluarkan Anggota',
    `Keluarkan ${count} anggota dari rombel ini?`,
    async () => {
      await rombelApi.removeUsers(selectedRombel.value!.id, [...selectedMemberIds.value])
      selectedMemberIds.value = []
      await fetchMembers()
      await fetchRombels()
    }
  )
}

// Filtered users for add modal (exclude current members)
const memberIds = computed(() => new Set(members.value.map(m => m.id)))

const filteredUsers = () => allUsers.value.filter(u => {
  if (memberIds.value.has(u.id)) return false
  const q = searchUser.value.toLowerCase()
  if (!q) return true
  return (
    u.name.toLowerCase().includes(q) ||
    u.username.toLowerCase().includes(q) ||
    (u.profile?.nis ?? '').toLowerCase().includes(q)
  )
})

onMounted(fetchRombels)
</script>

<template>
    <BasePageHeader
      title="Pengaitan Rombel"
      subtitle="Kelola anggota rombel belajar"
      :breadcrumbs="[{ label: 'Manajemen', to: '/admin' }, { label: 'Pengaitan Rombel' }]"
    />

    <div class="row g-3">
      <!-- Rombel list -->
      <div class="col-md-3">
        <div class="card h-100 p-0">
          <div class="card-header fw-semibold small text-muted text-uppercase">Daftar Rombel</div>
          <div class="list-group list-group-flush">
            <div v-if="loadingRombels" class="p-3 text-center text-muted small">Memuat...</div>
            <button
              v-for="r in rombels" :key="r.id"
              class="list-group-item list-group-item-action d-flex align-items-center gap-2 border-0"
              :class="{ active: selectedRombel?.id === r.id }"
              @click="selectRombel(r)"
            >
              <i class="ti ti-users"></i>
              <span class="flex-fill text-start">{{ r.name }}</span>
              <span
                v-if="(r as any).students_count != null"
                class="badge bg-secondary-lt"
              >{{ (r as any).students_count }}</span>
            </button>
            <div v-if="!loadingRombels && !rombels.length" class="p-3 text-center text-muted small">Belum ada rombel</div>
          </div>
        </div>
      </div>

      <!-- Members -->
      <div class="col">
        <div class="card">
        <div v-if="!selectedRombel" class="card-body">
          <div class="empty">
            <div class="empty-icon">
              <i class="ti ti-users-group" style="font-size: 3rem;"></i>
            </div>
            <p class="empty-title">Pilih rombel untuk melihat anggota</p>
            <p class="empty-subtitle text-muted">Klik salah satu rombel di sebelah kiri untuk menampilkan daftar anggotanya.</p>
          </div>
        </div>
        <template v-else>
          <div class="card-header d-flex align-items-center justify-content-between">
            <span class="card-title mb-0">{{ selectedRombel.name }}</span>
            <div class="d-flex align-items-center gap-2">
              <button
                v-if="selectedMemberIds.length"
                class="btn btn-sm btn-danger"
                @click="handleBulkRemove"
              >
                <i class="ti ti-user-minus me-1"></i>
                Keluarkan ({{ selectedMemberIds.length }})
              </button>
              <button class="btn btn-sm btn-primary" @click="openAddModal">
                <i class="ti ti-user-plus me-1"></i>
                Tambah Anggota
              </button>
            </div>
          </div>

          <!-- Search & filter bar -->
          <div class="card-body border-bottom py-2">
            <div class="row g-2 align-items-center">
              <div class="col-auto flex-fill">
                <div class="input-icon">
                  <span class="input-icon-addon"><i class="ti ti-search"></i></span>
                  <input
                    v-model="searchMember"
                    class="form-control form-control-sm"
                    placeholder="Cari nama, username, atau NIS..."
                    @input="onSearchChange"
                  />
                </div>
              </div>
              <div class="col-auto">
                <select
                  v-model="statusFilter"
                  class="form-select form-select-sm"
                  style="min-width: 140px"
                  @change="onStatusFilterChange"
                >
                  <option value="">Semua Status</option>
                  <option value="true">Aktif</option>
                  <option value="false">Tidak Aktif</option>
                </select>
              </div>
            </div>
          </div>

          <div class="table-responsive">
            <table class="table table-vcenter card-table">
              <thead>
                <tr>
                  <th style="width: 40px">
                    <input
                      type="checkbox"
                      class="form-check-input m-0 align-middle"
                      :checked="allSelected"
                      :indeterminate="someSelected"
                      @change="toggleSelectAll"
                    />
                  </th>
                  <th>Nama</th>
                  <th>Username</th>
                  <th>NIS</th>
                  <th style="width: 110px">Status</th>
                  <th style="width: 80px">Aksi</th>
                </tr>
              </thead>
              <tbody>
                <template v-if="loadingMembers">
                  <tr v-for="n in 5" :key="n">
                    <td v-for="c in 6" :key="c">
                      <div class="placeholder-glow"><span class="placeholder col-8" /></div>
                    </td>
                  </tr>
                </template>
                <template v-else-if="members.length">
                  <tr v-for="item in members" :key="item.id">
                    <td>
                      <input
                        type="checkbox"
                        class="form-check-input m-0"
                        :checked="selectedMemberIds.includes(item.id)"
                        @change="toggleMember(item.id)"
                      />
                    </td>
                    <td class="fw-medium">{{ item.name }}</td>
                    <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
                    <td>
                      <span v-if="item.profile?.nis" class="text-muted small">{{ item.profile.nis }}</span>
                      <span v-else class="text-muted small">-</span>
                    </td>
                    <td>
                      <span v-if="item.is_active" class="badge bg-green-lt">Aktif</span>
                      <span v-else class="badge bg-red-lt">Tidak Aktif</span>
                    </td>
                    <td>
                      <a href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="handleRemove(item)" title="Keluarkan">
                        <i class="ti ti-user-minus"></i>
                      </a>
                    </td>
                  </tr>
                </template>
                <tr v-else>
                  <td colspan="6" class="text-center text-muted py-5">
                    <i class="ti ti-database-off fs-4 mb-2 d-block opacity-50" aria-hidden="true" />
                    Belum ada anggota
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchMembers() }" />
        </template>
        </div>
      </div>
    </div>

    <!-- Add Member Modal -->
    <BaseModal v-if="showAddModal" title="Tambah Anggota Rombel" size="md" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchUser" class="form-control" placeholder="Cari peserta..." />
        </div>
        <div class="list-group list-group-flush overflow-auto mt-2" style="max-height:300px">
          <label v-for="u in filteredUsers()" :key="u.id" class="list-group-item list-group-item-action d-flex align-items-center gap-2">
            <input type="checkbox" class="form-check-input m-0" :value="u.id" v-model="selectedUserIds" />
            <span>
              {{ u.name }}
              <code class="text-muted small font-monospace">{{ u.username }}</code>
              <span v-if="u.profile?.nis" class="text-muted small ms-1">({{ u.profile.nis }})</span>
            </span>
          </label>
          <p v-if="!filteredUsers().length" class="text-center text-muted small p-3">Tidak ada peserta ditemukan</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="adding || !selectedUserIds.length" @click="handleAdd">
          <span v-if="adding" class="spinner-border spinner-border-sm me-1"></span>
          Tambahkan ({{ selectedUserIds.length }})
        </button>
      </template>
    </BaseModal>

    <!-- Confirm Modal -->
    <BaseConfirmModal
      v-if="confirmModal.show.value"
      :title="confirmModal.title.value"
      :message="confirmModal.message.value"
      confirm-label="Keluarkan"
      confirm-variant="danger"
      :loading="confirmModal.loading.value"
      @confirm="confirmModal.confirm()"
      @close="confirmModal.close()"
    />
</template>
