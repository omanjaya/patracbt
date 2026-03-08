<script setup lang="ts">
import { RouterLink } from 'vue-router'

defineProps<{
  items: { label: string; to?: string }[]
}>()
</script>

<template>
  <nav aria-label="Breadcrumb">
    <ol class="breadcrumb">
      <li
        v-for="(item, index) in items"
        :key="index"
        class="breadcrumb-item"
        :class="{ active: index === items.length - 1 }"
        :aria-current="index === items.length - 1 ? 'page' : undefined"
      >
        <RouterLink v-if="item.to && index !== items.length - 1" :to="item.to">
          {{ item.label }}
        </RouterLink>
        <span v-else>{{ item.label }}</span>
      </li>
    </ol>
  </nav>
</template>
