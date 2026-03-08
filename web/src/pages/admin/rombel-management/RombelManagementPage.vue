<script setup lang="ts">
import { ref, onMounted } from 'vue'
import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { userApi, type UserItem } from '../../../api/user.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const columns = [
  { key: 'name', label: 'Nama' },
  { key: 'username', label: 'Username' },
  { key: 'actions', label: 'Aksi', width: '80px' },
]

const rombels = ref<Rombel[]>([])
const selectedRombel = ref<Rombel | null>(null)
const members = ref<UserItem[]>([])
const loadingRombels = ref(false)
const loadingMembers = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

const showAddModal = ref(false)
const allUsers = ref<UserItem[]>([])
const selectedUserIds = ref<number[]>([])
const searchUser = ref('')
const adding = ref(false)

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
  await fetchMembers()
}

async function fetchMembers() {
  if (!selectedRombel.value) return
  loadingMembers.value = true
  try {
    const res = await userApi.list({ page: page.value, per_page: 20, rombel_id: selectedRombel.value.id })
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
  if (!selectedRombel.value || !selectedUserIds.value.length) return
  adding.value = true
  try {
    await rombelApi.assignUsers(selectedRombel.value.id, selectedUserIds.value)
    showAddModal.value = false
    await fetchMembers()
  } finally {
    adding.value = false
  }
}

async function handleRemove(user: UserItem) {
  if (!selectedRombel.value) return
  if (!confirm(`Keluarkan "${user.name}" dari rombel ini?`)) return
  await rombelApi.removeUsers(selectedRombel.value.id, [user.id])
  await fetchMembers()
}

const filteredUsers = () => allUsers.value.filter(u =>
  u.name.toLowerCase().includes(searchUser.value.toLowerCase()) ||
  u.username.toLowerCase().includes(searchUser.value.toLowerCase())
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
              {{ r.name }}
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
            <button class="btn btn-sm btn-primary" @click="openAddModal"><i class="ti ti-user-plus"></i>
              Tambah Anggota</button>
          </div>
          <BaseTable :columns="columns" :loading="loadingMembers" empty="Belum ada anggota">
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

    <BaseModal v-if="showAddModal" title="Tambah Anggota Rombel" size="md" @close="showAddModal = false">
      <div class="d-flex flex-column gap-3">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchUser" class="form-control" placeholder="Cari peserta..." />
        </div>
        <div class="list-group list-group-flush overflow-auto mt-2" style="max-height:300px">
          <label v-for="u in filteredUsers()" :key="u.id" class="list-group-item list-group-item-action d-flex align-items-center gap-2">
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

