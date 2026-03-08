<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getAvatarUrl } from '../../../utils/avatar'
import { examApi, type ExamSchedule, type ExamSession } from '../../../api/exam.api'
import { supervisionApi } from '../../../api/supervision.api'
import { useWebSocket } from '../../../composables/useWebSocket'
import client from '../../../api/client'
import { useToastStore } from '@/stores/toast.store'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()

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

// ─── Types ──────────────────────────────────────────────────────
interface LiveStudent {
  user_id: number
  session_id: number
  name: string
  nis?: string
  answered: number
  total: number
  violation_count: number
  last_violation_type: string
  status: 'online' | 'offline' | 'finished' | 'terminated' | 'not_started' | 'locked'
  last_seen: string
  schedule_id?: number
  schedule_name?: string
  room_name?: string
}

type ViewMode = 'list' | 'grid'

// ─── State ─────────────────────────────────────────────────────
const router = useRouter()

const schedules = ref<ExamSchedule[]>([])
const students = ref<Map<number, LiveStudent>>(new Map())
const loading = ref(true)
const tokenCopied = ref<Record<number, boolean>>({})

// Filters
const searchQuery = ref('')
const filterExamStatus = ref('')
const filterScheduleId = ref<number | ''>('')
const filterOnlineStatus = ref('')
const perPage = ref(50)
const currentPage = ref(1)
const viewMode = ref<ViewMode>('list')

// Stats
const statTotal = computed(() => students.value.size)
const statOngoing = computed(() => [...students.value.values()].filter(s => s.status === 'online').length)
const statCompleted = computed(() => [...students.value.values()].filter(s => s.status === 'finished').length)
const statNotStarted = computed(() => [...students.value.values()].filter(s => s.status === 'not_started').length)

// WS — kept as plain non-reactive variable to avoid tracking overhead
let wsConnections = new Map<number, ReturnType<typeof useWebSocket>>()
const wsConnected = ref(false)
let stopWatchers: (() => void)[] = []
let offlineTimer: ReturnType<typeof setInterval> | null = null

// Modals
const messageModal = ref(false)
const messageTarget = ref<LiveStudent | null>(null)
const messageText = ref('')
const messageSending = ref(false)
const messageBulk = ref(false)

const timeModal = ref(false)
const timeTarget = ref<LiveStudent | null>(null)
const timeMinutes = ref(10)
const timeSending = ref(false)
const timeBulk = ref(false)

const confirmModal = ref(false)
const confirmTitle = ref('')
const confirmMsg = ref('')
const confirmDanger = ref(true)
const confirmAction = ref<(() => Promise<void>) | null>(null)
const confirmLoading = ref(false)

const unfinishedModal = ref(false)
const unfinishedList = ref<LiveStudent[]>([])

const selectedSessions = ref<number[]>([])

// ─── Computed ──────────────────────────────────────────────────
const filteredStudents = computed(() => {
  let list = [...students.value.values()]

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(s => s.name.toLowerCase().includes(q) || s.nis?.toLowerCase().includes(q))
  }

  if (filterExamStatus.value) {
    list = list.filter(s => {
      if (filterExamStatus.value === 'ongoing') return s.status === 'online'
      if (filterExamStatus.value === 'locked') return s.status === 'locked'
      if (filterExamStatus.value === 'completed') return s.status === 'finished'
      if (filterExamStatus.value === 'terminated') return s.status === 'terminated'
      if (filterExamStatus.value === 'not_started') return s.status === 'not_started'
      return true
    })
  }

  if (filterScheduleId.value !== '') {
    list = list.filter(s => s.schedule_id === filterScheduleId.value)
  }

  if (filterOnlineStatus.value === 'online') {
    list = list.filter(s => s.status === 'online')
  } else if (filterOnlineStatus.value === 'offline') {
    list = list.filter(s => s.status === 'offline' || s.status === 'not_started')
  }

  return list
})

const paginatedStudents = computed(() => {
  if (perPage.value === 0) return filteredStudents.value
  const start = (currentPage.value - 1) * perPage.value
  return filteredStudents.value.slice(start, start + perPage.value)
})

const totalPages = computed(() => {
  if (perPage.value === 0) return 1
  return Math.ceil(filteredStudents.value.length / perPage.value)
})

const paginationInfo = computed(() => {
  const total = filteredStudents.value.length
  if (perPage.value === 0) return `Menampilkan semua ${total} peserta`
  const start = (currentPage.value - 1) * perPage.value + 1
  const end = Math.min(currentPage.value * perPage.value, total)
  return `Menampilkan ${start}–${end} dari ${total} peserta`
})

// ─── Watch ─────────────────────────────────────────────────────
watch(filteredStudents, () => { currentPage.value = 1 })

// ─── Helpers ───────────────────────────────────────────────────
function progressPct(s: LiveStudent) {
  if (!s.total) return 0
  return Math.round((s.answered / s.total) * 100)
}

function statusBadgeClass(status: string) {
  return {
    online: 'bg-success-lt text-success',
    offline: 'bg-secondary-lt text-secondary',
    finished: 'bg-primary-lt text-primary',
    terminated: 'bg-danger-lt text-danger',
    not_started: 'bg-secondary-lt text-muted',
    locked: 'bg-warning-lt text-warning',
  }[status] ?? 'bg-secondary-lt text-secondary'
}

function statusDotClass(status: string) {
  return {
    online: 'bg-success',
    offline: 'bg-secondary',
    finished: 'bg-primary',
    terminated: 'bg-danger',
    not_started: 'bg-secondary',
    locked: 'bg-warning',
  }[status] ?? 'bg-secondary'
}

function statusLabel(status: string) {
  return {
    online: 'Sedang Mengerjakan',
    offline: 'Offline',
    finished: 'Selesai',
    terminated: 'Terblokir',
    not_started: 'Belum Memulai',
    locked: 'Menunggu Buka Kunci',
  }[status] ?? status
}

// ─── Token ────────────────────────────────────────────────────
async function copyToken(scheduleId: number, tok: string) {
  try {
    await navigator.clipboard.writeText(tok)
    tokenCopied.value = { ...tokenCopied.value, [scheduleId]: true }
    setTimeout(() => {
      tokenCopied.value = { ...tokenCopied.value, [scheduleId]: false }
    }, 2000)
  } catch {}
}

async function regenerateToken(scheduleId: number) {
  showConfirm(
    'Ganti Token Baru',
    'Token baru akan dibuat dan token lama tidak bisa digunakan lagi. Lanjutkan?',
    true,
    async () => {
      const res = await client.post(`/exam-schedules/${scheduleId}/regenerate-token`)
      const newToken = res.data?.data?.token
      if (newToken) {
        const idx = schedules.value.findIndex(s => s.id === scheduleId)
        if (idx >= 0) schedules.value[idx] = { ...schedules.value[idx], token: newToken } as ExamSchedule
      }
    }
  )
}

// ─── Data Load ────────────────────────────────────────────────
async function loadData() {
  loading.value = true
  try {
    const res = await examApi.listSchedules({ status: 'active', per_page: 50 })
    schedules.value = res.data.data ?? []

    // Load all sessions for all active schedules
    await Promise.all(schedules.value.map(loadSessionsForSchedule))
    connectAllWS()
  } finally {
    loading.value = false
  }
}

async function loadSessionsForSchedule(schedule: ExamSchedule) {
  try {
    const [ongoingRes, notStartedRes] = await Promise.allSettled([
      examApi.listOngoingSessions(schedule.id),
      examApi.listNotStartedSessions(schedule.id),
    ])
    const ongoing: ExamSession[] = ongoingRes.status === 'fulfilled' ? (ongoingRes.value.data.data ?? []) : []
    const notStarted: ExamSession[] = notStartedRes.status === 'fulfilled' ? (notStartedRes.value.data.data ?? []) : []

    for (const s of [...ongoing, ...notStarted]) {
      students.value.set(s.user_id + schedule.id * 1000000, {
        user_id: s.user_id,
        session_id: s.id,
        name: (s as any).user?.name ?? `User ${s.user_id}`,
        nis: (s as any).user?.nis ?? '',
        answered: 0,
        total: 0,
        violation_count: s.violation_count ?? 0,
        last_violation_type: '',
        status: s.status === 'finished' ? 'finished' : s.status === 'terminated' ? 'terminated' : 'not_started',
        last_seen: '',
        schedule_id: schedule.id,
        schedule_name: schedule.name,
      })
    }
    students.value = new Map(students.value)
  } catch {}
}

function studentKey(userId: number, scheduleId: number) {
  return userId + scheduleId * 1000000
}

// ─── WebSocket ────────────────────────────────────────────────
function connectAllWS() {
  for (const schedule of schedules.value) {
    connectWSForSchedule(schedule.id)
  }

  offlineTimer = setInterval(() => {
    const now = Date.now()
    let changed = false
    students.value.forEach((s) => {
      if (s.status === 'online' && s.last_seen) {
        const diff = (now - new Date(s.last_seen).getTime()) / 1000
        if (diff > 90) {
          s.status = 'offline'
          changed = true
        }
      }
    })
    if (changed) students.value = new Map(students.value)
  }, 15000)
}

function connectWSForSchedule(scheduleId: number) {
  const ws = useWebSocket(scheduleId)
  wsConnections.set(scheduleId, ws)

  ws.on('student_joined', (data: { user_id: number; session_id: number }) => {
    const key = studentKey(data.user_id, scheduleId)
    const s = students.value.get(key)
    if (s) {
      s.status = 'online'
      s.last_seen = new Date().toISOString()
    } else {
      students.value.set(key, {
        user_id: data.user_id,
        session_id: data.session_id,
        name: `User ${data.user_id}`,
        answered: 0,
        total: 0,
        violation_count: 0,
        last_violation_type: '',
        status: 'online',
        last_seen: new Date().toISOString(),
        schedule_id: scheduleId,
        schedule_name: schedules.value.find(s => s.id === scheduleId)?.name,
      })
    }
    students.value = new Map(students.value)
    wsConnected.value = true
  })

  ws.on('student_left', (data: { user_id: number }) => {
    const key = studentKey(data.user_id, scheduleId)
    const s = students.value.get(key)
    if (s && s.status === 'online') {
      s.status = 'offline'
      students.value = new Map(students.value)
    }
  })

  ws.on('answer_saved', (data: { user_id: number; answered: number; total: number }) => {
    const key = studentKey(data.user_id, scheduleId)
    const s = students.value.get(key)
    if (s) {
      s.answered = data.answered
      s.total = data.total
      s.last_seen = new Date().toISOString()
      students.value = new Map(students.value)
    }
  })

  ws.on('answer_batch', (batch: { user_id: number; answered: number; total: number }[]) => {
    for (const data of batch) {
      const key = studentKey(data.user_id, scheduleId)
      const s = students.value.get(key)
      if (s) {
        s.answered = data.answered
        s.total = data.total
        s.last_seen = new Date().toISOString()
      }
    }
    students.value = new Map(students.value)
  })

  ws.on('violation_logged', (data: { user_id: number; violation_type: string; violation_count: number }) => {
    const key = studentKey(data.user_id, scheduleId)
    const s = students.value.get(key)
    if (s) {
      s.violation_count = data.violation_count
      s.last_violation_type = data.violation_type
      students.value = new Map(students.value)
      const label = violationLabels[data.violation_type] ?? data.violation_type
      toast.warning(`${s.name}: ${label} (${data.violation_count}x)`)
    }
  })

  ws.on('session_finished', (data: { user_id: number }) => {
    const key = studentKey(data.user_id, scheduleId)
    const s = students.value.get(key)
    if (s) {
      s.status = 'finished'
      students.value = new Map(students.value)
    }
  })

  const { connected } = ws
  const stop = watch(connected, (val) => {
    if (val) wsConnected.value = true
  })
  stopWatchers.push(stop)

  ws.connect()
}

function disconnectAll() {
  stopWatchers.forEach(fn => fn())
  stopWatchers = []
  wsConnections.forEach(ws => ws.disconnect())
  wsConnections.clear()
  wsConnected.value = false
  if (offlineTimer) {
    clearInterval(offlineTimer)
    offlineTimer = null
  }
}

// ─── Confirm Modal ────────────────────────────────────────────
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

// ─── Actions ─────────────────────────────────────────────────
async function lockStudent(s: LiveStudent) {
  if (!s.schedule_id) return
  showConfirm('Kunci Akses', `Kunci akses ujian untuk ${s.name}?`, true, async () => {
    await client.post(`/api/v1/monitoring/${s.schedule_id}/lock`, {
      target_user_id: s.user_id,
      message: 'Akses dikunci oleh pengawas',
    })
    s.status = 'locked'
    students.value = new Map(students.value)
  })
}

async function unlockStudent(s: LiveStudent) {
  if (!s.schedule_id) return
  try {
    await supervisionApi.unlock(s.schedule_id, s.session_id)
    s.status = 'online'
    students.value = new Map(students.value)
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Gagal')
  }
}

async function forceFinish(s: LiveStudent) {
  if (!s.schedule_id) return
  showConfirm(
    'Paksa Selesai',
    `Paksa selesaikan ujian ${s.name}? Tindakan ini tidak dapat dibatalkan.`,
    true,
    async () => {
      await supervisionApi.forceFinish(s.schedule_id!, s.session_id)
      s.status = 'finished'
      students.value = new Map(students.value)
    }
  )
}

function openMessageModal(s: LiveStudent, bulk = false) {
  messageTarget.value = s
  messageText.value = ''
  messageBulk.value = bulk
  messageModal.value = true
}

async function submitMessage() {
  if (!messageTarget.value || !messageText.value.trim()) return
  messageSending.value = true
  try {
    if (messageBulk.value && selectedSessions.value.length > 0) {
      // Bulk message for all selected
      await Promise.all(
        [...students.value.values()]
          .filter(s => selectedSessions.value.includes(s.session_id) && s.schedule_id)
          .map(s => supervisionApi.sendMessage(s.schedule_id!, s.session_id, messageText.value.trim()))
      )
    } else if (messageTarget.value.schedule_id) {
      await supervisionApi.sendMessage(messageTarget.value.schedule_id, messageTarget.value.session_id, messageText.value.trim())
    }
    messageModal.value = false
    messageText.value = ''
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Gagal')
  } finally {
    messageSending.value = false
  }
}

function openTimeModal(s: LiveStudent, bulk = false) {
  timeTarget.value = s
  timeMinutes.value = 10
  timeBulk.value = bulk
  timeModal.value = true
}

async function submitExtendTime() {
  if (!timeTarget.value) return
  if (timeMinutes.value < 1 || timeMinutes.value > 120) return
  timeSending.value = true
  try {
    if (timeBulk.value && selectedSessions.value.length > 0) {
      // Group by schedule
      const bySchedule = new Map<number, number[]>()
      for (const s of [...students.value.values()]) {
        if (selectedSessions.value.includes(s.session_id) && s.schedule_id) {
          const arr = bySchedule.get(s.schedule_id) ?? []
          arr.push(s.session_id)
          bySchedule.set(s.schedule_id, arr)
        }
      }
      await Promise.all([...bySchedule.entries()].map(([scheduleId, ids]) =>
        supervisionApi.bulkAction(scheduleId, 'extend_time', ids, timeMinutes.value)
      ))
    } else if (timeTarget.value.schedule_id) {
      await supervisionApi.extendTime(timeTarget.value.schedule_id, timeTarget.value.session_id, timeMinutes.value)
    }
    timeModal.value = false
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Gagal')
  } finally {
    timeSending.value = false
  }
}

// ─── Selection ────────────────────────────────────────────────
function toggleSelect(sessionId: number) {
  const idx = selectedSessions.value.indexOf(sessionId)
  if (idx >= 0) selectedSessions.value.splice(idx, 1)
  else selectedSessions.value.push(sessionId)
}

function clearSelection() {
  selectedSessions.value = []
}

async function bulkForceFinish() {
  if (selectedSessions.value.length === 0) return
  showConfirm(
    'Paksa Selesai',
    `Paksa selesaikan ${selectedSessions.value.length} peserta yang dipilih?`,
    true,
    async () => {
      const bySchedule = new Map<number, number[]>()
      for (const s of [...students.value.values()]) {
        if (selectedSessions.value.includes(s.session_id) && s.schedule_id) {
          const arr = bySchedule.get(s.schedule_id) ?? []
          arr.push(s.session_id)
          bySchedule.set(s.schedule_id, arr)
        }
      }
      await Promise.all([...bySchedule.entries()].map(([scheduleId, ids]) =>
        supervisionApi.bulkAction(scheduleId, 'force_finish', ids)
      ))
      clearSelection()
    }
  )
}

async function bulkUnlock() {
  if (selectedSessions.value.length === 0) return
  showConfirm('Buka Kunci', `Buka kunci ${selectedSessions.value.length} peserta?`, false, async () => {
    const bySchedule = new Map<number, number[]>()
    for (const s of [...students.value.values()]) {
      if (selectedSessions.value.includes(s.session_id) && s.schedule_id) {
        const arr = bySchedule.get(s.schedule_id) ?? []
        arr.push(s.session_id)
        bySchedule.set(s.schedule_id, arr)
      }
    }
    await Promise.all([...bySchedule.entries()].map(([scheduleId, ids]) =>
      supervisionApi.bulkAction(scheduleId, 'unlock', ids)
    ))
    clearSelection()
  })
}

// ─── Unfinished ───────────────────────────────────────────────
function showUnfinishedStudents() {
  unfinishedList.value = [...students.value.values()].filter(s => s.status === 'not_started')
  unfinishedModal.value = true
}

// ─── Exit Supervision ─────────────────────────────────────────
function exitSupervision() {
  disconnectAll()
  router.push({ name: 'Supervision' })
}

// ─── Refresh ─────────────────────────────────────────────────
async function refreshAll() {
  students.value = new Map()
  await Promise.all(schedules.value.map(loadSessionsForSchedule))
}

onMounted(loadData)
onUnmounted(disconnectAll)
</script>

<template>
  <!-- Page Header -->
  <BasePageHeader
    :title="schedules.map(s => s.name).join(', ') || 'Semua Ruang Ujian Aktif'"
    subtitle="Memantau seluruh ruangan secara real-time."
    :breadcrumbs="[{ label: 'Pengawasan', to: '/admin/supervision' }, { label: 'Global' }]"
  >
    <template #actions>
      <!-- WS Indicator -->
      <span :class="wsConnected ? 'badge bg-success text-white' : 'badge bg-secondary-lt text-secondary'">
        <i class="ti ti-activity me-1"></i>
        {{ wsConnected ? 'Live' : 'Menghubungkan...' }}
      </span>
      <button class="btn btn-ghost-secondary btn-sm" @click="refreshAll" title="Refresh">
        <i class="ti ti-refresh"></i>
      </button>
      <button class="btn btn-outline-danger btn-sm" @click="exitSupervision">
        <i class="ti ti-logout me-1"></i>
        Keluar Sesi
      </button>
    </template>
  </BasePageHeader>

  <!-- Token Cards per Schedule -->
  <div v-for="schedule in schedules.filter(s => s.token)" :key="schedule.id" class="card mb-3 border-primary-lt">
    <div class="card-body">
      <div class="d-flex justify-content-between align-items-center flex-wrap gap-3">
        <div>
          <h3 class="card-title mb-1">
            <i class="ti ti-key text-primary me-2"></i>
            {{ schedule.name }}
          </h3>
          <div class="text-muted small">
            Berikan token ini kepada siswa untuk memulai ujian.
          </div>
        </div>
        <div class="d-flex align-items-center gap-3">
          <span class="display-6 fw-bold font-monospace text-primary">{{ schedule.token }}</span>
          <div class="btn-list flex-nowrap">
            <button
              class="btn btn-icon btn-ghost-secondary"
              :title="tokenCopied[schedule.id] ? 'Tersalin!' : 'Salin Token'"
              @click="copyToken(schedule.id, schedule.token)"
            >
              <i v-if="tokenCopied[schedule.id]" class="ti ti-check text-success"></i>
              <i v-else class="ti ti-copy"></i>
            </button>
            <button
              class="btn btn-icon btn-ghost-warning"
              title="Ganti Token Baru"
              @click="regenerateToken(schedule.id)"
            >
              <i class="ti ti-refresh"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Stats -->
  <div class="row row-cards mb-3">
    <div class="col-sm-6 col-lg-3">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="bg-primary text-white avatar">
                <i class="ti ti-users"></i>
              </span>
            </div>
            <div class="col">
              <div class="font-weight-medium">Total Peserta</div>
              <div class="text-muted">
                <span class="fw-bold text-primary">{{ statTotal }}</span> Terdaftar
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-sm-6 col-lg-3">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="bg-success text-white avatar">
                <i class="ti ti-pencil"></i>
              </span>
            </div>
            <div class="col">
              <div class="font-weight-medium">Sedang Mengerjakan</div>
              <div class="text-muted">
                <span class="fw-bold text-success">{{ statOngoing }}</span> Siswa
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-sm-6 col-lg-3">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="bg-primary text-white avatar">
                <i class="ti ti-check"></i>
              </span>
            </div>
            <div class="col">
              <div class="font-weight-medium">Sudah Selesai</div>
              <div class="text-muted">
                <span class="fw-bold text-primary">{{ statCompleted }}</span> Siswa
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="col-sm-6 col-lg-3">
      <div class="card card-sm">
        <div class="card-body">
          <div class="row align-items-center">
            <div class="col-auto">
              <span class="bg-secondary text-white avatar">
                <i class="ti ti-clock"></i>
              </span>
            </div>
            <div class="col">
              <div class="font-weight-medium">Belum Mengerjakan</div>
              <div class="text-muted">
                <span class="fw-bold text-secondary">{{ statNotStarted }}</span> Siswa
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Filters -->
  <div class="card mb-3">
    <div class="card-body">
      <div class="row g-3 align-items-end">
        <div class="col-md-4">
          <label class="form-label">Cari Nama / NIS</label>
          <div class="input-group">
            <span class="input-group-text"><i class="ti ti-search"></i></span>
            <input
              v-model="searchQuery"
              type="text"
              class="form-control"
              placeholder="Ketik nama..."
            />
          </div>
        </div>
        <div class="col-md-3">
          <label class="form-label">Status Ujian</label>
          <select v-model="filterExamStatus" class="form-select">
            <option value="">Semua Status</option>
            <option value="ongoing">Sedang Mengerjakan</option>
            <option value="locked">Menunggu Buka Kunci</option>
            <option value="completed">Selesai</option>
            <option value="terminated">Terblokir</option>
            <option value="not_started">Belum Memulai</option>
          </select>
        </div>
        <div class="col-md-2">
          <label class="form-label">Jadwal Ujian</label>
          <select v-model="filterScheduleId" class="form-select">
            <option value="">Semua Jadwal</option>
            <option v-for="s in schedules" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
        </div>
        <div class="col-md-2">
          <label class="form-label">Status Online</label>
          <select v-model="filterOnlineStatus" class="form-select">
            <option value="">Semua</option>
            <option value="online">Online</option>
            <option value="offline">Offline</option>
          </select>
        </div>
        <div class="col-md-auto">
          <div class="btn-group" role="group">
            <button
              type="button"
              class="btn btn-outline-secondary"
              :class="{ active: viewMode === 'list' }"
              title="Tampilan List"
              @click="viewMode = 'list'"
            >
              <i class="ti ti-list"></i>
            </button>
            <button
              type="button"
              class="btn btn-outline-secondary"
              :class="{ active: viewMode === 'grid' }"
              title="Tampilan Grid"
              @click="viewMode = 'grid'"
            >
              <i class="ti ti-layout-grid"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Student Table/Grid -->
  <div class="card">
    <div class="card-header">
      <h3 class="card-title">Daftar Peserta</h3>
      <div class="ms-auto d-flex align-items-center gap-2">
        <div v-if="loading" class="spinner-border spinner-border-sm text-secondary" role="status"></div>
        <button type="button" class="btn btn-warning btn-sm" @click="showUnfinishedStudents">
          <i class="ti ti-list-details me-1"></i>
          Cek Belum Ujian
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="list-group list-group-flush">
      <div class="list-group-item p-5 text-center">
        <div class="spinner-border text-primary" role="status"></div>
        <p class="mt-2 text-muted">Memuat data...</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredStudents.length === 0">
      <div class="empty">
        <div class="empty-icon">
          <i class="ti ti-device-desktop-off" style="font-size: 3rem;"></i>
        </div>
        <p class="empty-title">Tidak ada sesi pengawasan aktif</p>
        <p class="empty-subtitle text-muted">Tidak ada peserta yang sesuai dengan filter saat ini, atau belum ada sesi ujian yang berlangsung.</p>
      </div>
    </div>

    <!-- LIST VIEW -->
    <template v-else-if="viewMode === 'list'">
      <div class="list-group list-group-flush list-group-hoverable">
        <div
          v-for="s in paginatedStudents"
          :key="s.session_id"
          class="list-group-item"
          :class="{ 'border-danger': s.violation_count > 0 }"
        >
          <div class="row align-items-center">
            <!-- Checkbox -->
            <div class="col-auto">
              <input
                type="checkbox"
                class="form-check-input"
                :checked="selectedSessions.includes(s.session_id)"
                @change="toggleSelect(s.session_id)"
              />
            </div>
            <!-- Avatar + Name -->
            <div class="col-auto">
              <span
                class="avatar"
                :style="`background-image: url(${getAvatarUrl(s.user_id)})`"
              >
                <span
                  class="badge"
                  :class="statusDotClass(s.status)"
                  style="width:10px;height:10px;position:absolute;bottom:0;right:0;border-radius:50%;padding:0;"
                ></span>
              </span>
            </div>
            <div class="col">
              <div class="d-flex align-items-center gap-2">
                <span class="fw-medium">{{ s.name }}</span>
                <span v-if="s.nis" class="text-muted small">{{ s.nis }}</span>
                <span v-if="s.schedule_name" class="badge bg-secondary-lt text-secondary small">{{ s.schedule_name }}</span>
                <span v-if="s.violation_count > 0" class="badge bg-danger-lt text-danger"
                  :title="s.last_violation_type ? (violationLabels[s.last_violation_type] ?? s.last_violation_type) : ''">
                  <i class="ti ti-alert-triangle me-1"></i>{{ s.violation_count }}x
                </span>
              </div>
              <div class="d-flex align-items-center gap-2 mt-1">
                <span class="badge" :class="statusBadgeClass(s.status)">{{ statusLabel(s.status) }}</span>
                <div v-if="s.total > 0" class="d-flex align-items-center gap-1 small text-muted">
                  <div class="progress progress-xs" style="width:80px">
                    <div
                      class="progress-bar"
                      :class="progressPct(s) >= 100 ? 'bg-success' : progressPct(s) >= 50 ? 'bg-primary' : 'bg-warning'"
                      :style="{ width: progressPct(s) + '%' }"
                    ></div>
                  </div>
                  <span>{{ s.answered }}/{{ s.total }}</span>
                </div>
              </div>
            </div>
            <!-- Actions -->
            <div class="col-auto">
              <div class="btn-list">
                <button
                  v-if="s.status === 'online'"
                  class="btn btn-sm btn-ghost-secondary"
                  title="Kunci peserta"
                  @click="lockStudent(s)"
                >
                  <i class="ti ti-lock"></i>
                </button>
                <button
                  v-if="s.status === 'locked'"
                  class="btn btn-sm btn-ghost-success"
                  title="Buka kunci"
                  @click="unlockStudent(s)"
                >
                  <i class="ti ti-lock-open"></i>
                </button>
                <button
                  v-if="s.status === 'online'"
                  class="btn btn-sm btn-ghost-secondary"
                  title="Tambah waktu"
                  @click="openTimeModal(s)"
                >
                  <i class="ti ti-clock-plus"></i>
                </button>
                <button
                  v-if="s.status === 'online' || s.status === 'offline'"
                  class="btn btn-sm btn-ghost-secondary"
                  title="Kirim pesan"
                  @click="openMessageModal(s)"
                >
                  <i class="ti ti-message"></i>
                </button>
                <button
                  v-if="s.status === 'online'"
                  class="btn btn-sm btn-ghost-danger"
                  title="Paksa selesai"
                  @click="forceFinish(s)"
                >
                  <i class="ti ti-player-stop"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- GRID VIEW -->
    <template v-else>
      <div class="row g-3 p-3">
        <div
          v-for="s in paginatedStudents"
          :key="s.session_id"
          class="col-6 col-sm-4 col-md-3 col-lg-2"
        >
          <div
            class="card h-100 text-center position-relative"
            :class="{ 'border-danger': s.violation_count > 0, 'border-primary': selectedSessions.includes(s.session_id) }"
          >
            <input
              type="checkbox"
              class="form-check-input position-absolute top-0 end-0 m-2"
              :checked="selectedSessions.includes(s.session_id)"
              @change="toggleSelect(s.session_id)"
            />
            <div
              class="avatar avatar-xs rounded-circle position-absolute top-0 start-0 mt-2 ms-2"
              :class="statusDotClass(s.status)"
            ></div>
            <div class="card-body pt-3 pb-1">
              <div
                class="avatar avatar-md rounded-circle mx-auto mb-2"
                :style="`background-image:url(${getAvatarUrl(s.user_id)})`"
              ></div>
              <p class="fw-medium mb-0 small text-truncate">{{ s.name }}</p>
              <span class="badge mt-1" :class="statusBadgeClass(s.status)">{{ statusLabel(s.status) }}</span>
              <div v-if="s.violation_count > 0" class="mt-1">
                <span class="badge bg-danger-lt text-danger"
                  :title="s.last_violation_type ? (violationLabels[s.last_violation_type] ?? s.last_violation_type) : ''">
                  <i class="ti ti-alert-triangle me-1"></i>{{ s.violation_count }}x
                </span>
              </div>
            </div>
            <div class="px-2 pb-2">
              <div class="progress progress-sm">
                <div
                  class="progress-bar"
                  :class="progressPct(s) >= 100 ? 'bg-success' : progressPct(s) >= 50 ? 'bg-primary' : 'bg-warning'"
                  :style="{ width: progressPct(s) + '%' }"
                ></div>
              </div>
              <span class="text-muted small">{{ s.answered }}/{{ s.total }}</span>
            </div>
            <div class="card-footer d-flex justify-content-center gap-1 p-1">
              <button v-if="s.status === 'online'" class="btn btn-sm btn-ghost-secondary p-1" title="Kunci" @click="lockStudent(s)">
                <i class="ti ti-lock"></i>
              </button>
              <button v-if="s.status === 'locked'" class="btn btn-sm btn-ghost-success p-1" title="Buka" @click="unlockStudent(s)">
                <i class="ti ti-lock-open"></i>
              </button>
              <button v-if="s.status === 'online'" class="btn btn-sm btn-ghost-secondary p-1" title="Pesan" @click="openMessageModal(s)">
                <i class="ti ti-message"></i>
              </button>
              <button v-if="s.status === 'online'" class="btn btn-sm btn-ghost-danger p-1" title="Paksa Selesai" @click="forceFinish(s)">
                <i class="ti ti-player-stop"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Pagination Footer -->
    <div class="card-footer">
      <div class="row align-items-center gy-2">
        <div class="col-12 col-md-auto text-center text-md-start">
          <p class="m-0 text-muted small">{{ paginationInfo }}</p>
        </div>
        <div class="col-12 col-md d-flex justify-content-center justify-content-md-end align-items-center gap-2">
          <select v-model.number="perPage" class="form-select form-select-sm w-auto">
            <option :value="10">10</option>
            <option :value="50">50</option>
            <option :value="0">Semua</option>
          </select>
          <ul v-if="totalPages > 1" class="pagination pagination-sm m-0 flex-wrap justify-content-center">
            <li class="page-item" :class="{ disabled: currentPage === 1 }">
              <button class="page-link" @click="currentPage--">
                <i class="ti ti-chevron-left"></i>
              </button>
            </li>
            <li
              v-for="p in totalPages"
              :key="p"
              class="page-item"
              :class="{ active: currentPage === p }"
            >
              <button class="page-link" @click="currentPage = p">{{ p }}</button>
            </li>
            <li class="page-item" :class="{ disabled: currentPage === totalPages }">
              <button class="page-link" @click="currentPage++">
                <i class="ti ti-chevron-right"></i>
              </button>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>

  <!-- Floating Bulk Action Bar -->
  <Transition name="slide-up">
    <div
      v-if="selectedSessions.length > 0"
      class="d-flex align-items-center bg-dark text-white rounded-pill shadow-lg px-4 py-2 gap-3"
      style="position: fixed; bottom: 30px; left: 50%; transform: translateX(-50%); z-index: 2000;"
    >
      <div class="d-flex align-items-center gap-2 border-end pe-3 border-secondary">
        <span class="fw-bold text-white small">{{ selectedSessions.length }} Dipilih</span>
        <button class="btn btn-sm btn-icon btn-dark border-0 shadow-none text-muted" @click="clearSelection" title="Batal Pilih">
          <i class="ti ti-x"></i>
        </button>
      </div>
      <div class="d-flex gap-2 flex-wrap">
        <button class="btn btn-sm btn-danger btn-pill" @click="bulkForceFinish">
          <i class="ti ti-player-stop me-1"></i>Akhiri
        </button>
        <button class="btn btn-sm btn-success btn-pill" @click="bulkUnlock">
          <i class="ti ti-lock-open me-1"></i>Buka Kunci
        </button>
        <button class="btn btn-sm btn-warning btn-pill" @click="openMessageModal(selectedSessions.length > 0 ? [...students.values()].find(s => s.session_id === selectedSessions[0])! : ({} as LiveStudent), true)">
          <i class="ti ti-message me-1"></i>Pesan
        </button>
        <button class="btn btn-sm btn-info btn-pill" @click="openTimeModal(selectedSessions.length > 0 ? [...students.values()].find(s => s.session_id === selectedSessions[0])! : ({} as LiveStudent), true)">
          <i class="ti ti-clock-plus me-1"></i>+ Waktu
        </button>
      </div>
    </div>
  </Transition>

  <!-- Message Modal -->
  <div v-if="messageModal" class="modal modal-blur show d-block" @click.self="messageModal = false">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">
            <span v-if="messageBulk">Kirim Pesan ke {{ selectedSessions.length }} Peserta</span>
            <span v-else>Kirim Pesan ke {{ messageTarget?.name }}</span>
          </h5>
          <button type="button" class="btn-close" @click="messageModal = false"></button>
        </div>
        <div class="modal-body">
          <textarea
            v-model="messageText"
            class="form-control"
            rows="3"
            placeholder="Contoh: Waktu tinggal 5 menit lagi!"
            @keydown.ctrl.enter="submitMessage"
          ></textarea>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost-secondary" @click="messageModal = false">Batal</button>
          <button
            class="btn btn-primary"
            :disabled="!messageText.trim() || messageSending"
            @click="submitMessage"
          >
            <i class="ti ti-send me-1"></i>
            {{ messageSending ? 'Mengirim...' : 'Kirim Pesan' }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Extend Time Modal -->
  <div v-if="timeModal" class="modal modal-blur show d-block" @click.self="timeModal = false">
    <div class="modal-dialog modal-dialog-centered modal-sm">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">
            <span v-if="timeBulk">Tambah Waktu — {{ selectedSessions.length }} Peserta</span>
            <span v-else>Tambah Waktu — {{ timeTarget?.name }}</span>
          </h5>
          <button type="button" class="btn-close" @click="timeModal = false"></button>
        </div>
        <div class="modal-body">
          <div class="mb-3">
            <label class="form-label">Durasi Tambahan (Menit)</label>
            <input
              type="number"
              v-model.number="timeMinutes"
              min="1"
              max="120"
              class="form-control"
            />
            <div class="form-hint">Maksimal 120 menit.</div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost-secondary" @click="timeModal = false">Batal</button>
          <button
            class="btn btn-primary"
            :disabled="timeSending || timeMinutes < 1"
            @click="submitExtendTime"
          >
            <i class="ti ti-clock-plus me-1"></i>
            {{ timeSending ? 'Menambahkan...' : 'Tambah' }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Confirm Modal -->
  <div v-if="confirmModal" class="modal modal-blur show d-block" @click.self="confirmModal = false">
    <div class="modal-dialog modal-dialog-centered modal-sm">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">{{ confirmTitle }}</h5>
          <button type="button" class="btn-close" @click="confirmModal = false" :disabled="confirmLoading"></button>
        </div>
        <div class="modal-body">
          <div class="text-center py-2">
            <i class="ti ti-alert-triangle fs-3 text-warning"></i>
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
      </div>
    </div>
  </div>

  <!-- Unfinished Students Modal -->
  <div v-if="unfinishedModal" class="modal modal-blur show d-block" @click.self="unfinishedModal = false">
    <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Peserta Belum Ujian</h5>
          <button type="button" class="btn-close" @click="unfinishedModal = false"></button>
        </div>
        <div class="modal-body p-0">
          <div v-if="unfinishedList.length === 0" class="text-center p-5">
            <i class="ti ti-confetti text-success fs-1 d-block mb-2"></i>
            <p class="empty-title fw-bold">Semua Selesai!</p>
            <p class="text-muted">Tidak ada siswa yang tertinggal.</p>
          </div>
          <div v-else class="list-group list-group-flush">
            <div
              v-for="s in unfinishedList"
              :key="s.session_id"
              class="list-group-item"
            >
              <div class="d-flex align-items-center gap-2">
                <span class="avatar avatar-sm" :style="`background-image:url(${getAvatarUrl(s.user_id)})`"></span>
                <div>
                  <div class="fw-medium">{{ s.name }}</div>
                  <div class="text-muted small">
                    {{ s.schedule_name ?? '' }}
                    <span v-if="s.nis"> · {{ s.nis }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <div class="text-muted small me-auto">
            {{ unfinishedList.length }} peserta belum mengerjakan
          </div>
          <button type="button" class="btn btn-secondary" @click="unfinishedModal = false">Tutup</button>
        </div>
      </div>
    </div>
  </div>
</template>
