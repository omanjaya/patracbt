<script setup lang="ts">
defineProps<{
  examTitle: string
  timerFormatted: string
  timerIsWarning: boolean
  timerIsDanger: boolean
  timerIsCritical: boolean
  saving: boolean
  lastSaved: boolean
  saveFailed: boolean
  isOnline: boolean
  violationCount: number
  maxViolations: number | string
  minTimeRemaining: number
  minTimeRemainingFormatted: string
  showShortcutHint: boolean
  answeredCount: number
  totalQuestions: number
  canFinish: boolean
  sidebarOpen: boolean
  timerIsPaused: boolean
}>()

const emit = defineEmits<{
  requestFinish: []
  toggleShortcutHint: [value: boolean]
  toggleSidebar: []
}>()
</script>

<template>
  <header class="exam-header">
    <div class="exam-header-left">
      <button class="btn-sidebar-toggle" @click="emit('toggleSidebar')" title="Navigasi Soal" aria-label="Buka navigasi soal">
        <i class="ti ti-layout-sidebar fs-4" />
      </button>
      <div class="exam-title">{{ examTitle }}</div>
    </div>
    <div class="exam-header-center">
      <div class="timer-group">
        <span class="network-dot" :class="isOnline ? 'network-online' : 'network-offline'" :title="isOnline ? 'Online' : 'Offline'"></span>
        <div class="timer" :class="{ 'timer--warning': timerIsWarning, 'timer--danger': timerIsDanger, 'timer--critical': timerIsCritical, 'timer--pulse': timerIsWarning, 'timer--paused': timerIsPaused }">
          {{ timerFormatted }}
        </div>
        <span v-if="timerIsPaused" class="paused-badge" title="Timer dijeda - menunggu koneksi">
          <i class="ti ti-player-pause"></i> Timer Dijeda
        </span>
        <span v-if="minTimeRemaining > 0" class="min-time-badge" title="Waktu minimal pengerjaan">
          Minimal: {{ minTimeRemainingFormatted }}
        </span>
        <span v-if="violationCount > 0" class="violation-badge" title="Jumlah pelanggaran">
          Pelanggaran: {{ violationCount }}/{{ maxViolations }}
        </span>
      </div>
    </div>
    <div class="exam-header-right">
      <!-- Keyboard shortcut hint -->
      <div class="shortcut-hint-wrapper">
        <button
          class="btn-shortcut-hint"
          title="Pintasan Keyboard"
          aria-label="Pintasan keyboard"
          @click="emit('toggleShortcutHint', !showShortcutHint)"
          @mouseenter="emit('toggleShortcutHint', true)"
          @mouseleave="emit('toggleShortcutHint', false)"
        >
          <i class="ti ti-keyboard" />
        </button>
        <div v-if="showShortcutHint" class="shortcut-tooltip">
          <div class="shortcut-tooltip-title">Pintasan Keyboard</div>
          <div class="shortcut-row"><kbd>&larr;</kbd> <kbd>&rarr;</kbd> <span>Soal sebelum / berikut</span></div>
          <div class="shortcut-row"><kbd>1</kbd> - <kbd>9</kbd> <span>Pilih opsi jawaban</span></div>
          <div class="shortcut-row"><kbd>F</kbd> <span>Tandai ragu-ragu</span></div>
          <div class="shortcut-row"><kbd>Ctrl</kbd>+<kbd>S</kbd> <span>Simpan jawaban</span></div>
          <div class="shortcut-row"><kbd>Esc</kbd> <span>Tutup modal</span></div>
        </div>
      </div>
      <!-- Save indicator -->
      <div class="save-indicator-wrap" aria-live="polite">
        <span class="save-indicator" v-if="saving">
          <span class="save-spinner" />
          <span class="save-text">Menyimpan...</span>
        </span>
        <span class="save-done" v-else-if="lastSaved">
          <span class="save-dot save-dot--ok"></span>
          <span class="save-text"><i class="ti ti-circle-check"></i> Tersimpan</span>
        </span>
        <span class="save-error" v-else-if="saveFailed">
          <span class="save-dot save-dot--fail"></span>
          <span class="save-text"><i class="ti ti-circle-x"></i> Gagal</span>
        </span>
      </div>
      <span v-if="violationCount > 0" class="violation-badge-mobile" :title="`Pelanggaran: ${violationCount}/${maxViolations}`">{{ violationCount }}</span>
      <span class="progress-text">{{ answeredCount }}/{{ totalQuestions }} dijawab</span>
      <button class="btn-finish" @click="emit('requestFinish')" :disabled="!canFinish" aria-label="Selesaikan ujian">Selesai</button>
    </div>
  </header>
</template>

<style scoped>
.exam-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  z-index: 20;
  gap: 0.75rem;
  flex-shrink: 0;
}
.exam-header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}
.btn-sidebar-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #475569;
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s;
}
.btn-sidebar-toggle:hover {
  background: #f1f5f9;
}
@media (max-width: 1024px) {
  .btn-sidebar-toggle { display: flex; }
}
.exam-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.exam-header-center {
  flex-shrink: 0;
}
.timer-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.timer {
  font-size: 1.15rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  padding: 0.25rem 0.75rem;
  border-radius: 8px;
  background: #f1f5f9;
  color: #334155;
}
.timer--warning {
  background: #fef3c7;
  color: #92400e;
}
.timer--danger {
  background: #fee2e2;
  color: #991b1b;
  animation: pulse-danger 1s infinite;
}
@keyframes pulse-danger {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}
.timer--pulse {
  animation: pulse-warning 1.5s ease-in-out infinite;
}
@keyframes pulse-warning {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}
.timer--critical {
  background: #7f1d1d;
  color: #fecaca;
  animation: pulse-critical 0.5s ease-in-out infinite;
}
@keyframes pulse-critical {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.08); opacity: 0.8; }
}
.network-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.network-online {
  background: #16a34a;
  box-shadow: 0 0 4px rgba(22, 163, 74, 0.5);
}
.network-offline {
  background: #dc2626;
  box-shadow: 0 0 4px rgba(220, 38, 38, 0.5);
  animation: blink-dot 1s infinite;
}
@keyframes blink-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
.min-time-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 6px;
  background: #dbeafe;
  color: #1e40af;
  white-space: nowrap;
}
.timer--paused {
  background: #fef9c3;
  color: #854d0e;
  animation: pulse-paused 1.5s ease-in-out infinite;
}
@keyframes pulse-paused {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
.paused-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.65rem;
  font-weight: 700;
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
  background: #fef08a;
  color: #854d0e;
  border: 1px solid #facc15;
  white-space: nowrap;
  animation: pulse-paused 1.5s ease-in-out infinite;
}
.paused-badge i {
  font-size: 0.75rem;
}
.violation-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 6px;
  background: #fef2f2;
  color: #dc2626;
  white-space: nowrap;
}
.exam-header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}
.save-indicator {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #64748b;
}
.save-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid #e2e8f0;
  border-top-color: var(--tblr-primary, #4f46e5);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.save-done {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #16a34a;
  font-weight: 500;
}
.save-error {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  color: #dc2626;
  font-weight: 500;
}
.progress-text {
  font-size: 0.8rem;
  color: #64748b;
  font-weight: 500;
  white-space: nowrap;
}
.save-indicator-wrap {
  display: flex;
  align-items: center;
}
.save-dot {
  display: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.save-dot--ok { background: #16a34a; box-shadow: 0 0 4px rgba(22,163,74,0.5); }
.save-dot--fail { background: #dc2626; box-shadow: 0 0 4px rgba(220,38,38,0.5); }
.violation-badge-mobile {
  display: none;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  font-size: 0.65rem;
  font-weight: 700;
  border-radius: 10px;
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}
@media (max-width: 767px) {
  .exam-header {
    padding: 0.375rem 0.5rem;
    gap: 0.375rem;
    touch-action: manipulation;
  }
  .progress-text { display: none; }
  .save-text { display: none; }
  .save-dot { display: block; }
  .save-indicator .save-spinner { display: block; }
  .exam-title { max-width: 80px; font-size: 0.75rem; }
  .min-time-badge { display: none; }
  .violation-badge { display: none; }
  .violation-badge-mobile { display: flex; }
  .shortcut-hint-wrapper { display: none; }
  .paused-badge { font-size: 0.55rem; padding: 0.15rem 0.4rem; }
  .timer { font-size: 0.95rem; padding: 0.2rem 0.5rem; }
  .exam-header-right { gap: 0.375rem; }
  .btn-finish {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
    min-height: 36px;
  }
  .save-indicator-wrap {
    min-width: 16px;
  }
}

/* Tablet */
@media (min-width: 768px) and (max-width: 1024px) {
  .exam-header {
    padding: 0.5rem 0.75rem;
  }
  .timer { font-size: 1.05rem; }
  .btn-finish {
    min-height: 40px;
    padding: 0.375rem 1rem;
  }
}
.btn-finish {
  padding: 0.375rem 1rem;
  font-size: 0.8rem;
  font-weight: 600;
  border: none;
  border-radius: 8px;
  background: #dc2626;
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}
.btn-finish:hover:not(:disabled) {
  background: #b91c1c;
}
.btn-finish:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Keyboard Shortcut Hint */
.shortcut-hint-wrapper {
  position: relative;
}
.btn-shortcut-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  flex-shrink: 0;
}
.btn-shortcut-hint:hover {
  background: #f1f5f9;
  border-color: #cbd5e1;
  color: #475569;
}
.shortcut-tooltip {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 240px;
  background: #1e293b;
  color: #e2e8f0;
  border-radius: 10px;
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  z-index: 60;
  animation: fade-in-down 0.15s ease;
}
@keyframes fade-in-down {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}
.shortcut-tooltip-title {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #94a3b8;
  margin-bottom: 0.5rem;
  padding-bottom: 0.375rem;
  border-bottom: 1px solid #334155;
}
.shortcut-row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0;
  line-height: 1.4;
}
.shortcut-row span {
  margin-left: auto;
  color: #cbd5e1;
  font-size: 0.7rem;
  text-align: right;
}
.shortcut-tooltip kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 20px;
  padding: 0 0.3rem;
  font-family: inherit;
  font-size: 0.65rem;
  font-weight: 700;
  color: #f1f5f9;
  background: #334155;
  border: 1px solid #475569;
  border-radius: 4px;
  line-height: 1;
}
</style>
