<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

interface Option {
  value: string
  label: string
  icon?: any
}

const props = defineProps<{
  options: Option[]
  modelValue: string
  class?: string
}>()
const emits = defineEmits<{ 'update:modelValue': [value: string] }>()

function select(v: string) {
  emits('update:modelValue', v)
}

const classes = computed(() =>
  cn('glass inline-flex items-center gap-1 rounded-xl p-1', props.class),
)
</script>

<template>
  <div :class="classes">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      @click="select(opt.value)"
      :class="
        cn(
          'relative inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-[13px] font-medium transition-all duration-200',
          modelValue === opt.value
            ? 'text-[var(--on-accent)] bg-[linear-gradient(135deg,var(--accent),var(--accent-2))] shadow-[0_4px_14px_-4px_var(--accent-soft)]'
            : 'text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-[var(--accent-soft)]',
        )
      "
    >
      <component v-if="opt.icon" :is="opt.icon" class="size-3.5" />
      <span>{{ opt.label }}</span>
    </button>
  </div>
</template>
