<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  questionBankApi,
  type QuestionBank,
  type Question,
  type Stimulus,
  QUESTION_TYPES,
} from '../../../api/question_bank.api'
import { sanitizeHtml } from '@/composables/useSafeHtml'

const route = useRoute()
const router = useRouter()
const bankId = Number(route.params.id)

const bank = ref<QuestionBank | null>(null)
const questions = ref<Question[]>([])
const stimuli = ref<Stimulus[]>([])
const loading = ref(true)
const withKey = ref(false)

async function fetchAll() {
  loading.value = true
  try {
    const [bankRes, qRes, sRes] = await Promise.all([
      questionBankApi.getById(bankId),
      questionBankApi.listQuestions(bankId, { per_page: 500 }),
      questionBankApi.listStimuli(bankId),
    ])
    bank.value = bankRes.data.data
    questions.value = qRes.data.data ?? []
    stimuli.value = sRes.data.data ?? []
  } finally {
    loading.value = false
  }
}

function getStimulusContent(stimulusId: number | null): string | null {
  if (!stimulusId) return null
  return stimuli.value.find(s => s.id === stimulusId)?.content ?? null
}

function qtLabel(t: string) {
  return QUESTION_TYPES.find(x => x.value === t)?.label ?? t
}

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleDateString('id-ID', {
    day: 'numeric', month: 'long', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

// Track which stimulus IDs have already been rendered to avoid duplicate stimulus boxes
const renderedStimulusIds = computed(() => {
  const seen = new Set<number>()
  return questions.value.map(q => {
    if (q.stimulus_id && !seen.has(q.stimulus_id)) {
      seen.add(q.stimulus_id)
      return q.stimulus_id
    }
    return null
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

function getOptionWeight(opt: any): number {
  return parseFloat(opt.weight ?? opt.score ?? 0)
}

function doPrint() {
  window.print()
}

onMounted(fetchAll)
</script>

<template>
  <!-- Toolbar (screen only) -->
  <div class="d-print-none mb-4">
    <div class="card">
      <div class="card-body py-2 d-flex align-items-center justify-content-between gap-3 flex-wrap">
        <div class="d-flex align-items-center gap-2">
          <button class="btn btn-sm btn-ghost-secondary" @click="router.push(`/admin/question-banks/${bankId}`)">
            <i class="ti ti-arrow-left me-1"></i>Kembali
          </button>
          <div v-if="bank">
            <span class="fw-semibold">{{ bank.name }}</span>
            <span class="text-muted ms-2 small">{{ questions.length }} soal</span>
          </div>
        </div>
        <div class="d-flex align-items-center gap-3">
          <label class="form-check form-switch mb-0 d-flex align-items-center gap-2">
            <input v-model="withKey" class="form-check-input" type="checkbox" role="switch" />
            <span class="form-check-label fw-semibold">Tampilkan Kunci Jawaban</span>
          </label>
          <button class="btn btn-primary" @click="doPrint">
            <i class="ti ti-printer me-1"></i>Cetak / Simpan PDF
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Loading -->
  <div v-if="loading" class="text-center py-5 text-muted d-print-none">
    <div class="spinner-border spinner-border-sm me-2"></div>Memuat soal...
  </div>

  <!-- Print Content -->
  <div v-else class="container-xl" style="max-width: 900px;">

    <!-- Identity Box -->
    <div class="border rounded p-4 mb-4" v-if="bank">
      <div class="fw-bold fs-4 border-bottom pb-2 mb-3">{{ bank.name }}</div>
      <table class="w-100" style="font-size: 0.95rem;">
        <tr>
          <td class="fw-semibold text-muted" style="width:160px; padding: 3px 0;">Mata Pelajaran</td>
          <td class="fw-bold" style="padding: 3px 0;">: {{ bank.subject?.name ?? '-' }}</td>
        </tr>
        <tr>
          <td class="fw-semibold text-muted" style="padding: 3px 0;">Jumlah Soal</td>
          <td style="padding: 3px 0;">: {{ questions.length }} soal</td>
        </tr>
        <tr>
          <td class="fw-semibold text-muted" style="padding: 3px 0;">Tanggal Cetak</td>
          <td style="padding: 3px 0;">: {{ formatDateTime(new Date().toISOString()) }}</td>
        </tr>
        <tr v-if="withKey">
          <td class="fw-semibold text-success" style="padding: 3px 0;">Mode Dokumen</td>
          <td class="text-success fw-bold" style="padding: 3px 0;">: PEGANGAN GURU (Kunci Jawaban Ditampilkan)</td>
        </tr>
      </table>
    </div>

    <!-- Empty state -->
    <div v-if="!questions.length" class="text-center py-5 text-muted d-print-none">
      <i class="ti ti-help-circle" style="font-size: 2rem;"></i>
      <p class="mt-2">Belum ada soal di bank soal ini.</p>
    </div>

    <!-- Questions -->
    <div
      v-for="(q, idx) in questions"
      :key="q.id"
      class="mb-4"
      style="break-inside: avoid; page-break-inside: avoid;"
    >
      <!-- Stimulus box (shown once per stimulus group) -->
      <div
        v-if="renderedStimulusIds[idx] !== null"
        class="p-3 mb-3 rounded"
        style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-left: 4px solid #3b82f6;"
      >
        <div class="fw-bold text-primary mb-2" style="font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.5px;">
          Wacana / Stimulus
        </div>
        <div v-html="sanitizeHtml(getStimulusContent(q.stimulus_id) ?? '')" class="small"></div>
      </div>

      <!-- Question item -->
      <div class="d-flex gap-2">
        <div class="fw-bold flex-shrink-0" style="width: 28px;">{{ idx + 1 }}.</div>
        <div class="flex-fill">

          <!-- Type + difficulty badges (screen only) -->
          <div class="d-print-none d-flex gap-1 mb-2 flex-wrap">
            <span class="badge bg-blue-lt text-blue small">{{ qtLabel(q.question_type) }}</span>
            <span class="badge bg-secondary-lt text-secondary small">{{ q.score }} poin</span>
          </div>

          <!-- Question body -->
          <div class="mb-3" v-html="sanitizeHtml(q.body ?? '')" style="line-height: 1.6;"></div>

          <!-- PG / PGK / Benar-Salah options -->
          <div v-if="['pg', 'pgk', 'benar_salah'].includes(q.question_type)" class="d-flex flex-column gap-2">
            <div
              v-for="opt in (q.options as any)"
              :key="opt.id"
              class="d-flex align-items-start gap-2"
              :class="withKey && isCorrectOption(q, opt.id) ? 'text-success' : ''"
            >
              <div
                class="flex-shrink-0 d-flex align-items-center justify-content-center fw-semibold rounded"
                style="width: 28px; height: 28px; font-size: 0.88rem;"
                :style="withKey && isCorrectOption(q, opt.id)
                  ? 'border: 2px solid #2fb344; background: #dcfce7; color: #166534;'
                  : 'border: 1.5px solid #cbd5e1; background: #fff; color: #334155;'"
              >
                {{ opt.id.toUpperCase() }}
              </div>
              <div class="pt-1 small" v-html="sanitizeHtml(opt.text ?? '')"></div>
              <span
                v-if="withKey && isCorrectOption(q, opt.id)"
                class="badge bg-success-lt text-success small ms-1 d-print-none"
              >
                {{ getOptionWeight(opt) === 1 ? 'Kunci' : Math.round(getOptionWeight(opt) * 100) + '%' }}
              </span>
            </div>
          </div>

          <!-- Menjodohkan -->
          <div v-else-if="q.question_type === 'menjodohkan'" class="mt-2">
            <table class="table table-sm table-bordered w-auto">
              <thead>
                <tr class="table-light">
                  <th>Premis (Kiri)</th>
                  <th>Pasangan (Kanan)</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in (q.options as any)?.prompts" :key="p.id">
                  <td>{{ p.id }}. {{ p.text }}</td>
                  <td v-if="withKey">
                    <strong>
                      {{
                        (q.options as any)?.answers?.find((a: any) =>
                          a.id === (q.correct_answer as any)?.[p.id]
                        )?.text ?? '–'
                      }}
                    </strong>
                  </td>
                  <td v-else class="text-muted" style="min-width: 120px;">.....................</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Isian Singkat -->
          <div v-else-if="q.question_type === 'isian_singkat'" class="mt-3">
            <div class="d-flex align-items-center gap-2 p-3 bg-light rounded border">
              <span class="small">Jawaban:</span>
              <div style="border-bottom: 2px dotted #999; width: 200px;"></div>
            </div>
            <div v-if="withKey && (q.options as any)?.accepted_answers?.length" class="mt-2 text-success small">
              <strong>Kunci:</strong>
              {{ (q.options as any).accepted_answers.join(', ') }}
            </div>
          </div>

          <!-- Esai -->
          <div v-else-if="q.question_type === 'esai'" class="mt-2">
            <div
              class="p-4 border rounded"
              style="background-image: linear-gradient(#e5e7eb 1px, transparent 1px); background-size: 100% 2em; line-height: 2em; min-height: 150px;"
            ></div>
            <div v-if="withKey" class="mt-2 text-warning small">
              <em>*Esai dinilai secara manual.</em>
            </div>
          </div>

          <!-- Matrix -->
          <div v-else-if="q.question_type === 'matrix'" class="mt-2">
            <table class="table table-sm table-bordered w-auto">
              <thead>
                <tr class="table-light">
                  <th>Pernyataan</th>
                  <th v-for="col in (q.options as any)?.columns" :key="col.id">{{ col.text }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in (q.options as any)?.rows" :key="row.id">
                  <td>{{ row.text }}</td>
                  <td
                    v-for="col in (q.options as any)?.columns"
                    :key="col.id"
                    class="text-center"
                    :class="withKey && (q.correct_answer as any)?.[row.id] === col.id ? 'table-success fw-bold' : ''"
                  >
                    <span v-if="withKey && (q.correct_answer as any)?.[row.id] === col.id">
                      <i class="ti ti-check text-success"></i>
                    </span>
                    <span v-else class="text-muted">○</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Metadata (screen only, with key) -->
          <div v-if="withKey" class="mt-2 text-muted small d-print-none">
            [Tipe: {{ q.question_type.toUpperCase() }} | Poin: {{ q.score }}]
          </div>

        </div>
      </div>
    </div>

  </div>
</template>
