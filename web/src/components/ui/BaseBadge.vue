<script setup lang="ts">
const props = withDefaults(defineProps<{
  variant?: 'default' | 'success' | 'warning' | 'danger' | 'info'
}>(), { variant: 'default' })

const colorMap: Record<string, string> = {
  default: 'secondary',
  success: 'success',
  warning: 'warning',
  danger: 'danger',
  info: 'info',
}
</script>

<template>
  <span class="badge" :class="`bg-${colorMap[props.variant]}-lt`">
    <slot />
  </span>
</template>
