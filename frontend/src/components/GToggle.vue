<script setup lang="ts">
import { computed } from 'vue'
import { SwitchRoot, SwitchThumb } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  modelValue?: boolean
  class?: string
  disabled?: boolean
}>()
const emits = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const checked = computed({
  get: () => !!props.modelValue,
  set: (v: boolean) => emits('update:modelValue', v),
})
</script>

<template>
  <SwitchRoot
    v-model="checked"
    :disabled="disabled"
    :class="
      cn(
        'relative inline-flex h-6 w-10 shrink-0 cursor-pointer items-center rounded-full border border-[var(--glass-border)] transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/60',
        checked ? 'bg-[var(--accent)]' : 'bg-[var(--track)]',
        props.class,
      )
    "
  >
    <SwitchThumb
      class="pointer-events-none block h-5 w-5 rounded-full bg-white shadow-[0_2px_6px_rgba(0,0,0,0.3)] transition-transform duration-200 data-[state=checked]:translate-x-[18px] data-[state=unchecked]:translate-x-[2px]"
    />
  </SwitchRoot>
</template>
