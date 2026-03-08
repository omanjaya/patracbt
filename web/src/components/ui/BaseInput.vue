<script setup lang="ts">
defineOptions({ inheritAttrs: false })

const props = defineProps<{
  label?: string
  error?: string
  hint?: string
}>()

const model = defineModel<string | number>()

const uid = Math.random().toString(36).slice(2, 9)
const inputId = `input-${uid}`
const errorId = `error-${uid}`
const hintId = `hint-${uid}`
</script>

<template>
  <div class="mb-3">
    <label v-if="label" :for="inputId" class="form-label">{{ label }}</label>
    <input
      :id="inputId"
      v-bind="$attrs"
      v-model="model"
      class="form-control"
      :class="{ 'is-invalid': error }"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="error ? errorId : hint ? hintId : undefined"
    />
    <div v-if="error" :id="errorId" class="invalid-feedback">{{ error }}</div>
    <div v-else-if="hint" :id="hintId" class="form-hint">{{ hint }}</div>
  </div>
</template>
