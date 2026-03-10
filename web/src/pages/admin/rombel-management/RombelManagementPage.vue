<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BaseConfirmModal from '../../../components/ui/BaseConfirmModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { userApi, type UserItem } from '../../../api/user.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useToastStore } from '@/stores/toast.store'

const toast = useToastStore()

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

function handleRemove(user: UserItem) {
  if (!selectedRombel.value) return
  confirmModal.ask(
    'Keluarkan Anggota',
    `Keluarkan "${user.name}" dari rombel ini?`,
    async () => {
      await rombelApi.removeUsers(selectedRombel.value!.id, [user.id])
      toast.success('Anggota berhasil dikeluarkan')
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
      toast.success(`${count} anggota berhasil dikeluarkan`)
      await fetchMembers()
      await fetchRombels()
    }
  )
}

// ============================
// Add Modal - Enhanced UX
// ============================
const showAddModal = ref(false)
const addTab = ref<'unassigned' | 'search' | 'transfer'>('unassigned')
const addUsers = ref<UserItem[]>([])
const addSelectedIds = ref<number[]>([])
const addSearch = ref('')
const addLoading = ref(false)
const adding = ref(false)
const addPage = ref(1)
const addTotalPages = ref(1)
const addTotal = ref(0)

// Transfer tab
const transferRombelId = ref<number | null>(null)

function openAddModal() {
  addTab.value = 'unassigned'
  addSelectedIds.value = []
  addSearch.value = ''
  addPage.value = 1
  transferRombelId.value = null
  showAddModal.value = true
  fetchAddUsers()
}

watch(addTab, () => {
  addSelectedIds.value = []
  addSearch.value = ''
  addPage.value = 1
  transferRombelId.value = null
  fetchAddUsers()
})

let addSearchTimeout: ReturnType<typeof setTimeout> | null = null
function onAddSearchChange() {
  if (addSearchTimeout) clearTimeout(addSearchTimeout)
  addSearchTimeout = setTimeout(() => {
    addPage.value = 1
    fetchAddUsers()
  }, 300)
}

async function fetchAddUsers() {
  if (!selectedRombel.value) return
  addLoading.value = true
  try {
    const params: Record<string, unknown> = {
      page: addPage.value,
      per_page: 20,
      role: 'peserta',
      exclude_rombel_id: selectedRombel.value.id,
    }
    if (addSearch.value.trim()) {
      params.search = addSearch.value.trim()
    }

    if (addTab.value === 'unassigned') {
      params.no_rombel = true
    } else if (addTab.value === 'transfer' && transferRombelId.value) {
      params.rombel_id = transferRombelId.value
    }

    const res = await userApi.list(params as Parameters<typeof userApi.list>[0])
    addUsers.value = res.data.data ?? []
    addTotalPages.value = res.data.meta?.total_pages ?? 1
    addTotal.value = res.data.meta?.total ?? 0
  } finally {
    addLoading.value = false
  }
}

// Select all in add modal
const addAllSelected = computed(() =>
  addUsers.value.length > 0 && addUsers.value.every(u => addSelectedIds.value.includes(u.id))
)
const addSomeSelected = computed(() =>
  !addAllSelected.value && addUsers.value.some(u => addSelectedIds.value.includes(u.id))
)

function toggleAddSelectAll() {
  if (addAllSelected.value) {
    // Deselect only current page
    const pageIds = new Set(addUsers.value.map(u => u.id))
    addSelectedIds.value = addSelectedIds.value.filter(id => !pageIds.has(id))
  } else {
    // Add current page to selection
    const current = new Set(addSelectedIds.value)
    for (const u of addUsers.value) current.add(u.id)
    addSelectedIds.value = Array.from(current)
  }
}

function toggleAddUser(id: number) {
  const idx = addSelectedIds.value.indexOf(id)
  if (idx >= 0) addSelectedIds.value.splice(idx, 1)
  else addSelectedIds.value.push(id)
}

function selectAllPages() {
  // Quick assign all from current filter (up to 500)
  confirmModal.ask(
    'Tambahkan Semua',
    `Tambahkan semua ${addTotal.value} siswa yang ditampilkan ke rombel "${selectedRombel.value?.name}"?`,
    async () => {
      try {
        // Fetch all IDs with large per_page
        const res = await userApi.list({
          per_page: 500,
          role: 'peserta',
          exclude_rombel_id: selectedRombel.value!.id,
          ...(addTab.value === 'unassigned' ? { no_rombel: true } : {}),
          ...(addTab.value === 'transfer' && transferRombelId.value ? { rombel_id: transferRombelId.value } : {}),
          ...(addSearch.value.trim() ? { search: addSearch.value.trim() } : {}),
        } as Parameters<typeof userApi.list>[0])
        const ids = (res.data.data ?? []).map((u: UserItem) => u.id)
        if (ids.length === 0) {
          toast.error('Tidak ada siswa untuk ditambahkan')
          return
        }
        await rombelApi.assignUsers(selectedRombel.value!.id, ids)
        toast.success(`${ids.length} siswa berhasil ditambahkan`)
        showAddModal.value = false
        await fetchMembers()
        await fetchRombels()
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menambahkan siswa')
      }
    }
  )
}

async function handleAdd() {
  if (!selectedRombel.value || !addSelectedIds.value.length) return
  adding.value = true
  try {
    await rombelApi.assignUsers(selectedRombel.value.id, addSelectedIds.value)
    toast.success(`${addSelectedIds.value.length} siswa berhasil ditambahkan`)
    showAddModal.value = false
    await fetchMembers()
    await fetchRombels()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menambahkan siswa')
  } finally {
    adding.value = false
  }
}

// Other rombels for transfer tab (excluding current)
const otherRombels = computed(() =>
  rombels.value.filter(r => r.id !== selectedRombel.value?.id)
)

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
              <span class="badge bg-secondary-lt">{{ r.students_count ?? 0 }}</span>
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
            <p class="empty-subtitle text-muted">Klik salah satu rombel di sebelah kiri.</p>
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

    <!-- Enhanced Add Member Modal -->
    <BaseModal v-if="showAddModal" title="Tambah Anggota Rombel" size="lg" @close="showAddModal = false">
      <!-- Tabs -->
      <ul class="nav nav-tabs mb-3">
        <li class="nav-item">
          <a class="nav-link" :class="{ active: addTab === 'unassigned' }" href="#" @click.prevent="addTab = 'unassigned'">
            <i class="ti ti-user-question me-1"></i>Belum Punya Rombel
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link" :class="{ active: addTab === 'search' }" href="#" @click.prevent="addTab = 'search'">
            <i class="ti ti-search me-1"></i>Cari Semua Peserta
          </a>
        </li>
        <li class="nav-item">
          <a class="nav-link" :class="{ active: addTab === 'transfer' }" href="#" @click.prevent="addTab = 'transfer'">
            <i class="ti ti-transfer me-1"></i>Dari Rombel Lain
          </a>
        </li>
      </ul>

      <!-- Transfer: rombel selector -->
      <div v-if="addTab === 'transfer'" class="mb-3">
        <select v-model.number="transferRombelId" class="form-select form-select-sm" @change="addPage = 1; addSelectedIds = []; fetchAddUsers()">
          <option :value="null">-- Pilih Rombel Asal --</option>
          <option v-for="r in otherRombels" :key="r.id" :value="r.id">{{ r.name }} ({{ r.students_count ?? 0 }} siswa)</option>
        </select>
      </div>

      <!-- Search -->
      <div class="input-icon mb-3">
        <span class="input-icon-addon"><i class="ti ti-search"></i></span>
        <input
          v-model="addSearch"
          class="form-control form-control-sm"
          placeholder="Cari nama, username, NIS..."
          @input="onAddSearchChange"
        />
      </div>

      <!-- Info bar -->
      <div class="d-flex align-items-center justify-content-between mb-2">
        <span class="text-muted small">
          <template v-if="addTab === 'unassigned'">Siswa yang belum terdaftar di rombel manapun</template>
          <template v-else-if="addTab === 'search'">Semua peserta (kecuali yang sudah di rombel ini)</template>
          <template v-else-if="addTab === 'transfer'">
            <template v-if="transferRombelId">Siswa dari rombel lain (akan ditambahkan juga ke rombel ini)</template>
            <template v-else>Pilih rombel asal di atas</template>
          </template>
        </span>
        <div class="d-flex align-items-center gap-2">
          <span class="badge bg-blue-lt">{{ addTotal }} siswa</span>
          <button
            v-if="addTotal > 0 && !(addTab === 'transfer' && !transferRombelId)"
            class="btn btn-sm btn-outline-primary"
            @click="selectAllPages"
          >
            <i class="ti ti-checks me-1"></i>Tambahkan Semua ({{ addTotal }})
          </button>
        </div>
      </div>

      <!-- User table -->
      <div class="table-responsive" style="max-height: 360px; overflow-y: auto;">
        <table class="table table-vcenter table-sm mb-0">
          <thead class="sticky-top bg-white">
            <tr>
              <th style="width: 40px">
                <input
                  type="checkbox"
                  class="form-check-input m-0 align-middle"
                  :checked="addAllSelected && addUsers.length > 0"
                  :indeterminate="addSomeSelected"
                  :disabled="addUsers.length === 0"
                  @change="toggleAddSelectAll"
                />
              </th>
              <th>Nama</th>
              <th>NIS</th>
              <th>Username</th>
            </tr>
          </thead>
          <tbody>
            <template v-if="addLoading">
              <tr v-for="n in 5" :key="n">
                <td v-for="c in 4" :key="c">
                  <div class="placeholder-glow"><span class="placeholder col-8" /></div>
                </td>
              </tr>
            </template>
            <template v-else-if="addUsers.length">
              <tr
                v-for="u in addUsers" :key="u.id"
                class="cursor-pointer"
                :class="{ 'bg-blue-lt': addSelectedIds.includes(u.id) }"
                @click="toggleAddUser(u.id)"
              >
                <td>
                  <input
                    type="checkbox"
                    class="form-check-input m-0"
                    :checked="addSelectedIds.includes(u.id)"
                    @click.stop
                    @change="toggleAddUser(u.id)"
                  />
                </td>
                <td class="fw-medium">{{ u.name }}</td>
                <td class="text-muted small">{{ u.profile?.nis ?? '-' }}</td>
                <td><code class="text-muted small font-monospace">{{ u.username }}</code></td>
              </tr>
            </template>
            <tr v-else>
              <td colspan="4" class="text-center text-muted py-4">
                <template v-if="addTab === 'transfer' && !transferRombelId">Pilih rombel asal terlebih dahulu</template>
                <template v-else>Tidak ada siswa ditemukan</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination in modal -->
      <div v-if="addTotalPages > 1" class="mt-2">
        <BasePagination :page="addPage" :total-pages="addTotalPages" :total="addTotal" :per-page="20" @change="p => { addPage = p; fetchAddUsers() }" />
      </div>

      <!-- Selected count -->
      <div v-if="addSelectedIds.length" class="alert alert-info mt-3 mb-0 py-2 d-flex align-items-center gap-2">
        <i class="ti ti-info-circle"></i>
        <span>{{ addSelectedIds.length }} siswa dipilih (bisa dari beberapa halaman)</span>
        <button class="btn btn-sm btn-ghost-secondary ms-auto" @click="addSelectedIds = []">Reset</button>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="adding || !addSelectedIds.length" @click="handleAdd">
          <span v-if="adding" class="spinner-border spinner-border-sm me-1"></span>
          Tambahkan ({{ addSelectedIds.length }})
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
