<script setup lang="ts">
import { watch, onBeforeUnmount } from 'vue'

const props = withDefaults(defineProps<{
  type?: 'success' | 'error' | 'warning' | 'info'
  dismissible?: boolean
}>(), {
  type: 'info',
  dismissible: true,
})

const emit = defineEmits<{
  dismiss: []
}>()

const model = defineModel<boolean>({ default: true })

const typeMap: Record<string, string> = {
  success: 'success',
  error: 'danger',
  warning: 'warning',
  info: 'info',
}

const iconMap: Record<string, string> = {
  success: 'ti ti-circle-check',
  error: 'ti ti-alert-circle',
  warning: 'ti ti-alert-triangle',
  info: 'ti ti-info-circle',
}

let autoDismissTimer: ReturnType<typeof setTimeout> | null = null

function dismiss() {
  model.value = false
  emit('dismiss')
}

function clearTimer() {
  if (autoDismissTimer) {
    clearTimeout(autoDismissTimer)
    autoDismissTimer = null
  }
}

watch(model, (visible) => {
  clearTimer()
  if (visible && props.type === 'success') {
    autoDismissTimer = setTimeout(dismiss, 5000)
  }
}, { immediate: true })

onBeforeUnmount(clearTimer)
</script>

<template>
  <div
    v-if="model"
    class="alert"
    :class="[
      `alert-${typeMap[type]}`,
      { 'alert-dismissible': dismissible },
    ]"
    role="alert"
  >
    <div class="d-flex align-items-center gap-2">
      <i :class="iconMap[type]" aria-hidden="true"></i>
      <div>
        <slot />
      </div>
    </div>
    <button
      v-if="dismissible"
      type="button"
      class="btn-close"
      aria-label="Tutup"
      @click="dismiss"
    />
  </div>
</template>
