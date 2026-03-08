<script setup lang="ts">
interface Props {
  /** Variant: text, circle, rect, button */
  variant?: 'text' | 'circle' | 'rect' | 'button'
  /** Width (CSS value, e.g. '100%', '200px') */
  width?: string
  /** Height (CSS value) */
  height?: string
  /** Number of text lines to show */
  lines?: number
  /** Animation: wave or pulse */
  animation?: 'wave' | 'pulse'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'text',
  width: '100%',
  lines: 1,
  animation: 'wave',
})
</script>

<template>
  <!-- Text variant -->
  <div
    v-if="variant === 'text'"
    :class="animation === 'wave' ? 'placeholder-wave' : 'placeholder-glow'"
  >
    <template v-if="lines <= 1">
      <span
        class="placeholder rounded col-12"
        :style="{ width: width, height: height }"
      />
    </template>
    <template v-else>
      <span
        v-for="n in lines"
        :key="n"
        class="placeholder rounded d-block"
        :class="n === lines ? 'mb-0' : 'mb-2'"
        :style="{
          width: n === lines ? '75%' : width,
          height: height,
        }"
      />
    </template>
  </div>

  <!-- Circle variant -->
  <div
    v-else-if="variant === 'circle'"
    :class="animation === 'wave' ? 'placeholder-wave' : 'placeholder-glow'"
  >
    <span
      class="placeholder rounded-circle d-inline-block"
      :style="{
        width: width ?? '3rem',
        height: height ?? width ?? '3rem',
      }"
    />
  </div>

  <!-- Rect variant -->
  <div
    v-else-if="variant === 'rect'"
    :class="animation === 'wave' ? 'placeholder-wave' : 'placeholder-glow'"
  >
    <span
      class="placeholder rounded d-block"
      :style="{
        width: width,
        height: height ?? '100px',
      }"
    />
  </div>

  <!-- Button variant -->
  <div
    v-else-if="variant === 'button'"
    :class="animation === 'wave' ? 'placeholder-wave' : 'placeholder-glow'"
  >
    <span
      class="placeholder rounded-2 d-inline-block"
      :style="{
        width: width ?? '120px',
        height: height ?? '2.25rem',
      }"
    />
  </div>
</template>
