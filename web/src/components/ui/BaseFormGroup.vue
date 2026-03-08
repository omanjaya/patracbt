<script setup lang="ts">
const props = defineProps<{
  label?: string
  error?: string
  required?: boolean
  id?: string
  hint?: string
}>()

const uid = Math.random().toString(36).slice(2, 9)
const groupId = props.id ?? `form-group-${uid}`
const errorId = `error-${uid}`
const hintId = `hint-${uid}`
</script>

<template>
  <div class="mb-3">
    <label v-if="label" :for="groupId" class="form-label">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <slot
      :id="groupId"
      :error-id="error ? errorId : undefined"
      :hint-id="!error && hint ? hintId : undefined"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="error ? errorId : hint ? hintId : undefined"
    />
    <div v-if="error" :id="errorId" class="invalid-feedback d-block">{{ error }}</div>
    <div v-else-if="hint" :id="hintId" class="form-hint">{{ hint }}</div>
  </div>
</template>
