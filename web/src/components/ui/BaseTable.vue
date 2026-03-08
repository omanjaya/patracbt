<script setup lang="ts">
import { computed } from 'vue'

export interface TableColumn {
  key: string
  label: string
  width?: string
  sortable?: boolean
  /** Hide column below this breakpoint: 'sm' | 'md' | 'lg' | 'xl' */
  hideOn?: 'sm' | 'md' | 'lg' | 'xl'
}

function hideClass(col: TableColumn): Record<string, boolean> {
  if (!col.hideOn) return {}
  return {
    'd-none': true,
    [`d-${col.hideOn}-table-cell`]: true,
  }
}

const props = withDefaults(defineProps<{
  columns: TableColumn[]
  loading?: boolean
  empty?: string
  rowCount?: number
  sortBy?: string
  sortDir?: 'asc' | 'desc' | ''
  selectable?: boolean
  selected?: number[]
}>(), {
  sortBy: '',
  sortDir: '',
  selectable: false,
  selected: () => [],
})

const emit = defineEmits<{
  sort: [payload: { key: string; dir: 'asc' | 'desc' | '' }]
  'update:selected': [ids: number[]]
}>()

const visibleColumns = computed(() => {
  return props.columns
})

const isAllSelected = computed(() => {
  if (!props.selectable || !props.rowCount || props.rowCount === 0) return false
  return props.selected.length >= props.rowCount
})

const isSomeSelected = computed(() => {
  if (!props.selectable) return false
  return props.selected.length > 0 && !isAllSelected.value
})

function handleSort(col: TableColumn) {
  if (!col.sortable) return
  let nextDir: 'asc' | 'desc' | '' = 'asc'
  if (props.sortBy === col.key) {
    if (props.sortDir === 'asc') nextDir = 'desc'
    else if (props.sortDir === 'desc') nextDir = ''
    else nextDir = 'asc'
  }
  emit('sort', { key: nextDir ? col.key : '', dir: nextDir })
}

function sortIcon(col: TableColumn): string {
  if (!col.sortable) return ''
  if (props.sortBy === col.key && props.sortDir === 'asc') return 'ti-sort-ascending'
  if (props.sortBy === col.key && props.sortDir === 'desc') return 'ti-sort-descending'
  return 'ti-arrows-sort'
}

function toggleSelectAll() {
  if (isAllSelected.value) {
    emit('update:selected', [])
  } else {
    // Parent is responsible for providing the full list via selected prop
    // Emit empty to signal "select all" — parent handles it
    emit('update:selected', [])
  }
}

function toggleRow(id: number) {
  const idx = props.selected.indexOf(id)
  if (idx >= 0) {
    emit('update:selected', props.selected.filter((v) => v !== id))
  } else {
    emit('update:selected', [...props.selected, id])
  }
}

const isEmpty = computed(() => {
  if (props.loading) return false
  if (props.rowCount === undefined) return false // let slot handle emptiness
  return props.rowCount === 0
})

const colSpan = computed(() => {
  let count = props.columns.length
  if (props.selectable) count += 1
  return count
})

defineExpose({ toggleRow, toggleSelectAll, isAllSelected, isSomeSelected, hideClass })
</script>

<template>
  <div class="table-responsive">
    <table class="table table-vcenter card-table">
      <thead>
        <tr>
          <th v-if="selectable" style="width: 1%">
            <input
              type="checkbox"
              class="form-check-input m-0 align-middle"
              :checked="isAllSelected"
              :indeterminate="isSomeSelected"
              aria-label="Pilih semua baris"
              @change="toggleSelectAll"
            />
          </th>
          <th
            v-for="col in visibleColumns"
            :key="col.key"
            :style="col.width ? `width:${col.width}` : ''"
            :class="[{ 'cursor-pointer user-select-none': col.sortable }, hideClass(col)]"
            @click="handleSort(col)"
          >
            <span class="d-inline-flex align-items-center gap-1">
              {{ col.label }}
              <i
                v-if="col.sortable"
                class="ti opacity-50"
                :class="sortIcon(col)"
                aria-hidden="true"
              />
            </span>
          </th>
        </tr>
      </thead>
      <tbody>
        <template v-if="loading">
          <tr v-for="n in 5" :key="n">
            <td v-if="selectable">
              <div class="placeholder-glow"><span class="placeholder col-12" /></div>
            </td>
            <td v-for="col in columns" :key="col.key" :class="hideClass(col)">
              <div class="placeholder-glow"><span class="placeholder col-8" /></div>
            </td>
          </tr>
        </template>
        <slot v-else-if="!isEmpty" />
        <tr v-if="isEmpty">
          <td :colspan="colSpan" class="text-center text-muted py-5">
            <slot name="empty">
              <i class="ti ti-database-off fs-4 mb-2 d-block opacity-50" aria-hidden="true" />
              {{ empty ?? 'Tidak ada data' }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.cursor-pointer {
  cursor: pointer;
}
</style>
