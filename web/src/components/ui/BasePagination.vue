<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  page: number
  totalPages: number
  total: number
  perPage: number
}>()

const emit = defineEmits<{
  change: [page: number]
  'update:perPage': [value: number]
}>()

const from = computed(() => (props.page - 1) * props.perPage + 1)
const to = computed(() => Math.min(props.page * props.perPage, props.total))

const pages = computed(() => {
  const range: (number | '...')[] = []
  const total = props.totalPages
  if (total <= 7) {
    for (let i = 1; i <= total; i++) range.push(i)
  } else {
    range.push(1)
    if (props.page > 3) range.push('...')
    for (let i = Math.max(2, props.page - 1); i <= Math.min(total - 1, props.page + 1); i++) range.push(i)
    if (props.page < total - 2) range.push('...')
    range.push(total)
  }
  return range
})
</script>

<template>
  <nav aria-label="Pagination" class="d-flex align-items-center flex-wrap gap-2 p-3 border-top">
    <p class="m-0 text-muted small">
      Menampilkan <strong>{{ from }}&ndash;{{ to }}</strong> dari <strong>{{ total }}</strong> data
    </p>
    <div class="d-flex align-items-center gap-2 ms-3">
      <label class="text-muted small text-nowrap m-0" for="per-page-select">Tampilkan</label>
      <select
        id="per-page-select"
        class="form-select form-select-sm"
        style="width: auto"
        :value="perPage"
        @change="emit('update:perPage', Number(($event.target as HTMLSelectElement).value))"
      >
        <option v-for="opt in [10, 20, 50, 100]" :key="opt" :value="opt">{{ opt }}</option>
      </select>
      <span class="text-muted small text-nowrap">per halaman</span>
    </div>
    <ul class="pagination pagination-sm m-0 ms-auto">
      <li class="page-item" :class="{ disabled: page <= 1 }">
        <button
          type="button"
          class="page-link"
          :disabled="page <= 1"
          :aria-disabled="page <= 1"
          aria-label="Halaman sebelumnya"
          @click="page > 1 && emit('change', page - 1)"
        >
          <i class="ti ti-chevron-left" aria-hidden="true" />
        </button>
      </li>
      <template v-for="p in pages" :key="p">
        <li v-if="p !== '...'" class="page-item" :class="{ active: p === page }">
          <button
            type="button"
            class="page-link"
            :aria-label="`Halaman ${p}`"
            :aria-current="p === page ? 'page' : undefined"
            @click="emit('change', p as number)"
          >
            {{ p }}
          </button>
        </li>
        <li v-else class="page-item disabled">
          <span class="page-link" aria-hidden="true">&hellip;</span>
        </li>
      </template>
      <li class="page-item" :class="{ disabled: page >= totalPages }">
        <button
          type="button"
          class="page-link"
          :disabled="page >= totalPages"
          :aria-disabled="page >= totalPages"
          aria-label="Halaman berikutnya"
          @click="page < totalPages && emit('change', page + 1)"
        >
          <i class="ti ti-chevron-right" aria-hidden="true" />
        </button>
      </li>
    </ul>
  </nav>
</template>
