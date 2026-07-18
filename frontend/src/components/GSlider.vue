<script setup lang="ts">
import { computed } from 'vue'
import { SliderRoot, SliderTrack, SliderRange, SliderThumb } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  modelValue?: number
  min?: number
  max?: number
  step?: number
  class?: string
  disabled?: boolean
}>()
const emits = defineEmits<{ 'update:modelValue': [value: number] }>()

// reka-ui SliderRoot uses a number[] model; surface a plain number instead.
const arr = computed({
  get: () => (props.modelValue != null ? [props.modelValue] : [props.min ?? 0]),
  set: (v: number[]) => emits('update:modelValue', v[0] ?? 0),
})
</script>

<template>
  <SliderRoot
    v-model="arr"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    :class="cn('relative flex w-full touch-none select-none items-center h-5', props.class)"
  >
    <SliderTrack class="relative h-1.5 w-full grow overflow-hidden rounded-full bg-[var(--track)]">
      <SliderRange
        class="absolute h-full rounded-full bg-[linear-gradient(90deg,var(--accent),var(--accent-2))] shadow-[0_0_12px_var(--accent-soft)]"
      />
    </SliderTrack>
    <SliderThumb
      class="block h-4 w-4 rounded-full bg-[var(--thumb)] shadow-[0_2px_8px_rgba(0,0,0,0.3)] ring-1 ring-[var(--glass-border)] transition-transform hover:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]"
    />
  </SliderRoot>
</template>
