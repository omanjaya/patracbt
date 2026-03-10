<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { reportApi, type PersonalReport, type AnswerDetail } from '../../../api/report.api'
import { sanitizeHtml } from '@/composables/useSafeHtml'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()
const sessionId = Number(route.params.sessionId)

const report = ref<PersonalReport | null>(null)
const loading = ref(true)
const error = ref('')

// Which question is scrolled-to / highlighted
const activeIndex = ref<number | null>(null)

async function fetchReport() {
  loading.value = true
  try {
    const res = await reportApi.getMyReport(sessionId)
    report.value = res.data.data
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat pembahasan ujian.'
  } finally {
    loading.value = false
  }
}

// ── Computed helpers ──────────────────────────────────────────
const session = computed(() => report.value?.session)
const answers = computed(() => report.value?.answers ?? [])

const totalQuestions = computed(() => answers.value.length)
const answeredCount = computed(() => answers.value.filter(a => a.user_answer != null && a.user_answer !== '').length)
const correctCount = computed(() => answers.value.filter(a => a.is_correct).length)
const wrongCount = computed(() => answeredCount.value - correctCount.value)

const scorePercent = computed(() => {
  if (!session.value || !session.value.max_score) return 0
  return Math.round((session.value.score / session.value.max_score) * 100)
})

const totalEarned = computed(() => answers.value.reduce((sum, a) => sum + (a.earned_score ?? 0), 0))
const totalMax = computed(() => answers.value.reduce((sum, a) => sum + a.score, 0))

const gradeLabel = computed(() => {
  const p = scorePercent.value
  if (p >= 90) return { label: 'Sangat Baik', color: 'green' }
  if (p >= 75) return { label: 'Baik', color: 'blue' }
  if (p >= 60) return { label: 'Cukup', color: 'yellow' }
  return { label: 'Perlu Perbaikan', color: 'red' }
})

// ── Formatting helpers ───────────────────────────────────────
function formatDate(d: string | null) {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function duration() {
  if (!session.value?.start_time || !session.value?.finished_at) return '-'
  const diff = Math.floor((new Date(session.value.finished_at).getTime() - new Date(session.value.start_time).getTime()) / 1000)
  const m = Math.floor(diff / 60)
  const s = diff % 60
  return `${m} menit ${s} detik`
}

function typeLabel(t: string) {
  const map: Record<string, string> = {
    pg: 'Pilihan Ganda',
    pgk: 'PG Kompleks',
    esai: 'Esai',
    isian: 'Isian',
    menjodohkan: 'Menjodohkan',
    benar_salah: 'Benar/Salah',
  }
  return map[t] ?? t
}

function typeBadgeClass(t: string) {
  const map: Record<string, string> = {
    pg: 'bg-blue-lt text-blue',
    pgk: 'bg-indigo-lt text-indigo',
    esai: 'bg-purple-lt text-purple',
    isian: 'bg-cyan-lt text-cyan',
    menjodohkan: 'bg-orange-lt text-orange',
    benar_salah: 'bg-teal-lt text-teal',
  }
  return map[t] ?? 'bg-secondary-lt text-secondary'
}

// ── Answer rendering helpers ─────────────────────────────────
function parsePGOptions(ans: AnswerDetail): { key: string; text: string }[] {
  try {
    if (Array.isArray(ans.options)) return ans.options
    if (typeof ans.options === 'object' && ans.options !== null) {
      return Object.entries(ans.options as Record<string, string>).map(([k, v]) => ({ key: k, text: v }))
    }
  } catch (e) {
    console.warn('Failed to parse PG options:', e)
  }
  return []
}

function isOptionSelected(ans: AnswerDetail, optionKey: string): boolean {
  if (ans.question_type === 'pgk') {
    // PGK: user_answer is array of keys
    if (Array.isArray(ans.user_answer)) return ans.user_answer.includes(optionKey)
    return false
  }
  // PG: user_answer is single key string
  return ans.user_answer === optionKey
}

function isOptionCorrect(ans: AnswerDetail, optionKey: string): boolean {
  if (ans.question_type === 'pgk') {
    if (Array.isArray(ans.correct_answer)) return ans.correct_answer.includes(optionKey)
    return false
  }
  return ans.correct_answer === optionKey
}

function parseMatchingPairs(ans: AnswerDetail): { left: string; right: string; userRight: string; isCorrect: boolean }[] {
  try {
    const correct = ans.correct_answer as Record<string, string> | null
    const user = ans.user_answer as Record<string, string> | null
    if (!correct) return []
    return Object.entries(correct).map(([left, right]) => ({
      left,
      right,
      userRight: user?.[left] ?? '-',
      isCorrect: user?.[left] === right,
    }))
  } catch (e) {
    console.warn('Failed to parse matching pairs:', e)
    return []
  }
}

function parseIsianAccepted(ans: AnswerDetail): string[] {
  try {
    if (Array.isArray(ans.correct_answer)) return ans.correct_answer as string[]
    if (typeof ans.correct_answer === 'string') return [ans.correct_answer]
  } catch (e) {
    console.warn('Failed to parse isian accepted answers:', e)
  }
  return []
}

function displayUserAnswer(ans: AnswerDetail): string {
  if (ans.user_answer == null || ans.user_answer === '') return '(Tidak dijawab)'
  if (typeof ans.user_answer === 'string') return ans.user_answer
  return JSON.stringify(ans.user_answer)
}

function scrollToQuestion(idx: number) {
  activeIndex.value = idx
  const el = document.getElementById(`question-${idx}`)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(fetchReport)
</script>

<template>
  <!-- Loading -->
  <div v-if="loading" class="p-5 text-center text-muted">
    <span class="spinner-border spinner-border-sm me-2"></span>Memuat pembahasan...
  </div>

  <!-- Error -->
  <div v-else-if="error" class="alert alert-danger d-flex align-items-center gap-2">
    <i class="ti ti-alert-circle"></i>
    <div>
      {{ error }}
      <div class="mt-2">
        <button class="btn btn-sm btn-outline-danger" @click="router.push('/peserta')">Kembali ke Dashboard</button>
      </div>
    </div>
  </div>

  <!-- Report content -->
  <div v-else-if="report" class="pb-4">
    <!-- Header -->
    <BasePageHeader
      title="Pembahasan Ujian"
      :subtitle="session?.exam_schedule?.name ?? `Sesi #${sessionId}`"
      :breadcrumbs="[{ label: 'Laporan Pribadi' }]"
    >
      <template #actions>
        <button class="btn btn-icon btn-ghost-secondary btn-sm" @click="router.push('/peserta')" title="Kembali">
          <i class="ti ti-arrow-left"></i>
        </button>
        <span class="badge fs-5" :class="`bg-${gradeLabel.color}-lt text-${gradeLabel.color}`">
          {{ scorePercent }}% &mdash; {{ gradeLabel.label }}
        </span>
      </template>
    </BasePageHeader>

    <!-- Summary Cards -->
    <div class="row g-3 mb-4">
      <div class="col-6 col-sm-4 col-lg">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center gap-2">
              <span class="avatar avatar-sm bg-primary-lt"><i class="ti ti-list-numbers text-primary"></i></span>
              <div>
                <div class="text-muted small">Total Soal</div>
                <div class="fw-bold">{{ totalQuestions }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center gap-2">
              <span class="avatar avatar-sm bg-azure-lt"><i class="ti ti-edit text-azure"></i></span>
              <div>
                <div class="text-muted small">Dijawab</div>
                <div class="fw-bold">{{ answeredCount }} / {{ totalQuestions }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center gap-2">
              <span class="avatar avatar-sm bg-green-lt"><i class="ti ti-circle-check text-green"></i></span>
              <div>
                <div class="text-muted small">Benar</div>
                <div class="fw-bold text-green">{{ correctCount }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center gap-2">
              <span class="avatar avatar-sm bg-red-lt"><i class="ti ti-circle-x text-red"></i></span>
              <div>
                <div class="text-muted small">Salah</div>
                <div class="fw-bold text-red">{{ wrongCount }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-6 col-sm-4 col-lg">
        <div class="card card-sm">
          <div class="card-body">
            <div class="d-flex align-items-center gap-2">
              <span class="avatar avatar-sm bg-yellow-lt"><i class="ti ti-star text-yellow"></i></span>
              <div>
                <div class="text-muted small">Skor</div>
                <div class="fw-bold">{{ totalEarned.toFixed(1) }} / {{ totalMax.toFixed(1) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Exam info row -->
    <div class="card mb-4">
      <div class="card-body py-2">
        <div class="row g-2 text-muted small">
          <div class="col-sm-4 d-flex align-items-center gap-2">
            <i class="ti ti-calendar"></i>
            <span>Dikerjakan: {{ formatDate(session?.start_time) }}</span>
          </div>
          <div class="col-sm-4 d-flex align-items-center gap-2">
            <i class="ti ti-clock"></i>
            <span>Durasi: {{ duration() }}</span>
          </div>
          <div class="col-sm-4 d-flex align-items-center gap-2">
            <i class="ti ti-circle-check"></i>
            <span>Selesai: {{ formatDate(session?.finished_at) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Question Navigator (horizontal scroll) -->
    <div class="card mb-4">
      <div class="card-body py-2">
        <div class="d-flex align-items-center gap-2 mb-2">
          <i class="ti ti-map-pin text-muted"></i>
          <span class="text-muted small fw-medium">Navigasi Soal</span>
        </div>
        <div class="d-flex flex-wrap gap-1">
          <button
            v-for="(ans, idx) in answers"
            :key="ans.question_id"
            class="btn btn-sm btn-icon"
            :class="[
              ans.is_correct ? 'btn-success' : (ans.user_answer != null && ans.user_answer !== '' ? 'btn-danger' : 'btn-outline-secondary'),
              activeIndex === idx ? 'ring ring-primary' : '',
            ]"
            style="width: 36px; height: 36px; font-size: 0.75rem; position: relative;"
            @click="scrollToQuestion(idx)"
            :title="`Soal ${idx + 1} - ${ans.is_correct ? 'Benar' : 'Salah'}`"
          >
            {{ idx + 1 }}
            <span
              v-if="ans.is_flagged"
              class="position-absolute top-0 end-0 translate-middle-y"
              style="font-size: 0.6rem;"
            >
              <i class="ti ti-flag-filled text-warning"></i>
            </span>
          </button>
        </div>
      </div>
    </div>

    <!-- Question List -->
    <div class="d-flex flex-column gap-3">
      <div
        v-for="(ans, idx) in answers"
        :key="ans.question_id"
        :id="`question-${idx}`"
        class="card"
        :class="{ 'border-success': ans.is_correct, 'border-danger': !ans.is_correct && ans.user_answer != null && ans.user_answer !== '' }"
      >
        <!-- Question Header -->
        <div class="card-header">
          <div class="d-flex align-items-center gap-2 flex-grow-1">
            <span
              class="avatar avatar-sm rounded"
              :class="ans.is_correct ? 'bg-green-lt' : (ans.user_answer != null && ans.user_answer !== '' ? 'bg-red-lt' : 'bg-secondary-lt')"
            >
              <span class="fw-bold" style="font-size: 0.75rem;">{{ idx + 1 }}</span>
            </span>
            <span class="badge" :class="typeBadgeClass(ans.question_type)">
              {{ typeLabel(ans.question_type) }}
            </span>
            <span v-if="ans.is_flagged" class="badge bg-warning-lt text-warning">
              <i class="ti ti-flag-filled me-1"></i>Ditandai
            </span>
          </div>
          <div class="card-actions">
            <span class="fw-medium" :class="ans.is_correct ? 'text-green' : 'text-red'">
              {{ (ans.earned_score ?? 0).toFixed(1) }} / {{ ans.score.toFixed(1) }}
            </span>
          </div>
        </div>

        <div class="card-body">
          <!-- Question body (HTML) -->
          <div class="mb-3 question-body" v-html="sanitizeHtml(ans.body)" />

          <hr class="my-3" />

          <!-- PG / PGK: show options -->
          <template v-if="ans.question_type === 'pg' || ans.question_type === 'pgk'">
            <div class="d-flex flex-column gap-2">
              <div
                v-for="opt in parsePGOptions(ans)"
                :key="opt.key"
                class="d-flex align-items-start gap-2 p-2 rounded"
                :class="{
                  'bg-green-lt': isOptionCorrect(ans, opt.key),
                  'bg-red-lt': isOptionSelected(ans, opt.key) && !isOptionCorrect(ans, opt.key),
                  'bg-light': !isOptionSelected(ans, opt.key) && !isOptionCorrect(ans, opt.key),
                }"
              >
                <!-- Radio/Checkbox indicator -->
                <span class="mt-1" style="min-width: 20px;">
                  <i
                    v-if="isOptionCorrect(ans, opt.key) && isOptionSelected(ans, opt.key)"
                    class="ti ti-circle-check-filled text-green"
                  ></i>
                  <i
                    v-else-if="isOptionSelected(ans, opt.key) && !isOptionCorrect(ans, opt.key)"
                    class="ti ti-circle-x-filled text-red"
                  ></i>
                  <i
                    v-else-if="isOptionCorrect(ans, opt.key)"
                    class="ti ti-circle-check text-green"
                  ></i>
                  <i
                    v-else-if="ans.question_type === 'pgk'"
                    class="ti ti-square text-muted"
                  ></i>
                  <i v-else class="ti ti-circle text-muted"></i>
                </span>
                <div>
                  <span class="fw-bold me-1">{{ opt.key }}.</span>
                  <span v-html="sanitizeHtml(opt.text)" />
                </div>
              </div>
            </div>
          </template>

          <!-- Esai -->
          <template v-else-if="ans.question_type === 'esai'">
            <div class="mb-3">
              <div class="d-flex align-items-center gap-2 mb-2">
                <i class="ti ti-pencil text-muted"></i>
                <span class="fw-medium small text-muted">Jawaban Anda:</span>
              </div>
              <div class="p-3 rounded" :class="ans.user_answer ? 'bg-light' : 'bg-light text-muted fst-italic'">
                {{ ans.user_answer ? displayUserAnswer(ans) : '(Tidak dijawab)' }}
              </div>
            </div>
            <div class="alert alert-info mb-0 small">
              <i class="ti ti-info-circle me-1"></i>
              Soal esai dinilai oleh pengajar. Skor: <strong>{{ (ans.earned_score ?? 0).toFixed(1) }} / {{ ans.score.toFixed(1) }}</strong>
            </div>
          </template>

          <!-- Menjodohkan -->
          <template v-else-if="ans.question_type === 'menjodohkan'">
            <div class="table-responsive">
              <table class="table table-vcenter table-bordered mb-0">
                <thead>
                  <tr>
                    <th class="w-33">Pernyataan</th>
                    <th class="w-33">Jawaban Anda</th>
                    <th class="w-33">Jawaban Benar</th>
                    <th style="width:40px"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="pair in parseMatchingPairs(ans)" :key="pair.left">
                    <td class="fw-medium">{{ pair.left }}</td>
                    <td :class="pair.isCorrect ? 'text-green' : 'text-red'">
                      {{ pair.userRight }}
                    </td>
                    <td class="text-green">{{ pair.right }}</td>
                    <td class="text-center">
                      <i v-if="pair.isCorrect" class="ti ti-check text-green"></i>
                      <i v-else class="ti ti-x text-red"></i>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </template>

          <!-- Isian -->
          <template v-else-if="ans.question_type === 'isian'">
            <div class="mb-3">
              <div class="d-flex align-items-center gap-2 mb-2">
                <i class="ti ti-pencil text-muted"></i>
                <span class="fw-medium small text-muted">Jawaban Anda:</span>
              </div>
              <div
                class="p-3 rounded fw-medium"
                :class="ans.is_correct ? 'bg-green-lt text-green' : 'bg-red-lt text-red'"
              >
                {{ displayUserAnswer(ans) }}
              </div>
            </div>
            <div>
              <div class="d-flex align-items-center gap-2 mb-2">
                <i class="ti ti-check text-green"></i>
                <span class="fw-medium small text-muted">Jawaban yang diterima:</span>
              </div>
              <div class="d-flex flex-wrap gap-1">
                <span
                  v-for="accepted in parseIsianAccepted(ans)"
                  :key="accepted"
                  class="badge bg-green-lt text-green"
                >
                  {{ accepted }}
                </span>
              </div>
            </div>
          </template>

          <!-- Benar/Salah -->
          <template v-else-if="ans.question_type === 'benar_salah'">
            <div class="d-flex flex-column gap-2">
              <div
                v-for="opt in ['benar', 'salah']"
                :key="opt"
                class="d-flex align-items-center gap-2 p-2 rounded"
                :class="{
                  'bg-green-lt': ans.correct_answer === opt,
                  'bg-red-lt': ans.user_answer === opt && ans.correct_answer !== opt,
                  'bg-light': ans.user_answer !== opt && ans.correct_answer !== opt,
                }"
              >
                <i
                  v-if="ans.correct_answer === opt && ans.user_answer === opt"
                  class="ti ti-circle-check-filled text-green"
                ></i>
                <i
                  v-else-if="ans.user_answer === opt && ans.correct_answer !== opt"
                  class="ti ti-circle-x-filled text-red"
                ></i>
                <i
                  v-else-if="ans.correct_answer === opt"
                  class="ti ti-circle-check text-green"
                ></i>
                <i v-else class="ti ti-circle text-muted"></i>
                <span class="fw-medium text-capitalize">{{ opt }}</span>
              </div>
            </div>
          </template>

          <!-- Fallback for unknown types -->
          <template v-else>
            <div class="mb-2">
              <span class="fw-medium small text-muted">Jawaban Anda:</span>
              <div class="p-2 bg-light rounded mt-1">{{ displayUserAnswer(ans) }}</div>
            </div>
            <div>
              <span class="fw-medium small text-muted">Jawaban Benar:</span>
              <div class="p-2 bg-green-lt rounded mt-1">{{ typeof ans.correct_answer === 'string' ? ans.correct_answer : JSON.stringify(ans.correct_answer) }}</div>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- Bottom nav -->
    <div class="d-flex justify-content-center mt-4">
      <button class="btn btn-primary" @click="router.push('/peserta')">
        <i class="ti ti-arrow-left me-1"></i>Kembali ke Dashboard
      </button>
    </div>
  </div>
</template>

<style scoped>
.question-body :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 0.375rem;
}
.question-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
}
.question-body :deep(td),
.question-body :deep(th) {
  border: 1px solid var(--tblr-border-color);
  padding: 0.25rem 0.5rem;
}
.ring {
  box-shadow: 0 0 0 2px var(--tblr-primary);
}

/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  .page-header .row {
    flex-direction: column;
    gap: 0.5rem;
  }
  .page-header .col-auto {
    align-self: flex-start;
  }
  /* Summary cards stack 2x2 */
  .row.g-3.mb-4 > [class*="col-lg"] {
    flex: 0 0 50%;
    max-width: 50%;
  }
  /* Exam info row stack */
  .card-body .row.g-2 .col-sm-4 {
    flex: 0 0 100%;
    max-width: 100%;
  }
  /* Question navigator: horizontal scroll */
  .d-flex.flex-wrap.gap-1 {
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 0.375rem;
    gap: 0.375rem;
  }
  .d-flex.flex-wrap.gap-1 .btn {
    flex-shrink: 0;
    min-width: 36px;
    min-height: 36px;
  }
  /* Question body readable */
  .question-body {
    font-size: 0.95rem;
    line-height: 1.75;
    overflow-x: auto;
  }
  .question-body :deep(img) {
    max-width: 100%;
    height: auto;
  }
  /* Table scroll on matching type */
  .table-responsive {
    -webkit-overflow-scrolling: touch;
  }
  /* Card header flex wrap */
  .card-header .d-flex.align-items-center.gap-2.flex-grow-1 {
    flex-wrap: wrap;
    gap: 0.375rem;
  }
  .card-actions {
    margin-top: 0.25rem;
  }
  /* Options touch-friendly */
  .d-flex.align-items-start.gap-2.p-2.rounded {
    padding: 0.625rem !important;
    min-height: 44px;
  }
  /* Bottom button */
  .d-flex.justify-content-center.mt-4 .btn {
    min-height: 48px;
    width: 100%;
    max-width: 320px;
  }
}

/* ── Tablet Responsive ── */
@media (min-width: 768px) and (max-width: 1024px) {
  .row.g-3.mb-4 > [class*="col-lg"] {
    flex: 0 0 33.333%;
    max-width: 33.333%;
  }
  .d-flex.align-items-start.gap-2.p-2.rounded {
    min-height: 44px;
  }
}
</style>
