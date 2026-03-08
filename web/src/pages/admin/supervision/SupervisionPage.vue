<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { getAvatarUrl, getIllustration } from '../../../utils/avatar'
import { examApi, type ExamSchedule, type ExamSession } from '../../../api/exam.api'
import { supervisionApi } from '../../../api/supervision.api'
import { useWebSocket } from '../../../composables/useWebSocket'
import client from '../../../api/client'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()

// ─── Types ──────────────────────────────────────────────────────
interface LiveStudent {
  user_id: number
  session_id: number
  name: string
  answered: number
  total: number
  violation_count: number
  last_violation_type: string
  status: 'online' | 'offline' | 'finished' | 'terminated'
  last_seen: string
}

const violationLabels: Record<string, string> = {
  tab_switch: 'Pindah Tab',
  blur_extended: 'Tab Tidak Aktif Lama',
  multi_tab: 'Multi-Tab Terdeteksi',
  popup_detected: 'Buka Popup/Window',
  background_detected: 'Split-Screen / Background',
  external_paste: 'Paste Teks Eksternal',
  alt_tab: 'Alt+Tab / Pindah Window',
  fullscreen_exit: 'Keluar Fullscreen / Floating App',
  window_resize: 'Split-Screen / Floating App',
  focus_lost: 'Window Kehilangan Fokus',
}

// ─── State ─────────────────────────────────────────────────────
const schedules = ref<ExamSchedule[]>([])
const selectedScheduleId = ref<number | null>(null)
const sessions = ref<ExamSession[]>([])
const students = reactive<Record<number, LiveStudent>>({})
const loading = ref(false)
const filterStatus = ref<string>('all')
const viewMode = ref<'grid' | 'list'>('grid')
const searchQuery = ref('')

let ws: ReturnType<typeof useWebSocket> | null = null
const wsConnected = ref(false)
let offlineTimer: ReturnType<typeof setInterval> | null = null
let stopWsWatch: (() => void) | null = null

const studentList = computed(() => {
  let list = Object.values(students)
  if (filterStatus.value !== 'all') {
    list = list.filter((s) => s.status === filterStatus.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter((s) => s.name.toLowerCase().includes(q) || String(s.user_id).includes(q))
  }
  return list
})

const onlineCount = computed(() => Object.values(students).filter((s) => s.status === 'online').length)
const finishedCount = computed(() => Object.values(students).filter((s) => s.status === 'finished').length)
const offlineCount = computed(() => Object.values(students).filter((s) => s.status === 'offline').length)
const violationCount = computed(() => Object.values(students).filter((s) => s.violation_count > 0).length)

// ─── API ───────────────────────────────────────────────────────
async function loadSchedules() {
  try {
    const res = await examApi.listSchedules({ status: 'active', per_page: 50 })
    schedules.value = res.data.data ?? []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat daftar jadwal')
  }
}

async function selectSchedule(scheduleId: number) {
  selectedScheduleId.value = scheduleId
  // Clear reactive object
  for (const key of Object.keys(students)) delete students[Number(key)]
  disconnectWS()
  await loadSessions(scheduleId)
  connectWS(scheduleId)
}

async function loadSessions(scheduleId: number) {
  loading.value = true
  try {
    // Load ongoing + not-started sessions combined
    const [ongoingRes, notStartedRes] = await Promise.allSettled([
      examApi.listOngoingSessions(scheduleId),
      examApi.listNotStartedSessions(scheduleId),
    ])
    const ongoing: ExamSession[] = ongoingRes.status === 'fulfilled' ? (ongoingRes.value.data.data ?? []) : []
    const notStarted: ExamSession[] = notStartedRes.status === 'fulfilled' ? (notStartedRes.value.data.data ?? []) : []
    sessions.value = [...ongoing, ...notStarted]
    // Seed student map from sessions
    for (const s of sessions.value) {
      const existing = students[s.user_id]
      students[s.user_id] = {
        user_id: s.user_id,
        session_id: s.id,
        name: (s as any).user?.name ?? `User ${s.user_id}`,
        answered: 0,
        total: 0,
        violation_count: s.violation_count ?? 0,
        last_violation_type: existing?.last_violation_type ?? '',
        status: s.status === 'finished' ? 'finished' : s.status === 'terminated' ? 'terminated' : 'offline',
        last_seen: existing?.last_seen ?? '',
      }
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat data sesi')
  } finally {
    loading.value = false
  }
}

async function lockStudent(student: LiveStudent) {
  if (!selectedScheduleId.value) return
  showConfirm(
    `Kunci Akses`,
    `Kunci akses ujian untuk ${student.name}? Peserta tidak bisa melanjutkan ujian.`,
    false,
    async () => {
      await client.post(`/api/v1/monitoring/${selectedScheduleId.value}/lock`, {
        target_user_id: student.user_id,
        message: 'Akses dikunci oleh pengawas',
      })
    }
  )
}

async function forceFinish(student: LiveStudent) {
  if (!selectedScheduleId.value) return
  showConfirm(
    `Paksa Selesai`,
    `Paksa selesaikan ujian ${student.name}? Tindakan ini tidak dapat dibatalkan dan sesi akan langsung dinilai.`,
    true,
    async () => {
      await supervisionApi.forceFinish(selectedScheduleId.value!, student.session_id)
      student.status = 'finished'
    }
  )
}

async function extendTime(student: LiveStudent) {
  openTimeModal(student)
}

async function submitExtendTime() {
  if (!selectedScheduleId.value || !timeTarget.value) return
  if (timeMinutes.value < 1 || timeMinutes.value > 120) return
  timeSending.value = true
  try {
    await supervisionApi.extendTime(selectedScheduleId.value, timeTarget.value.session_id, timeMinutes.value)
    toast.success('Waktu berhasil ditambahkan')
    timeModal.value = false
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menambah waktu')
  } finally {
    timeSending.value = false
  }
}

async function sendMessage(student: LiveStudent) {
  openMessageModal(student)
}

async function submitMessage() {
  if (!selectedScheduleId.value || !messageTarget.value || !messageText.value.trim()) return
  messageSending.value = true
  try {
    await supervisionApi.sendMessage(selectedScheduleId.value, messageTarget.value.session_id, messageText.value.trim())
    toast.success('Pesan berhasil dikirim')
    messageModal.value = false
    messageText.value = ''
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengirim pesan')
  } finally {
    messageSending.value = false
  }
}

async function returnToExam(student: LiveStudent) {
  if (!selectedScheduleId.value) return
  showConfirm(
    'Kembalikan ke Ujian',
    `Siswa ${student.name} akan dikembalikan ke ujian. Nilai akan direset tetapi jawaban tetap tersimpan.`,
    false,
    async () => {
      await supervisionApi.returnToExam(selectedScheduleId.value!, student.session_id)
      student.status = 'online'
    }
  )
}

async function forceLogout(student: LiveStudent) {
  if (!selectedScheduleId.value) return
  showConfirm(
    'Paksa Logout',
    `Siswa ${student.name} akan dikeluarkan paksa dari sistem. Pelanggaran akan ditambahkan.`,
    true,
    async () => {
      await supervisionApi.forceLogout(selectedScheduleId.value!, student.session_id)
      student.status = 'terminated'
    }
  )
}

async function unlockStudent(student: LiveStudent) {
  if (!selectedScheduleId.value) return
  try {
    await supervisionApi.unlock(selectedScheduleId.value, student.session_id)
    toast.success('Akses berhasil dibuka')
    student.status = 'online'
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal membuka kunci')
  }
}

const selectedStudents = ref<number[]>([])
const exportLoading = ref(false)

// Fix #5: Modal konfirmasi khusus untuk aksi destruktif
const confirmModal = ref(false)
const confirmTitle = ref('')
const confirmMsg = ref('')
const confirmDanger = ref(true)
const confirmAction = ref<(() => Promise<void>) | null>(null)
const confirmLoading = ref(false)

function showConfirm(title: string, msg: string, danger: boolean, action: () => Promise<void>) {
  confirmTitle.value = title
  confirmMsg.value = msg
  confirmDanger.value = danger
  confirmAction.value = action
  confirmModal.value = true
}

async function executeConfirm() {
  if (!confirmAction.value) return
  confirmLoading.value = true
  try {
    await confirmAction.value()
  } finally {
    confirmLoading.value = false
    confirmModal.value = false
    confirmAction.value = null
  }
}

// ─── Modals ────────────────────────────────────────────────────
const messageModal = ref(false)
const messageTarget = ref<LiveStudent | null>(null)
const messageText = ref('')
const messageSending = ref(false)

const timeModal = ref(false)
const timeTarget = ref<LiveStudent | null>(null)
const timeMinutes = ref(10)
const timeSending = ref(false)

const tokenCopied = ref(false)

function openMessageModal(student: LiveStudent) {
  messageTarget.value = student
  messageText.value = ''
  messageModal.value = true
}

function openTimeModal(student: LiveStudent) {
  timeTarget.value = student
  timeMinutes.value = 10
  timeModal.value = true
}

async function copyToken(token: string) {
  try {
    await navigator.clipboard.writeText(token)
    tokenCopied.value = true
    setTimeout(() => { tokenCopied.value = false }, 2000)
  } catch {
    toast.info('Token: ' + token)
  }
}

function toggleSelectStudent(sessionId: number) {
  const idx = selectedStudents.value.indexOf(sessionId)
  if (idx >= 0) selectedStudents.value.splice(idx, 1)
  else selectedStudents.value.push(sessionId)
}

async function bulkForceFinish() {
  if (!selectedScheduleId.value || selectedStudents.value.length === 0) return
  showConfirm(
    'Paksa Selesai Massal',
    `Paksa selesaikan ${selectedStudents.value.length} peserta?`,
    true,
    async () => {
      try {
        await supervisionApi.bulkAction(selectedScheduleId.value!, 'force_finish', selectedStudents.value)
        toast.success(`${selectedStudents.value.length} peserta berhasil diselesaikan`)
        selectedStudents.value = []
        await loadSessions(selectedScheduleId.value!)
      } catch (e: any) {
        toast.error(e?.response?.data?.message ?? 'Gagal menyelesaikan peserta')
      }
    }
  )
}

async function bulkExtendTime() {
  if (!selectedScheduleId.value || selectedStudents.value.length === 0) return
  const input = prompt(`Tambah waktu untuk ${selectedStudents.value.length} peserta (menit):`, '10')
  const minutes = parseInt(input ?? '')
  if (!minutes || minutes < 1) return
  try {
    await supervisionApi.bulkAction(selectedScheduleId.value, 'extend_time', selectedStudents.value, minutes)
    toast.success(`Waktu berhasil ditambahkan untuk ${selectedStudents.value.length} peserta`)
    selectedStudents.value = []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menambah waktu')
  }
}

async function finishAllOngoing() {
  if (!selectedScheduleId.value) return
  showConfirm(
    'Selesaikan Semua Sesi',
    `Selesaikan SEMUA sesi yang sedang berlangsung? Semua peserta akan langsung dinilai dan tidak bisa melanjutkan ujian.`,
    true,
    async () => {
      await examApi.finishAllOngoing(selectedScheduleId.value!)
      await loadSessions(selectedScheduleId.value!)
    }
  )
}

// ─── WebSocket ─────────────────────────────────────────────────
function connectWS(scheduleId: number) {
  ws = useWebSocket(scheduleId)
  // Track connection state via watch
  ws.on('time_sync', () => { wsConnected.value = true })

  // Track connection state
  const { connected } = ws
  stopWsWatch = watch(connected, (val) => { wsConnected.value = val })

  ws.on('student_joined', (data: { user_id: number; session_id: number }) => {
    const s = students[data.user_id]
    if (s) {
      s.status = 'online'
      s.last_seen = new Date().toISOString()
    } else {
      students[data.user_id] = {
        user_id: data.user_id,
        session_id: data.session_id,
        name: `User ${data.user_id}`,
        answered: 0,
        total: 0,
        violation_count: 0,
        last_violation_type: '',
        status: 'online',
        last_seen: new Date().toISOString(),
      }
    }
  })

  ws.on('student_left', (data: { user_id: number }) => {
    const s = students[data.user_id]
    if (s && s.status === 'online') {
      s.status = 'offline'
    }
  })

  ws.on('answer_saved', (data: { user_id: number; answered: number; total: number }) => {
    const s = students[data.user_id]
    if (s) {
      s.answered = data.answered
      s.total = data.total
      s.last_seen = new Date().toISOString()
    }
  })

  ws.on('answer_batch', (batch: { user_id: number; answered: number; total: number }[]) => {
    for (const data of batch) {
      const s = students[data.user_id]
      if (s) {
        s.answered = data.answered
        s.total = data.total
        s.last_seen = new Date().toISOString()
      }
    }
  })

  ws.on('violation_logged', (data: { user_id: number; violation_type: string; violation_count: number }) => {
    const s = students[data.user_id]
    if (s) {
      s.violation_count = data.violation_count
      s.last_violation_type = data.violation_type
      const label = violationLabels[data.violation_type] ?? data.violation_type
      toast.warning(`${s.name}: ${label} (${data.violation_count}x)`)
    }
  })

  ws.on('session_finished', (data: { user_id: number }) => {
    const s = students[data.user_id]
    if (s) {
      s.status = 'finished'
    }
  })

  ws.connect()

  // Offline detection: mark as offline if no activity for 90s
  offlineTimer = setInterval(() => {
    const now = Date.now()
    for (const s of Object.values(students)) {
      if (s.status === 'online' && s.last_seen) {
        const diff = (now - new Date(s.last_seen).getTime()) / 1000
        if (diff > 90) {
          s.status = 'offline'
        }
      }
    }
  }, 15000)
}

function disconnectWS() {
  stopWsWatch?.()
  stopWsWatch = null
  ws?.disconnect()
  ws = null
  wsConnected.value = false
  if (offlineTimer) {
    clearInterval(offlineTimer)
    offlineTimer = null
  }
}

function progressPct(s: LiveStudent) {
  if (!s.total) return 0
  return Math.round((s.answered / s.total) * 100)
}

function progressRingColor(pct: number) {
  if (pct > 80) return 'var(--tblr-success)'
  if (pct >= 50) return 'var(--tblr-warning)'
  return 'var(--tblr-primary)'
}

function statusBgClass(status: string) {
  return { online: 'bg-success', offline: 'bg-secondary', finished: 'bg-primary', terminated: 'bg-danger' }[status] ?? 'bg-secondary'
}

function statusTextClass(status: string) {
  return { online: 'text-success', offline: 'text-secondary', finished: 'text-primary', terminated: 'text-danger' }[status] ?? 'text-secondary'
}

function statusLabel(status: string) {
  return { online: 'Online', offline: 'Offline', finished: 'Selesai', terminated: 'Dihentikan' }[status] ?? status
}

function exportUnfinished() {
  if (!selectedScheduleId.value) return
  exportLoading.value = true
  try {
    const url = `/api/v1/reports/${selectedScheduleId.value}/unfinished/export`
    const token = localStorage.getItem('access_token')
    const a = document.createElement('a')
    a.href = url + `?token=${token}`
    a.download = ''
    a.click()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengekspor data')
  } finally {
    setTimeout(() => { exportLoading.value = false }, 2000)
  }
}

onMounted(loadSchedules)
onUnmounted(disconnectWS)
</script>

<template>
    <!-- Header -->
    <BasePageHeader
      title="Ruang Pengawasan"
      subtitle="Monitor peserta ujian secara real-time"
      :breadcrumbs="[{ label: 'Ruang Pengawasan' }]"
    >
      <template #actions>
        <template v-if="selectedScheduleId">
          <button class="btn btn-ghost-secondary" @click="exportUnfinished" :disabled="exportLoading">
            <span v-if="exportLoading" class="spinner-border spinner-border-sm me-1"></span>
            <i v-else class="ti ti-user-exclamation me-1"></i>Export Belum Selesai
          </button>
          <button class="btn btn-ghost-secondary" @click="loadSessions(selectedScheduleId!)">
            <i class="ti ti-refresh"></i>
            Refresh
          </button>
        </template>
      </template>
    </BasePageHeader>

    <!-- Schedule Selector -->
    <div class="mb-3">
      <label class="form-label fw-medium">Pilih Jadwal Ujian Aktif</label>
      <div class="d-flex flex-wrap gap-2">
        <button
          v-for="s in schedules"
          :key="s.id"
          class="btn btn-outline-secondary"
          :class="{ active: s.id === selectedScheduleId }"
          @click="selectSchedule(s.id)"
        >
          <i class="ti ti-device-desktop-analytics"></i>
          <span>{{ s.name }}</span>
          <span class="badge bg-secondary-lt text-secondary ms-1">{{ s.status }}</span>
        </button>
        <div v-if="schedules.length === 0" class="text-muted small p-2">
          Tidak ada ujian aktif saat ini
        </div>
      </div>
    </div>

    <!-- Token Card -->
    <div v-if="selectedScheduleId" class="card mb-3">
      <div class="card-body">
        <p class="fw-semibold mb-0">Token Ujian</p>
        <p class="text-muted small mb-0">Berikan token ini kepada siswa untuk memulai ujian</p>
      </div>
      <div class="card-body border-start d-flex align-items-center gap-3">
        <span class="h2 font-monospace mb-0 text-primary" style="letter-spacing:0.1em">{{ schedules.find(s => s.id === selectedScheduleId)?.token ?? '-' }}</span>
        <button
          class="btn btn-sm btn-outline-primary"

          @click="copyToken(schedules.find(s => s.id === selectedScheduleId)?.token ?? '')"
          :title="tokenCopied ? 'Tersalin!' : 'Salin token'"
        >
          <i class="ti ti-check"></i>
          <i class="ti ti-copy"></i>
          {{ tokenCopied ? 'Tersalin' : 'Salin' }}
        </button>
      </div>
    </div>

    <!-- Stats Bar -->
    <div v-if="selectedScheduleId" class="d-flex flex-wrap align-items-center gap-2 mb-3">
      <div class="badge bg-green-lt text-green d-flex align-items-center gap-1">
        <i class="ti ti-wifi"></i>
        <span>{{ onlineCount }} Online</span>
      </div>
      <div class="badge bg-blue-lt text-blue d-flex align-items-center gap-1">
        <i class="ti ti-check"></i>
        <span>{{ finishedCount }} Selesai</span>
      </div>
      <div class="badge bg-secondary-lt text-secondary d-flex align-items-center gap-1">
        <i class="ti ti-wifi"></i>
        <span>{{ offlineCount }} Offline</span>
      </div>
      <div class="badge bg-red-lt text-red d-flex align-items-center gap-1">
        <i class="ti ti-alert-triangle"></i>
        <span>{{ violationCount }} Pelanggaran</span>
      </div>

      <!-- WS Status -->
      <div :class="wsConnected ? 'badge bg-green text-white ms-auto' : 'badge bg-secondary-lt text-secondary ms-auto'">
        <i class="ti ti-activity"></i>
        {{ wsConnected ? 'Live' : 'Menghubungkan...' }}
      </div>

      <!-- Filter -->
      <div class="btn-group btn-group-sm">
        <button v-for="f in ['all','online','offline','finished']" :key="f"
          :class="['btn', filterStatus === f ? 'btn-primary' : 'btn-outline-secondary']"
          @click="filterStatus = f">
          {{ f === 'all' ? 'Semua' : statusLabel(f) }}
        </button>
      </div>

      <!-- Bulk actions -->
      <div v-if="selectedStudents.length > 0" class="d-flex align-items-center gap-2 ms-2">
        <span class="badge bg-secondary-lt text-secondary">{{ selectedStudents.length }} dipilih</span>
        <button class="btn btn-sm btn-outline-secondary" @click="bulkForceFinish">Paksa Selesai</button>
        <button class="btn btn-sm btn-outline-secondary" @click="bulkExtendTime">+ Waktu</button>
      </div>

      <!-- Finish all -->
      <button v-if="selectedScheduleId" class="btn btn-sm btn-outline-danger ms-auto" @click="finishAllOngoing">
        <i class="ti ti-check"></i> Selesaikan Semua
      </button>
    </div>

    <!-- Search + View Toggle Bar -->
    <div v-if="selectedScheduleId" class="d-flex align-items-center gap-2 mb-3">
      <div class="input-icon flex-grow-1" style="max-width: 320px;">
        <span class="input-icon-addon">
          <i class="ti ti-search"></i>
        </span>
        <input
          v-model="searchQuery"
          type="text"
          class="form-control"
          placeholder="Cari nama atau NIS peserta..."
        />
      </div>
      <div class="btn-group btn-group-sm ms-auto">
        <button
          class="btn"
          :class="viewMode === 'grid' ? 'btn-primary' : 'btn-outline-secondary'"
          @click="viewMode = 'grid'"
          title="Tampilan Grid"
        >
          <i class="ti ti-layout-grid"></i>
        </button>
        <button
          class="btn"
          :class="viewMode === 'list' ? 'btn-primary' : 'btn-outline-secondary'"
          @click="viewMode = 'list'"
          title="Tampilan List"
        >
          <i class="ti ti-list"></i>
        </button>
      </div>
    </div>

    <!-- Content Area -->
    <div v-if="selectedScheduleId">
      <div v-if="loading" class="text-center text-muted py-4">Memuat data peserta...</div>
      <div v-else-if="studentList.length === 0">
        <div class="text-center py-5">
          <img :src="getIllustration('searching-for-a-signal')" class="img-fluid mb-3 opacity-75" style="max-height:160px" alt="">
          <p class="text-muted">Belum ada peserta yang terhubung</p>
        </div>
      </div>

      <!-- ════════════ GRID VIEW ════════════ -->
      <div v-else-if="viewMode === 'grid'" class="row g-3">
        <div
          v-for="s in studentList" :key="s.user_id" class="col-md-6 col-lg-4 col-xl-3">
        <div class="card h-100" :class="{ 'border-danger': s.violation_count > 0, 'border-primary': selectedStudents.includes(s.session_id) }"
        >
          <!-- Select checkbox -->
          <input type="checkbox" class="form-check-input position-absolute top-0 end-0 m-2"
            :checked="selectedStudents.includes(s.session_id)"
            @change="toggleSelectStudent(s.session_id)" />

          <!-- Status dot -->
          <div class="avatar avatar-xs rounded-circle position-absolute top-0 start-0 mt-2 ms-2" :class="statusBgClass(s.status)"></div>

          <!-- Avatar -->
          <div class="avatar avatar-md rounded-circle mx-auto mb-2"
            :style="`background-image:url(${getAvatarUrl(s.user_id)})`">
          </div>

          <!-- Info -->
          <div class="card-body pt-0 text-center">
            <p class="fw-medium mb-0 small">{{ s.name }}</p>
            <p class="d-flex align-items-center justify-content-center gap-1 text-muted small mt-1">
              <span :class="statusTextClass(s.status)">{{ statusLabel(s.status) }}</span>
              <span v-if="s.violation_count > 0" class="badge bg-danger-lt text-danger d-flex align-items-center gap-1"
                :title="s.last_violation_type ? (violationLabels[s.last_violation_type] ?? s.last_violation_type) : ''">
                <i class="ti ti-alert-triangle small"></i>
                {{ s.violation_count }}x
              </span>
            </p>
          </div>

          <!-- Circular Progress Ring -->
          <div class="d-flex justify-content-center pb-2">
            <div class="progress-ring" :style="{
              '--progress-value': progressPct(s),
              '--progress-color': progressRingColor(progressPct(s)),
            }">
              <span class="progress-ring__label">{{ progressPct(s) }}%</span>
            </div>
          </div>
          <div class="text-center pb-2">
            <span class="text-muted small">{{ s.answered }}/{{ s.total }}</span>
          </div>

          <!-- Action buttons -->
          <div class="card-footer d-flex justify-content-center gap-1 p-2">
            <button
              v-if="s.status === 'online'"
              class="btn btn-sm btn-ghost-secondary"
              title="Kunci peserta"
              @click="lockStudent(s)"
            >
              <i class="ti ti-lock"></i>
            </button>
            <button
              v-if="s.status === 'online'"
              class="btn btn-sm btn-ghost-secondary"
              title="Tambah waktu"
              @click="extendTime(s)"
            >
              <i class="ti ti-clock"></i>
            </button>
            <button
              v-if="s.status === 'online' || s.status === 'offline'"
              class="btn btn-sm btn-ghost-secondary"
              title="Kirim pesan"
              @click="sendMessage(s)"
            >
              <i class="ti ti-message"></i>
            </button>
            <button
              v-if="s.status === 'online'"
              class="btn btn-sm btn-ghost-success"
              title="Paksa selesai"
              @click="forceFinish(s)"
            >
              <i class="ti ti-check"></i>
            </button>
            <button
              v-if="s.status === 'offline'"
              class="btn btn-sm btn-ghost-secondary"
              title="Buka kunci"
              @click="unlockStudent(s)"
            >
              <i class="ti ti-lock-open"></i>
            </button>
            <button
              v-if="s.status === 'finished' || s.status === 'terminated'"
              class="btn btn-sm btn-ghost-primary"
              title="Kembalikan ke Ujian"
              @click="returnToExam(s)"
            >
              <i class="ti ti-arrow-back-up"></i>
            </button>
            <button
              class="btn btn-sm btn-ghost-danger"
              title="Paksa Logout"
              @click="forceLogout(s)"
            >
              <i class="ti ti-logout"></i>
            </button>
          </div>
        </div>
        </div>
      </div>

      <!-- ════════════ LIST VIEW ════════════ -->
      <div v-else class="card">
        <div class="table-responsive">
          <table class="table table-vcenter card-table">
            <thead>
              <tr>
                <th style="width:1%"></th>
                <th>Nama</th>
                <th>NIS</th>
                <th style="width:120px">Progress</th>
                <th>Status</th>
                <th>Pelanggaran</th>
                <th style="width:200px">Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in studentList" :key="s.user_id" :class="{ 'table-danger': s.violation_count > 0 }">
                <td>
                  <input type="checkbox" class="form-check-input"
                    :checked="selectedStudents.includes(s.session_id)"
                    @change="toggleSelectStudent(s.session_id)" />
                </td>
                <td>
                  <div class="d-flex align-items-center gap-2">
                    <span class="avatar avatar-sm rounded-circle" :style="`background-image:url(${getAvatarUrl(s.user_id)})`"></span>
                    <span class="fw-medium">{{ s.name }}</span>
                  </div>
                </td>
                <td class="text-muted">{{ s.user_id }}</td>
                <td>
                  <div class="d-flex align-items-center gap-2">
                    <div class="progress-ring progress-ring--sm" :style="{
                      '--progress-value': progressPct(s),
                      '--progress-color': progressRingColor(progressPct(s)),
                    }">
                      <span class="progress-ring__label" style="font-size:0.6rem">{{ progressPct(s) }}%</span>
                    </div>
                    <span class="text-muted small">{{ s.answered }}/{{ s.total }}</span>
                  </div>
                </td>
                <td>
                  <div class="d-flex align-items-center gap-1">
                    <span class="badge" :class="statusBgClass(s.status)">
                      <span class="d-none d-md-inline">{{ statusLabel(s.status) }}</span>
                    </span>
                  </div>
                </td>
                <td>
                  <span v-if="s.violation_count > 0" class="badge bg-danger-lt text-danger d-flex align-items-center gap-1" style="width:fit-content"
                    :title="s.last_violation_type ? (violationLabels[s.last_violation_type] ?? s.last_violation_type) : ''">
                    <i class="ti ti-alert-triangle"></i>
                    {{ s.violation_count }}x
                  </span>
                  <span v-else class="text-muted">-</span>
                </td>
                <td>
                  <div class="d-flex gap-1">
                    <button
                      v-if="s.status === 'online'"
                      class="btn btn-sm btn-ghost-secondary"
                      title="Kunci peserta"
                      @click="lockStudent(s)"
                    >
                      <i class="ti ti-lock"></i>
                    </button>
                    <button
                      v-if="s.status === 'online'"
                      class="btn btn-sm btn-ghost-secondary"
                      title="Tambah waktu"
                      @click="extendTime(s)"
                    >
                      <i class="ti ti-clock"></i>
                    </button>
                    <button
                      v-if="s.status === 'online' || s.status === 'offline'"
                      class="btn btn-sm btn-ghost-secondary"
                      title="Kirim pesan"
                      @click="sendMessage(s)"
                    >
                      <i class="ti ti-message"></i>
                    </button>
                    <button
                      v-if="s.status === 'online'"
                      class="btn btn-sm btn-ghost-success"
                      title="Paksa selesai"
                      @click="forceFinish(s)"
                    >
                      <i class="ti ti-check"></i>
                    </button>
                    <button
                      v-if="s.status === 'offline'"
                      class="btn btn-sm btn-ghost-secondary"
                      title="Buka kunci"
                      @click="unlockStudent(s)"
                    >
                      <i class="ti ti-lock-open"></i>
                    </button>
                    <button
                      v-if="s.status === 'finished' || s.status === 'terminated'"
                      class="btn btn-sm btn-ghost-primary"
                      title="Kembalikan ke Ujian"
                      @click="returnToExam(s)"
                    >
                      <i class="ti ti-arrow-back-up"></i>
                    </button>
                    <button
                      class="btn btn-sm btn-ghost-danger"
                      title="Paksa Logout"
                      @click="forceLogout(s)"
                    >
                      <i class="ti ti-logout"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else>
      <div class="text-center py-5">
        <img :src="getIllustration('searching-for-a-signal')" class="img-fluid mb-3 opacity-75" style="max-height:160px" alt="">
        <p class="text-muted">Pilih jadwal ujian untuk memulai pengawasan</p>
      </div>
    </div>

    <!-- Message Modal -->
    <div v-if="messageModal" class="modal modal-blur show d-block" @click.self="messageModal = false">
      <div class="modal-dialog modal-dialog-centered"><div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Kirim Pesan ke {{ messageTarget?.name }}</h5>
          <button type="button" class="btn-close" @click="messageModal = false"></button>
        </div>
        <div class="modal-body">
          <textarea
            v-model="messageText"
            class="form-control"
            rows="3"
            placeholder="Contoh: Waktu tinggal 5 menit lagi!"
            @keydown.ctrl.enter="submitMessage"
          />
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost-secondary" @click="messageModal = false">Batal</button>
          <button class="btn btn-primary" :disabled="!messageText.trim() || messageSending" @click="submitMessage">
            {{ messageSending ? 'Mengirim...' : 'Kirim Pesan' }}
          </button>
        </div>
      </div></div>
    </div>

    <!-- Extend Time Modal -->
    <div v-if="timeModal" class="modal modal-blur show d-block" @click.self="timeModal = false">
      <div class="modal-dialog modal-dialog-centered modal-sm"><div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Tambah Waktu — {{ timeTarget?.name }}</h5>
          <button type="button" class="btn-close" @click="timeModal = false"></button>
        </div>
        <div class="modal-body">
          <div class="mb-3">
            <label class="form-label">Durasi Tambahan (Menit)</label>
            <input type="number" v-model.number="timeMinutes" min="1" max="120" class="form-control" />
            <p class="form-text text-muted">Maksimal 120 menit.</p>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost-secondary" @click="timeModal = false">Batal</button>
          <button class="btn btn-primary" :disabled="timeSending || timeMinutes < 1" @click="submitExtendTime">
            {{ timeSending ? 'Menambahkan...' : 'Tambah' }}
          </button>
        </div>
      </div></div>
    </div>
    <!-- Fix #5: Confirm modal untuk aksi destruktif -->
    <div v-if="confirmModal" class="modal modal-blur show d-block" @click.self="confirmModal = false">
      <div class="modal-dialog modal-dialog-centered modal-sm"><div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">{{ confirmTitle }}</h5>
          <button type="button" class="btn-close" @click="confirmModal = false" :disabled="confirmLoading"></button>
        </div>
        <div class="modal-body">
          <div class="text-center py-2">
            <i class="ti ti-alert-triangle fs-3"></i>
          </div>
          <p class="text-muted text-center mb-0">{{ confirmMsg }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost-secondary" @click="confirmModal = false" :disabled="confirmLoading">Batal</button>
          <button
            class="btn"
            :class="confirmDanger ? 'btn-danger' : 'btn-primary'"
            :disabled="confirmLoading"
            @click="executeConfirm"
          >
            {{ confirmLoading ? 'Memproses...' : 'Ya, Lanjutkan' }}
          </button>
        </div>
      </div></div>
    </div>
</template>

<style scoped>
/* ─── Circular Progress Ring ─────────────────────────────────── */
.progress-ring {
  --progress-value: 0;
  --progress-color: var(--tblr-primary);
  --ring-size: 56px;
  --ring-width: 5px;

  position: relative;
  width: var(--ring-size);
  height: var(--ring-size);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: conic-gradient(
    var(--progress-color) calc(var(--progress-value) * 1%),
    var(--tblr-border-color, #e6e7e9) calc(var(--progress-value) * 1%)
  );
  flex-shrink: 0;
}

.progress-ring::after {
  content: '';
  position: absolute;
  inset: var(--ring-width);
  border-radius: 50%;
  background: var(--tblr-card-bg, #fff);
}

.progress-ring__label {
  position: relative;
  z-index: 1;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--tblr-body-color);
}

/* Small variant for list view */
.progress-ring--sm {
  --ring-size: 36px;
  --ring-width: 3px;
}
</style>
