<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import BaseModal from '../../../components/ui/BaseModal.vue'
import BasePagination from '../../../components/ui/BasePagination.vue'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import { useRoute, useRouter } from 'vue-router'
import { useToastStore } from '../../../stores/toast.store'
import { sanitizeHtml } from '@/composables/useSafeHtml'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'
import {
  questionBankApi,
  type QuestionBank,
  type Question,
  type Stimulus,
  type CreateQuestionPayload,
  QUESTION_TYPES,
  DIFFICULTY_LABELS,
  BLOOM_LEVELS,
} from '../../../api/question_bank.api'

const toast = useToastStore()

// Confirm modal state
const confirmModal = ref(false)
const confirmLoading = ref(false)
const confirmTitle = ref('Konfirmasi Hapus')
const confirmMessage = ref('')
const pendingAction = ref<(() => Promise<void>) | null>(null)

function askConfirm(title: string, message: string, action: () => Promise<void>) {
  confirmTitle.value = title
  confirmMessage.value = message
  pendingAction.value = action
  confirmModal.value = true
}

async function doConfirm() {
  if (!pendingAction.value) return
  confirmLoading.value = true
  try {
    await pendingAction.value()
  } finally {
    confirmLoading.value = false
    confirmModal.value = false
  }
}

const route = useRoute()
const router = useRouter()
const bankId = Number(route.params.id)

if (isNaN(bankId) || bankId <= 0) {
  router.replace('/admin/question-banks')
}

const bank = ref<QuestionBank | null>(null)
const questions = ref<Question[]>([])
const stimuli = ref<Stimulus[]>([])
const loadingBank = ref(true)
const loadingQuestions = ref(false)
const page = ref(1)
const totalPages = ref(1)
const total = ref(0)

// Question modal
const showModal = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const editId = ref<number | null>(null)
const expandedId = ref<number | null>(null)

// Audio state
const audioFile = ref<File | null>(null)
const removeAudio = ref(false)
const existingAudioPath = ref<string | null>(null)

const form = reactive<{
  stimulus_id: string
  question_type: string
  body: string
  score: number
  difficulty: string
  order_index: number
  audio_limit: number
  bloom_level: number
  topic_code: string
  // PG / PGK / BS options
  options: { id: string; text: string; score: number }[]
  correct_single: string
  correct_multi: string[]
  // Menjodohkan
  match_prompts: { id: string; text: string }[]
  match_answers: { id: string; text: string }[]
  match_pairs: Record<string, string>
  // Isian singkat
  accepted_answers: string[]
  // Matrix
  matrix_rows: { id: string; text: string }[]
  matrix_cols: { id: string; text: string }[]
  matrix_correct: Record<string, string>
}>({
  stimulus_id: '',
  question_type: 'pg',
  body: '',
  score: 1,
  difficulty: 'medium',
  order_index: 0,
  audio_limit: 2,
  bloom_level: 0,
  topic_code: '',
  options: [
    { id: 'a', text: '', score: 1 },
    { id: 'b', text: '', score: 0 },
    { id: 'c', text: '', score: 0 },
    { id: 'd', text: '', score: 0 },
  ],
  correct_single: 'a',
  correct_multi: [],
  match_prompts: [{ id: '1', text: '' }, { id: '2', text: '' }],
  match_answers: [{ id: 'a', text: '' }, { id: 'b', text: '' }],
  match_pairs: {},
  accepted_answers: [''],
  matrix_rows: [{ id: '1', text: '' }],
  matrix_cols: [{ id: 'a', text: '' }, { id: 'b', text: '' }],
  matrix_correct: {},
})

// Question form errors (real-time validation)
const formErrors = reactive<Record<string, string>>({})

function validateQuestionField(field: string) {
  delete formErrors[field]
  switch (field) {
    case 'body':
      if (!form.body.trim()) formErrors.body = 'Soal / pertanyaan wajib diisi'
      break
    case 'question_type':
      if (!form.question_type) formErrors.question_type = 'Tipe soal wajib dipilih'
      break
    case 'score':
      if (form.score <= 0) formErrors.score = 'Bobot nilai harus lebih dari 0'
      break
  }
}

// Stimulus modal
const showStimulusModal = ref(false)
const editStimulusId = ref<number | null>(null)
const stimulusContent = ref('')
const savingStimulus = ref(false)

async function fetchBank() {
  try {
    const res = await questionBankApi.getById(bankId)
    bank.value = res.data.data
  } finally {
    loadingBank.value = false
  }
}

async function fetchQuestions() {
  loadingQuestions.value = true
  try {
    const res = await questionBankApi.listQuestions(bankId, { page: page.value, per_page: 20 })
    questions.value = res.data.data ?? []
    totalPages.value = res.data.meta?.total_pages ?? 1
    total.value = res.data.meta?.total ?? 0
  } finally {
    loadingQuestions.value = false
  }
}

async function fetchStimuli() {
  const res = await questionBankApi.listStimuli(bankId)
  stimuli.value = res.data.data ?? []
}

function resetForm() {
  Object.keys(formErrors).forEach(k => delete formErrors[k])
  Object.assign(form, {
    stimulus_id: '', question_type: 'pg', body: '', score: 1,
    difficulty: 'medium', order_index: questions.value.length,
    audio_limit: 2,
    bloom_level: 0,
    topic_code: '',
    options: [
      { id: 'a', text: '', score: 1 }, { id: 'b', text: '', score: 0 },
      { id: 'c', text: '', score: 0 }, { id: 'd', text: '', score: 0 },
    ],
    correct_single: 'a', correct_multi: [],
    match_prompts: [{ id: '1', text: '' }, { id: '2', text: '' }],
    match_answers: [{ id: 'a', text: '' }, { id: 'b', text: '' }],
    match_pairs: {},
    accepted_answers: [''],
    matrix_rows: [{ id: '1', text: '' }],
    matrix_cols: [{ id: 'a', text: '' }, { id: 'b', text: '' }],
    matrix_correct: {},
  })
  audioFile.value = null
  removeAudio.value = false
  existingAudioPath.value = null
}

function openCreate() {
  isEdit.value = false; editId.value = null
  resetForm()
  showModal.value = true
}

function openEdit(q: Question) {
  isEdit.value = true; editId.value = q.id
  resetForm()
  form.question_type = q.question_type
  form.body = q.body
  form.score = q.score
  form.difficulty = q.difficulty
  form.order_index = q.order_index
  form.stimulus_id = q.stimulus_id ? String(q.stimulus_id) : ''
  form.audio_limit = q.audio_limit ?? 2
  form.bloom_level = q.bloom_level ?? 0
  form.topic_code = q.topic_code ?? ''
  existingAudioPath.value = q.audio_path ?? null

  // Restore type-specific fields from options/correct_answer
  try {
    const opts = q.options as any
    const ans = q.correct_answer as any

    if (q.question_type === 'pg' || q.question_type === 'pgk' || q.question_type === 'benar_salah') {
      if (Array.isArray(opts)) form.options = opts
      if (q.question_type === 'pgk') form.correct_multi = Array.isArray(ans) ? ans : []
      else form.correct_single = typeof ans === 'string' ? ans : ''
    } else if (q.question_type === 'menjodohkan') {
      form.match_prompts = opts?.prompts ?? form.match_prompts
      form.match_answers = opts?.answers ?? form.match_answers
      form.match_pairs = ans ?? {}
    } else if (q.question_type === 'isian_singkat') {
      form.accepted_answers = opts?.accepted_answers ?? ['']
    } else if (q.question_type === 'matrix') {
      form.matrix_rows = opts?.rows ?? form.matrix_rows
      form.matrix_cols = opts?.columns ?? form.matrix_cols
      form.matrix_correct = ans ?? {}
    }
  } catch {}

  showModal.value = true
}

function buildPayload(): CreateQuestionPayload {
  const { question_type: qt } = form
  let options: unknown = null
  let correct_answer: unknown = null

  if (qt === 'pg' || qt === 'pgk' || qt === 'benar_salah') {
    options = qt === 'benar_salah'
      ? [{ id: 'true', text: 'Benar' }, { id: 'false', text: 'Salah' }]
      : form.options
    correct_answer = qt === 'pgk' ? form.correct_multi : form.correct_single
  } else if (qt === 'menjodohkan') {
    options = { prompts: form.match_prompts, answers: form.match_answers }
    correct_answer = form.match_pairs
  } else if (qt === 'isian_singkat') {
    options = { accepted_answers: form.accepted_answers.filter(a => a.trim()) }
    correct_answer = null
  } else if (qt === 'matrix') {
    options = { rows: form.matrix_rows, columns: form.matrix_cols }
    correct_answer = form.matrix_correct
  } else if (qt === 'esai') {
    options = null
    correct_answer = null
  }

  return {
    stimulus_id: form.stimulus_id ? Number(form.stimulus_id) : undefined,
    question_type: qt,
    body: form.body,
    score: form.score,
    difficulty: form.difficulty,
    order_index: form.order_index,
    options,
    correct_answer,
    audio_limit: form.audio_limit,
    bloom_level: form.bloom_level,
    topic_code: form.topic_code,
  }
}

function onAudioFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  audioFile.value = input.files?.[0] ?? null
}

function handleRemoveAudio() {
  removeAudio.value = true
  existingAudioPath.value = null
  audioFile.value = null
}

function audioStreamUrl(path: string): string {
  const base = import.meta.env.VITE_API_BASE_URL?.replace('/api/v1', '') ?? ''
  return `${base}/audio-stream/${path}`
}

async function handleSave() {
  saving.value = true
  try {
    const payload = buildPayload()
    if (isEdit.value && editId.value) {
      await questionBankApi.updateQuestion(editId.value, { ...payload, remove_audio: removeAudio.value }, audioFile.value ?? undefined)
    } else {
      await questionBankApi.createQuestion(bankId, payload, audioFile.value ?? undefined)
    }
    showModal.value = false
    toast.success('Soal berhasil disimpan')
    await fetchQuestions()
    await fetchBank()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan soal')
  } finally {
    saving.value = false
  }
}

function handleDelete(q: Question) {
  askConfirm(
    'Hapus Soal',
    'Apakah Anda yakin ingin menghapus soal ini? Tindakan ini tidak dapat dibatalkan.',
    async () => {
      try {
        await questionBankApi.deleteQuestion(q.id)
        toast.success('Soal berhasil dihapus')
        await fetchQuestions()
        await fetchBank()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus soal')
      }
    },
  )
}

function toggleExpand(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}

// Options helpers
function addOption() {
  const ids = 'abcdefghijklmnopqrstuvwxyz'
  const next = ids[form.options.length] ?? String(form.options.length)
  form.options.push({ id: next, text: '', score: 0 })
}
function removeOption(idx: number) {
  form.options.splice(idx, 1)
}
function addMatchRow() {
  const n = String(form.match_prompts.length + 1)
  form.match_prompts.push({ id: n, text: '' })
  const ids = 'abcdefghijklmnopqrstuvwxyz'
  const k = ids[form.match_answers.length] ?? n
  form.match_answers.push({ id: k, text: '' })
}
function addMatrixRow() {
  form.matrix_rows.push({ id: String(form.matrix_rows.length + 1), text: '' })
}
function addMatrixCol() {
  const ids = 'abcdefghijklmnopqrstuvwxyz'
  form.matrix_cols.push({ id: ids[form.matrix_cols.length] ?? String(form.matrix_cols.length), text: '' })
}
function addAcceptedAnswer() { form.accepted_answers.push('') }
function removeAcceptedAnswer(i: number) { form.accepted_answers.splice(i, 1) }

// Stimulus handlers
function openCreateStimulus() {
  editStimulusId.value = null; stimulusContent.value = ''
  showStimulusModal.value = true
}
function openEditStimulus(s: Stimulus) {
  editStimulusId.value = s.id; stimulusContent.value = s.content
  showStimulusModal.value = true
}
async function handleSaveStimulus() {
  savingStimulus.value = true
  try {
    if (editStimulusId.value) await questionBankApi.updateStimulus(editStimulusId.value, { content: stimulusContent.value })
    else await questionBankApi.createStimulus(bankId, { content: stimulusContent.value })
    showStimulusModal.value = false
    toast.success('Stimulus berhasil disimpan')
    await fetchStimuli()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan stimulus')
  } finally {
    savingStimulus.value = false
  }
}
function handleDeleteStimulus(s: Stimulus) {
  askConfirm(
    'Hapus Stimulus',
    'Apakah Anda yakin ingin menghapus stimulus ini?',
    async () => {
      try {
        await questionBankApi.deleteStimulus(s.id)
        toast.success('Stimulus berhasil dihapus')
        await fetchStimuli()
      } catch (e: any) {
        toast.error(e.response?.data?.message ?? 'Gagal menghapus stimulus')
      }
    },
  )
}

const diffBadge: Record<string, 'success' | 'warning' | 'danger'> = {
  easy: 'success', medium: 'warning', hard: 'danger',
}

const isLocked = computed(() => bank.value?.is_locked === true)

const qtLabel = (t: string) => QUESTION_TYPES.find(x => x.value === t)?.label ?? t

// Reorder state
const reorderMode = ref(false)
const reorderList = ref<Question[]>([])
const savingReorder = ref(false)

function enterReorderMode() {
  reorderList.value = [...questions.value]
  reorderMode.value = true
}

function cancelReorder() {
  reorderMode.value = false
}

function onDragStart(e: DragEvent, idx: number) {
  e.dataTransfer?.setData('text/plain', String(idx))
}

function onDrop(e: DragEvent, toIdx: number) {
  e.preventDefault()
  const fromIdx = Number(e.dataTransfer?.getData('text/plain') ?? -1)
  if (fromIdx < 0 || fromIdx === toIdx) return
  const arr = [...reorderList.value]
  const [moved] = arr.splice(fromIdx, 1)
  arr.splice(toIdx, 0, moved!)
  reorderList.value = arr
}

async function saveReorder() {
  savingReorder.value = true
  try {
    const items = reorderList.value.map((q, idx) => ({ id: q.id, order_index: idx }))
    await questionBankApi.reorderQuestions(bankId, items)
    reorderMode.value = false
    toast.success('Urutan soal berhasil disimpan')
    await fetchQuestions()
  } catch (e: any) {
    toast.error(e.response?.data?.message ?? 'Gagal menyimpan urutan soal')
  } finally {
    savingReorder.value = false
  }
}

// Filter state
const filterType = ref('')
const filterDifficulty = ref('')

watch([filterType, filterDifficulty], () => { page.value = 1 })

const filteredQuestions = computed(() => {
  return questions.value.filter(q => {
    if (filterType.value && q.question_type !== filterType.value) return false
    if (filterDifficulty.value && q.difficulty !== filterDifficulty.value) return false
    return true
  })
})

function isCorrectOption(q: Question, optId: string): boolean {
  if (q.question_type === 'pg' || q.question_type === 'benar_salah') {
    return q.correct_answer === optId
  }
  if (q.question_type === 'pgk') {
    return Array.isArray(q.correct_answer) && (q.correct_answer as string[]).includes(optId)
  }
  return false
}

onMounted(() => { fetchBank(); fetchQuestions(); fetchStimuli() })
</script>

<template>
    <!-- Header -->
    <BasePageHeader
      :title="bank?.name ?? ''"
      :subtitle="bank ? `${bank.subject?.name ?? 'Tanpa mata pelajaran'} · ${total} soal` : ''"
      :breadcrumbs="[{ label: 'Bank Soal', to: '/admin/question-banks' }, { label: bank?.name ?? 'Detail' }]"
    >
      <template #actions>
        <button v-if="!reorderMode && !isLocked && questions.length > 1"
                class="btn btn-secondary" @click="enterReorderMode">
          <i class="ti ti-arrows-sort"></i> Atur Urutan
        </button>
        <button class="btn btn-secondary" @click="openCreateStimulus" :disabled="isLocked"><i class="ti ti-stack"></i>
          Tambah Stimulus</button>
        <button class="btn btn-primary" @click="openCreate" :disabled="isLocked"><i class="ti ti-plus"></i>
          Tambah Soal</button>
      </template>
    </BasePageHeader>

    <!-- Locked Banner -->
    <div v-if="isLocked" class="alert alert-warning mb-3">
      <div class="d-flex align-items-center gap-2">
        <i class="ti ti-lock fs-3"></i>
        <div>
          <div class="fw-bold">Bank soal ini digunakan dalam jadwal ujian dan tidak dapat diubah</div>
          <div class="text-muted small">Hapus jadwal ujian terkait terlebih dahulu untuk membuka kunci.</div>
        </div>
      </div>
    </div>

    <!-- Stimuli -->
    <div v-if="stimuli.length" class="card mb-3">
      <div class="card-header d-flex align-items-center gap-2 fw-medium">
        <i class="ti ti-stack"></i>
        <span>Stimulus ({{ stimuli.length }})</span>
      </div>
      <div class="card-body d-flex flex-column gap-3">
        <div v-for="s in stimuli" :key="s.id" class="border rounded p-3 position-relative">
          <p class="mb-2 small" v-html="sanitizeHtml(s.content)" />
          <div class="d-flex gap-1 mt-2" v-if="!isLocked">
            <a href="#" class="btn btn-sm btn-ghost-secondary" @click.prevent="openEditStimulus(s)"><i class="ti ti-pencil"></i></a>
            <a href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="handleDeleteStimulus(s)"><i class="ti ti-trash"></i></a>
          </div>
        </div>
      </div>
    </div>

    <!-- Filter bar -->
    <div v-if="!reorderMode" class="card mb-3">
      <div class="card-body py-2 d-flex align-items-center gap-2 flex-wrap">
        <span class="text-muted small fw-semibold">Filter:</span>
        <select v-model="filterType" class="form-select form-select-sm" style="width:auto">
          <option value="">Semua Tipe</option>
          <option v-for="t in QUESTION_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
        <select v-model="filterDifficulty" class="form-select form-select-sm" style="width:auto">
          <option value="">Semua Kesulitan</option>
          <option value="easy">Mudah</option>
          <option value="medium">Sedang</option>
          <option value="hard">Sulit</option>
        </select>
        <span class="ms-auto text-muted small">{{ filteredQuestions.length }} soal ditampilkan</span>
      </div>
    </div>

    <!-- Reorder mode UI -->
    <div v-if="reorderMode" class="card mb-3">
      <div class="card-header d-flex align-items-center gap-2">
        <i class="ti ti-arrows-sort text-primary"></i>
        <span class="fw-semibold">Mode Atur Urutan</span>
        <span class="text-muted small ms-2">Seret soal untuk mengubah urutan</span>
        <div class="ms-auto d-flex gap-2">
          <button class="btn btn-sm btn-secondary" @click="cancelReorder">Batal</button>
          <button class="btn btn-sm btn-primary" :disabled="savingReorder" @click="saveReorder">
            <span v-if="savingReorder" class="spinner-border spinner-border-sm me-1"></span>
            Simpan Urutan
          </button>
        </div>
      </div>
      <div class="list-group list-group-flush">
        <div
          v-for="(q, idx) in reorderList"
          :key="q.id"
          class="list-group-item px-3 py-2 d-flex align-items-center gap-3"
          draggable="true"
          @dragstart="onDragStart($event, idx)"
          @dragover.prevent
          @drop="onDrop($event, idx)"
          style="cursor:grab"
        >
          <i class="ti ti-grip-vertical text-muted"></i>
          <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0">{{ idx + 1 }}</span>
          <span class="badge bg-blue-lt text-blue small">{{ qtLabel(q.question_type) }}</span>
          <div class="text-truncate flex-fill text-muted small" v-html="sanitizeHtml(q.body.substring(0, 80) + '...')" />
        </div>
      </div>
    </div>

    <!-- Questions list -->
    <div class="card">
      <div v-if="loadingQuestions" class="p-4 text-center text-muted">Memuat soal...</div>
      <div v-else-if="!questions.length" class="p-5 text-center text-muted d-flex flex-column align-items-center gap-2">
        <i class="ti ti-help-circle"></i>
        <p>Belum ada soal. Klik "Tambah Soal" untuk mulai.</p>
      </div>
      <div v-else class="list-group list-group-flush">
        <div v-for="(q, idx) in filteredQuestions" :key="q.id" class="list-group-item px-3 py-2">
          <div class="d-flex align-items-center gap-2 cursor-pointer" @click="toggleExpand(q.id)">
            <div class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ (page - 1) * 20 + idx + 1 }}</div>
            <div class="d-flex align-items-center gap-1 flex-wrap">
              <span class="badge bg-blue-lt text-blue small">{{ qtLabel(q.question_type) }}</span>
              <span class="badge" :class="`bg-${ {default:'secondary',success:'success',warning:'warning',danger:'danger',info:'info'}[diffBadge[q.difficulty] ?? 'default'] ?? 'secondary'}-lt`">{{ DIFFICULTY_LABELS[q.difficulty] }}</span>
              <span class="badge bg-secondary-lt text-secondary small ms-1">{{ q.score }} poin</span>
              <span v-if="q.audio_path" class="badge bg-cyan-lt text-cyan small ms-1"><i class="ti ti-volume"></i></span>
            </div>
            <div class="text-muted small flex-fill text-truncate" v-html="sanitizeHtml(q.body.substring(0, 120) + (q.body.length > 120 ? '...' : ''))" />
            <div class="d-flex gap-1 flex-shrink-0 ms-auto" @click.stop>
              <a v-if="!isLocked" href="#" class="btn btn-sm btn-ghost-secondary" @click.prevent="openEdit(q)"><i class="ti ti-pencil"></i></a>
              <a v-if="!isLocked" href="#" class="btn btn-sm btn-ghost-danger" @click.prevent="handleDelete(q)"><i class="ti ti-trash"></i></a>
              <a href="#" class="btn btn-sm btn-ghost-secondary" @click.prevent="toggleExpand(q.id)">
                <i :class="expandedId === q.id ? 'ti ti-chevron-up' : 'ti ti-chevron-down'"></i>
              </a>
            </div>
          </div>
          <div v-if="expandedId === q.id" class="mt-2 ps-4">
            <div class="mb-3" v-html="sanitizeHtml(q.body)" />
            <!-- Audio player -->
            <div v-if="q.audio_path" class="mb-3 d-flex align-items-center gap-2">
              <i class="ti ti-volume text-blue"></i>
              <audio controls :src="audioStreamUrl(q.audio_path)" class="flex-fill"></audio>
              <span class="badge bg-secondary-lt text-secondary small">Batas: {{ q.audio_limit === 0 ? 'Tanpa batas' : q.audio_limit + 'x' }}</span>
            </div>
            <!-- Bloom & Topic -->
            <div v-if="q.bloom_level || q.topic_code" class="d-flex gap-2 mb-2">
              <span v-if="q.bloom_level && q.bloom_level > 0" class="badge bg-purple-lt text-purple small">
                {{ BLOOM_LEVELS.find(b => b.value === q.bloom_level)?.label ?? `C${q.bloom_level}` }}
              </span>
              <span v-if="q.topic_code" class="badge bg-cyan-lt text-cyan small">
                <i class="ti ti-tag me-1"></i>{{ q.topic_code }}
              </span>
            </div>
            <div v-if="q.options" class="d-flex flex-column gap-1">
              <template v-if="q.question_type === 'pg' || q.question_type === 'pgk' || q.question_type === 'benar_salah'">
                <div v-for="opt in (q.options as any)" :key="opt.id" :class="['d-flex align-items-center gap-2 p-2 rounded small', isCorrectOption(q, opt.id) ? 'bg-green-lt text-green' : '']">
                  <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0">{{ opt.id.toUpperCase() }}</span>
                  <span>{{ opt.text }}</span>
                </div>
              </template>
              <template v-else-if="q.question_type === 'menjodohkan'">
                <div class="row g-3">
                  <div class="col">
                    <p class="fw-semibold small text-muted mb-1">Pernyataan</p>
                    <div v-for="p in (q.options as any)?.prompts" :key="p.id" class="small border rounded p-1 px-2 mb-1">{{ p.id }}. {{ p.text }}</div>
                  </div>
                  <div class="col">
                    <p class="fw-semibold small text-muted mb-1">Jawaban</p>
                    <div v-for="a in (q.options as any)?.answers" :key="a.id" class="small border rounded p-1 px-2 mb-1">{{ a.id.toUpperCase() }}. {{ a.text }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="q.question_type === 'isian_singkat'">
                <p class="text-muted small fw-medium mb-1">Jawaban yang diterima:</p>
                <div class="d-flex flex-wrap gap-1">
                  <span v-for="a in (q.options as any)?.accepted_answers" :key="a" class="badge bg-primary-lt text-primary">{{ a }}</span>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
      <BasePagination v-if="totalPages > 1" :page="page" :total-pages="totalPages" :total="total" :per-page="20" @change="p => { page = p; fetchQuestions() }" />
    </div>

    <!-- Question Modal -->
    <BaseModal v-if="showModal" :title="isEdit ? 'Edit Soal' : 'Tambah Soal'" size="lg" @close="showModal = false">
      <form @submit.prevent="handleSave">
        <!-- Type & config row -->
        <div class="row g-3">
          <div class="mb-3">
            <label class="form-label">Tipe Soal *</label>
            <select v-model="form.question_type" class="form-select" :class="{ 'is-invalid': formErrors.question_type }" :disabled="isEdit" @blur="validateQuestionField('question_type')" @change="formErrors.question_type = ''">
              <option v-for="t in QUESTION_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
            <div v-if="formErrors.question_type" class="invalid-feedback">{{ formErrors.question_type }}</div>
          </div>
          <div class="mb-3">
            <label class="form-label">Bobot Nilai</label>
            <input v-model.number="form.score" type="number" min="0" step="0.5" class="form-control" :class="{ 'is-invalid': formErrors.score }" @blur="validateQuestionField('score')" @input="formErrors.score = ''" />
            <div v-if="formErrors.score" class="invalid-feedback">{{ formErrors.score }}</div>
          </div>
          <div class="mb-3">
            <label class="form-label">Tingkat Kesulitan</label>
            <select v-model="form.difficulty" class="form-select">
              <option value="easy">Mudah</option>
              <option value="medium">Sedang</option>
              <option value="hard">Sulit</option>
            </select>
          </div>
          <!-- Bloom Level -->
          <div class="mb-3">
            <label class="form-label">Level Bloom (Taksonomi)</label>
            <select v-model.number="form.bloom_level" class="form-select">
              <option v-for="b in BLOOM_LEVELS" :key="b.value" :value="b.value">{{ b.label }}</option>
            </select>
          </div>
          <!-- Topic/KD Code -->
          <div class="mb-3">
            <label class="form-label">Kode Topik / KD <span class="text-muted">(opsional)</span></label>
            <input v-model="form.topic_code" class="form-control" placeholder="Contoh: 3.5.1 atau MATEMATIKA-KD3" />
          </div>
        </div>

        <!-- Stimulus selector -->
        <div class="mb-3" v-if="stimuli.length">
          <label class="form-label">Stimulus (opsional)</label>
          <select v-model="form.stimulus_id" class="form-select">
            <option value="">Tanpa stimulus</option>
            <option v-for="s in stimuli" :key="s.id" :value="s.id">Stimulus #{{ s.id }}</option>
          </select>
        </div>

        <!-- Body -->
        <div class="mb-3">
          <label class="form-label">Soal / Pertanyaan *</label>
          <textarea v-model="form.body" class="form-control" :class="{ 'is-invalid': formErrors.body }" rows="4" placeholder="Tulis pertanyaan di sini..." required @blur="validateQuestionField('body')" @input="formErrors.body = ''" />
          <div v-if="formErrors.body" class="invalid-feedback">{{ formErrors.body }}</div>
        </div>

        <!-- PG options -->
        <template v-if="form.question_type === 'pg' || form.question_type === 'pgk'">
          <div class="mb-3">
            <label class="form-label">Pilihan Jawaban</label>
            <div class="d-flex flex-column gap-2">
              <div v-for="(opt, i) in form.options" :key="opt.id" class="d-flex align-items-center gap-2">
                <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ opt.id.toUpperCase() }}</span>
                <input v-model="opt.text" class="form-control form-control-sm flex-fill" :placeholder="`Opsi ${opt.id.toUpperCase()}`" />
                <input v-if="form.question_type === 'pgk'" v-model.number="opt.score" type="number" min="0" step="0.25" class="form-control form-control-sm" style="width:70px" title="Bobot opsi ini" />
                <button type="button" class="btn btn-sm btn-ghost-danger flex-shrink-0" @click="removeOption(i)" v-if="form.options.length > 2">×</button>
              </div>
            </div>
            <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addOption" v-if="form.options.length < 8">+ Tambah Opsi</button>
          </div>
          <!-- Correct answer for PG -->
          <div class="mb-3" v-if="form.question_type === 'pg'">
            <label class="form-label">Jawaban Benar</label>
            <select v-model="form.correct_single" class="form-select">
              <option v-for="opt in form.options" :key="opt.id" :value="opt.id">{{ opt.id.toUpperCase() }} — {{ opt.text || '(kosong)' }}</option>
            </select>
          </div>
          <!-- Correct answer for PGK -->
          <div class="mb-3" v-else>
            <label class="form-label">Jawaban Benar (boleh lebih dari satu)</label>
            <div class="d-flex flex-column gap-1">
              <label v-for="opt in form.options" :key="opt.id" class="form-check d-flex align-items-center gap-2">
                <input type="checkbox" :value="opt.id" v-model="form.correct_multi" />
                {{ opt.id.toUpperCase() }} — {{ opt.text || '(kosong)' }}
              </label>
            </div>
          </div>
        </template>

        <!-- Benar / Salah -->
        <template v-else-if="form.question_type === 'benar_salah'">
          <div class="mb-3">
            <label class="form-label">Jawaban Benar</label>
            <div class="d-flex flex-column gap-1">
              <label class="form-check d-flex align-items-center gap-2"><input type="radio" value="true" v-model="form.correct_single" /> Benar</label>
              <label class="form-check d-flex align-items-center gap-2"><input type="radio" value="false" v-model="form.correct_single" /> Salah</label>
            </div>
          </div>
        </template>

        <!-- Menjodohkan -->
        <template v-else-if="form.question_type === 'menjodohkan'">
          <div class="border rounded p-3 mb-3">
            <div class="d-flex flex-column gap-2">
              <p class="form-label">Pernyataan (Kiri)</p>
              <div v-for="(p, i) in form.match_prompts" :key="i" class="d-flex align-items-center gap-2 mb-2">
                <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ p.id }}</span>
                <input v-model="p.text" class="form-control" :placeholder="`Pernyataan ${p.id}`" />
              </div>
            </div>
            <div class="d-flex flex-column gap-2">
              <p class="form-label">Pasangan (Kanan)</p>
              <div v-for="(a, i) in form.match_answers" :key="i" class="d-flex align-items-center gap-2 mb-2">
                <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ a.id.toUpperCase() }}</span>
                <input v-model="a.text" class="form-control" :placeholder="`Jawaban ${a.id.toUpperCase()}`" />
              </div>
            </div>
          </div>
          <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addMatchRow">+ Tambah Pasangan</button>
          <div class="mb-3">
            <p class="form-label">Kunci Jawaban (Pernyataan → Pasangan)</p>
            <div v-for="p in form.match_prompts" :key="p.id" class="d-flex align-items-center gap-2 mb-2">
              <span>{{ p.id }}.</span>
              <select v-model="form.match_pairs[p.id]" class="form-select form-select-sm">
                <option value="">Pilih pasangan</option>
                <option v-for="a in form.match_answers" :key="a.id" :value="a.id">{{ a.id.toUpperCase() }}</option>
              </select>
            </div>
          </div>
        </template>

        <!-- Isian Singkat -->
        <template v-else-if="form.question_type === 'isian_singkat'">
          <div class="mb-3">
            <label class="form-label">Jawaban yang Diterima</label>
            <div v-for="(_, i) in form.accepted_answers" :key="i" class="d-flex align-items-center gap-2 mb-2">
              <input v-model="form.accepted_answers[i]" class="form-control" :placeholder="`Jawaban ${i + 1}`" />
              <button type="button" class="btn btn-sm btn-ghost-danger flex-shrink-0" @click="removeAcceptedAnswer(i)" v-if="form.accepted_answers.length > 1">×</button>
            </div>
            <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addAcceptedAnswer">+ Tambah Variasi Jawaban</button>
          </div>
        </template>

        <!-- Matrix -->
        <template v-else-if="form.question_type === 'matrix'">
          <div class="mb-3">
            <label class="form-label">Baris</label>
            <div v-for="(r, i) in form.matrix_rows" :key="i" class="d-flex align-items-center gap-2">
              <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ r.id }}</span>
              <input v-model="r.text" class="form-control form-control-sm flex-fill" :placeholder="`Baris ${r.id}`" />
            </div>
            <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addMatrixRow">+ Tambah Baris</button>
          </div>
          <div class="mb-3">
            <label class="form-label">Kolom</label>
            <div v-for="(c, i) in form.matrix_cols" :key="i" class="d-flex align-items-center gap-2">
              <span class="badge bg-secondary-lt text-secondary fw-bold flex-shrink-0 text-center">{{ c.id.toUpperCase() }}</span>
              <input v-model="c.text" class="form-control form-control-sm flex-fill" :placeholder="`Kolom ${c.id.toUpperCase()}`" />
            </div>
            <button type="button" class="btn btn-sm btn-outline-secondary mt-1" @click="addMatrixCol">+ Tambah Kolom</button>
          </div>
          <div class="mb-3">
            <label class="form-label">Kunci Jawaban (per baris)</label>
            <div v-for="r in form.matrix_rows" :key="r.id" class="d-flex align-items-center gap-2 mb-2">
              <span>{{ r.id }}.</span>
              <select v-model="form.matrix_correct[r.id]" class="form-select form-select-sm">
                <option value="">Pilih kolom benar</option>
                <option v-for="c in form.matrix_cols" :key="c.id" :value="c.id">{{ c.id.toUpperCase() }} — {{ c.text }}</option>
              </select>
            </div>
          </div>
        </template>

        <!-- Esai: no options needed -->
        <template v-else-if="form.question_type === 'esai'">
          <div class="alert alert-info alert-sm">Soal esai tidak memiliki opsi jawaban. Penilaian dilakukan secara manual atau via AI.</div>
        </template>

        <!-- Audio Section -->
        <div class="border rounded p-3 mb-3 mt-3">
          <label class="form-label fw-semibold d-flex align-items-center gap-2">
            <i class="ti ti-volume"></i> Audio Soal (opsional)
          </label>

          <!-- Existing audio preview -->
          <div v-if="existingAudioPath && !removeAudio" class="mb-2">
            <audio controls class="w-100 mb-2" :src="audioStreamUrl(existingAudioPath)"></audio>
            <button type="button" class="btn btn-sm btn-outline-danger" @click="handleRemoveAudio">
              <i class="ti ti-trash me-1"></i>Hapus Audio
            </button>
          </div>

          <!-- New audio preview -->
          <div v-if="audioFile" class="mb-2">
            <p class="text-muted small mb-1">File baru: {{ audioFile.name }}</p>
          </div>

          <!-- Upload input -->
          <div class="mb-2">
            <input type="file" accept=".mp3,.wav,.m4a,.ogg,.aac" class="form-control form-control-sm" @change="onAudioFileChange" />
            <small class="text-muted">Format: mp3, wav, m4a, ogg, aac. Maks 10MB.</small>
          </div>

          <!-- Playback limit -->
          <div class="d-flex align-items-center gap-2">
            <label class="form-label mb-0 small">Batas putar:</label>
            <input v-model.number="form.audio_limit" type="number" min="0" class="form-control form-control-sm" style="width:80px" />
            <small class="text-muted">0 = tanpa batas</small>
          </div>
        </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="showModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="saving" @click="handleSave"><span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>Simpan Soal</button>
      </template>
    </BaseModal>

    <!-- Stimulus Modal -->
    <BaseModal v-if="showStimulusModal" title="Stimulus / Wacana" @close="showStimulusModal = false">
      <form @submit.prevent="handleSaveStimulus">
        <div class="mb-3">
          <label class="form-label">Isi Stimulus *</label>
          <textarea v-model="stimulusContent" class="form-control" rows="6" placeholder="Teks bacaan, grafik deskripsi, atau materi stimulus..." required />
        </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="showStimulusModal = false">Batal</button>
        <button class="btn btn-primary" :disabled="savingStimulus" @click="handleSaveStimulus"><span v-if="savingStimulus" class="spinner-border spinner-border-sm me-1"></span>Simpan</button>
      </template>
    </BaseModal>

    <BaseConfirmModal
      v-if="confirmModal"
      :title="confirmTitle"
      :message="confirmMessage"
      @confirm="doConfirm"
      @close="confirmModal = false"
      :loading="confirmLoading"
    />
</template>


