<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'

import BaseTable from '../../../components/ui/BaseTable.vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import {
  examApi, type ExamSchedule, type CreateSchedulePayload,
  STATUS_LABELS, STATUS_COLORS,
} from '../../../api/exam.api'
import { questionBankApi, type QuestionBank } from '../../../api/question_bank.api'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { tagApi, type Tag as TagType } from '../../../api/tag.api'
import client from '../../../api/client'
import { useToastStore } from '../../../stores/toast.store'
import { useConfirmModal } from '@/composables/useConfirmModal'
import { useTableFilters } from '@/composables/useTableFilters'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()
const router = useRouter()
const confirmModal = useConfirmModal()

const columns = [
  { key: 'name', label: 'Nama Ujian' },
  { key: 'time', label: 'Waktu Pelaksanaan' },
  { key: 'status', label: 'Status' },
  { key: 'token', label: 'Token' },
  { key: 'actions', label: 'Aksi', width: '180px' },
]

const activeTab = ref<'active' | 'trashed'>('active')
const list = ref<ExamSchedule[]>([])
const { searchRaw, search, page, total, totalPages, loading } = useTableFilters(fetchList)
const statusFilter = ref('')

watch(statusFilter, () => { page.value = 1; fetchList() })

const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)

const allBanks = ref<QuestionBank[]>([])
const allRombels = ref<Rombel[]>([])
const allTags = ref<TagType[]>([])
const allRooms = ref<{id: number, name: string}[]>([])

// Cache warming state
const warmingId = ref<number | null>(null)
async function handleWarmCache(item: ExamSchedule) {
  warmingId.value = item.id
  try {
    const res = await client.post(`/api/v1/exam-schedules/${item.id}/warm-cache`)
    toast.success(`Cache soal berhasil dimuat (${res.data.data?.cached ?? 0} soal)`)
  } catch {
    toast.error('Gagal memuat cache soal')
  } finally {
    warmingId.value = null
  }
}

// Token modal states
const showTokenModal = ref(false)
const selectedScheduleToken = ref<ExamSchedule | null>(null)
const globalToken = ref('')
const roomTokens = ref<{room_id: number; room_name: string; token: string}[]>([])

interface BankRow { question_bank_id: number; question_count: number; weight: number }
const form = reactive({
  name: '',
  start_time: '',
  end_time: '',
  duration_minutes: 60,
  allow_see_result: true,
  max_violations: 3,
  randomize_questions: false,
  randomize_options: false,
  next_exam_schedule_id: null as number | null,
  selected_banks: [] as BankRow[],
  rombel_ids: [] as number[],
  tag_ids: [] as number[],
})

let searchController: AbortController | null = null

async function fetchList() {
  searchController?.abort()
  searchController = new AbortController()
  loading.value = true
  try {
    if (activeTab.value === 'active') {
      const res = await examApi.listSchedules({ page: page.value, per_page: 20, search: search.value, status: statusFilter.value || undefined }, { signal: searchController.signal })
      list.value = res.data.data ?? []
      total.value = res.data.meta?.total ?? 0
    } else {
      const res = await examApi.listTrashedSchedules({ page: page.value, per_page: 20, search: search.value }, { signal: searchController.signal })
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
  activeTab.value = tab; page.value = 1; search.value = ''; fetchList()
}

async function fetchDependencies() {
  const [banksRes, rombelsRes, tagsRes, roomsRes] = await Promise.all([
    questionBankApi.list({ per_page: 200 }),
    rombelApi.list({ per_page: 200 }),
    tagApi.listAll(),
    client.get('/admin/rooms', { params: { per_page: 200 } }),
  ])
  allBanks.value = banksRes.data.data ?? []
  allRombels.value = rombelsRes.data.data ?? []
  allTags.value = tagsRes.data.data ?? []
  allRooms.value = roomsRes.data.data ?? []
}

function resetForm() {
  Object.assign(form, {
    name: '', start_time: '', end_time: '',
    duration_minutes: 60, allow_see_result: true,
    max_violations: 3, randomize_questions: false,
    randomize_options: false, next_exam_schedule_id: null,
    selected_banks: [], rombel_ids: [], tag_ids: [],
  })
}

function openCreate() {
  isEdit.value = false; editId.value = null
  resetForm()
  showModal.value = true
}

function openEdit(item: ExamSchedule) {
  isEdit.value = true; editId.value = item.id
  form.name = item.name
  form.start_time = item.start_time.slice(0, 16)
  form.end_time = item.end_time.slice(0, 16)
  form.duration_minutes = item.duration_minutes
  form.allow_see_result = item.allow_see_result
  form.max_violations = item.max_violations
  form.randomize_questions = item.randomize_questions
  form.randomize_options = item.randomize_options
  form.next_exam_schedule_id = item.next_exam_schedule_id ?? null
  form.selected_banks = (item.question_banks ?? []).map(b => ({ question_bank_id: b.question_bank_id, question_count: b.question_count, weight: b.weight ?? 1 }))
  form.rombel_ids = (item.rombels ?? []).map(r => r.rombel_id)
  form.tag_ids = (item.tags ?? []).map(t => t.tag_id)
  showModal.value = true
}

async function handleSave() {
  saving.value = true
  try {
    const payload: CreateSchedulePayload = {
      name: form.name,
      start_time: new Date(form.start_time).toISOString(),
      end_time: new Date(form.end_time).toISOString(),
      duration_minutes: form.duration_minutes,
      allow_see_result: form.allow_see_result,
      max_violations: form.max_violations,
      randomize_questions: form.randomize_questions,
      randomize_options: form.randomize_options,
      next_exam_schedule_id: form.next_exam_schedule_id ?? undefined,
      question_banks: form.selected_banks.filter(b => b.question_bank_id).map(b => ({ ...b, weight: b.weight ?? 1 })),
      rombel_ids: form.rombel_ids,
      tag_ids: form.tag_ids,
    }
    if (isEdit.value && editId.value) await examApi.updateSchedule(editId.value, payload)
    else await examApi.createSchedule(payload)
    showModal.value = false
    toast.success('Jadwal ujian berhasil disimpan')
    await fetchList()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan jadwal ujian')
  } finally {
    saving.value = false
  }
}

function handleDelete(item: ExamSchedule) {
  confirmModal.ask(
    'Hapus Jadwal',
    `Hapus jadwal ujian "${item.name}"? Akan masuk ke sampah.`,
    async () => {
      try {
        await examApi.deleteSchedule(item.id)
        toast.success('Jadwal berhasil dihapus')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus jadwal')
      }
    },
  )
}

function handleRestore(item: ExamSchedule) {
  confirmModal.ask(
    'Pulihkan Jadwal',
    `Pulihkan jadwal "${item.name}"?`,
    async () => {
      try {
        await examApi.restoreSchedule(item.id)
        toast.success('Jadwal berhasil dipulihkan')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal memulihkan jadwal')
      }
    },
  )
}

function handleForceDelete(item: ExamSchedule) {
  confirmModal.ask(
    'Hapus Permanen',
    `Hapus permanen jadwal "${item.name}"? Tindakan ini tidak dapat dibatalkan.`,
    async () => {
      try {
        await examApi.forceDeleteSchedule(item.id)
        toast.success('Jadwal berhasil dihapus permanen')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus jadwal')
      }
    },
  )
}

function changeStatus(item: ExamSchedule, status: string) {
  const statusLabels: Record<string, string> = { published: 'Publikasi', active: 'Aktifkan', finished: 'Selesaikan' }
  confirmModal.ask(
    'Konfirmasi Ubah Status',
    `Ubah status jadwal "${item.name}" menjadi "${statusLabels[status] ?? status}"?`,
    async () => {
      try {
        await examApi.updateStatus(item.id, status)
        toast.success('Status jadwal berhasil diubah')
        await fetchList()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal mengubah status')
      }
    },
  )
}

const cloningId = ref<number | null>(null)
async function handleClone(item: ExamSchedule) {
  cloningId.value = item.id
  try {
    const res = await examApi.cloneSchedule(item.id)
    const cloned = res.data.data
    toast.success(`Jadwal "${item.name}" berhasil diduplikat`)
    await fetchList()
    // Open edit for the cloned schedule
    if (cloned?.id) {
      const detailRes = await examApi.getSchedule(cloned.id)
      openEdit(detailRes.data.data)
    }
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menduplikat jadwal')
  } finally {
    cloningId.value = null
  }
}

function addBank() { form.selected_banks.push({ question_bank_id: 0, question_count: 0, weight: 1 }) }
function removeBank(i: number) { form.selected_banks.splice(i, 1) }

function formatDate(d: string) {
  return new Date(d).toLocaleString('id-ID', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// Token Setup Methods
async function openTokenSetup(item: ExamSchedule) {
  selectedScheduleToken.value = item
  globalToken.value = ''
  roomTokens.value = allRooms.value.map(r => ({ room_id: r.id, room_name: r.name, token: '' }))
  showTokenModal.value = true
  
  try {
    const res = await examApi.getRoomTokens(item.id)
    const data = res.data.data
    globalToken.value = data.global_token || ''
    const assignedRooms: any[] = data.rooms || []
    
    assignedRooms.forEach(ar => {
      const idx = roomTokens.value.findIndex(rt => rt.room_id === ar.room_id)
      if (idx !== -1 && roomTokens.value[idx]) roomTokens.value[idx].token = ar.token
    })
  } catch (e) {
    toast.error('Gagal memuat token ruangan')
  }
}

async function handleSaveTokens() {
  if (!selectedScheduleToken.value) return
  saving.value = true
  try {
    const payload = {
      global_token: globalToken.value,
      rooms: roomTokens.value.filter(r => r.token.trim() !== '').map(r => ({
        room_id: r.room_id,
        token: r.token.trim()
      }))
    }
    await examApi.saveRoomTokens(selectedScheduleToken.value.id, payload)
    toast.success('Token pengawas berhasil disimpan')
    showTokenModal.value = false
  } catch (e) {
    toast.error('Gagal menyimpan token')
  } finally {
    saving.value = false
  }
}

function generateRandomToken(length = 6) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function setGlobalRandomToken() {
  globalToken.value = generateRandomToken()
}

function setRoomRandomToken(room: any) {
  room.token = generateRandomToken()
}

onMounted(() => { fetchList(); fetchDependencies() })
</script>

<template>
    <BasePageHeader
      title="Penjadwalan Ujian"
      subtitle="Buat dan kelola jadwal ujian CBT"
      :breadcrumbs="[{ label: 'Jadwal Ujian' }]"
    >
      <template #actions>
        <button class="btn btn-primary" v-if="activeTab === 'active'" @click="openCreate"><i class="ti ti-plus"></i>
          Buat Jadwal</button>
      </template>
    </BasePageHeader>

    <!-- Tabs -->
    <ul class="nav nav-tabs mb-3">
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'active' }" href="#" @click.prevent="switchTab('active')">Jadwal Aktif</a>
      </li>
      <li class="nav-item">
        <a class="nav-link" :class="{ active: activeTab === 'trashed' }" href="#" @click.prevent="switchTab('trashed')">Sampah</a>
      </li>
    </ul>

    <div class="card">
      <div class="card-header d-flex align-items-center gap-2 flex-wrap">
        <div class="input-group">
          <span class="input-group-text"><i class="ti ti-search"></i></span>
          <input v-model="searchRaw" class="form-control" placeholder="Cari jadwal..." />
        </div>
        <select v-model="statusFilter" class="form-select form-select-sm">
          <option value="">Semua Status</option>
          <option value="draft">Draft</option>
          <option value="published">Dipublikasi</option>
          <option value="active">Aktif</option>
          <option value="finished">Selesai</option>
        </select>
      </div>

      <BaseTable :columns="columns" :loading="loading" :row-count="list.length" empty="Belum ada jadwal ujian">
        <template #empty>
          <i class="ti ti-calendar-off fs-1 mb-2 d-block opacity-50"></i>
          <p class="text-muted mb-0">Belum ada jadwal ujian</p>
        </template>
        <template #default>
          <tr v-for="item in list" :key="item.id">
            <td>
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-calendar-clock"></i>
                <div>
                  <p class="fw-medium">{{ item.name }}</p>
                  <p class="text-muted small">{{ item.duration_minutes }} menit</p>
                </div>
              </div>
            </td>
            <td>
              <p class="small">{{ formatDate(item.start_time) }}</p>
              <p class="text-muted small">s/d {{ formatDate(item.end_time) }}</p>
            </td>
            <td>
              <span class="badge" :class="`bg-${STATUS_COLORS[item.status as keyof typeof STATUS_COLORS] === 'default' ? 'secondary' : STATUS_COLORS[item.status as keyof typeof STATUS_COLORS]}-lt`">{{ STATUS_LABELS[item.status] }}</span>
            </td>
            <td>
              <code class="badge bg-secondary-lt text-secondary font-monospace fs-6">{{ item.token }}</code>
            </td>
            <td>
              <div class="d-flex gap-1">
                <template v-if="activeTab === 'active'">
                  <!-- 3 primary actions: Preview, Edit, Hapus -->
                  <router-link :to="`${router.currentRoute.value.path}/${item.id}/preview`" class="btn btn-sm btn-ghost-cyan" aria-label="Preview soal">
                    <i class="ti ti-eye"></i>
                  </router-link>
                  <button class="btn btn-sm btn-ghost-secondary" :aria-label="`Edit jadwal ${item.name}`" @click="openEdit(item)">
                    <i class="ti ti-pencil"></i>
                  </button>
                  <button class="btn btn-sm btn-ghost-danger" :aria-label="`Hapus jadwal ${item.name}`" @click="handleDelete(item)">
                    <i class="ti ti-trash"></i>
                  </button>
                  <!-- Dropdown "Lainnya" -->
                  <div class="dropdown">
                    <button class="btn btn-sm btn-ghost-secondary" data-bs-toggle="dropdown" :aria-label="`Aksi lainnya untuk ${item.name}`" aria-expanded="false">
                      <i class="ti ti-dots-vertical"></i>
                    </button>
                    <div class="dropdown-menu dropdown-menu-end">
                      <button v-if="item.status === 'draft'" class="dropdown-item" @click="changeStatus(item, 'published')">
                        <i class="ti ti-users me-2"></i>Publikasi
                      </button>
                      <button v-if="item.status === 'published'" class="dropdown-item" @click="changeStatus(item, 'active')">
                        <i class="ti ti-calendar-clock me-2"></i>Aktifkan
                      </button>
                      <button class="dropdown-item" :disabled="warmingId === item.id" @click="handleWarmCache(item)">
                        <span v-if="warmingId === item.id" class="spinner-border spinner-border-sm me-2"></span>
                        <i v-else class="ti ti-bolt me-2"></i>Warm Cache Soal
                      </button>
                      <button class="dropdown-item" @click="openTokenSetup(item)">
                        <i class="ti ti-key me-2"></i>Atur Token Ruangan
                      </button>
                      <button class="dropdown-item" :disabled="cloningId === item.id" @click="handleClone(item)">
                        <span v-if="cloningId === item.id" class="spinner-border spinner-border-sm me-2"></span>
                        <i v-else class="ti ti-copy me-2"></i>Duplikat
                      </button>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <button class="btn btn-sm btn-ghost-secondary" :aria-label="`Pulihkan jadwal ${item.name}`" @click="handleRestore(item)">
                    <i class="ti ti-rotate-clockwise"></i>
                  </button>
                  <button class="btn btn-sm btn-ghost-danger" :aria-label="`Hapus permanen jadwal ${item.name}`" @click="handleForceDelete(item)">
                    <i class="ti ti-eraser"></i>
                  </button>
                </template>
              </div>
            </td>
          </tr>
        </template>
      </BaseTable>

      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchList() }" />
    </div>

    <BaseModal v-if="showModal" :title="isEdit ? 'Edit Jadwal Ujian' : 'Buat Jadwal Ujian'" size="lg" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <div class="mb-3">
              <label class="form-label">Nama Ujian *</label>
              <input class="form-control" type="text" v-model="form.name" placeholder="Contoh: UTS Matematika XII IPA" required />
            </div>

        <div class="row g-2">
          <div class="mb-3">
            <label class="form-label">Waktu Mulai *</label>
            <input v-model="form.start_time" type="datetime-local" class="form-control" required />
          </div>
          <div class="mb-3">
            <label class="form-label">Waktu Selesai *</label>
            <input v-model="form.end_time" type="datetime-local" class="form-control" required />
          </div>
        </div>

        <div class="row g-2">
          <div class="mb-3">
            <label class="form-label">Durasi Pengerjaan (menit) *</label>
            <input v-model.number="form.duration_minutes" type="number" min="1" class="form-control" required />
          </div>
          <div class="mb-3">
            <label class="form-label">Maks. Pelanggaran</label>
            <input v-model.number="form.max_violations" type="number" min="1" class="form-control" />
          </div>
        </div>

        <!-- Options -->
        <div class="d-flex flex-wrap gap-3 mb-2">
          <label class="form-check mb-0">
            <input type="checkbox" class="form-check-input" v-model="form.allow_see_result" />
            <span class="form-check-label">Peserta bisa lihat hasil</span>
          </label>
          <label class="form-check mb-0">
            <input type="checkbox" class="form-check-input" v-model="form.randomize_questions" />
            <span class="form-check-label">Acak urutan soal</span>
          </label>
          <label class="form-check mb-0">
            <input type="checkbox" class="form-check-input" v-model="form.randomize_options" />
            <span class="form-check-label">Acak pilihan jawaban</span>
          </label>
        </div>

        <!-- Multi-stage: next section -->
        <div class="mb-3">
          <label class="form-label">Bagian Berikutnya (Multi-Tahap)</label>
          <select v-model="form.next_exam_schedule_id" class="form-select">
            <option :value="null">— Tidak ada (ujian tunggal) —</option>
            <option v-for="s in list.filter(s => s.id !== editId)" :key="s.id" :value="s.id">
              {{ s.name }}
            </option>
          </select>
          <p class="form-text text-muted">Isi jika ujian ini dilanjutkan ke sesi berikutnya otomatis setelah selesai.</p>
        </div>

        <!-- Question Banks -->
        <div class="border rounded p-3 mb-3">
          <div class="d-flex align-items-center gap-2 small fw-semibold text-muted mb-2">
            <i class="ti ti-books"></i>
            Bank Soal
          </div>
          <div v-for="(bank, i) in form.selected_banks" :key="i" class="d-flex gap-2 align-items-center mb-2">
            <select v-model="bank.question_bank_id" class="form-select">
              <option :value="0">Pilih bank soal</option>
              <option v-for="b in allBanks" :key="b.id" :value="b.id">{{ b.name }} ({{ b.question_count }} soal)</option>
            </select>
            <input v-model.number="bank.question_count" type="number" min="0" class="form-control" style="width:100px" placeholder="0=semua" />
            <button type="button" class="btn btn-sm btn-ghost-danger flex-shrink-0" @click="removeBank(i)">×</button>
          </div>
          <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addBank">
            <i class="ti ti-plus"></i> Tambah Bank Soal
          </button>
        </div>

        <!-- Rombel -->
        <div class="border rounded p-3 mb-3">
          <div class="d-flex align-items-center gap-2 small fw-semibold text-muted mb-2">
            <i class="ti ti-users"></i>
            Peserta (Rombel — kosongkan = semua)
          </div>
          <div class="d-flex flex-wrap gap-2">
            <label v-for="r in allRombels" :key="r.id" class="form-check-label d-flex align-items-center gap-1">
              <input type="checkbox" :value="r.id" v-model="form.rombel_ids" />
              {{ r.name }}
            </label>
          </div>
        </div>

        <!-- Tags -->
        <div class="border rounded p-3 mb-3" v-if="allTags.length">
          <div class="d-flex align-items-center gap-2 small fw-semibold text-muted mb-2">
            <i class="ti ti-tag"></i>
            Peserta (Tag — kosongkan = semua)
          </div>
          <div class="d-flex flex-wrap gap-2">
            <label v-for="t in allTags" :key="t.id" class="form-check-label d-flex align-items-center gap-1">
              <input type="checkbox" :value="t.id" v-model="form.tag_ids" />
              <span class="d-inline-block rounded-circle flex-shrink-0" style="width:10px;height:10px" :style="{ background: t.color || 'var(--tblr-primary)' }" />
              {{ t.name }}
            </label>
          </div>
        </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="showModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSave"><span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan</button>
      </template>
    </BaseModal>

    <!-- Modal Token Pengawas -->
    <BaseModal v-if="showTokenModal" :title="`Token Pengawas: ${selectedScheduleToken?.name}`" @close="showTokenModal = false">
      <div class="mb-4">
        <label class="form-label d-flex justify-content-between align-items-center">
          <span>Token Global (Semua Ruangan)</span>
          <button class="btn btn-sm btn-outline-secondary" @click="setGlobalRandomToken">
            <i class="ti ti-refresh me-1"></i> Acak
          </button>
        </label>
        <input type="text" class="form-control font-monospace" v-model="globalToken" placeholder="Opsional (misal: GLOBAL)" maxlength="6" />
        <small class="form-hint">Token ini bisa digunakan oleh pengawas untuk masuk ke ruangan manapun tanpa token spesifik ruangan.</small>
      </div>

      <hr />

      <h4 class="mb-3">Token Per Ruangan</h4>
      <div class="alert alert-info">
        Tetapkan token spesifik untuk masing-masing ruangan. Kosongkan jika ruangan tidak digunakan untuk ujian ini.
      </div>

      <div class="table-responsive" style="max-height: 400px; overflow-y: auto;">
        <table class="table table-vcenter">
          <thead>
            <tr>
              <th>Ruangan</th>
              <th style="width: 200px">Token Pengawas</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="room in roomTokens" :key="room.room_id">
              <td>{{ room.room_name }}</td>
              <td>
                <div class="input-group input-group-sm">
                  <input type="text" class="form-control font-monospace text-uppercase" v-model="room.token" maxlength="6" placeholder="Kosong" />
                  <button class="btn btn-icon" @click="setRoomRandomToken(room)" title="Acak Token">
                    <i class="ti ti-refresh"></i>
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="roomTokens.length === 0">
              <td colspan="2" class="text-center text-muted py-3">Belum ada data ruangan di master data</td>
            </tr>
          </tbody>
        </table>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showTokenModal = false">Tutup</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSaveTokens">
          <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan Token
        </button>
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
</template>

