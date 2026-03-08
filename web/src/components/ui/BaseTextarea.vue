<script setup lang="ts">
defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  label?: string
  error?: string
  hint?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  rows?: number
}>(), {
  rows: 3,
})

const model = defineModel<string>()

const uid = Math.random().toString(36).slice(2, 9)
const textareaId = `textarea-${uid}`
const errorId = `error-${uid}`
const hintId = `hint-${uid}`
</script>

<template>
  <div class="mb-3">
    <label v-if="label" :for="textareaId" class="form-label">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <textarea
      :id="textareaId"
      v-bind="$attrs"
      v-model="model"
      class="form-control"
      :class="{ 'is-invalid': error }"
      :placeholder="placeholder"
      :disabled="disabled"
      :required="required"
      :rows="rows"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="error ? errorId : hint ? hintId : undefined"
    />
    <div v-if="error" :id="errorId" class="invalid-feedback">{{ error }}</div>
    <div v-else-if="hint" :id="hintId" class="form-hint">{{ hint }}</div>
  </div>
</template>
