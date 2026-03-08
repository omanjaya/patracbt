<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { reportApi, type PersonalReport, type AnswerDetail } from '../../../api/report.api'
import { useToastStore } from '../../../stores/toast.store'
import { sanitizeHtml } from '@/composables/useSafeHtml'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const scheduleId = Number(route.params.scheduleId)
const sessionId = Number(route.params.sessionId)

const report = ref<PersonalReport | null>(null)
const loading = ref(true)
const error = ref('')
const savingMap = ref<Record<number, boolean>>({})
const savedMap = ref<Record<number, boolean>>({})
const aiLoadingMap = ref<Record<number, boolean>>({})
const aiResultMap = ref<Record<number, { score: number; reason: string } | null>>({})
const scoreInputs = ref<Record<number, number>>({})
const batchLoading = ref(false)
const expandedReasons = ref<Record<number, boolean>>({})

async function load() {
  try {
    const res = await reportApi.getPersonalReport(scheduleId, sessionId)
    report.value = res.data.data
    // Init score inputs from existing earned_score
    for (const ans of res.data.data.answers ?? []) {
      if (ans.question_type === 'esai') {
        scoreInputs.value[ans.question_id] = ans.earned_score ?? 0
      }
    }
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat data.'
  } finally {
    loading.value = false
  }
}

async function saveScore(ans: AnswerDetail) {
  const score = scoreInputs.value[ans.question_id] ?? 0
  savingMap.value[ans.question_id] = true
  savedMap.value[ans.question_id] = false
  try {
    await reportApi.gradeEssay(sessionId, ans.question_id, score)
    savedMap.value[ans.question_id] = true
    toast.success('Nilai tersimpan')
    setTimeout(() => { savedMap.value[ans.question_id] = false }, 3000)
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menyimpan nilai')
  } finally {
    savingMap.value[ans.question_id] = false
  }
}

async function askAI(ans: AnswerDetail) {
  const answerText = typeof ans.user_answer === 'string' ? ans.user_answer : JSON.stringify(ans.user_answer ?? '')
  aiLoadingMap.value[ans.question_id] = true
  aiResultMap.value[ans.question_id] = null
  try {
    const res = await reportApi.aiGradeEssay(sessionId, ans.question_id, answerText)
    const data = res.data as any
    const result = data.data ?? data
    aiResultMap.value[ans.question_id] = { score: result.score ?? result.suggested_points ?? 0, reason: result.reason ?? result.feedback ?? '' }
    expandedReasons.value[ans.question_id] = true
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menghubungi AI. Pastikan konfigurasi AI sudah diatur di menu Pengaturan.')
  } finally {
    aiLoadingMap.value[ans.question_id] = false
  }
}

async function askAIBatch() {
  batchLoading.value = true
  // Mark all essay questions as loading
  for (const ans of essayAnswers()) {
    if (ans.user_answer) {
      aiLoadingMap.value[ans.question_id] = true
    }
  }
  try {
    const res = await reportApi.aiGradeBatchEssay(sessionId)
    const results = (res.data as any).data ?? res.data
    if (Array.isArray(results)) {
      for (const item of results) {
        if (item.error) {
          toast.error(`Soal #${item.question_id}: ${item.error}`)
        } else {
          aiResultMap.value[item.question_id] = { score: item.score, reason: item.reason }
          scoreInputs.value[item.question_id] = item.score
          expandedReasons.value[item.question_id] = true
        }
        aiLoadingMap.value[item.question_id] = false
      }
      toast.success(`${results.filter((r: any) => !r.error).length} esai berhasil dinilai AI`)
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menghubungi AI. Pastikan konfigurasi AI sudah diatur di menu Pengaturan.')
  } finally {
    batchLoading.value = false
    for (const ans of essayAnswers()) {
      aiLoadingMap.value[ans.question_id] = false
    }
  }
}

function applyAIScore(qId: number) {
  const aiResult = aiResultMap.value[qId]
  if (aiResult != null) {
    scoreInputs.value[qId] = aiResult.score
  }
}

function toggleReason(qId: number) {
  expandedReasons.value[qId] = !expandedReasons.value[qId]
}

const essayAnswers = () => (report.value?.answers ?? []).filter(a => a.question_type === 'esai')
onMounted(load)
</script>

<template>
  <BasePageHeader
    :title="(report?.session as any)?.user?.name ?? `Sesi #${sessionId}`"
    subtitle="Koreksi Esai"
    :breadcrumbs="[{ label: 'Penilaian', to: '/guru/reports' }, { label: 'Detail', to: `/guru/grading/${scheduleId}` }, { label: (report?.session as any)?.user?.name ?? 'Koreksi' }]"
  >
    <template #actions>
      <button
        v-if="essayAnswers().length > 0"
        class="btn btn-purple"
        :disabled="batchLoading"
        @click="askAIBatch"
      >
        <span v-if="batchLoading" class="spinner-border spinner-border-sm me-1"></span>
        <i v-else class="ti ti-sparkles me-1"></i>
        {{ batchLoading ? 'Menilai dengan AI...' : 'AI Grade Semua Esai' }}
      </button>
      <button class="btn btn-ghost-secondary" @click="router.push(`/guru/grading/${scheduleId}`)">
        <i class="ti ti-arrow-left me-1"></i>Kembali
      </button>
    </template>
  </BasePageHeader>

  <div v-if="loading" class="d-flex flex-column gap-3">
    <div v-for="n in 3" :key="n" class="card placeholder-glow">
      <div class="card-header">
        <div class="placeholder col-3"></div>
      </div>
      <div class="card-body">
        <div class="placeholder col-8 mb-3"></div>
        <div class="placeholder col-6 mb-2"></div>
        <hr class="my-3" />
        <div class="placeholder col-10 mb-2"></div>
        <div class="placeholder col-4"></div>
      </div>
    </div>
  </div>

  <div v-else-if="error" class="alert alert-danger">{{ error }}</div>

  <div v-else-if="essayAnswers().length === 0" class="card">
    <div class="card-body text-center text-muted py-5">
      <i class="ti ti-file-off fs-4 mb-2 d-block opacity-50"></i>
      Tidak ada soal esai di ujian ini.
    </div>
  </div>

  <div v-else class="d-flex flex-column gap-3">
    <div v-for="(ans, index) in essayAnswers()" :key="ans.question_id" class="card">
      <div class="card-header">
        <h3 class="card-title">Soal No. {{ index + 1 }}</h3>
        <div class="card-options d-flex gap-2 align-items-center">
          <span v-if="aiResultMap[ans.question_id]" class="badge bg-purple-lt text-purple">
            <i class="ti ti-sparkles me-1"></i>AI Graded
          </span>
          <span class="badge bg-blue-lt text-blue">Bobot Max: {{ ans.score }}</span>
        </div>
      </div>
      <div class="card-body">
        <!-- Question body -->
        <div class="mb-3" v-html="sanitizeHtml(ans.body)" />

        <hr class="my-3" />

        <!-- Student answer -->
        <div class="mb-3">
          <p class="form-label fw-medium">Jawaban Siswa:</p>
          <div v-if="ans.user_answer" class="p-3 bg-light rounded">
            {{ typeof ans.user_answer === 'string' ? ans.user_answer : (ans.user_answer as any)?.text ?? JSON.stringify(ans.user_answer) }}
          </div>
          <div v-else class="text-muted fst-italic p-3 bg-light rounded">Siswa tidak menjawab (Kosong).</div>
        </div>

        <!-- AI Reasoning (expanded section) -->
        <div v-if="aiResultMap[ans.question_id]" class="mb-3">
          <div class="alert alert-purple mb-0">
            <div class="d-flex align-items-center justify-content-between mb-1">
              <div class="fw-medium">
                <i class="ti ti-sparkles me-1"></i>Saran AI: {{ aiResultMap[ans.question_id]!.score }} poin
              </div>
              <button class="btn btn-sm btn-ghost-purple p-1" @click="toggleReason(ans.question_id)">
                <i :class="expandedReasons[ans.question_id] ? 'ti ti-chevron-up' : 'ti ti-chevron-down'"></i>
              </button>
            </div>
            <div v-if="expandedReasons[ans.question_id]">
              <p class="small mb-2 text-muted">{{ aiResultMap[ans.question_id]!.reason }}</p>
              <button class="btn btn-sm btn-purple" @click="applyAIScore(ans.question_id)">
                <i class="ti ti-check me-1"></i>Gunakan Nilai AI ({{ aiResultMap[ans.question_id]!.score }})
              </button>
            </div>
          </div>
        </div>

        <!-- Score + AI row -->
        <div class="row g-3">
          <div class="col-md-6">
            <p class="form-label">Berikan Nilai (Poin)</p>
            <div class="input-group">
              <input
                type="number"
                v-model.number="scoreInputs[ans.question_id]"
                :min="0"
                :max="ans.score"
                step="0.01"
                class="form-control"
                :aria-label="`Nilai untuk soal nomor ${index + 1}, maksimal ${ans.score} poin`"
              />
              <span class="input-group-text">/ {{ ans.score }}</span>
              <button
                class="btn"
                :class="savedMap[ans.question_id] ? 'btn-success' : 'btn-primary'"
                :disabled="savingMap[ans.question_id]"
                @click="saveScore(ans)"
              >
                <template v-if="savingMap[ans.question_id]">
                  <span class="spinner-border spinner-border-sm me-1"></span>Menyimpan...
                </template>
                <template v-else-if="savedMap[ans.question_id]">
                  <i class="ti ti-check me-1"></i>Tersimpan
                </template>
                <template v-else>Simpan</template>
              </button>
            </div>
          </div>

          <div v-if="ans.user_answer" class="col-md-6">
            <p class="form-label">Bantuan AI</p>
            <button
              class="btn btn-outline-purple w-100"
              :disabled="aiLoadingMap[ans.question_id]"
              @click="askAI(ans)"
            >
              <span v-if="aiLoadingMap[ans.question_id]" class="spinner-border spinner-border-sm me-1"></span>
              <i v-else class="ti ti-sparkles me-1"></i>
              {{ aiLoadingMap[ans.question_id] ? 'Menilai dengan AI...' : 'AI Grade' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
