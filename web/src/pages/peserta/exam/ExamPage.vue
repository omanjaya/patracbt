<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { examApi, type ExamSession, type SafeQuestion, type ExamAnswer, type Stimulus } from '../../../api/exam.api'
import { useExamTimer } from '../../../composables/useExamTimer'
import { useWebSocket } from '../../../composables/useWebSocket'
import { useAntiCheat } from '@/composables/useAntiCheat'

import ExamHeader from './components/ExamHeader.vue'
import ExamSidebar from './components/ExamSidebar.vue'
import ExamQuestionPanel from './components/ExamQuestionPanel.vue'
import ExamSummaryModal from './components/ExamSummaryModal.vue'
import ExamViolationBanner from './components/ExamViolationBanner.vue'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.id)

const session = ref<ExamSession | null>(null)
const questions = ref<SafeQuestion[]>([])
const answersMap = ref<Map<number, unknown>>(new Map())
const flaggedSet = ref<Set<number>>(new Set())
const currentIdx = ref(0)
const loading = ref(true)
const saving = ref(false)
const saveFailed = ref(false)
const lastSaved = ref(false)
const finishing = ref(false)
const showFinishModal = ref(false)
const showSummaryModal = ref(false)
const error = ref('')
const locked = ref(false)
const lockMessage = ref('Akses dikunci oleh pengawas')
const chatNotif = ref<{ sender: string; message: string } | null>(null)
let chatNotifTimer: ReturnType<typeof setTimeout> | null = null

// Anti-cheat composable
const antiCheat = useAntiCheat({
  onViolation: (event) => {
    examApi.logViolation(sessionId, { violation_type: event.type, description: event.description })
  },
  onBlur: () => flushPendingAnswers(),
})
const violationWarning = antiCheat.violationWarning
const violationCount = antiCheat.violationCount
const sidebarOpen = ref(false)
const showShortcutHint = ref(false)

// Offline detection
const isOnline = ref(navigator.onLine)
const offlineQueueKey = computed(() => `exam_offline_queue_${sessionId}`)
function loadOfflineQueue(): Array<{ question_id: number; answer: unknown; is_flagged: boolean }> {
  try {
    const raw = localStorage.getItem(`exam_offline_queue_${sessionId}`)
    if (raw) return JSON.parse(raw)
  } catch { /* ignore parse errors */ }
  return []
}
const offlineQueue = ref<Array<{ question_id: number; answer: unknown; is_flagged: boolean }>>(loadOfflineQueue())
watch(offlineQueue, (val) => {
  try {
    if (val.length > 0) {
      localStorage.setItem(offlineQueueKey.value, JSON.stringify(val))
    } else {
      localStorage.removeItem(offlineQueueKey.value)
    }
  } catch { /* storage full or unavailable */ }
}, { deep: true })
let flushingQueue = false

// Batch save
const pendingAnswers = ref<Map<number, { question_id: number; answer: unknown; is_flagged: boolean }>>(new Map())
let flushTimer: ReturnType<typeof setTimeout> | null = null
let lastSavedTimer: ReturnType<typeof setTimeout> | null = null

function onOnline() {
  isOnline.value = true
  flushOfflineQueue()
}
function onOffline() {
  isOnline.value = false
}
async function flushOfflineQueue() {
  if (flushingQueue || offlineQueue.value.length === 0) return
  flushingQueue = true
  let processed = 0
  while (processed < offlineQueue.value.length && isOnline.value) {
    const item = offlineQueue.value[processed]
    try {
      await examApi.saveAnswer(sessionId, item!)
      processed++
    } catch {
      break
    }
  }
  if (processed > 0) {
    offlineQueue.value = offlineQueue.value.slice(processed)
  }
  flushingQueue = false
  if (offlineQueue.value.length === 0) {
    saveFailed.value = false
  }
}

async function flushPendingAnswers() {
  if (pendingAnswers.value.size === 0 || saving.value) return
  if (flushTimer) { clearTimeout(flushTimer); flushTimer = null }
  saving.value = true
  lastSaved.value = false

  const answers = Array.from(pendingAnswers.value.values())
  pendingAnswers.value.clear()

  if (!isOnline.value) {
    answers.forEach(a => offlineQueue.value.push(a))
    saveFailed.value = true
    saving.value = false
    return
  }

  let success = false
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      await examApi.batchSaveAnswers(sessionId, answers)
      success = true
      break
    } catch {
      if (attempt < 2) {
        await new Promise(r => setTimeout(r, Math.pow(2, attempt) * 1000))
      }
    }
  }

  if (success) {
    saveFailed.value = false
    lastSaved.value = true
    if (lastSavedTimer) clearTimeout(lastSavedTimer)
    lastSavedTimer = setTimeout(() => { lastSaved.value = false }, 2000)
  } else {
    answers.forEach(a => offlineQueue.value.push(a))
    saveFailed.value = true
  }

  saving.value = false
}

// Timer warning banners
const timerWarningBanner = ref('')
let timerWarningBannerTimeout: ReturnType<typeof setTimeout> | null = null

// Stimulus
const stimulusError = ref(false)
const stimulusRetryId = ref<number | null>(null)

// Min working time
const minWorkingTime = ref(0)
const examStartedAt = ref<Date | null>(null)
const elapsedMinutes = ref(0)
let minTimeInterval: ReturnType<typeof setInterval> | null = null
const canFinish = computed(() => {
  if (minWorkingTime.value <= 0) return true
  return elapsedMinutes.value >= minWorkingTime.value
})
const minTimeRemaining = computed(() => {
  if (minWorkingTime.value <= 0) return 0
  return Math.max(0, minWorkingTime.value - elapsedMinutes.value)
})
const minTimeRemainingFormatted = computed(() => {
  const totalSec = Math.ceil(minTimeRemaining.value * 60)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

// Stimulus cache
const stimulusCache = ref<Map<number, Stimulus>>(new Map())
const currentStimulus = ref<Stimulus | null>(null)

// Audio replay limits
const audioPlayCounts = ref<Map<string, number>>(new Map())
const MAX_AUDIO_PLAYS = 2

function attachCustomAudioPlayers() {
  const audios = document.querySelectorAll('.question-body audio, .stimulus-content audio')
  audios.forEach((audio: any) => {
    if (audio.dataset.initialized) return
    audio.dataset.initialized = '1'
    audio.controls = false
    audio.style.display = 'none'

    const src = audio.getAttribute('src') || ''
    const qId = currentQuestion.value?.id ?? 0
    const audioKey = `${qId}:${src}`
    let currentPlays = audioPlayCounts.value.get(audioKey) || 0

    const wrapper = document.createElement('div')
    wrapper.className = 'custom-audio-player mb-3'
    const ui = document.createElement('div')
    ui.className = 'd-flex align-items-center gap-3 p-2 border rounded bg-white shadow-sm'
    const playBtn = document.createElement('button')
    playBtn.className = 'btn btn-primary btn-sm'
    playBtn.innerHTML = '&#9654; Putar'
    const textInfo = document.createElement('span')
    textInfo.className = 'small fw-medium'
    textInfo.textContent = `Sisa putar: ${MAX_AUDIO_PLAYS - currentPlays}x`
    const progressContainer = document.createElement('div')
    progressContainer.className = 'progress flex-grow-1'
    progressContainer.style.height = '6px'
    const progressBar = document.createElement('div')
    progressBar.className = 'progress-bar bg-primary'
    progressBar.style.width = '0%'
    progressContainer.appendChild(progressBar)

    if (currentPlays >= MAX_AUDIO_PLAYS) {
      playBtn.disabled = true
      playBtn.innerHTML = 'Habis'
      textInfo.textContent = 'Batas putar tercapai'
      textInfo.className = 'small fw-medium text-danger'
    }

    playBtn.onclick = () => {
      if (currentPlays >= MAX_AUDIO_PLAYS) return
      if (audio.paused) {
        audio.play()
        playBtn.innerHTML = '&#9208; Jeda'
      } else {
        audio.pause()
        playBtn.innerHTML = '&#9654; Putar'
      }
    }

    audio.onended = () => {
      currentPlays++
      audioPlayCounts.value.set(audioKey, currentPlays)
      playBtn.innerHTML = '&#9654; Putar'
      progressBar.style.width = '0%'
      if (currentPlays >= MAX_AUDIO_PLAYS) {
        playBtn.disabled = true
        playBtn.innerHTML = 'Habis'
        textInfo.textContent = 'Batas putar tercapai'
        textInfo.className = 'small fw-medium text-danger'
      } else {
        textInfo.textContent = `Sisa putar: ${MAX_AUDIO_PLAYS - currentPlays}x`
      }
    }

    audio.ontimeupdate = () => {
      if (audio.duration) {
        const pct = (audio.currentTime / audio.duration) * 100
        progressBar.style.width = `${pct}%`
      }
    }

    ui.appendChild(playBtn)
    ui.appendChild(progressContainer)
    ui.appendChild(textInfo)
    audio.parentNode?.insertBefore(wrapper, audio)
    wrapper.appendChild(audio)
    wrapper.appendChild(ui)
  })
}

// WebSocket
let ws: ReturnType<typeof useWebSocket> | null = null

function connectWS(scheduleId: number) {
  ws = useWebSocket(scheduleId, sessionId)

  ws.on('lock_client', (data: { target_user_id: number; message: string }) => {
    locked.value = true
    lockMessage.value = data.message || 'Akses dikunci oleh pengawas'
  })

  ws.on('force_finish', async () => {
    if (saveTimeout) clearTimeout(saveTimeout)
    if (flushTimer) { clearTimeout(flushTimer); flushTimer = null }
    await flushPendingAnswers()
    ws?.disconnect()
    antiCheat.stop()
    router.replace(`/peserta/result/${sessionId}`)
  })

  ws.on('time_extended', (data: { new_end_time: string }) => {
    if (data.new_end_time) {
      timerEndTime.value = data.new_end_time
    }
  })

  ws.on('chat_message', (data: { sender_name: string; message: string }) => {
    chatNotif.value = { sender: data.sender_name, message: data.message }
    if (chatNotifTimer) clearTimeout(chatNotifTimer)
    chatNotifTimer = setTimeout(() => { chatNotif.value = null }, 8000)
  })

  ws.connect()
}

// Timer
const timerEndTime = ref<string | null>(null)
const timer = useExamTimer(timerEndTime, {
  onExpire: () => { finishExam() },
  onWarning: () => {
    timerWarningBanner.value = 'Waktu tersisa 5 menit!'
    if (timerWarningBannerTimeout) clearTimeout(timerWarningBannerTimeout)
    timerWarningBannerTimeout = setTimeout(() => { timerWarningBanner.value = '' }, 8000)
  },
  onDanger: () => {
    timerWarningBanner.value = 'Waktu tersisa 1 menit!'
    if (timerWarningBannerTimeout) clearTimeout(timerWarningBannerTimeout)
    timerWarningBannerTimeout = setTimeout(() => { timerWarningBanner.value = '' }, 8000)
  },
})

const currentQuestion = computed(() => questions.value[currentIdx.value] ?? null)
const isFlagged = computed(() => flaggedSet.value.has(currentQuestion.value?.id ?? 0))
const answeredCount = computed(() => answersMap.value.size)
const unansweredCount = computed(() => questions.value.length - answeredCount.value)
const flaggedCount = computed(() => flaggedSet.value.size)
const progressPercent = computed(() => answeredCount.value / Math.max(questions.value.length, 1) * 100)

// Section grouping
interface QuestionSection {
  bankId: number
  bankName: string
  startIdx: number
  endIdx: number
}
const questionSections = computed<QuestionSection[]>(() => {
  if (!questions.value.length) return []
  const banks = session.value?.exam_schedule?.question_banks
  const bankNameMap = new Map<number, string>()
  if (banks) {
    for (const b of banks) {
      bankNameMap.set(b.question_bank_id, b.question_bank?.name ?? `Bank ${b.question_bank_id}`)
    }
  }
  const sections: QuestionSection[] = []
  let currentBankId = -1
  for (let i = 0; i < questions.value.length; i++) {
    const q = questions.value[i]!
    if (q.question_bank_id !== currentBankId) {
      currentBankId = q.question_bank_id
      sections.push({
        bankId: currentBankId,
        bankName: bankNameMap.get(currentBankId) ?? `Bagian ${sections.length + 1}`,
        startIdx: i,
        endIdx: i,
      })
    } else {
      sections[sections.length - 1]!.endIdx = i
    }
  }
  return sections
})
const hasMultipleSections = computed(() => questionSections.value.length > 1)

// Stimulus fetch
async function fetchStimulus(stimulusId: number | null) {
  currentStimulus.value = null
  stimulusError.value = false
  stimulusRetryId.value = null
  if (!stimulusId) return
  if (stimulusCache.value.has(stimulusId)) {
    currentStimulus.value = stimulusCache.value.get(stimulusId)!
    return
  }
  try {
    const res = await examApi.getStimulus(stimulusId, { timeout: 10000 })
    const s = res.data.data
    stimulusCache.value.set(stimulusId, s)
    currentStimulus.value = s
    nextTick(attachCustomAudioPlayers)
  } catch {
    stimulusError.value = true
    stimulusRetryId.value = stimulusId
  }
}
function retryStimulus() {
  if (stimulusRetryId.value) {
    fetchStimulus(stimulusRetryId.value)
  }
}

async function loadSession() {
  loading.value = true
  try {
    const res = await examApi.loadSession(sessionId)
    const data = res.data.data
    session.value = data.session
    questions.value = data.questions

    for (const ans of (data.answers as ExamAnswer[])) {
      answersMap.value.set(ans.question_id, ans.answer)
      if (ans.is_flagged) flaggedSet.value.add(ans.question_id)
    }

    if (data.session.min_working_time && data.session.min_working_time > 0) {
      minWorkingTime.value = data.session.min_working_time
      examStartedAt.value = data.session.start_time ? new Date(data.session.start_time) : new Date()
      const updateElapsed = () => {
        if (examStartedAt.value) {
          elapsedMinutes.value = (Date.now() - examStartedAt.value.getTime()) / 60000
        }
      }
      updateElapsed()
      minTimeInterval = setInterval(updateElapsed, 1000)
    }

    timerEndTime.value = data.session.end_time

    if (data.session.exam_schedule_id) {
      connectWS(data.session.exam_schedule_id)
    }

    antiCheat.start()
    window.addEventListener('beforeunload', onBeforeUnload)
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat sesi ujian.'
  } finally {
    loading.value = false
  }
}

// Current answer state per type
const pgAnswer = ref('')
const pgkAnswers = ref<string[]>([])
const bsAnswer = ref('')
const matchPairs = ref<Record<string, string>>({})
const isianText = ref('')
const matrixAnswers = ref<Record<string, string>>({})
const esaiText = ref('')

function loadAnswerIntoState() {
  const q = currentQuestion.value
  if (!q) return
  const ans = answersMap.value.get(q.id) as any

  pgAnswer.value = ''
  pgkAnswers.value = []
  bsAnswer.value = ''
  matchPairs.value = {}
  isianText.value = ''
  matrixAnswers.value = {}
  esaiText.value = ''

  if (!ans || typeof ans !== 'object') return

  switch (q.question_type) {
    case 'pg': pgAnswer.value = ans.option_id ?? ''; break
    case 'pgk': pgkAnswers.value = Array.isArray(ans.option_ids) ? ans.option_ids : []; break
    case 'benar_salah': bsAnswer.value = ans.option_id ?? ''; break
    case 'menjodohkan': matchPairs.value = ans.pairs && typeof ans.pairs === 'object' ? ans.pairs : {}; break
    case 'isian_singkat': isianText.value = ans.text ?? ''; break
    case 'matrix': matrixAnswers.value = ans.answers && typeof ans.answers === 'object' ? ans.answers : {}; break
    case 'esai': esaiText.value = ans.text ?? ''; break
  }
}

watch(currentIdx, () => {
  loadAnswerIntoState()
  fetchStimulus(currentQuestion.value?.stimulus_id ?? null)
  nextTick(() => {
    attachCustomAudioPlayers()
    const btn = document.querySelector<HTMLElement>(`.num-btn[data-idx="${currentIdx.value}"]`)
    btn?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
})

function buildAnswer(): unknown {
  const q = currentQuestion.value
  if (!q) return null
  switch (q.question_type) {
    case 'pg': return pgAnswer.value ? { option_id: pgAnswer.value } : null
    case 'pgk': return pgkAnswers.value.length ? { option_ids: pgkAnswers.value } : null
    case 'benar_salah': return bsAnswer.value ? { option_id: bsAnswer.value } : null
    case 'menjodohkan': return Object.keys(matchPairs.value).length ? { pairs: matchPairs.value } : null
    case 'isian_singkat': return isianText.value.trim() ? { text: isianText.value.trim() } : null
    case 'matrix': return Object.keys(matrixAnswers.value).length ? { answers: matrixAnswers.value } : null
    case 'esai': return esaiText.value.trim() ? { text: esaiText.value.trim() } : null
  }
  return null
}

let saveTimeout: ReturnType<typeof setTimeout> | null = null

function onAnswerChange() {
  const q = currentQuestion.value
  if (!q) return
  const ans = buildAnswer()
  if (ans !== null) answersMap.value.set(q.id, ans)
  else answersMap.value.delete(q.id)

  pendingAnswers.value.set(q.id, {
    question_id: q.id,
    answer: buildAnswer(),
    is_flagged: flaggedSet.value.has(q.id),
  })

  if (flushTimer) clearTimeout(flushTimer)
  flushTimer = setTimeout(() => flushPendingAnswers(), 3000)

  if (pendingAnswers.value.size >= 10) {
    flushPendingAnswers()
  }
}

async function autoSave() {
  const q = currentQuestion.value
  if (!q) return
  pendingAnswers.value.set(q.id, {
    question_id: q.id,
    answer: buildAnswer(),
    is_flagged: flaggedSet.value.has(q.id),
  })
  await flushPendingAnswers()
}

function toggleFlag() {
  const q = currentQuestion.value
  if (!q) return
  if (flaggedSet.value.has(q.id)) flaggedSet.value.delete(q.id)
  else flaggedSet.value.add(q.id)
  autoSave()
}

async function goTo(idx: number) {
  await flushPendingAnswers()
  currentIdx.value = idx
  if (window.innerWidth < 768) sidebarOpen.value = false
}
async function prev() { if (currentIdx.value > 0) { await flushPendingAnswers(); currentIdx.value-- } }
async function next() { if (currentIdx.value < questions.value.length - 1) { await flushPendingAnswers(); currentIdx.value++ } }

function requestFinish() {
  showSummaryModal.value = true
}

const opts = computed(() => {
  const q = currentQuestion.value
  if (!q) return []
  const o = q.options as any
  if (!Array.isArray(o)) return []
  const orderMap = session.value?.option_order
  if (orderMap && orderMap[q.id]) {
    const orderedIds: string[] = orderMap[q.id] ?? []
    const optMap = new Map(o.map((opt: any) => [opt.id, opt]))
    const reordered = orderedIds.map(id => optMap.get(id)).filter(Boolean)
    return reordered.length === o.length ? reordered : o
  }
  return o
})

// Keyboard shortcuts
function handleKeydown(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement)?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
    if (e.key === 'Escape') {
      if (showSummaryModal.value) { showSummaryModal.value = false; e.preventDefault() }
      if (showFinishModal.value) { showFinishModal.value = false; e.preventDefault() }
    }
    return
  }

  if (showSummaryModal.value || showFinishModal.value) {
    if (e.key === 'Escape') {
      if (showSummaryModal.value) showSummaryModal.value = false
      if (showFinishModal.value) showFinishModal.value = false
      e.preventDefault()
    }
    return
  }

  switch (e.key) {
    case 'ArrowLeft':
    case 'PageUp':
      e.preventDefault()
      prev()
      break
    case 'ArrowRight':
    case 'PageDown':
      e.preventDefault()
      next()
      break
    case 'f':
    case 'F':
      e.preventDefault()
      toggleFlag()
      break
    case 's':
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault()
        flushPendingAnswers()
      }
      break
    case 'Escape':
      if (showShortcutHint.value) showShortcutHint.value = false
      break
    default:
      if (/^[1-9]$/.test(e.key)) {
        e.preventDefault()
        selectOptionByIndex(parseInt(e.key) - 1)
      }
  }
}

function selectOptionByIndex(index: number) {
  const q = currentQuestion.value
  if (!q) return
  const type = q.question_type
  const options = opts.value

  if (type === 'pg') {
    if (options[index]) {
      pgAnswer.value = options[index].id
      onAnswerChange()
    }
    return
  }

  if (type === 'benar_salah') {
    const bsOpts = [{ id: 'true' }, { id: 'false' }]
    if (bsOpts[index]) {
      bsAnswer.value = bsOpts[index].id
      onAnswerChange()
    }
    return
  }

  if (type === 'pgk') {
    if (options[index]) {
      const optId = options[index].id
      const idx = pgkAnswers.value.indexOf(optId)
      if (idx >= 0) {
        pgkAnswers.value.splice(idx, 1)
      } else {
        pgkAnswers.value.push(optId)
      }
      onAnswerChange()
    }
    return
  }
}

async function finishExam() {
  finishing.value = true
  try {
    if (saveTimeout) clearTimeout(saveTimeout)
    if (flushTimer) { clearTimeout(flushTimer); flushTimer = null }
    await autoSave()
    await flushPendingAnswers()
    await examApi.finishExam(sessionId)
    antiCheat.stop()
    ws?.disconnect()
    const hasNext = session.value?.exam_schedule?.next_exam_schedule_id != null
    if (hasNext) {
      router.replace(`/peserta/exam/${sessionId}/transition`)
    } else {
      router.replace(`/peserta/result/${sessionId}`)
    }
  } finally {
    finishing.value = false
  }
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  e.preventDefault()
  if (saveTimeout) clearTimeout(saveTimeout)
  if (flushTimer) { clearTimeout(flushTimer); flushTimer = null }

  const beaconAnswers: { question_id: number; answer: unknown; is_flagged: boolean }[] =
    Array.from(pendingAnswers.value.values())

  if (beaconAnswers.length > 0) {
    let token = ''
    try { token = localStorage.getItem('access_token') ?? '' } catch { /* storage unavailable */ }
    if (!token) return
    const url = `/api/v1/exam/sessions/${sessionId}/answers/batch`
    const blob = new Blob([JSON.stringify(beaconAnswers)], { type: 'application/json' })
    fetch(url, {
      method: 'POST',
      body: blob,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      keepalive: true,
    })
    pendingAnswers.value.clear()
  }
}

let cleaned = false
function cleanup() {
  if (cleaned) return
  cleaned = true
  ws?.disconnect()
  antiCheat.stop()
  window.removeEventListener('beforeunload', onBeforeUnload)
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('online', onOnline)
  window.removeEventListener('offline', onOffline)
  if (chatNotifTimer) clearTimeout(chatNotifTimer)
  if (minTimeInterval) clearInterval(minTimeInterval)
  if (timerWarningBannerTimeout) clearTimeout(timerWarningBannerTimeout)
  if (flushTimer) clearTimeout(flushTimer)
  if (saveTimeout) clearTimeout(saveTimeout)
  if (lastSavedTimer) clearTimeout(lastSavedTimer)
}

onUnmounted(cleanup)
onBeforeRouteLeave(() => { cleanup() })

// Touch swipe navigation
let touchStartX = 0
let touchStartY = 0
function onTouchStart(e: TouchEvent) {
  touchStartX = e.touches[0]!.clientX
  touchStartY = e.touches[0]!.clientY
}
function onTouchEnd(e: TouchEvent) {
  const dx = e.changedTouches[0]!.clientX - touchStartX
  const dy = e.changedTouches[0]!.clientY - touchStartY
  // Only trigger if horizontal swipe > 50px and more horizontal than vertical (don't interfere with scroll)
  if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.5) {
    if (dx < 0) next()   // swipe left = next
    else prev()           // swipe right = prev
  }
}

onMounted(() => {
  loadSession()
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('online', onOnline)
  window.addEventListener('offline', onOffline)
})

// Handle answer updates from QuestionPanel child component
function onUpdatePgAnswer(val: string) { pgAnswer.value = val }
function onUpdatePgkAnswers(val: string[]) { pgkAnswers.value = val }
function onUpdateBsAnswer(val: string) { bsAnswer.value = val }
function onUpdateMatchPairs(val: Record<string, string>) { matchPairs.value = val }
function onUpdateIsianText(val: string) { isianText.value = val }
function onUpdateMatrixAnswers(val: Record<string, string>) { matrixAnswers.value = val }
function onUpdateEsaiText(val: string) { esaiText.value = val }

function onSummaryConfirmFinish() {
  showSummaryModal.value = false
  showFinishModal.value = true
}
</script>

<template>
  <div class="exam-shell">
    <!-- Chat notification toast -->
    <div v-if="chatNotif" class="chat-toast">
      <i class="ti ti-message-circle"></i>
      <div class="chat-toast-body">
        <p class="chat-toast-sender">{{ chatNotif.sender }}</p>
        <p class="chat-toast-msg">{{ chatNotif.message }}</p>
      </div>
      <button class="chat-toast-close" @click="chatNotif = null">&#x2715;</button>
    </div>

    <!-- Banners -->
    <ExamViolationBanner
      :show-violation="violationWarning"
      :violation-message="antiCheat.latestMessage.value"
      :violation-count="violationCount"
      :show-save-failed="saveFailed"
      :show-offline="!isOnline"
      :timer-warning-banner="timerWarningBanner"
      @dismiss-violation="violationWarning = false"
      @dismiss-timer-warning="timerWarningBanner = ''"
    />

    <!-- Lock overlay -->
    <div v-if="locked" class="lock-overlay">
      <div class="lock-box">
        <i class="ti ti-lock"></i>
        <h2>Akses Dikunci</h2>
        <p>{{ lockMessage }}</p>
        <p class="lock-sub">Hubungi pengawas untuk informasi lebih lanjut.</p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="exam-loading">
      <div class="loading-spinner" />
      <p>Memuat ujian...</p>
    </div>

    <div v-else-if="error" class="exam-error">
      <i class="ti ti-alert-circle"></i>
      <p>{{ error }}</p>
    </div>

    <template v-else>
      <!-- Header -->
      <ExamHeader
        :exam-title="session?.exam_schedule?.name ?? 'Ujian'"
        :timer-formatted="timer.formatted.value"
        :timer-is-warning="timer.isWarning.value"
        :timer-is-danger="timer.isDanger.value"
        :timer-is-critical="timer.isCritical.value"
        :timer-is-paused="timer.isPaused.value"
        :saving="saving"
        :last-saved="lastSaved"
        :save-failed="saveFailed"
        :is-online="isOnline"
        :violation-count="violationCount"
        :max-violations="session?.exam_schedule?.max_violations ?? '?'"
        :min-time-remaining="minTimeRemaining"
        :min-time-remaining-formatted="minTimeRemainingFormatted"
        :show-shortcut-hint="showShortcutHint"
        :answered-count="answeredCount"
        :total-questions="questions.length"
        :can-finish="canFinish"
        :sidebar-open="sidebarOpen"
        @request-finish="requestFinish"
        @toggle-shortcut-hint="(v: boolean) => showShortcutHint = v"
        @toggle-sidebar="sidebarOpen = !sidebarOpen"
      />

      <!-- Progress bar -->
      <div class="exam-progress" role="progressbar" :aria-valuenow="answeredCount" :aria-valuemax="questions.length">
        <div class="exam-progress-bar" :style="{ width: progressPercent + '%' }" />
      </div>

      <div class="exam-body" @touchstart.passive="onTouchStart" @touchend.passive="onTouchEnd">
        <!-- Question panel -->
        <ExamQuestionPanel
          :question="currentQuestion"
          :opts="opts"
          :stimulus="currentStimulus"
          :stimulus-loading="!currentStimulus && !stimulusError && !!currentQuestion?.stimulus_id"
          :stimulus-error="stimulusError"
          :is-flagged="isFlagged"
          :current-idx="currentIdx"
          :total-questions="questions.length"
          :pg-answer="pgAnswer"
          :pgk-answers="pgkAnswers"
          :bs-answer="bsAnswer"
          :match-pairs="matchPairs"
          :isian-text="isianText"
          :matrix-answers="matrixAnswers"
          :esai-text="esaiText"
          @answer-change="onAnswerChange"
          @toggle-flag="toggleFlag"
          @retry-stimulus="retryStimulus"
          @prev="prev"
          @next="next"
          @update:pg-answer="onUpdatePgAnswer"
          @update:pgk-answers="onUpdatePgkAnswers"
          @update:bs-answer="onUpdateBsAnswer"
          @update:match-pairs="onUpdateMatchPairs"
          @update:isian-text="onUpdateIsianText"
          @update:matrix-answers="onUpdateMatrixAnswers"
          @update:esai-text="onUpdateEsaiText"
        />

        <!-- Sidebar -->
        <ExamSidebar
          :questions="questions"
          :current-idx="currentIdx"
          :answers-map="answersMap"
          :flagged-set="flaggedSet"
          :question-sections="questionSections"
          :has-multiple-sections="hasMultipleSections"
          :sidebar-open="sidebarOpen"
          :progress-percent="progressPercent"
          :answered-count="answeredCount"
          :unanswered-count="unansweredCount"
          :flagged-count="flaggedCount"
          :can-finish="canFinish"
          :min-time-remaining="minTimeRemaining"
          @go-to="goTo"
          @toggle-sidebar="sidebarOpen = false"
          @request-finish="requestFinish"
        />
      </div>

      <!-- Summary modal -->
      <ExamSummaryModal
        :show="showSummaryModal"
        :questions="questions"
        :answers-map="answersMap"
        :flagged-set="flaggedSet"
        :answered-count="answeredCount"
        :unanswered-count="unansweredCount"
        :flagged-count="flaggedCount"
        :finishing="finishing"
        :question-sections="questionSections"
        :has-multiple-sections="hasMultipleSections"
        @close="showSummaryModal = false"
        @go-to="goTo"
        @confirm-finish="onSummaryConfirmFinish"
      />

      <!-- Final Finish Confirmation modal -->
      <Teleport to="body">
        <div v-if="showFinishModal" class="modal-overlay" @click.self="showFinishModal = false">
          <div class="modal-dialog modal-dialog-centered">
            <div class="modal-content">
              <div class="modal-icon-wrap">
                <i class="ti ti-alert-triangle" style="font-size:2.25rem" />
              </div>
              <h2 class="modal-title">Yakin Selesaikan Ujian?</h2>
              <p class="modal-desc">
                Anda telah menjawab <strong>{{ answeredCount }}</strong> dari <strong>{{ questions.length }}</strong> soal.
                <span v-if="unansweredCount > 0" class="modal-warn"> Masih ada <strong>{{ unansweredCount }}</strong> soal belum dijawab.</span>
                <br/>Setelah dikumpulkan, jawaban tidak dapat diubah lagi.
              </p>
              <div class="modal-actions">
                <button class="btn-cancel" @click="showFinishModal = false">Batal</button>
                <button class="btn-confirm" @click="finishExam" :disabled="finishing">
                  {{ finishing ? 'Menyelesaikan...' : 'Ya, Kumpulkan' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </Teleport>
    </template>
  </div>
</template>

<style scoped>
/* Shell Layout */
.exam-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: #f1f5f9;
  font-family: var(--tblr-font-sans-serif, 'Inter', sans-serif);
}

/* Progress Bar */
.exam-progress {
  height: 3px;
  background: #e2e8f0;
  flex-shrink: 0;
}
.exam-progress-bar {
  height: 100%;
  background: var(--tblr-primary, #4f46e5);
  transition: width 0.3s ease;
  border-radius: 0 2px 2px 0;
}

/* Body Layout */
.exam-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  position: relative;
}

/* Lock Overlay */
.lock-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
}
.lock-box {
  text-align: center;
  background: #fff;
  padding: 3rem 2.5rem;
  border-radius: 16px;
  max-width: 400px;
}
.lock-box i { font-size: 3rem; color: #dc2626; }
.lock-box h2 { margin: 1rem 0 0.5rem; font-size: 1.25rem; color: #1e293b; }
.lock-box p { font-size: 0.9rem; color: #475569; margin: 0; }
.lock-sub { font-size: 0.8rem !important; color: #94a3b8 !important; margin-top: 0.5rem !important; }

/* Loading / Error */
.exam-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  color: #64748b;
}
.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e2e8f0;
  border-top-color: var(--tblr-primary, #4f46e5);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.exam-error {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: #dc2626;
  font-size: 0.95rem;
}

/* Chat Toast */
.chat-toast {
  position: fixed;
  top: 4.5rem;
  right: 1rem;
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  z-index: 50;
  max-width: 320px;
  animation: slide-in-right 0.3s ease;
}
@keyframes slide-in-right {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
.chat-toast-body { flex: 1; }
.chat-toast-sender {
  font-size: 0.75rem;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 0.125rem;
}
.chat-toast-msg {
  font-size: 0.8rem;
  color: #475569;
  margin: 0;
}
.chat-toast-close {
  border: none;
  background: none;
  color: #94a3b8;
  cursor: pointer;
  font-size: 0.8rem;
  padding: 0;
}

/* Finish Confirmation Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.modal-dialog {
  width: 100%;
  max-width: 420px;
}
.modal-content {
  background: #fff;
  border-radius: 16px;
  padding: 2rem;
  text-align: center;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}
.modal-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin: 0 auto 1rem;
  border-radius: 50%;
  background: #fef2f2;
  color: #dc2626;
}
.modal-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 0.5rem;
}
.modal-desc {
  font-size: 0.85rem;
  color: #64748b;
  line-height: 1.6;
  margin: 0 0 1.5rem;
}
.modal-warn {
  color: #dc2626;
  font-weight: 600;
}
.modal-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
}
.btn-cancel {
  padding: 0.5rem 1.25rem;
  font-size: 0.85rem;
  font-weight: 600;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-cancel:hover { background: #f1f5f9; }
.btn-confirm {
  padding: 0.5rem 1.5rem;
  font-size: 0.85rem;
  font-weight: 700;
  border: none;
  border-radius: 8px;
  background: #dc2626;
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-confirm:hover:not(:disabled) { background: #b91c1c; }
.btn-confirm:disabled { opacity: 0.5; cursor: not-allowed; }

/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  .exam-shell {
    overscroll-behavior: none;
    touch-action: manipulation;
  }
  .exam-body {
    flex-direction: column;
  }
  .chat-toast {
    top: auto;
    bottom: 1rem;
    right: 0.5rem;
    left: 0.5rem;
    max-width: none;
  }
  .lock-box {
    margin: 1rem;
    padding: 2rem 1.5rem;
  }
  .modal-content {
    padding: 1.5rem 1rem;
    margin: 0 0.5rem;
  }
  .modal-actions {
    flex-direction: column;
  }
  .btn-cancel,
  .btn-confirm {
    width: 100%;
    padding: 0.75rem;
    min-height: 48px;
    font-size: 0.9rem;
  }
}

/* ── Tablet Responsive ── */
@media (min-width: 768px) and (max-width: 1024px) {
  .modal-content {
    padding: 1.75rem;
  }
  .btn-cancel,
  .btn-confirm {
    padding: 0.625rem 1.5rem;
    min-height: 44px;
  }
}
</style>
