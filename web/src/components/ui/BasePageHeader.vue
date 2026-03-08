<script setup lang="ts">
import BaseBreadcrumb from './BaseBreadcrumb.vue'

defineProps<{
  title: string
  subtitle?: string
  breadcrumbs?: { label: string; to?: string }[]
}>()
</script>

<template>
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <BaseBreadcrumb v-if="breadcrumbs?.length" :items="breadcrumbs" />
        <h2 class="page-title">{{ title }}</h2>
        <p v-if="subtitle" class="text-muted mb-0">{{ subtitle }}</p>
      </div>
      <div class="col-auto ms-auto d-print-none d-flex gap-2">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
