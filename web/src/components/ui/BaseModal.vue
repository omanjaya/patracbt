<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, onUnmounted, nextTick } from 'vue'

const props = withDefaults(defineProps<{
  title?: string
  size?: 'sm' | 'md' | 'lg'
  fullscreenMobile?: boolean
}>(), { size: 'md', fullscreenMobile: false })

const emit = defineEmits<{ close: [] }>()

const modalRef = ref<HTMLElement | null>(null)
const titleId = `modal-title-${Math.random().toString(36).slice(2, 9)}`
const triggerElement = ref<Element | null>(null)
const isVisible = ref(false)

function getFocusableElements(): HTMLElement[] {
  if (!modalRef.value) return []
  return Array.from(
    modalRef.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  )
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }
  if (e.key === 'Tab') {
    const focusable = getFocusableElements()
    if (focusable.length === 0) {
      e.preventDefault()
      return
    }
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey) {
      if (document.activeElement === first && last) {
        e.preventDefault()
        last.focus()
      }
    } else {
      if (document.activeElement === last && first) {
        e.preventDefault()
        first.focus()
      }
    }
  }
}

onMounted(async () => {
  triggerElement.value = document.activeElement
  document.body.style.overflow = 'hidden'
  document.addEventListener('keydown', handleKeydown)
  await nextTick()
  isVisible.value = true
  await nextTick()
  const focusable = getFocusableElements()
  if (focusable.length > 0) {
    focusable[0]!.focus()
  }
})

onBeforeUnmount(() => {
  if (triggerElement.value && triggerElement.value instanceof HTMLElement) {
    triggerElement.value.focus()
  }
})

onUnmounted(() => {
  document.body.style.overflow = ''
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div
        v-if="isVisible"
        class="modal modal-blur show d-block modal-backdrop-custom"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @click.self="emit('close')"
      >
        <div
          ref="modalRef"
          class="modal-dialog modal-dialog-centered modal-wrapper"
          :class="[
            size === 'lg' ? 'modal-lg' : size === 'sm' ? 'modal-sm' : '',
            fullscreenMobile ? 'modal-fullscreen-md-down' : '',
          ]"
        >
          <div class="modal-content">
            <div class="modal-header">
              <h5 :id="titleId" class="modal-title">{{ title }}</h5>
              <button type="button" class="btn-close" @click="emit('close')" aria-label="Close" />
            </div>
            <div class="modal-body">
              <slot />
            </div>
            <div v-if="$slots.footer" class="modal-footer">
              <slot name="footer" />
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-backdrop-custom {
  background: rgba(var(--tblr-body-bg-rgb, 0, 0, 0), 0.5);
}
.modal-wrapper {
  z-index: 1080;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.modal-fade-enter-active .modal-wrapper,
.modal-fade-leave-active .modal-wrapper {
  transition: transform 0.2s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-from .modal-wrapper {
  transform: scale(0.95);
}
.modal-fade-leave-to .modal-wrapper {
  transform: scale(0.95);
}
</style>
