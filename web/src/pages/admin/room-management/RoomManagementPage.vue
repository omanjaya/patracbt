<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BaseConfirmModal from '../../../components/ui/BaseConfirmModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { roomApi, type Room } from '../../../api/room.api'
import { userApi, type UserItem } from '../../../api/user.api'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import { useConfirmModal } from '@/composables/useConfirmModal'

const rooms = ref<Room[]>([])
const selectedRoom = ref<Room | null>(null)
const members = ref<UserItem[]>([])
const loadingRooms = ref(false)
const loadingMembers = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

// Search & filter
const searchMember = ref('')
const statusFilter = ref<'' | 'true' | 'false'>('')
const rombelFilter = ref<number | ''>('')
const rombelsList = ref<Rombel[]>([])

// Bulk selection
const selectedMemberIds = ref<number[]>([])

// Confirm modal
const confirmModal = useConfirmModal()

// ─── Add Member Modal ───────────────────────────────────────────
const showAddModal = ref(false)
const addTab = ref<'no_room' | 'all' | 'from_room'>('no_room')
const addSearch = ref('')
const addPage = ref(1)
const addTotal = ref(0)
const addTotalPages = ref(1)
const addUsers = ref<UserItem[]>([])
const addLoading = ref(false)
const addSelected = ref<number[]>([])
const adding = ref(false)
const sourceRoomId = ref<number | ''>('')

// Debounce
let searchTimeout: ReturnType<typeof setTimeout> | null = null
function onSearchChange() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => { page.value = 1; fetchMembers() }, 300)
}

let addSearchTimeout: ReturnType<typeof setTimeout> | null = null
function onAddSearchChange() {
  if (addSearchTimeout) clearTimeout(addSearchTimeout)
  addSearchTimeout = setTimeout(() => { addPage.value = 1; fetchAddUsers() }, 300)
}

function onStatusFilterChange() { page.value = 1; fetchMembers() }
function onRombelFilterChange() { page.value = 1; fetchMembers() }

// Select all logic — member table
const allSelected = computed(() =>
  members.value.length > 0 && members.value.every(m => selectedMemberIds.value.includes(m.id))
)
const someSelected = computed(() =>
  !allSelected.value && members.value.some(m => selectedMemberIds.value.includes(m.id))
)
function toggleSelectAll() {
  if (allSelected.value) selectedMemberIds.value = []
  else selectedMemberIds.value = members.value.map(m => m.id)
}
function toggleMember(id: number) {
  const idx = selectedMemberIds.value.indexOf(id)
  if (idx >= 0) selectedMemberIds.value.splice(idx, 1)
  else selectedMemberIds.value.push(id)
}

// Select all logic — add modal
const addAllSelected = computed(() =>
  addUsers.value.length > 0 && addUsers.value.every(u => addSelected.value.includes(u.id))
)
const addSomeSelected = computed(() =>
  !addAllSelected.value && addUsers.value.some(u => addSelected.value.includes(u.id))
)
function toggleAddSelectAll() {
  if (addAllSelected.value) {
    const pageIds = new Set(addUsers.value.map(u => u.id))
    addSelected.value = addSelected.value.filter(id => !pageIds.has(id))
  } else {
    const existing = new Set(addSelected.value)
    for (const u of addUsers.value) existing.add(u.id)
    addSelected.value = [...existing]
  }
}
function toggleAddUser(id: number) {
  const idx = addSelected.value.indexOf(id)
  if (idx >= 0) addSelected.value.splice(idx, 1)
  else addSelected.value.push(id)
}

// ─── Data Fetching ──────────────────────────────────────────────

async function fetchRooms() {
  loadingRooms.value = true
  try {
    const res = await roomApi.list({ per_page: 100 })
    rooms.value = res.data.data ?? []
  } finally {
    loadingRooms.value = false
  }
}

async function fetchRombels() {
  try {
    const res = await rombelApi.list({ per_page: 100 })
    rombelsList.value = res.data.data ?? []
  } catch { /* silently fail */ }
}

async function selectRoom(r: Room) {
  selectedRoom.value = r
  page.value = 1
  searchMember.value = ''
  statusFilter.value = ''
  rombelFilter.value = ''
  selectedMemberIds.value = []
  await fetchMembers()
}

async function fetchMembers() {
  if (!selectedRoom.value) return
  loadingMembers.value = true
  selectedMemberIds.value = []
  try {
    const params: Record<string, unknown> = {
      page: page.value, per_page: 20, room_id: selectedRoom.value.id,
    }
    if (searchMember.value.trim()) params.search = searchMember.value.trim()
    if (statusFilter.value !== '') params.is_active = statusFilter.value === 'true'
    if (rombelFilter.value !== '') params.rombel_id = rombelFilter.value
    const res = await userApi.list(params as Parameters<typeof userApi.list>[0])
    members.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } finally {
    loadingMembers.value = false
  }
}

async function fetchAddUsers() {
  if (!selectedRoom.value) return
  addLoading.value = true
  try {
    const params: Record<string, unknown> = {
      page: addPage.value, per_page: 20, role: 'peserta',
    }
    if (addSearch.value.trim()) params.search = addSearch.value.trim()

    if (addTab.value === 'no_room') {
      params.no_room = true
    } else if (addTab.value === 'all') {
      params.exclude_room_id = selectedRoom.value.id
    } else if (addTab.value === 'from_room' && sourceRoomId.value) {
      params.room_id = sourceRoomId.value
    }

    const res = await userApi.list(params as Parameters<typeof userApi.list>[0])
    addUsers.value = res.data.data ?? []
    addTotal.value = res.data.meta?.total ?? 0
    addTotalPages.value = res.data.meta?.total_pages ?? 1
  } finally {
    addLoading.value = false
  }
}

function openAddModal() {
  addTab.value = 'no_room'
  addSearch.value = ''
  addPage.value = 1
  addSelected.value = []
  sourceRoomId.value = ''
  showAddModal.value = true
  fetchAddUsers()
}

watch(addTab, () => { addPage.value = 1; addSearch.value = ''; sourceRoomId.value = ''; fetchAddUsers() })
watch(sourceRoomId, () => { addPage.value = 1; fetchAddUsers() })

async function handleAdd() {
  if (!selectedRoom.value || !addSelected.value.length) return
  adding.value = true
  try {
    await roomApi.assignUsers(selectedRoom.value.id, addSelected.value)
    showAddModal.value = false
    await fetchMembers()
    await fetchRooms()
  } finally {
    adding.value = false
  }
}

function handleRemove(user: UserItem) {
  if (!selectedRoom.value) return
  confirmModal.ask(
    'Keluarkan Peserta',
    `Keluarkan "${user.name}" dari ruangan ini?`,
    async () => {
      await roomApi.removeUsers(selectedRoom.value!.id, [user.id])
      await fetchMembers()
      await fetchRooms()
    }
  )
}

function handleBulkRemove() {
  if (!selectedRoom.value || !selectedMemberIds.value.length) return
  confirmModal.ask(
    'Keluarkan Peserta',
    `Keluarkan ${selectedMemberIds.value.length} peserta dari ruangan ini?`,
    async () => {
      await roomApi.removeUsers(selectedRoom.value!.id, [...selectedMemberIds.value])
      selectedMemberIds.value = []
      await fetchMembers()
      await fetchRooms()
    }
  )
}

const capacityText = computed(() => {
  if (!selectedRoom.value) return ''
  return `${total.value}/${selectedRoom.value.capacity} siswa`
})

const otherRooms = computed(() => rooms.value.filter(r => r.id !== selectedRoom.value?.id))

onMounted(() => { fetchRooms(); fetchRombels() })
</script>

<template>
    <BasePageHeader
      title="Pengaitan Ruangan"
      subtitle="Kelola penempatan peserta di ruangan ujian"
      :breadcrumbs="[{ label: 'Manajemen', to: '/admin' }, { label: 'Pengaitan Ruangan' }]"
    />

    <div class="row g-3">
      <!-- Room list sidebar -->
      <div class="col-md-3">
        <div class="card h-100 p-0">
          <div class="card-header fw-semibold small text-muted text-uppercase">Daftar Ruangan</div>
          <div class="list-group list-group-flush">
            <div v-if="loadingRooms" class="p-3 text-center text-muted small">Memuat...</div>
            <button
              v-for="r in rooms" :key="r.id"
              class="list-group-item list-group-item-action d-flex align-items-center gap-2 border-0"
              :class="{ active: selectedRoom?.id === r.id }"
              @click="selectRoom(r)"
            >
              <i class="ti ti-door"></i>
              <span class="flex-fill text-start">{{ r.name }}</span>
              <span class="badge bg-secondary-lt">{{ r.students_count ?? 0 }}/{{ r.capacity }}</span>
            </button>
            <div v-if="!loadingRooms && !rooms.length" class="p-3 text-center text-muted small">Belum ada ruangan</div>
          </div>
        </div>
      </div>

      <!-- Members -->
      <div class="col">
        <div class="card">
        <div v-if="!selectedRoom" class="card-body">
          <div class="empty">
            <div class="empty-icon">
              <i class="ti ti-door" style="font-size: 3rem;"></i>
            </div>
            <p class="empty-title">Pilih ruangan untuk melihat anggota</p>
            <p class="empty-subtitle text-muted">Klik salah satu ruangan di sebelah kiri.</p>
          </div>
        </div>
        <template v-else>
          <div class="card-header d-flex align-items-center justify-content-between">
            <div class="d-flex align-items-center gap-2">
              <span class="card-title mb-0">{{ selectedRoom.name }}</span>
              <span class="badge bg-blue-lt">{{ capacityText }}</span>
            </div>
            <div class="d-flex align-items-center gap-2">
              <button v-if="selectedMemberIds.length" class="btn btn-sm btn-danger" @click="handleBulkRemove">
                <i class="ti ti-user-minus me-1"></i>Keluarkan ({{ selectedMemberIds.length }})
              </button>
              <button class="btn btn-sm btn-primary" @click="openAddModal">
                <i class="ti ti-user-plus me-1"></i>Tambah Peserta
              </button>
            </div>
          </div>

          <!-- Search & filter bar -->
          <div class="card-body border-bottom py-2">
            <div class="row g-2 align-items-center">
              <div class="col-auto flex-fill">
                <div class="input-icon">
                  <span class="input-icon-addon"><i class="ti ti-search"></i></span>
                  <input v-model="searchMember" class="form-control form-control-sm" placeholder="Cari nama, username, atau NIS..." @input="onSearchChange" />
                </div>
              </div>
              <div class="col-auto">
                <select v-model="rombelFilter" class="form-select form-select-sm" style="min-width:150px" @change="onRombelFilterChange">
                  <option value="">Semua Rombel</option>
                  <option v-for="rb in rombelsList" :key="rb.id" :value="rb.id">{{ rb.name }}</option>
                </select>
              </div>
              <div class="col-auto">
                <select v-model="statusFilter" class="form-select form-select-sm" style="min-width:140px" @change="onStatusFilterChange">
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
                  <th style="width:40px">
                    <input type="checkbox" class="form-check-input m-0 align-middle" :checked="allSelected" :indeterminate="someSelected" @change="toggleSelectAll" />
                  </th>
                  <th>Nama</th>
                  <th>Username</th>
                  <th>NIS</th>
                  <th style="width:110px">Status</th>
                  <th style="width:80px">Aksi</th>
                </tr>
              </thead>
              <tbody>
                <template v-if="loadingMembers">
                  <tr v-for="n in 5" :key="n">
                    <td v-for="c in 6" :key="c"><div class="placeholder-glow"><span class="placeholder col-8" /></div></td>
                  </tr>
                </template>
                <template v-else-if="members.length">
                  <tr v-for="item in members" :key="item.id">
                    <td><input type="checkbox" class="form-check-input m-0" :checked="selectedMemberIds.includes(item.id)" @change="toggleMember(item.id)" /></td>
                    <td class="fw-medium">{{ item.name }}</td>
                    <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
                    <td><span class="text-muted small">{{ item.profile?.nis ?? '-' }}</span></td>
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
                    <i class="ti ti-database-off fs-4 mb-2 d-block opacity-50" />Belum ada peserta
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
    <BaseModal v-if="showAddModal" title="Tambah Peserta ke Ruangan" size="lg" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <!-- Tabs -->
        <ul class="nav nav-tabs">
          <li class="nav-item">
            <a class="nav-link" :class="{ active: addTab === 'no_room' }" href="#" @click.prevent="addTab = 'no_room'">
              <i class="ti ti-user-question me-1"></i>Belum Punya Ruangan
            </a>
          </li>
          <li class="nav-item">
            <a class="nav-link" :class="{ active: addTab === 'all' }" href="#" @click.prevent="addTab = 'all'">
              <i class="ti ti-users me-1"></i>Cari Semua Peserta
            </a>
          </li>
          <li class="nav-item">
            <a class="nav-link" :class="{ active: addTab === 'from_room' }" href="#" @click.prevent="addTab = 'from_room'">
              <i class="ti ti-transfer me-1"></i>Dari Ruangan Lain
            </a>
          </li>
        </ul>

        <!-- Source room selector -->
        <select v-if="addTab === 'from_room'" v-model="sourceRoomId" class="form-select form-select-sm">
          <option value="">-- Pilih ruangan sumber --</option>
          <option v-for="r in otherRooms" :key="r.id" :value="r.id">{{ r.name }} ({{ r.students_count ?? 0 }})</option>
        </select>

        <!-- Search -->
        <div class="input-icon">
          <span class="input-icon-addon"><i class="ti ti-search"></i></span>
          <input v-model="addSearch" class="form-control form-control-sm" placeholder="Cari nama, username, atau NIS..." @input="onAddSearchChange" />
        </div>

        <!-- User list -->
        <div class="table-responsive" style="max-height:350px;overflow-y:auto">
          <table class="table table-vcenter table-sm mb-0">
            <thead class="sticky-top bg-white">
              <tr>
                <th style="width:40px">
                  <input type="checkbox" class="form-check-input m-0" :checked="addAllSelected" :indeterminate="addSomeSelected" @change="toggleAddSelectAll" :disabled="!addUsers.length" />
                </th>
                <th>Nama</th>
                <th>Username</th>
                <th>NIS</th>
              </tr>
            </thead>
            <tbody>
              <template v-if="addLoading">
                <tr v-for="n in 5" :key="n">
                  <td v-for="c in 4" :key="c"><div class="placeholder-glow"><span class="placeholder col-8" /></div></td>
                </tr>
              </template>
              <template v-else-if="addUsers.length">
                <tr
                  v-for="u in addUsers" :key="u.id"
                  class="cursor-pointer"
                  :class="{ 'bg-primary-lt': addSelected.includes(u.id) }"
                  @click="toggleAddUser(u.id)"
                >
                  <td><input type="checkbox" class="form-check-input m-0" :checked="addSelected.includes(u.id)" @click.stop @change="toggleAddUser(u.id)" /></td>
                  <td class="fw-medium">{{ u.name }}</td>
                  <td><code class="text-muted small font-monospace">{{ u.username }}</code></td>
                  <td><span class="text-muted small">{{ u.profile?.nis ?? '-' }}</span></td>
                </tr>
              </template>
              <tr v-else>
                <td colspan="4" class="text-center text-muted py-4">
                  <template v-if="addTab === 'from_room' && !sourceRoomId">Pilih ruangan sumber terlebih dahulu</template>
                  <template v-else>Tidak ada peserta ditemukan</template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div v-if="addTotalPages > 1" class="d-flex align-items-center justify-content-between">
          <span class="text-muted small">{{ addTotal }} peserta</span>
          <BasePagination :page="addPage" :total-pages="addTotalPages" :total="addTotal" :per-page="20" @change="p => { addPage = p; fetchAddUsers() }" />
        </div>
      </div>

      <template #footer>
        <span v-if="addSelected.length" class="text-muted small me-auto">{{ addSelected.length }} dipilih</span>
        <button class="btn btn-secondary" @click="showAddModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="adding || !addSelected.length" @click="handleAdd">
          <span v-if="adding" class="spinner-border spinner-border-sm me-1"></span>
          <i v-else class="ti ti-user-plus me-1"></i>Tambahkan ({{ addSelected.length }})
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
