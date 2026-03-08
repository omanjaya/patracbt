<script setup lang="ts">
import BaseModal from './BaseModal.vue'
import BaseButton from './BaseButton.vue'

withDefaults(defineProps<{
  title?: string
  message: string
  confirmLabel?: string
  confirmVariant?: 'primary' | 'danger' | 'warning'
  loading?: boolean
}>(), {
  title: 'Konfirmasi',
  confirmLabel: 'Hapus',
  confirmVariant: 'danger',
})

const emit = defineEmits<{ confirm: []; close: [] }>()
</script>

<template>
  <BaseModal :title="title" size="sm" @close="emit('close')">
    <div class="text-center py-3">
      <i
        class="ti ti-alert-triangle text-warning mb-3 d-block"
        aria-hidden="true"
        style="font-size: 3rem;"
      />
      <p class="mb-0">{{ message }}</p>
    </div>
    <template #footer>
      <BaseButton variant="ghost" class="me-auto" @click="emit('close')">
        Batal
      </BaseButton>
      <BaseButton
        :variant="confirmVariant"
        :loading="loading"
        @click="emit('confirm')"
      >
        {{ confirmLabel }}
      </BaseButton>
    </template>
  </BaseModal>
</template>
