<script setup lang="ts">
import type { SafeQuestion, Stimulus } from '@/api/exam.api'
import { sanitizeHtml } from '@/composables/useSafeHtml'

const props = defineProps<{
  question: SafeQuestion | null
  opts: Array<{ id: string; text: string }>
  stimulus: Stimulus | null
  stimulusLoading: boolean
  stimulusError: boolean
  isFlagged: boolean
  currentIdx: number
  totalQuestions: number
  // Answer state
  pgAnswer: string
  pgkAnswers: string[]
  bsAnswer: string
  matchPairs: Record<string, string>
  isianText: string
  matrixAnswers: Record<string, string>
  esaiText: string
}>()

const emit = defineEmits<{
  answerChange: []
  toggleFlag: []
  retryStimulus: []
  prev: []
  next: []
  'update:pgAnswer': [value: string]
  'update:pgkAnswers': [value: string[]]
  'update:bsAnswer': [value: string]
  'update:matchPairs': [value: Record<string, string>]
  'update:isianText': [value: string]
  'update:matrixAnswers': [value: Record<string, string>]
  'update:esaiText': [value: string]
}>()

function onPgChange(optId: string) {
  emit('update:pgAnswer', optId)
  emit('answerChange')
}

function onPgkToggle(optId: string) {
  const current = [...props.pgkAnswers]
  const idx = current.indexOf(optId)
  if (idx >= 0) {
    current.splice(idx, 1)
  } else {
    current.push(optId)
  }
  emit('update:pgkAnswers', current)
  emit('answerChange')
}

function onBsChange(optId: string) {
  emit('update:bsAnswer', optId)
  emit('answerChange')
}

function onMatchChange(promptId: string, value: string) {
  const updated = { ...props.matchPairs, [promptId]: value }
  emit('update:matchPairs', updated)
  emit('answerChange')
}

function onIsianInput(e: Event) {
  emit('update:isianText', (e.target as HTMLInputElement).value)
  emit('answerChange')
}

function onMatrixChange(rowId: string, value: string) {
  const updated = { ...props.matrixAnswers, [rowId]: value }
  emit('update:matrixAnswers', updated)
  emit('answerChange')
}

function onEsaiInput(e: Event) {
  emit('update:esaiText', (e.target as HTMLTextAreaElement).value)
  emit('answerChange')
}
</script>

<template>
  <div class="question-panel">
    <div class="question-num-badge">Soal {{ currentIdx + 1 }} dari {{ totalQuestions }}</div>

    <!-- Stimulus / Wacana -->
    <div v-if="question?.stimulus_id" class="stimulus-box">
      <p class="stimulus-label">Bacaan / Stimulus</p>
      <div v-if="stimulus" class="stimulus-content" v-html="sanitizeHtml(stimulus.content ?? '')" />
      <div v-else-if="stimulusError" class="stimulus-error">
        <i class="ti ti-alert-triangle"></i>
        <span>Gagal memuat stimulus.</span>
        <button class="stimulus-retry-btn" @click="emit('retryStimulus')">Klik untuk coba lagi</button>
      </div>
      <div v-else class="stimulus-loading">
        <span class="loading-dot" /><span class="loading-dot" /><span class="loading-dot" />
      </div>
    </div>

    <!-- Question body -->
    <div class="question-body" v-html="sanitizeHtml(question?.body ?? '')" />

    <!-- Answer inputs per type -->
    <div class="answer-area">
      <!-- PG -->
      <div v-if="question?.question_type === 'pg'" class="options-list">
        <label
          v-for="opt in opts" :key="opt.id"
          class="option-item"
          :class="{ 'option-selected': pgAnswer === opt.id }"
        >
          <input type="radio" :value="opt.id" :checked="pgAnswer === opt.id" @change="onPgChange(opt.id)" class="sr-only" />
          <span class="opt-key">{{ opt.id.toUpperCase() }}</span>
          <span class="opt-text">{{ opt.text }}</span>
        </label>
      </div>

      <!-- PGK -->
      <div v-else-if="question?.question_type === 'pgk'" class="options-list">
        <label
          v-for="opt in opts" :key="opt.id"
          class="option-item"
          :class="{ 'option-selected': pgkAnswers.includes(opt.id) }"
        >
          <input type="checkbox" :value="opt.id" :checked="pgkAnswers.includes(opt.id)" @change="onPgkToggle(opt.id)" class="sr-only" />
          <span class="opt-key">{{ opt.id.toUpperCase() }}</span>
          <span class="opt-text">{{ opt.text }}</span>
        </label>
      </div>

      <!-- Benar/Salah -->
      <div v-else-if="question?.question_type === 'benar_salah'" class="options-list">
        <label
          v-for="opt in [{ id: 'true', text: 'Benar' }, { id: 'false', text: 'Salah' }]" :key="opt.id"
          class="option-item"
          :class="{ 'option-selected': bsAnswer === opt.id }"
        >
          <input type="radio" :value="opt.id" :checked="bsAnswer === opt.id" @change="onBsChange(opt.id)" class="sr-only" />
          <span class="opt-key">{{ opt.id === 'true' ? '\u2713' : '\u2717' }}</span>
          <span class="opt-text">{{ opt.text }}</span>
        </label>
      </div>

      <!-- Menjodohkan -->
      <div v-else-if="question?.question_type === 'menjodohkan'" class="match-answer">
        <div class="match-prompts">
          <div v-for="p in (question.options as any)?.prompts" :key="p.id" class="match-row">
            <span class="match-label">{{ p.id }}. {{ p.text }}</span>
            <select :value="matchPairs[p.id] ?? ''" @change="onMatchChange(p.id, ($event.target as HTMLSelectElement).value)" class="match-select">
              <option value="">Pilih...</option>
              <option v-for="a in (question.options as any)?.answers" :key="a.id" :value="a.id">
                {{ a.id.toUpperCase() }}. {{ a.text }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- Isian Singkat -->
      <div v-else-if="question?.question_type === 'isian_singkat'" class="isian-answer">
        <input :value="isianText" @input="onIsianInput" class="isian-input" placeholder="Ketik jawaban Anda..." />
      </div>

      <!-- Matrix -->
      <div v-else-if="question?.question_type === 'matrix'" class="matrix-answer">
        <div v-for="r in (question.options as any)?.rows" :key="r.id" class="match-row">
          <span class="match-label">{{ r.id }}. {{ r.text }}</span>
          <select :value="matrixAnswers[r.id] ?? ''" @change="onMatrixChange(r.id, ($event.target as HTMLSelectElement).value)" class="match-select">
            <option value="">Pilih...</option>
            <option v-for="c in (question.options as any)?.columns" :key="c.id" :value="c.id">
              {{ c.id.toUpperCase() }}. {{ c.text }}
            </option>
          </select>
        </div>
      </div>

      <!-- Esai -->
      <div v-else-if="question?.question_type === 'esai'" class="esai-answer">
        <textarea :value="esaiText" @input="onEsaiInput" class="esai-textarea" rows="8" placeholder="Tulis jawaban esai Anda di sini..." />
      </div>
    </div>

    <!-- Navigation buttons -->
    <div class="nav-buttons">
      <button class="btn-nav" @click="emit('prev')" :disabled="currentIdx === 0" aria-label="Soal sebelumnya">
        <i class="ti ti-chevron-left" />
        <span class="nav-label">Sebelumnya</span>
      </button>
      <button class="btn-flag" :class="{ 'btn-flag--active': isFlagged }" @click="emit('toggleFlag')" :aria-label="isFlagged ? 'Batalkan ragu-ragu' : 'Tandai ragu-ragu'">
        <i class="ti ti-flag" />
        <span class="nav-label">{{ isFlagged ? 'Batalkan Ragu' : 'Ragu-ragu' }}</span>
      </button>
      <button class="btn-nav" @click="emit('next')" :disabled="currentIdx === totalQuestions - 1" aria-label="Soal berikutnya">
        <span class="nav-label">Berikutnya</span>
        <i class="ti ti-chevron-right" />
      </button>
    </div>
  </div>
</template>

<style scoped>
/* Question Panel */
.question-panel {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
}
.question-num-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--tblr-primary, #4f46e5);
  background: var(--tblr-primary-lt, #eef2ff);
  border-radius: 999px;
  margin-bottom: 1rem;
  align-self: flex-start;
}

/* Stimulus box */
.stimulus-box {
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 10px;
  padding: 1rem 1.25rem;
  margin-bottom: 1.25rem;
}
.stimulus-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #92400e;
  margin-bottom: 0.5rem;
}
.stimulus-content {
  font-size: 0.9rem;
  line-height: 1.7;
  color: #451a03;
}
.stimulus-loading {
  display: flex;
  gap: 4px;
  padding: 0.5rem 0;
}
.loading-dot {
  width: 6px;
  height: 6px;
  background: #d97706;
  border-radius: 50%;
  animation: dot-bounce 0.6s infinite alternate;
}
.loading-dot:nth-child(2) { animation-delay: 0.2s; }
.loading-dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes dot-bounce {
  to { opacity: 0.3; transform: translateY(-3px); }
}
.stimulus-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  color: #9a3412;
  font-size: 0.85rem;
}
.stimulus-retry-btn {
  border: 1px solid #fdba74;
  background: #fff7ed;
  color: #9a3412;
  font-size: 0.8rem;
  font-weight: 600;
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}
.stimulus-retry-btn:hover {
  background: #ffedd5;
}

/* Question body */
.question-body {
  font-size: 0.95rem;
  line-height: 1.75;
  color: #1e293b;
  margin-bottom: 1.5rem;
}
.question-body :deep(img) {
  max-width: 100%;
  border-radius: 8px;
}

/* Answer Area */
.answer-area {
  flex: 1;
}
.options-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.option-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 2px solid #e2e8f0;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
  background: #fff;
}
.option-item:hover {
  border-color: #c7d2fe;
  background: #f5f3ff;
}
.option-selected {
  border-color: var(--tblr-primary, #4f46e5);
  background: var(--tblr-primary-lt, #eef2ff);
}
.opt-key {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #f1f5f9;
  font-size: 0.75rem;
  font-weight: 700;
  color: #475569;
  flex-shrink: 0;
}
.option-selected .opt-key {
  background: var(--tblr-primary, #4f46e5);
  color: #fff;
}
.opt-text {
  padding-top: 0.2rem;
  font-size: 0.9rem;
  line-height: 1.5;
}
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0,0,0,0);
  white-space: nowrap;
  border: 0;
}

/* Matching / Matrix */
.match-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.75rem;
}
.match-label {
  flex: 1;
  font-size: 0.9rem;
}
.match-select {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 0.85rem;
  background: #fff;
}

/* Isian / Esai */
.isian-input {
  width: 100%;
  padding: 0.625rem 0.875rem;
  border: 2px solid #e2e8f0;
  border-radius: 10px;
  font-size: 0.9rem;
  transition: border-color 0.15s;
}
.isian-input:focus {
  outline: none;
  border-color: var(--tblr-primary, #4f46e5);
}
.esai-textarea {
  width: 100%;
  padding: 0.75rem 1rem;
  border: 2px solid #e2e8f0;
  border-radius: 10px;
  font-size: 0.9rem;
  line-height: 1.6;
  resize: vertical;
  min-height: 160px;
  transition: border-color 0.15s;
}
.esai-textarea:focus {
  outline: none;
  border-color: var(--tblr-primary, #4f46e5);
}

/* Navigation Buttons */
.nav-buttons {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-top: auto;
  padding-top: 1.5rem;
  border-top: 1px solid #e2e8f0;
}
.btn-nav {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  font-size: 0.8rem;
  font-weight: 600;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-nav:hover:not(:disabled) {
  background: #f8fafc;
  border-color: #cbd5e1;
}
.btn-nav:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.btn-flag {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  font-size: 0.8rem;
  font-weight: 600;
  border: 1px solid #fbbf24;
  border-radius: 8px;
  background: #fff;
  color: #92400e;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-flag:hover {
  background: #fffbeb;
}
.btn-flag--active {
  background: #f59e0b;
  color: #fff;
  border-color: #f59e0b;
}
.btn-flag--active:hover {
  background: #d97706;
}
@media (max-width: 480px) {
  .nav-label { display: none; }
  .btn-nav, .btn-flag { padding: 0.5rem 0.75rem; }
}

/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  .question-panel {
    padding: 1rem;
    touch-action: manipulation;
  }
  .question-body {
    font-size: 1rem;
    line-height: 1.8;
    /* Prevent iOS zoom on text < 16px */
  }
  .option-item {
    padding: 0.875rem 1rem;
    min-height: 48px;
    gap: 0.625rem;
  }
  .opt-key {
    width: 32px;
    height: 32px;
    font-size: 0.8rem;
  }
  .opt-text {
    font-size: 0.95rem;
    line-height: 1.55;
  }
  .stimulus-content {
    font-size: 0.95rem;
  }
  .match-row {
    flex-direction: column;
    gap: 0.5rem;
    align-items: stretch;
  }
  .match-select {
    width: 100%;
    min-height: 44px;
    font-size: 1rem;
  }
  .isian-input {
    min-height: 48px;
    font-size: 1rem;
  }
  .esai-textarea {
    font-size: 1rem;
    min-height: 140px;
  }
  .nav-buttons {
    padding-top: 1rem;
    position: sticky;
    bottom: 0;
    background: #f1f5f9;
    margin: 0 -1rem;
    padding: 0.75rem 1rem;
    border-top: 1px solid #e2e8f0;
    z-index: 5;
  }
  .btn-nav,
  .btn-flag {
    min-height: 44px;
    touch-action: manipulation;
  }
}

/* ── Tablet Responsive ── */
@media (min-width: 768px) and (max-width: 1024px) {
  .question-panel {
    padding: 1.25rem;
  }
  .option-item {
    padding: 0.875rem 1rem;
    min-height: 44px;
  }
  .opt-key {
    width: 30px;
    height: 30px;
  }
  .btn-nav,
  .btn-flag {
    min-height: 44px;
  }
  .match-select {
    min-height: 44px;
  }
}

/* Custom Audio Player */
.custom-audio-player {
  margin-bottom: 0.75rem;
}
</style>
