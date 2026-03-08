<script setup lang="ts">
import { computed } from 'vue'
import type { SafeQuestion } from '@/api/exam.api'

interface QuestionSection {
  bankId: number
  bankName: string
  startIdx: number
  endIdx: number
}

const props = defineProps<{
  show: boolean
  questions: SafeQuestion[]
  answersMap: Map<number, unknown>
  flaggedSet: Set<number>
  answeredCount: number
  unansweredCount: number
  flaggedCount: number
  finishing: boolean
  questionSections?: QuestionSection[]
  hasMultipleSections?: boolean
}>()

const emit = defineEmits<{
  close: []
  goTo: [idx: number]
  confirmFinish: []
}>()

const unansweredQuestions = computed(() =>
  props.questions
    .map((q, i) => ({ ...q, idx: i }))
    .filter(q => !props.answersMap.has(q.id))
)

const flaggedQuestions = computed(() =>
  props.questions
    .map((q, i) => ({ ...q, idx: i }))
    .filter(q => props.flaggedSet.has(q.id))
)

const sectionBreakdown = computed(() => {
  if (!props.hasMultipleSections || !props.questionSections?.length) return []
  return props.questionSections.map(sec => {
    const sectionQuestions = props.questions.slice(sec.startIdx, sec.endIdx + 1)
    const total = sectionQuestions.length
    const answered = sectionQuestions.filter(q => props.answersMap.has(q.id)).length
    return { bankName: sec.bankName, total, answered }
  })
})

function summaryGoTo(idx: number) {
  emit('close')
  emit('goTo', idx)
}
</script>

<template>
  <!-- Answer Summary Modal -->
  <Teleport to="body">
    <div v-if="show" class="summary-overlay" @click.self="emit('close')">
      <div class="summary-dialog">
        <div class="summary-header">
          <h2 class="summary-title">Ringkasan Jawaban</h2>
          <button class="summary-close" @click="emit('close')">
            <i class="ti ti-x" style="font-size: 20px;" />
          </button>
        </div>

        <div class="summary-body">
          <!-- Stats cards -->
          <div class="summary-stats">
            <div class="stat-card stat-total">
              <div class="stat-number">{{ questions.length }}</div>
              <div class="stat-label">Total Soal</div>
            </div>
            <div class="stat-card stat-answered">
              <div class="stat-number">{{ answeredCount }}</div>
              <div class="stat-label">Dijawab</div>
            </div>
            <div class="stat-card stat-unanswered">
              <div class="stat-number">{{ unansweredCount }}</div>
              <div class="stat-label">Belum Dijawab</div>
            </div>
            <div class="stat-card stat-flagged">
              <div class="stat-number">{{ flaggedCount }}</div>
              <div class="stat-label">Ragu-ragu</div>
            </div>
          </div>

          <!-- Per-section breakdown -->
          <div v-if="sectionBreakdown.length > 0" class="summary-section">
            <h3 class="summary-section-title" style="color: #334155;">
              <i class="ti ti-list-details" />
              Progress per Bagian
            </h3>
            <div class="section-breakdown">
              <div v-for="(sec, idx) in sectionBreakdown" :key="idx" class="section-breakdown-row">
                <span class="section-breakdown-name">{{ sec.bankName }}</span>
                <span class="section-breakdown-progress">
                  <span class="section-breakdown-bar">
                    <span class="section-breakdown-fill" :style="{ width: (sec.answered / Math.max(sec.total, 1) * 100) + '%' }" />
                  </span>
                  Dijawab {{ sec.answered }} dari {{ sec.total }}
                </span>
              </div>
            </div>
          </div>

          <!-- Unanswered list -->
          <div v-if="unansweredQuestions.length > 0" class="summary-section" role="alert">
            <h3 class="summary-section-title summary-section-title--danger">
              <i class="ti ti-alert-triangle" />
              Soal Belum Dijawab ({{ unansweredQuestions.length }})
            </h3>
            <div class="summary-question-list">
              <button
                v-for="q in unansweredQuestions" :key="q.id"
                class="summary-question-btn summary-question-btn--unanswered"
                @click="summaryGoTo(q.idx)"
              >
                Soal {{ q.idx + 1 }}
              </button>
            </div>
          </div>

          <!-- Flagged list -->
          <div v-if="flaggedQuestions.length > 0" class="summary-section">
            <h3 class="summary-section-title summary-section-title--warning">
              <i class="ti ti-flag" />
              Soal Ditandai Ragu ({{ flaggedQuestions.length }})
            </h3>
            <div class="summary-question-list">
              <button
                v-for="q in flaggedQuestions" :key="q.id"
                class="summary-question-btn summary-question-btn--flagged"
                @click="summaryGoTo(q.idx)"
              >
                Soal {{ q.idx + 1 }}
              </button>
            </div>
          </div>

          <!-- All good message -->
          <div v-if="unansweredQuestions.length === 0 && flaggedQuestions.length === 0" class="summary-allgood">
            <i class="ti ti-circle-check" style="font-size:2.5rem" />
            <p>Semua soal telah dijawab dan tidak ada yang ditandai ragu.</p>
          </div>
        </div>

        <div class="summary-footer">
          <button class="btn-review" @click="emit('close')">
            <i class="ti ti-chevron-left" />
            Kembali Review
          </button>
          <button class="btn-submit" @click="emit('confirmFinish')">
            Kumpulkan Ujian
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- Final Finish Confirmation modal -->
  <slot name="finishModal" />
</template>

<style scoped>
/* Summary Modal */
.summary-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1090;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.summary-dialog {
  width: 100%;
  max-width: 540px;
  max-height: 85vh;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.summary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid #e2e8f0;
}
.summary-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: #1e293b;
  margin: 0;
}
.summary-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: #f1f5f9;
  border-radius: 8px;
  color: #475569;
  cursor: pointer;
}
.summary-close:hover { background: #e2e8f0; }

.summary-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.25rem;
}

/* Stats grid */
.summary-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}
@media (max-width: 480px) {
  .summary-stats { grid-template-columns: repeat(2, 1fr); }
}
.stat-card {
  text-align: center;
  padding: 0.75rem 0.5rem;
  border-radius: 10px;
  border: 1px solid;
}
.stat-number {
  font-size: 1.5rem;
  font-weight: 800;
  line-height: 1;
  margin-bottom: 0.25rem;
}
.stat-label {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.stat-total {
  border-color: #e2e8f0;
  background: #f8fafc;
}
.stat-total .stat-number { color: #334155; }
.stat-total .stat-label { color: #64748b; }
.stat-answered {
  border-color: #bbf7d0;
  background: #f0fdf4;
}
.stat-answered .stat-number { color: #16a34a; }
.stat-answered .stat-label { color: #15803d; }
.stat-unanswered {
  border-color: #fecaca;
  background: #fef2f2;
}
.stat-unanswered .stat-number { color: #dc2626; }
.stat-unanswered .stat-label { color: #b91c1c; }
.stat-flagged {
  border-color: #fed7aa;
  background: #fff7ed;
}
.stat-flagged .stat-number { color: #ea580c; }
.stat-flagged .stat-label { color: #c2410c; }

/* Summary sections */
.summary-section {
  margin-bottom: 1rem;
}
.summary-section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  font-weight: 700;
  margin: 0 0 0.5rem;
}
.summary-section-title--danger { color: #dc2626; }
.summary-section-title--warning { color: #d97706; }

.summary-question-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}
.summary-question-btn {
  padding: 0.3rem 0.6rem;
  font-size: 0.75rem;
  font-weight: 600;
  border: 1px solid;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}
.summary-question-btn--unanswered {
  border-color: #fecaca;
  background: #fef2f2;
  color: #dc2626;
}
.summary-question-btn--unanswered:hover {
  background: #fee2e2;
}
.summary-question-btn--flagged {
  border-color: #fed7aa;
  background: #fff7ed;
  color: #ea580c;
}
.summary-question-btn--flagged:hover {
  background: #ffedd5;
}

.summary-allgood {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 2rem 1rem;
  color: #16a34a;
  text-align: center;
}
.summary-allgood p {
  margin: 0;
  font-size: 0.9rem;
  color: #475569;
}

/* Section breakdown */
.section-breakdown {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.section-breakdown-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  font-size: 0.8rem;
  color: #475569;
}
.section-breakdown-name {
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.section-breakdown-progress {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
  font-size: 0.75rem;
  color: #64748b;
  white-space: nowrap;
}
.section-breakdown-bar {
  width: 60px;
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}
.section-breakdown-fill {
  display: block;
  height: 100%;
  background: var(--tblr-primary, #4f46e5);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.summary-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid #e2e8f0;
  background: #f8fafc;
}
.btn-review {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  font-size: 0.85rem;
  font-weight: 600;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-review:hover {
  background: #f1f5f9;
}
.btn-submit {
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
.btn-submit:hover {
  background: #b91c1c;
}

/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  .summary-overlay {
    padding: 0;
    align-items: flex-end;
  }
  .summary-dialog {
    max-width: none;
    max-height: 90vh;
    border-radius: 16px 16px 0 0;
    touch-action: manipulation;
  }
  .summary-body {
    padding: 1rem;
    -webkit-overflow-scrolling: touch;
  }
  .summary-stats {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }
  .stat-number {
    font-size: 1.25rem;
  }
  .summary-footer {
    flex-direction: column;
    padding: 0.75rem 1rem;
    gap: 0.5rem;
  }
  .btn-review,
  .btn-submit {
    width: 100%;
    justify-content: center;
    min-height: 48px;
    font-size: 0.9rem;
  }
  .summary-question-btn {
    min-height: 36px;
    padding: 0.375rem 0.75rem;
    font-size: 0.8rem;
  }
  .section-breakdown-row {
    flex-direction: column;
    gap: 0.25rem;
    align-items: flex-start;
  }
}

/* ── Tablet Responsive ── */
@media (min-width: 768px) and (max-width: 1024px) {
  .summary-dialog {
    max-width: 500px;
  }
  .btn-review,
  .btn-submit {
    min-height: 44px;
  }
  .summary-question-btn {
    min-height: 36px;
  }
}
</style>
