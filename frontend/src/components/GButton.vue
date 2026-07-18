<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'glass' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg' | 'icon'
  class?: string
  disabled?: boolean
}>(), {
  variant: 'glass',
  size: 'md',
})

const base = 'relative inline-flex items-center justify-center gap-2 font-medium rounded-xl transition-all duration-200 select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/60 disabled:opacity-40 disabled:cursor-not-allowed active:scale-[0.97]'

const variants = {
  // Primary: accent gradient fill, glows on hover.
  primary: 'text-[var(--on-accent)] shadow-[0_6px_20px_-4px_var(--accent-soft)] bg-[linear-gradient(135deg,var(--accent),var(--accent-2))] hover:brightness-110 hover:shadow-[0_8px_28px_-4px_var(--accent-soft)]',
  // Glass: translucent panel button.
  glass: 'glass text-[var(--text)] hover:bg-[var(--glass-bg-strong)]',
  // Ghost: bare, hover highlight.
  ghost: 'text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-[var(--accent-soft)]',
  // Danger.
  danger: 'text-[#ff7a7a] glass hover:bg-[rgba(255,90,90,0.12)]',
}

const sizes = {
  sm: 'h-8 px-3 text-[13px]',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-6 text-[15px]',
  icon: 'h-9 w-9',
}

const classes = computed(() => cn(base, variants[props.variant], sizes[props.size], props.class))
</script>

<template>
  <button :class="classes" :disabled="disabled">
    <slot />
  </button>
</template>
