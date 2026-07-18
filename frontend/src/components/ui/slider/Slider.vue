<script setup lang="ts">
import { computed } from "vue"
import { SliderRoot, SliderTrack, SliderRange, SliderThumb } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<{
  modelValue?: number[]
  min?: number
  max?: number
  step?: number
  class?: string
  disabled?: boolean
}>()

const emits = defineEmits<{ "update:modelValue": [value: number[]] }>()

const value = computed({
  get: () => props.modelValue,
  set: (v) => emits("update:modelValue", v as number[]),
})
</script>

<template>
  <SliderRoot
    v-model="value"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    :class="
      cn(
        'relative flex w-full touch-none select-none items-center data-[disabled]:opacity-50',
        props.class,
      )
    "
  >
    <SliderTrack class="relative h-1.5 w-full grow overflow-hidden rounded-full bg-secondary">
      <SliderRange class="absolute h-full bg-primary" />
    </SliderTrack>
    <SliderThumb
      class="block h-4 w-4 rounded-full border-2 border-primary bg-background shadow transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background hover:border-primary/70"
    />
  </SliderRoot>
</template>
