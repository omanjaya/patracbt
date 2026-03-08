<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { examApi, type ExamSchedule, STATUS_LABELS, STATUS_COLORS } from '../../../api/exam.api'
import { sanitizeHtml } from '../../../composables/useSafeHtml'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const errorMsg = ref('')
const schedule = ref<ExamSchedule | null>(null)

interface PreviewQuestion {
  id: number
  question_type: string
  body: string
  score: number
  difficulty: string
  options: any
  correct_answer: any
  order_index: number
}

interface PreviewBank {
  question_bank_id: number
  question_bank_name: string
  total_questions: number
  question_count: number
  weight: number
  sample_questions: PreviewQuestion[]
}

const banks = ref<PreviewBank[]>([])

const QUESTION_TYPE_LABELS: Record<string, string> = {
  pg: 'Pilihan Ganda',
  pgk: 'Pilihan Ganda Kompleks',
  esai: 'Esai',
  menjodohkan: 'Menjodohkan',
  isian_singkat: 'Isian Singkat',
  matrix: 'Matrix',
  benar_salah: 'Benar/Salah',
}

const DIFFICULTY_LABELS: Record<string, string> = {
  easy: 'Mudah',
  medium: 'Sedang',
  hard: 'Sulit',
}

const DIFFICULTY_COLORS: Record<string, string> = {
  easy: 'green',
  medium: 'yellow',
  hard: 'red',
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('id-ID', {
    day: 'numeric', month: 'long', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function parseOptions(options: any): string[] {
  if (!options) return []
  if (Array.isArray(options)) return options.map(o => typeof o === 'string' ? o : (o.text ?? o.label ?? JSON.stringify(o)))
  if (typeof options === 'string') {
    try { return parseOptions(JSON.parse(options)) } catch { return [] }
  }
  return []
}

function parseCorrectAnswer(answer: any): any {
  if (answer === null || answer === undefined) return null
  if (typeof answer === 'string') {
    try { return JSON.parse(answer) } catch { return answer }
  }
  return answer
}

function isCorrectOption(question: PreviewQuestion, index: number): boolean {
  const correct = parseCorrectAnswer(question.correct_answer)
  if (question.question_type === 'pg') {
    return correct === index || correct === String(index)
  }
  if (question.question_type === 'pgk') {
    if (Array.isArray(correct)) return correct.includes(index) || correct.includes(String(index))
  }
  return false
}

function getMatchingPairs(question: PreviewQuestion): { left: string; right: string }[] {
  const options = question.options
  const correct = parseCorrectAnswer(question.correct_answer)
  if (!options) return []

  let parsed = options
  if (typeof parsed === 'string') {
    try { parsed = JSON.parse(parsed) } catch { return [] }
  }

  if (parsed.left && parsed.right) {
    const left = Array.isArray(parsed.left) ? parsed.left : []
    const right = Array.isArray(parsed.right) ? parsed.right : []
    return left.map((l: string, i: number) => ({
      left: l,
      right: correct && correct[i] !== undefined ? right[correct[i]] ?? '-' : right[i] ?? '-',
    }))
  }
  return []
}

function getMatrixData(question: PreviewQuestion): { rows: string[]; columns: string[] } {
  const options = question.options
  if (!options) return { rows: [], columns: [] }

  let parsed = options
  if (typeof parsed === 'string') {
    try { parsed = JSON.parse(parsed) } catch { return { rows: [], columns: [] } }
  }

  return {
    rows: Array.isArray(parsed.rows) ? parsed.rows : [],
    columns: Array.isArray(parsed.columns) ? parsed.columns : [],
  }
}

function getMatrixCorrect(question: PreviewQuestion): Record<number, number> {
  const correct = parseCorrectAnswer(question.correct_answer)
  if (!correct || typeof correct !== 'object') return {}
  const result: Record<number, number> = {}
  for (const [k, v] of Object.entries(correct)) {
    result[Number(k)] = Number(v)
  }
  return result
}

function goBack() {
  router.back()
}

onMounted(async () => {
  const id = Number(route.params.id)
  if (!id) {
    errorMsg.value = 'ID jadwal tidak valid'
    loading.value = false
    return
  }
  try {
    const res = await examApi.previewSchedule(id)
    schedule.value = res.data.data.schedule
    banks.value = res.data.data.banks ?? []
  } catch (e: any) {
    errorMsg.value = e.response?.data?.message ?? 'Gagal memuat data preview'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <!-- Page Header -->
    <BasePageHeader
      title="Preview Soal Ujian"
      subtitle="Tampilan soal seperti yang akan dilihat peserta"
      :breadcrumbs="[{ label: 'Jadwal Ujian', to: '/admin/exam-schedules' }, { label: 'Preview' }]"
    >
      <template #actions>
        <button class="btn btn-outline-secondary" @click="goBack">
          <i class="ti ti-arrow-bar-left me-1"></i>
          Kembali
        </button>
      </template>
    </BasePageHeader>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" role="status"></div>
      <p class="text-muted mt-2">Memuat preview...</p>
    </div>

    <!-- Error -->
    <div v-else-if="errorMsg" class="alert alert-danger">
      <i class="ti ti-alert-circle me-2"></i>{{ errorMsg }}
    </div>

    <!-- Content -->
    <template v-else-if="schedule">
      <!-- Schedule Info Header -->
      <div class="card mb-4">
        <div class="card-body">
          <div class="row g-3">
            <div class="col-lg-6">
              <h3 class="mb-1">{{ schedule.name }}</h3>
              <span class="badge mb-2" :class="`bg-${STATUS_COLORS[schedule.status] === 'default' ? 'secondary' : STATUS_COLORS[schedule.status]}-lt`">
                {{ STATUS_LABELS[schedule.status] }}
              </span>
            </div>
            <div class="col-lg-6">
              <div class="row g-2 text-muted small">
                <div class="col-sm-6">
                  <div class="d-flex align-items-center gap-2 mb-1">
                    <i class="ti ti-calendar"></i>
                    <span>{{ formatDate(schedule.start_time) }}</span>
                  </div>
                  <div class="d-flex align-items-center gap-2">
                    <i class="ti ti-calendar-off"></i>
                    <span>{{ formatDate(schedule.end_time) }}</span>
                  </div>
                </div>
                <div class="col-sm-6">
                  <div class="d-flex align-items-center gap-2 mb-1">
                    <i class="ti ti-clock"></i>
                    <span>{{ schedule.duration_minutes }} menit</span>
                  </div>
                  <div class="d-flex flex-wrap gap-2">
                    <span v-if="schedule.randomize_questions" class="badge bg-blue-lt">Acak Soal</span>
                    <span v-if="schedule.randomize_options" class="badge bg-blue-lt">Acak Opsi</span>
                    <span v-if="schedule.detect_cheating" class="badge bg-orange-lt">Deteksi Kecurangan</span>
                    <span v-if="schedule.allow_see_result" class="badge bg-green-lt">Lihat Hasil</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- No banks -->
      <div v-if="banks.length === 0" class="card">
        <div class="empty">
          <div class="empty-icon">
            <i class="ti ti-file-off" style="font-size: 3rem;"></i>
          </div>
          <p class="empty-title">Tidak ada soal untuk ditampilkan</p>
          <p class="empty-subtitle text-muted">Belum ada bank soal yang ditambahkan ke jadwal ini.</p>
        </div>
      </div>

      <!-- Question Banks -->
      <div v-for="(bank, bankIdx) in banks" :key="bank.question_bank_id" class="card mb-4">
        <div class="card-header">
          <div class="d-flex align-items-center justify-content-between w-100">
            <div>
              <h3 class="card-title mb-0">
                <i class="ti ti-books me-2"></i>
                {{ bank.question_bank_name || `Bank Soal #${bank.question_bank_id}` }}
              </h3>
              <p class="text-muted small mb-0 mt-1">
                {{ bank.total_questions }} soal total
                <span v-if="bank.question_count > 0"> &middot; {{ bank.question_count }} soal dipilih</span>
                <span v-if="bank.weight !== 1"> &middot; Bobot: {{ bank.weight }}x</span>
              </p>
            </div>
            <span class="badge bg-primary-lt">Bagian {{ bankIdx + 1 }}</span>
          </div>
        </div>
        <div class="card-body">
          <!-- Sample Questions -->
          <div v-for="(q, qIdx) in bank.sample_questions" :key="q.id" class="border rounded p-3 mb-3">
            <!-- Question Header -->
            <div class="d-flex align-items-start justify-content-between mb-2">
              <div class="d-flex align-items-center gap-2">
                <span class="badge bg-dark-lt fw-bold">{{ qIdx + 1 }}</span>
                <span class="badge bg-azure-lt">{{ QUESTION_TYPE_LABELS[q.question_type] ?? q.question_type }}</span>
                <span class="badge" :class="`bg-${DIFFICULTY_COLORS[q.difficulty] ?? 'secondary'}-lt`">
                  {{ DIFFICULTY_LABELS[q.difficulty] ?? q.difficulty }}
                </span>
              </div>
              <span class="text-muted small">{{ q.score }} poin</span>
            </div>

            <!-- Question Body -->
            <div class="mb-3" v-html="sanitizeHtml(q.body)"></div>

            <!-- PG: Radio buttons -->
            <template v-if="q.question_type === 'pg'">
              <div v-for="(opt, optIdx) in parseOptions(q.options)" :key="optIdx" class="d-flex align-items-start gap-2 mb-2 p-2 rounded" :class="{ 'bg-green-lt border border-green': isCorrectOption(q, optIdx) }">
                <input type="radio" class="form-check-input mt-1" disabled :checked="isCorrectOption(q, optIdx)" />
                <div>
                  <span v-html="sanitizeHtml(opt)"></span>
                  <i v-if="isCorrectOption(q, optIdx)" class="ti ti-circle-check text-green ms-1"></i>
                </div>
              </div>
            </template>

            <!-- PGK: Checkboxes -->
            <template v-else-if="q.question_type === 'pgk'">
              <div v-for="(opt, optIdx) in parseOptions(q.options)" :key="optIdx" class="d-flex align-items-start gap-2 mb-2 p-2 rounded" :class="{ 'bg-green-lt border border-green': isCorrectOption(q, optIdx) }">
                <input type="checkbox" class="form-check-input mt-1" disabled :checked="isCorrectOption(q, optIdx)" />
                <div>
                  <span v-html="sanitizeHtml(opt)"></span>
                  <i v-if="isCorrectOption(q, optIdx)" class="ti ti-circle-check text-green ms-1"></i>
                </div>
              </div>
            </template>

            <!-- Esai -->
            <template v-else-if="q.question_type === 'esai'">
              <textarea class="form-control" rows="4" disabled placeholder="Peserta mengetik jawaban esai di sini..."></textarea>
              <div v-if="parseCorrectAnswer(q.correct_answer)" class="mt-2 p-2 bg-green-lt rounded small">
                <strong class="text-green"><i class="ti ti-circle-check me-1"></i>Kunci Jawaban:</strong>
                <div class="mt-1" v-html="sanitizeHtml(typeof parseCorrectAnswer(q.correct_answer) === 'string' ? parseCorrectAnswer(q.correct_answer) : JSON.stringify(parseCorrectAnswer(q.correct_answer)))"></div>
              </div>
            </template>

            <!-- Menjodohkan -->
            <template v-else-if="q.question_type === 'menjodohkan'">
              <div class="table-responsive">
                <table class="table table-vcenter table-bordered">
                  <thead>
                    <tr>
                      <th style="width:50%">Pernyataan</th>
                      <th style="width:50%">Jawaban (Kunci)</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(pair, pIdx) in getMatchingPairs(q)" :key="pIdx">
                      <td v-html="sanitizeHtml(pair.left)"></td>
                      <td>
                        <span class="badge bg-green-lt">
                          <i class="ti ti-circle-check me-1"></i>
                          <span v-html="sanitizeHtml(pair.right)"></span>
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>

            <!-- Isian Singkat -->
            <template v-else-if="q.question_type === 'isian_singkat'">
              <input type="text" class="form-control" disabled placeholder="Peserta mengetik jawaban singkat di sini..." />
              <div v-if="parseCorrectAnswer(q.correct_answer)" class="mt-2 p-2 bg-green-lt rounded small">
                <strong class="text-green"><i class="ti ti-circle-check me-1"></i>Kunci Jawaban:</strong>
                <span class="ms-1">{{ typeof parseCorrectAnswer(q.correct_answer) === 'string' ? parseCorrectAnswer(q.correct_answer) : JSON.stringify(parseCorrectAnswer(q.correct_answer)) }}</span>
              </div>
            </template>

            <!-- Matrix -->
            <template v-else-if="q.question_type === 'matrix'">
              <div class="table-responsive">
                <table class="table table-vcenter table-bordered">
                  <thead>
                    <tr>
                      <th></th>
                      <th v-for="(col, cIdx) in getMatrixData(q).columns" :key="cIdx" class="text-center">
                        <span v-html="sanitizeHtml(col)"></span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, rIdx) in getMatrixData(q).rows" :key="rIdx">
                      <td v-html="sanitizeHtml(row)"></td>
                      <td v-for="(_, cIdx) in getMatrixData(q).columns" :key="cIdx" class="text-center">
                        <input
                          type="radio"
                          class="form-check-input"
                          disabled
                          :checked="getMatrixCorrect(q)[rIdx] === cIdx"
                        />
                        <i v-if="getMatrixCorrect(q)[rIdx] === cIdx" class="ti ti-circle-check text-green ms-1"></i>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>

            <!-- Benar/Salah -->
            <template v-else-if="q.question_type === 'benar_salah'">
              <div class="d-flex gap-3">
                <div class="d-flex align-items-center gap-2 p-2 rounded" :class="{ 'bg-green-lt border border-green': parseCorrectAnswer(q.correct_answer) === true || parseCorrectAnswer(q.correct_answer) === 'true' }">
                  <input type="radio" class="form-check-input" disabled :checked="parseCorrectAnswer(q.correct_answer) === true || parseCorrectAnswer(q.correct_answer) === 'true'" />
                  <span>Benar</span>
                  <i v-if="parseCorrectAnswer(q.correct_answer) === true || parseCorrectAnswer(q.correct_answer) === 'true'" class="ti ti-circle-check text-green"></i>
                </div>
                <div class="d-flex align-items-center gap-2 p-2 rounded" :class="{ 'bg-green-lt border border-green': parseCorrectAnswer(q.correct_answer) === false || parseCorrectAnswer(q.correct_answer) === 'false' }">
                  <input type="radio" class="form-check-input" disabled :checked="parseCorrectAnswer(q.correct_answer) === false || parseCorrectAnswer(q.correct_answer) === 'false'" />
                  <span>Salah</span>
                  <i v-if="parseCorrectAnswer(q.correct_answer) === false || parseCorrectAnswer(q.correct_answer) === 'false'" class="ti ti-circle-check text-green"></i>
                </div>
              </div>
            </template>
          </div>

          <!-- More questions indicator -->
          <div v-if="bank.total_questions > bank.sample_questions.length" class="text-center text-muted py-2">
            <i class="ti ti-dots me-1"></i>
            ... dan {{ bank.total_questions - bank.sample_questions.length }} soal lainnya
          </div>

          <!-- No questions -->
          <div v-if="bank.sample_questions.length === 0" class="text-center text-muted py-3">
            <i class="ti ti-file-off fs-3 d-block mb-1"></i>
            Belum ada soal di bank soal ini.
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
