<script setup lang="ts">
import { useToastStore } from '../../stores/toast.store'

const toast = useToastStore()

const iconMap: Record<string, string> = {
  success: 'ti-circle-check',
  error: 'ti-alert-circle',
  warning: 'ti-alert-triangle',
  info: 'ti-info-circle',
}

const typeClass: Record<string, string> = {
  success: 'bg-success-lt text-success',
  error: 'bg-danger-lt text-danger',
  warning: 'bg-warning-lt text-warning',
  info: 'bg-info-lt text-info',
}

const progressColor: Record<string, string> = {
  success: 'bg-success',
  error: 'bg-danger',
  warning: 'bg-warning',
  info: 'bg-info',
}
</script>

<template>
  <Teleport to="body">
    <div
      class="toast-container position-fixed bottom-0 end-0 p-3"
      style="z-index:9999"
      aria-live="polite"
      aria-atomic="true"
    >
      <TransitionGroup name="toast-slide">
        <div
          v-for="t in toast.toasts"
          :key="t.id"
          class="toast show mb-2 overflow-hidden"
          :class="typeClass[t.type]"
          :style="{ '--duration': `${t.duration}ms` }"
          role="alert"
          @mouseenter="toast.pause(t.id)"
          @mouseleave="toast.resume(t.id)"
        >
          <div class="d-flex align-items-center gap-2 p-3">
            <i class="ti" :class="iconMap[t.type]" aria-hidden="true"></i>
            <span class="me-auto">{{ t.message }}</span>
            <button
              type="button"
              class="btn-close ms-2"
              aria-label="Tutup notifikasi"
              @click="toast.remove(t.id)"
            ></button>
          </div>
          <div class="toast-progress">
            <div class="toast-progress-bar" :class="progressColor[t.type]"></div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-progress {
  height: 3px;
  width: 100%;
  position: relative;
}

.toast-progress-bar {
  height: 100%;
  width: 100%;
  animation: toast-shrink var(--duration, 5000ms) linear forwards;
  transform-origin: left;
}

.toast:hover .toast-progress-bar {
  animation-play-state: paused;
}

@keyframes toast-shrink {
  from {
    transform: scaleX(1);
  }
  to {
    transform: scaleX(0);
  }
}

/* TransitionGroup animations */
.toast-slide-enter-active {
  transition: all 0.3s ease-out;
}
.toast-slide-leave-active {
  transition: all 0.25s ease-in;
}
.toast-slide-enter-from {
  opacity: 0;
  transform: translateX(100%);
}
.toast-slide-leave-to {
  opacity: 0;
  transform: translateX(50%);
}
.toast-slide-move {
  transition: transform 0.25s ease;
}
</style>
