<script setup lang="ts">
import { computed } from "vue"
import { cn } from "@/lib/utils"

const props = defineProps<{
  modelValue?: string | number
  type?: string
  class?: string
  placeholder?: string
  min?: number | string
  max?: number | string
  step?: number | string
  disabled?: boolean
}>()

const emits = defineEmits<{ "update:modelValue": [value: string | number] }>()

const value = computed({
  get: () => props.modelValue,
  set: (v) => emits("update:modelValue", v as string | number),
})
</script>

<template>
  <input
    v-model="value"
    :type="type ?? 'text'"
    :placeholder="placeholder"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    :class="
      cn(
        'flex h-9 w-full rounded-md border border-input bg-black/30 px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
