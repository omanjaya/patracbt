<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'danger' | 'warning' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
})
</script>

<template>
  <button
    :type="type"
    class="btn"
    :class="[
      variant === 'primary' ? 'btn-primary' :
      variant === 'secondary' ? 'btn-secondary' :
      variant === 'danger' ? 'btn-danger' :
      variant === 'warning' ? 'btn-warning' : 'btn-ghost-secondary',
      size === 'sm' ? 'btn-sm' : size === 'lg' ? 'btn-lg' : '',
    ]"
    :disabled="disabled || loading"
  >
    <template v-if="loading">
      <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true" />
      <span class="visually-hidden">Memproses...</span>
    </template>
    <slot v-else />
  </button>
</template>
