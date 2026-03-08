<script setup lang="ts">
withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  actionLabel?: string
}>(), {
  icon: 'ti-database-off',
})

const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <div class="empty py-5">
    <div class="empty-icon">
      <i class="ti" :class="icon" aria-hidden="true" style="font-size: 3rem;" />
    </div>
    <p class="empty-title mt-3">{{ title }}</p>
    <p v-if="description" class="empty-subtitle text-muted">{{ description }}</p>
    <div v-if="actionLabel" class="empty-action mt-3">
      <button type="button" class="btn btn-primary" @click="emit('action')">
        {{ actionLabel }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.empty-icon {
  opacity: 0.5;
}
</style>
