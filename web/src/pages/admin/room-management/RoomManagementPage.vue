<script setup lang="ts">
import { ref, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { roomApi, type Room } from '../../../api/room.api'
import { userApi, type UserItem } from '../../../api/user.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const columns = [
  { key: 'name', label: 'Nama' },
  { key: 'username', label: 'Username' },
  { key: 'actions', label: 'Aksi', width: '80px' },
]

const rooms = ref<Room[]>([])
const selectedRoom = ref<Room | null>(null)
const members = ref<UserItem[]>([])
const loadingRooms = ref(false)
const loadingMembers = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

const showAddModal = ref(false)
const allUsers = ref<UserItem[]>([])
const selectedUserIds = ref<number[]>([])
const searchUser = ref('')
const adding = ref(false)

async function fetchRooms() {
  loadingRooms.value = true
  try {
    const res = await roomApi.list({ per_page: 100 })
    rooms.value = res.data.data ?? []
  } finally {
    loadingRooms.value = false
  }
}

async function selectRoom(r: Room) {
  selectedRoom.value = r
  page.value = 1
  await fetchMembers()
}

async function fetchMembers() {
  if (!selectedRoom.value) return
  loadingMembers.value = true
  try {
    const res = await roomApi.getUsers(selectedRoom.value.id, { page: page.value, per_page: 20 })
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
  showAddModal.value = true
}

async function handleAdd() {
  if (!selectedRoom.value || !selectedUserIds.value.length) return
  adding.value = true
  try {
    await roomApi.assignUsers(selectedRoom.value.id, selectedUserIds.value)
    showAddModal.value = false
    await fetchMembers()
  } finally {
    adding.value = false
  }
}

async function handleRemove(user: UserItem) {
  if (!selectedRoom.value) return
  if (!confirm(`Keluarkan "${user.name}" dari ruangan ini?`)) return
  await roomApi.removeUsers(selectedRoom.value.id, [user.id])
  await fetchMembers()
}

const filteredUsers = () => allUsers.value.filter(u =>
  u.name.toLowerCase().includes(searchUser.value.toLowerCase()) ||
  u.username.toLowerCase().includes(searchUser.value.toLowerCase())
)

onMounted(fetchRooms)
</script>

<template>
    <BasePageHeader
      title="Pengaitan Ruangan"
      subtitle="Kelola penempatan peserta di ruangan ujian"
      :breadcrumbs="[{ label: 'Manajemen', to: '/admin' }, { label: 'Pengaitan Ruangan' }]"
    />

    <div class="row g-3">
      <div class="col-md-3">
        <div class="card h-100">
          <div class="card-header fw-semibold small text-muted text-uppercase">Daftar Ruangan</div>
          <div v-if="loadingRooms" class="p-3 text-center text-muted small">Memuat...</div>
          <div class="list-group list-group-flush">
            <a
              v-for="r in rooms" :key="r.id"
              class="list-group-item list-group-item-action d-flex align-items-center gap-2"
              :class="{ active: selectedRoom?.id === r.id }"
              href="#"
              @click.prevent="selectRoom(r)"
            >
              <i class="ti ti-door"></i>
              <span>{{ r.name }}</span>
              <span class="badge bg-secondary-lt text-secondary ms-auto">{{ r.capacity }}</span>
            </a>
          </div>
          <div v-if="!loadingRooms && !rooms.length" class="p-3 text-center text-muted small">Belum ada ruangan</div>
        </div>
      </div>

      <div class="col">
        <div class="card h-100">
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
              <span class="card-title mb-0">{{ selectedRoom.name }}</span>
              <button class="btn btn-sm btn-primary" @click="openAddModal"><i class="ti ti-user-plus"></i>
                Tambah Peserta</button>
            </div>
            <BaseTable :columns="columns" :loading="loadingMembers" empty="Belum ada peserta">
              <tr v-for="item in members" :key="item.id">
                <td class="fw-medium">{{ item.name }}</td>
                <td><code class="text-muted small font-monospace">{{ item.username }}</code></td>
                <td>
                  <a href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="handleRemove(item)" title="Keluarkan">
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

    <BaseModal v-if="showAddModal" title="Tambah Peserta ke Ruangan" size="md" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchUser" class="form-control" placeholder="Cari peserta..." />
        </div>
        <div class="border rounded overflow-auto mt-2" style="max-height:18rem">
          <label v-for="u in filteredUsers()" :key="u.id" class="d-flex align-items-center gap-2 px-3 py-2 cursor-pointer">
            <input type="checkbox" :value="u.id" v-model="selectedUserIds" />
            <span>{{ u.name }} <code class="text-muted small font-monospace">{{ u.username }}</code></span>
          </label>
          <p v-if="!filteredUsers().length" class="text-center text-muted small p-3">Tidak ada peserta ditemukan</p>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="adding" @click="handleAdd"><span v-if="adding" class="spinner-border spinner-border-sm me-1"></span>Tambahkan ({{ selectedUserIds.length }})</button>
      </template>
    </BaseModal>
</template>

