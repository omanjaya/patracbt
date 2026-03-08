<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { examApi, type ExamSchedule } from '../../../api/exam.api'
import { supervisionApi } from '../../../api/supervision.api'
import client from '../../../api/client'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

// ─── Types ──────────────────────────────────────────────────────
interface Room {
  id: number
  name: string
}

interface RoomToken {
  room_id: number
  room_name: string
  token: string
}

// ─── State ─────────────────────────────────────────────────────
const router = useRouter()

const activeSchedules = ref<ExamSchedule[]>([])
const rooms = ref<Room[]>([])
const selectedScheduleIds = ref<number[]>([])
const selectedRoomId = ref<string>('')
const token = ref('')
const loading = ref(false)
const submitting = ref(false)
const errorMsg = ref('')

// Token management
const roomTokens = ref<RoomToken[]>([])
const tokenLoading = ref(false)
const tokenGenerating = ref(false)
const tokenCopied = ref<number | null>(null)

// Suppose admin/supervisor skip token (simplified permission check via role)
const isAdmin = ref(false)

const canSubmit = computed(() => {
  const hasSchedule = selectedScheduleIds.value.length > 0
  const hasRoom = !!selectedRoomId.value
  const hasToken = isAdmin.value || token.value.trim().length === 6
  return hasSchedule && hasRoom && hasToken && !submitting.value
})

// ─── API ───────────────────────────────────────────────────────
async function loadData() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [schedulesRes, roomsRes, profileRes] = await Promise.allSettled([
      examApi.listSchedules({ status: 'active', per_page: 100 }),
      client.get('/rooms', { params: { per_page: 200 } }),
      client.get('/auth/me'),
    ])

    if (schedulesRes.status === 'fulfilled') {
      activeSchedules.value = schedulesRes.value.data.data ?? []
    }
    if (roomsRes.status === 'fulfilled') {
      rooms.value = roomsRes.value.data.data ?? []
    }
    if (profileRes.status === 'fulfilled') {
      const role = profileRes.value.data?.data?.role ?? ''
      isAdmin.value = role === 'admin'
    }
  } finally {
    loading.value = false
  }
}

function toggleSchedule(id: number) {
  const idx = selectedScheduleIds.value.indexOf(id)
  if (idx >= 0) {
    selectedScheduleIds.value.splice(idx, 1)
  } else {
    selectedScheduleIds.value.push(id)
  }
}

async function submitClaim() {
  if (!canSubmit.value) return
  submitting.value = true
  errorMsg.value = ''
  try {
    await client.post('/supervision/claim', {
      exam_schedule_ids: selectedScheduleIds.value,
      room_id: selectedRoomId.value,
      token: isAdmin.value ? undefined : token.value.trim(),
    })
    // Navigate to supervision page after successful claim
    router.push({ name: 'Supervision' })
  } catch (e: any) {
    errorMsg.value = e?.response?.data?.message ?? 'Gagal masuk ruangan. Periksa kembali data yang diisi.'
  } finally {
    submitting.value = false
  }
}

// Fetch room tokens when a schedule is selected (first one)
watch(selectedScheduleIds, async (ids) => {
  if (ids.length > 0) {
    await fetchRoomTokens(ids[0]!)
  } else {
    roomTokens.value = []
  }
}, { deep: true })

async function fetchRoomTokens(scheduleId: number) {
  tokenLoading.value = true
  try {
    const res = await supervisionApi.getRoomTokens(scheduleId)
    roomTokens.value = res.data.data ?? []
  } catch {
    roomTokens.value = []
  } finally {
    tokenLoading.value = false
  }
}

async function generateTokens() {
  if (selectedScheduleIds.value.length === 0) return
  const scheduleId = selectedScheduleIds.value[0]!
  const roomIds = rooms.value.map((r: Room) => r.id)
  if (roomIds.length === 0) return
  tokenGenerating.value = true
  try {
    const res = await supervisionApi.generateRoomTokens(scheduleId, roomIds)
    roomTokens.value = res.data.data ?? []
  } catch (e: any) {
    errorMsg.value = e?.response?.data?.message ?? 'Gagal generate token.'
  } finally {
    tokenGenerating.value = false
  }
}

async function copyRoomToken(token: string, roomId: number) {
  try {
    await navigator.clipboard.writeText(token)
    tokenCopied.value = roomId
    setTimeout(() => { tokenCopied.value = null }, 2000)
  } catch {}
}

onMounted(loadData)
</script>

<template>
  <BasePageHeader
    title="Klaim Sesi Pengawasan"
    subtitle="Pilih jadwal ujian dan ruangan untuk memulai pengawasan"
    :breadcrumbs="[{ label: 'Pengawasan', to: '/admin/supervision' }, { label: 'Klaim Sesi' }]"
  />

  <div class="container-tight py-4">
    <div class="card card-md">
      <!-- Header -->
      <div class="card-body text-center py-4 p-sm-5">
        <div class="mb-3">
          <span class="avatar avatar-lg bg-primary-lt">
            <i class="ti ti-door-enter fs-2 text-primary"></i>
          </span>
        </div>
        <h1 class="mt-2">Masuk Ruang Ujian</h1>
        <p class="text-muted mb-0">
          Silakan pilih jadwal ujian dan ruangan untuk memulai pengawasan.
          <span class="text-primary">Anda dapat memilih beberapa jadwal sekaligus.</span>
        </p>
      </div>

      <hr class="m-0" />

      <div class="card-body">
        <!-- Error Alert -->
        <div v-if="errorMsg" class="alert alert-danger" role="alert">
          <div class="d-flex">
            <div>
              <i class="ti ti-alert-circle icon alert-icon me-2"></i>
            </div>
            <div>{{ errorMsg }}</div>
          </div>
        </div>

        <!-- Loading skeleton -->
        <div v-if="loading" class="text-center py-4">
          <div class="spinner-border text-primary" role="status"></div>
          <p class="mt-2 text-muted">Memuat data...</p>
        </div>

        <template v-else>
          <!-- 1. Pilih Jadwal Ujian (multi) -->
          <div class="mb-3">
            <label class="form-label fw-medium">
              Jadwal Ujian Aktif
              <small class="text-muted fw-normal">(Dapat memilih lebih dari satu)</small>
            </label>

            <div v-if="activeSchedules.length === 0">
              <div class="empty py-3">
                <div class="empty-icon">
                  <i class="ti ti-calendar-off" style="font-size: 3rem;"></i>
                </div>
                <p class="empty-title">Tidak ada sesi yang perlu diklaim</p>
                <p class="empty-subtitle text-muted">Belum ada ujian yang sedang berlangsung saat ini.</p>
              </div>
            </div>

            <div v-else class="d-flex flex-wrap gap-2">
              <button
                v-for="s in activeSchedules"
                :key="s.id"
                type="button"
                class="btn"
                :class="selectedScheduleIds.includes(s.id) ? 'btn-primary' : 'btn-outline-secondary'"
                @click="toggleSchedule(s.id)"
              >
                <i class="ti ti-device-desktop-analytics me-1"></i>
                {{ s.name }}
                <span
                  v-if="selectedScheduleIds.includes(s.id)"
                  class="badge bg-white text-primary ms-1"
                >
                  <i class="ti ti-check"></i>
                </span>
              </button>
            </div>

            <div v-if="selectedScheduleIds.length > 0" class="mt-2">
              <span class="badge bg-primary-lt text-primary">
                {{ selectedScheduleIds.length }} jadwal dipilih
              </span>
            </div>
          </div>

          <!-- 2. Pilih Ruangan -->
          <div class="mb-3">
            <label class="form-label fw-medium">Pilih Ruangan</label>
            <select
              v-model="selectedRoomId"
              class="form-select"
              :disabled="activeSchedules.length === 0"
            >
              <option value="">-- Pilih Ruangan --</option>
              <option v-if="isAdmin" value="GLOBAL_ALL">
                MONITOR SEMUA RUANGAN (GLOBAL)
              </option>
              <option v-for="r in rooms" :key="r.id" :value="r.id">
                {{ r.name }}
              </option>
            </select>
          </div>

          <!-- 3. Token (hidden for admin/supervisor) -->
          <div v-if="!isAdmin" class="mb-3">
            <label class="form-label fw-medium">Token Ruangan</label>
            <input
              v-model="token"
              type="text"
              class="form-control font-monospace"
              placeholder="6 Digit Token"
              maxlength="6"
              autocomplete="off"
            />
            <div class="form-hint">
              <i class="ti ti-info-circle me-1"></i>
              Masukkan token yang tertera di papan tulis/meja ruangan.
            </div>
          </div>

          <div v-else class="mb-3">
            <div class="alert alert-info" role="alert">
              <div class="d-flex">
                <div>
                  <i class="ti ti-info-circle icon alert-icon me-2"></i>
                </div>
                <div>
                  <strong>Akses Supervisor:</strong> Anda dapat masuk tanpa token.
                </div>
              </div>
            </div>
          </div>

          <!-- Room Tokens Table (admin only) -->
          <div v-if="isAdmin && selectedScheduleIds.length > 0" class="mb-3">
            <div class="d-flex align-items-center justify-content-between mb-2">
              <label class="form-label fw-medium mb-0">Token per Ruangan</label>
              <button
                type="button"
                class="btn btn-sm btn-outline-primary"
                :disabled="tokenGenerating"
                @click="generateTokens"
              >
                <span v-if="tokenGenerating">
                  <span class="spinner-border spinner-border-sm me-1" role="status"></span>
                  Generating...
                </span>
                <span v-else>
                  <i class="ti ti-key me-1"></i>
                  Generate Token
                </span>
              </button>
            </div>

            <div v-if="tokenLoading" class="text-center py-3">
              <span class="spinner-border spinner-border-sm me-2"></span>
              <span class="text-muted">Memuat token...</span>
            </div>

            <div v-else-if="roomTokens.length > 0" class="table-responsive">
              <table class="table table-vcenter card-table table-sm">
                <thead>
                  <tr>
                    <th>Ruangan</th>
                    <th>Token</th>
                    <th class="w-1"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="rt in roomTokens" :key="rt.room_id">
                    <td>{{ rt.room_name }}</td>
                    <td>
                      <code class="font-monospace fs-5">{{ rt.token }}</code>
                    </td>
                    <td>
                      <button
                        type="button"
                        class="btn btn-sm btn-ghost-secondary"
                        @click="copyRoomToken(rt.token, rt.room_id)"
                        :title="tokenCopied === rt.room_id ? 'Tersalin!' : 'Salin token'"
                      >
                        <i :class="tokenCopied === rt.room_id ? 'ti ti-check text-success' : 'ti ti-copy'"></i>
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div v-else class="text-muted small text-center py-2">
              <i class="ti ti-key-off me-1"></i>
              Belum ada token. Klik "Generate Token" untuk membuat.
            </div>
          </div>

          <!-- Submit -->
          <div class="form-footer">
            <button
              type="button"
              class="btn btn-primary w-100"
              :disabled="!canSubmit"
              @click="submitClaim"
            >
              <span v-if="submitting">
                <span class="spinner-border spinner-border-sm me-2" role="status"></span>
                Memproses...
              </span>
              <span v-else>
                <i class="ti ti-door-enter me-1"></i>
                Masuk Ruangan
              </span>
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
