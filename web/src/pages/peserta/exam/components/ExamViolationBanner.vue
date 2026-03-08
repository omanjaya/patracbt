<script setup lang="ts">
defineProps<{
  showViolation: boolean
  violationMessage: string
  violationCount: number
  showSaveFailed: boolean
  showOffline: boolean
  timerWarningBanner: string
}>()

const emit = defineEmits<{
  dismissViolation: []
  dismissTimerWarning: []
}>()
</script>

<template>
  <!-- Violation warning banner -->
  <div v-if="showViolation" class="violation-banner" role="alert" aria-live="assertive">
    <i class="ti ti-alert-circle"></i>
    <span>Peringatan! {{ violationMessage }} ({{ violationCount }}x). Pelanggaran berlebihan dapat membatalkan ujian Anda.</span>
    <button class="violation-close" @click="emit('dismissViolation')">&#x2715;</button>
  </div>

  <!-- Save failed persistent banner -->
  <div v-if="showSaveFailed" class="save-failed-banner">
    <i class="ti ti-alert-triangle"></i>
    <span>Jawaban belum tersimpan, mencoba ulang...</span>
  </div>

  <!-- Offline persistent banner -->
  <div v-if="showOffline" class="offline-banner">
    <i class="ti ti-wifi-off"></i>
    <span>Koneksi terputus. Jawaban akan disimpan saat online kembali.</span>
  </div>

  <!-- Timer warning banner -->
  <div v-if="timerWarningBanner" class="timer-warning-banner">
    <i class="ti ti-clock-exclamation"></i>
    <span>{{ timerWarningBanner }}</span>
    <button class="timer-warning-close" @click="emit('dismissTimerWarning')">&#x2715;</button>
  </div>
</template>

<style scoped>
/* Violation Banner */
.violation-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #fef2f2;
  border-bottom: 2px solid #fecaca;
  color: #991b1b;
  font-size: 0.8rem;
  font-weight: 500;
  z-index: 21;
  animation: flash-banner 0.3s ease;
}
@keyframes flash-banner {
  0% { background: #dc2626; color: #fff; }
  100% { background: #fef2f2; color: #991b1b; }
}
.violation-close {
  margin-left: auto;
  border: none;
  background: none;
  color: #991b1b;
  cursor: pointer;
  font-size: 0.85rem;
}

/* Save Failed Banner */
.save-failed-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #fef2f2;
  border-bottom: 2px solid #fecaca;
  color: #991b1b;
  font-size: 0.8rem;
  font-weight: 500;
  z-index: 21;
}

/* Offline Banner */
.offline-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #fef3c7;
  border-bottom: 2px solid #fde68a;
  color: #92400e;
  font-size: 0.8rem;
  font-weight: 500;
  z-index: 21;
}

/* Timer Warning Banner */
.timer-warning-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #fff7ed;
  border-bottom: 2px solid #fdba74;
  color: #9a3412;
  font-size: 0.85rem;
  font-weight: 600;
  z-index: 21;
  animation: flash-banner 0.5s ease;
}
.timer-warning-close {
  margin-left: auto;
  border: none;
  background: none;
  color: #9a3412;
  cursor: pointer;
  font-size: 0.85rem;
}

/* ── Mobile Responsive ── */
@media (max-width: 767px) {
  .violation-banner,
  .save-failed-banner,
  .offline-banner,
  .timer-warning-banner {
    font-size: 0.75rem;
    padding: 0.5rem 0.75rem;
  }
  .violation-close,
  .timer-warning-close {
    min-width: 32px;
    min-height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    touch-action: manipulation;
  }
}
</style>
