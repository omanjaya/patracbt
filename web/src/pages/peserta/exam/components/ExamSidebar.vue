<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SafeQuestion } from '@/api/exam.api'

interface QuestionSection {
  bankId: number
  bankName: string
  startIdx: number
  endIdx: number
}

const props = defineProps<{
  questions: SafeQuestion[]
  currentIdx: number
  answersMap: Map<number, unknown>
  flaggedSet: Set<number>
  questionSections: QuestionSection[]
  hasMultipleSections: boolean
  sidebarOpen: boolean
  progressPercent: number
  answeredCount: number
  unansweredCount: number
  flaggedCount: number
  canFinish: boolean
  minTimeRemaining: number
}>()

const emit = defineEmits<{
  goTo: [idx: number]
  toggleSidebar: []
  requestFinish: []
}>()

// Filter state: 'all' | 'unanswered' | 'flagged'
const activeFilter = ref<'all' | 'unanswered' | 'flagged'>('all')

function questionStatus(q: SafeQuestion, idx: number): 'answered' | 'flagged' | 'current' | 'empty' {
  if (props.currentIdx === idx) return 'current'
  if (props.flaggedSet.has(q.id)) return 'flagged'
  if (props.answersMap.has(q.id)) return 'answered'
  return 'empty'
}

function questionIcon(q: SafeQuestion): string {
  if (props.flaggedSet.has(q.id)) return '?'
  if (props.answersMap.has(q.id)) return '\u2713'
  return '-'
}

function questionAriaLabel(q: SafeQuestion, idx: number): string {
  const num = idx + 1
  if (props.flaggedSet.has(q.id)) return `Soal ${num}, ragu-ragu`
  if (props.answersMap.has(q.id)) return `Soal ${num}, sudah dijawab`
  return `Soal ${num}, belum dijawab`
}

function isVisible(q: SafeQuestion): boolean {
  if (activeFilter.value === 'all') return true
  if (activeFilter.value === 'unanswered') return !props.answersMap.has(q.id)
  if (activeFilter.value === 'flagged') return props.flaggedSet.has(q.id)
  return true
}

const hasVisibleQuestions = computed(() => props.questions.some((q) => isVisible(q)))
</script>

<template>
  <!-- Sidebar overlay for mobile -->
  <div v-if="sidebarOpen" class="sidebar-overlay" @click="emit('toggleSidebar')" />

  <!-- Navigation sidebar -->
  <aside class="nav-sidebar" :class="{ 'nav-sidebar--open': sidebarOpen }">
    <div class="nav-sidebar-header">
      <span>Navigasi Soal</span>
      <button class="btn-sidebar-close" @click="emit('toggleSidebar')">
        <i class="ti ti-x" style="font-size: 18px;" />
      </button>
    </div>

    <!-- Progress mini -->
    <div class="sidebar-progress">
      <div class="sidebar-progress-bar">
        <div class="sidebar-progress-fill" :style="{ width: progressPercent + '%' }" />
      </div>
      <span class="sidebar-progress-text">{{ answeredCount }}/{{ questions.length }}</span>
    </div>

    <!-- Filter toggles -->
    <div class="filter-toggles">
      <button class="filter-btn" :class="{ 'filter-btn--active': activeFilter === 'all' }" @click="activeFilter = 'all'">Semua</button>
      <button class="filter-btn" :class="{ 'filter-btn--active': activeFilter === 'unanswered' }" @click="activeFilter = 'unanswered'">Belum Dijawab</button>
      <button class="filter-btn" :class="{ 'filter-btn--active': activeFilter === 'flagged' }" @click="activeFilter = 'flagged'">Ragu-ragu</button>
    </div>

    <!-- Question grid with section labels -->
    <div class="num-grid-wrapper">
      <template v-if="hasMultipleSections">
        <div v-for="sec in questionSections" :key="sec.bankId" class="section-group">
          <div class="section-label">{{ sec.bankName }}</div>
          <div class="num-grid">
            <template v-for="i in (sec.endIdx - sec.startIdx + 1)" :key="questions[sec.startIdx + i - 1]!.id">
              <button
                v-if="isVisible(questions[sec.startIdx + i - 1]!)"
                class="num-btn"
                :class="questionStatus(questions[sec.startIdx + i - 1]!, sec.startIdx + i - 1)"
                :data-idx="sec.startIdx + i - 1"
                :aria-label="questionAriaLabel(questions[sec.startIdx + i - 1]!, sec.startIdx + i - 1)"
                @click="emit('goTo', sec.startIdx + i - 1)"
              ><span class="num-icon">{{ questionIcon(questions[sec.startIdx + i - 1]!) }}</span>{{ sec.startIdx + i }}</button>
            </template>
          </div>
        </div>
      </template>
      <template v-else>
        <div class="num-grid">
          <template v-for="(q, i) in questions" :key="q.id">
            <button
              v-if="isVisible(q)"
              class="num-btn"
              :class="questionStatus(q, i)"
              :data-idx="i"
              :aria-label="questionAriaLabel(q, i)"
              @click="emit('goTo', i)"
            ><span class="num-icon">{{ questionIcon(q) }}</span>{{ i + 1 }}</button>
          </template>
        </div>
      </template>
      <div v-if="!hasVisibleQuestions" class="empty-filter-state">
        <small>Tidak ada soal yang sesuai filter</small>
      </div>
    </div>

    <div class="legend">
      <div class="legend-item"><span class="legend-dot current" />Saat ini</div>
      <div class="legend-item"><span class="legend-dot answered" />Dijawab</div>
      <div class="legend-item"><span class="legend-dot flagged" />Ragu-ragu</div>
      <div class="legend-item"><span class="legend-dot empty" />Belum</div>
    </div>
    <div class="nav-sidebar-summary">
      <div class="summary-row">
        <span>Dijawab</span>
        <strong class="text-success">{{ answeredCount }}</strong>
      </div>
      <div class="summary-row">
        <span>Belum</span>
        <strong class="text-muted">{{ unansweredCount }}</strong>
      </div>
      <div class="summary-row">
        <span>Ragu-ragu</span>
        <strong class="text-warning">{{ flaggedCount }}</strong>
      </div>
    </div>
    <p v-if="!canFinish" class="min-time-notice">
      Tombol selesai akan aktif dalam {{ Math.ceil(minTimeRemaining) }} menit
    </p>
    <button class="btn-finish-side" @click="emit('requestFinish')" :disabled="!canFinish">Selesaikan Ujian</button>
  </aside>
</template>

<style scoped>
/* Sidebar Overlay (mobile) */
.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 39;
}
@media (max-width: 1024px) {
  .sidebar-overlay { display: block; }
}

/* Navigation Sidebar */
.nav-sidebar {
  width: 260px;
  background: #fff;
  border-left: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  flex-shrink: 0;
}
@media (max-width: 767px) {
  .nav-sidebar {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    top: auto;
    width: 100%;
    max-height: 70vh;
    z-index: 40;
    border-radius: 16px 16px 0 0;
    border-left: none;
    border-top: 1px solid #e2e8f0;
    box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.15);
    transform: translateY(100%);
    transition: transform 0.3s ease;
    touch-action: manipulation;
    overscroll-behavior: contain;
  }
  .nav-sidebar--open {
    transform: translateY(0);
  }
  .nav-sidebar-header {
    padding: 0.5rem 1rem;
    position: relative;
  }
  .nav-sidebar-header::before {
    content: '';
    position: absolute;
    top: 6px;
    left: 50%;
    transform: translateX(-50%);
    width: 36px;
    height: 4px;
    background: #cbd5e1;
    border-radius: 2px;
  }
  .num-grid {
    grid-template-columns: repeat(8, 1fr);
    gap: 4px;
  }
  .num-btn {
    font-size: 0.65rem;
    border-radius: 6px;
    min-height: 36px;
  }
  .num-grid-wrapper {
    padding: 0.5rem 0.75rem;
    max-height: 35vh;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
  .filter-toggles {
    padding: 0.375rem 0.75rem 0;
    overflow-x: auto;
    flex-wrap: nowrap;
    -webkit-overflow-scrolling: touch;
  }
  .filter-btn {
    flex: 0 0 auto;
    min-height: 32px;
    padding: 0.3rem 0.625rem;
  }
  .legend {
    padding: 0.375rem 0.75rem;
    gap: 0.375rem 0.75rem;
  }
  .nav-sidebar-summary {
    padding: 0.5rem 0.75rem;
    flex-direction: row;
    justify-content: space-around;
    gap: 0.5rem;
  }
  .summary-row {
    flex-direction: column;
    align-items: center;
    font-size: 0.7rem;
    gap: 0.125rem;
  }
  .btn-finish-side {
    margin: 0.5rem 0.75rem 0.75rem;
    min-height: 48px;
    font-size: 0.85rem;
    touch-action: manipulation;
  }
}
.nav-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  font-weight: 700;
  font-size: 0.85rem;
  color: #1e293b;
  border-bottom: 1px solid #e2e8f0;
  flex-shrink: 0;
}
.btn-sidebar-close {
  display: none;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: #f1f5f9;
  border-radius: 6px;
  color: #475569;
  cursor: pointer;
}
.btn-sidebar-close:hover { background: #e2e8f0; }
@media (max-width: 1024px) {
  .btn-sidebar-close { display: flex; }
}

/* Sidebar progress */
.sidebar-progress {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem 0.5rem;
}
.sidebar-progress-bar {
  flex: 1;
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}
.sidebar-progress-fill {
  height: 100%;
  background: var(--tblr-primary, #4f46e5);
  border-radius: 3px;
  transition: width 0.3s ease;
}
.sidebar-progress-text {
  font-size: 0.7rem;
  font-weight: 700;
  color: #64748b;
  white-space: nowrap;
}

/* Filter toggles */
.filter-toggles {
  display: flex;
  gap: 4px;
  padding: 0.5rem 1rem 0;
}
.filter-btn {
  flex: 1;
  padding: 0.3rem 0.25rem;
  font-size: 0.6rem;
  font-weight: 600;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  transition: all 0.15s;
  text-align: center;
  white-space: nowrap;
}
.filter-btn:hover { background: #f1f5f9; }
.filter-btn--active {
  background: var(--tblr-primary, #4f46e5);
  color: #fff;
  border-color: var(--tblr-primary, #4f46e5);
}

/* Question number grid */
.num-grid-wrapper {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 1rem;
}
.empty-filter-state {
  text-align: center;
  color: #94a3b8;
  padding: 1rem 0.5rem;
}
.section-group {
  margin-bottom: 0.75rem;
}
.section-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: #64748b;
  padding: 0.25rem 0;
  margin-bottom: 0.375rem;
  border-bottom: 1px dashed #e2e8f0;
}
.num-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 6px;
  margin-bottom: 0.5rem;
}
.num-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  border: 2px solid #e2e8f0;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
  background: #fff;
  color: #94a3b8;
  transition: all 0.15s;
}
.num-icon {
  position: absolute;
  top: 1px;
  right: 2px;
  font-size: 0.5rem;
  line-height: 1;
  opacity: 0.7;
}
.num-btn:hover {
  border-color: #c7d2fe;
  background: #f5f3ff;
}
.num-btn.current {
  border-color: var(--tblr-primary, #4f46e5);
  background: var(--tblr-primary, #4f46e5);
  color: #fff;
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.2);
}
.num-btn.answered {
  border-color: #86efac;
  background: #dcfce7;
  color: #166534;
}
.num-btn.flagged {
  border-color: #fdba74;
  background: #fff7ed;
  color: #9a3412;
}
.num-btn.empty {
  border-color: #e2e8f0;
  background: #fff;
  color: #94a3b8;
}

/* Legend */
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  padding: 0.5rem 1rem;
  border-top: 1px solid #f1f5f9;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.65rem;
  color: #64748b;
}
.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  border: 2px solid;
}
.legend-dot.current {
  border-color: var(--tblr-primary, #4f46e5);
  background: var(--tblr-primary, #4f46e5);
}
.legend-dot.answered {
  border-color: #86efac;
  background: #dcfce7;
}
.legend-dot.flagged {
  border-color: #fdba74;
  background: #fff7ed;
}
.legend-dot.empty {
  border-color: #e2e8f0;
  background: #fff;
}

/* Sidebar summary */
.nav-sidebar-summary {
  padding: 0.75rem 1rem;
  border-top: 1px solid #f1f5f9;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.8rem;
  color: #475569;
}
.text-success { color: #16a34a; }
.text-warning { color: #d97706; }
.text-muted { color: #94a3b8; }

.min-time-notice {
  font-size: 0.7rem;
  color: #dc2626;
  padding: 0 1rem;
  margin: 0;
}
.btn-finish-side {
  margin: 0.75rem 1rem 1rem;
  padding: 0.5rem;
  font-size: 0.8rem;
  font-weight: 700;
  border: none;
  border-radius: 8px;
  background: #dc2626;
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-finish-side:hover:not(:disabled) {
  background: #b91c1c;
}
.btn-finish-side:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Tablet Responsive ── */
@media (min-width: 768px) and (max-width: 1024px) {
  .nav-sidebar {
    position: fixed;
    top: 0;
    right: -260px;
    bottom: 0;
    width: 260px;
    z-index: 40;
    box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
    transition: right 0.25s ease;
  }
  .nav-sidebar--open {
    right: 0;
  }
  .num-btn {
    min-height: 36px;
  }
  .filter-btn {
    min-height: 32px;
  }
  .btn-finish-side {
    min-height: 44px;
  }
}
</style>
