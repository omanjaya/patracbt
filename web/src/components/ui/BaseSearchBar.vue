<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'

const props = withDefaults(defineProps<{
  placeholder?: string
  debounceMs?: number
}>(), {
  placeholder: 'Cari...',
  debounceMs: 300,
})

const model = defineModel<string>({ default: '' })
const raw = ref(model.value)
let timer: ReturnType<typeof setTimeout> | null = null

watch(raw, (val) => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    model.value = val
  }, props.debounceMs)
})

watch(model, (val) => {
  if (val !== raw.value) raw.value = val
})

function clear() {
  raw.value = ''
  model.value = ''
  if (timer) clearTimeout(timer)
}

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})
</script>

<template>
  <div class="input-group">
    <span class="input-group-text" aria-hidden="true">
      <i class="ti ti-search"></i>
    </span>
    <input
      v-model="raw"
      type="text"
      class="form-control"
      :placeholder="placeholder"
      aria-label="Pencarian"
    />
    <button
      v-if="raw"
      type="button"
      class="btn btn-ghost-secondary input-group-text"
      aria-label="Hapus pencarian"
      @click="clear"
    >
      <i class="ti ti-x"></i>
    </button>
  </div>
</template>
