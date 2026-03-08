<script setup lang="ts">
defineOptions({ inheritAttrs: false })

interface SelectOption {
  value: string | number
  label: string
}

const props = defineProps<{
  label?: string
  error?: string
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  required?: boolean
}>()

const model = defineModel<string | number | null>()

const uid = Math.random().toString(36).slice(2, 9)
const selectId = `select-${uid}`
const errorId = `error-${uid}`
</script>

<template>
  <div class="mb-3">
    <label v-if="label" :for="selectId" class="form-label">
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>
    <select
      :id="selectId"
      v-bind="$attrs"
      v-model="model"
      class="form-select"
      :class="{ 'is-invalid': error }"
      :disabled="disabled"
      :required="required"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="error ? errorId : undefined"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value">
        {{ opt.label }}
      </option>
    </select>
    <div v-if="error" :id="errorId" class="invalid-feedback">{{ error }}</div>
  </div>
</template>
