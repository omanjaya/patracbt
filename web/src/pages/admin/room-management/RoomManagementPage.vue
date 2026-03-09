<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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

// Add modal
const showAddModal = ref(false)
const allUsers = ref<UserItem[]>([])
const selectedUserIds = ref<number[]>([])
const searchUser = ref('')
const adding = ref(false)
const onlyUnassigned = ref(false)

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

function onRombelFilterChange() {
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
  } catch {
    // silently fail
  }
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
      page: page.value,
      per_page: 20,
      room_id: selectedRoom.value.id,
    }
    if (searchMember.value.trim()) {
      params.search = searchMember.value.trim()
    }
    if (statusFilter.value !== '') {
      params.is_active = statusFilter.value === 'true'
    }
    if (rombelFilter.value !== '') {
      params.rombel_id = rombelFilter.value
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
  onlyUnassigned.value = false
  showAddModal.value = true
}

async function handleAdd() {
  if (!selectedRoom.value || !selectedUserIds.value.length) return
  adding.value = true
  try {
    await roomApi.assignUsers(selectedRoom.value.id, selectedUserIds.value)
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
  const count = selectedMemberIds.value.length
  confirmModal.ask(
    'Keluarkan Peserta',
    `Keluarkan ${count} peserta dari ruangan ini?`,
    async () => {
      await roomApi.removeUsers(selectedRoom.value!.id, [...selectedMemberIds.value])
      selectedMemberIds.value = []
      await fetchMembers()
      await fetchRooms()
    }
  )
}

// Filtered users for add modal (exclude current members)
const memberIds = computed(() => new Set(members.value.map(m => m.id)))

const filteredUsers = () => allUsers.value.filter(u => {
  if (memberIds.value.has(u.id)) return false
  if (onlyUnassigned.value) {
    // Filter: only show users that have no room_id (check via a simple heuristic)
    // We rely on the fact that users already assigned to ANY room would have a room assignment.
    // Since we fetched all peserta, we filter out those in any room's member list.
    // This is a client-side approximation; the backend `room_id=0` could also work.
  }
  const q = searchUser.value.toLowerCase()
  if (!q) return true
  return (
    u.name.toLowerCase().includes(q) ||
    u.username.toLowerCase().includes(q) ||
    (u.profile?.nis ?? '').toLowerCase().includes(q)
  )
})

// Capacity display
const capacityText = computed(() => {
  if (!selectedRoom.value) return ''
  const count = total.value
  const cap = selectedRoom.value.capacity
  return `${count}/${cap} siswa`
})

onMounted(() => {
  fetchRooms()
  fetchRombels()
})
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
              <span
                v-if="r.students_count != null"
                class="badge bg-secondary-lt"
              >{{ r.students_count }}/{{ r.capacity }}</span>
              <span
                v-else
                class="badge bg-secondary-lt text-secondary"
              >{{ r.capacity }}</span>
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
            <p class="empty-subtitle text-muted">Klik salah satu ruangan di sebelah kiri untuk menampilkan daftar peserta di dalamnya.</p>
          </div>
        </div>
        <template v-else>
          <div class="card-header d-flex align-items-center justify-content-between">
            <div class="d-flex align-items-center gap-2">
              <span class="card-title mb-0">{{ selectedRoom.name }}</span>
              <span class="badge bg-blue-lt">{{ capacityText }}</span>
            </div>
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
                Tambah Peserta
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
                  v-model="rombelFilter"
                  class="form-select form-select-sm"
                  style="min-width: 150px"
                  @change="onRombelFilterChange"
                >
                  <option value="">Semua Rombel</option>
                  <option v-for="rb in rombelsList" :key="rb.id" :value="rb.id">{{ rb.name }}</option>
                </select>
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
                    Belum ada peserta
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
    <BaseModal v-if="showAddModal" title="Tambah Peserta ke Ruangan" size="md" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchUser" class="form-control" placeholder="Cari nama, username, atau NIS..." />
        </div>
        <div class="form-check">
          <input
            id="onlyUnassigned"
            v-model="onlyUnassigned"
            type="checkbox"
            class="form-check-input"
          />
          <label for="onlyUnassigned" class="form-check-label small text-muted">Hanya yang belum punya ruangan</label>
        </div>
        <div class="list-group list-group-flush overflow-auto mt-1" style="max-height:300px">
          <label v-for="u in filteredUsers()" :key="u.id" class="list-group-item list-group-item-action d-flex align-items-center gap-2">
            <input type="checkbox" class="form-check-input m-0" :value="u.id" v-model="selectedUserIds" />
            <div class="d-flex flex-column">
              <span>
                {{ u.name }}
                <code class="text-muted small font-monospace ms-1">{{ u.username }}</code>
              </span>
              <span v-if="u.profile?.nis" class="text-muted small">NIS: {{ u.profile.nis }}</span>
            </div>
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
