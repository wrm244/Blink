<script setup lang="ts">
// GButton - 通用按钮组件，支持多种视觉风格和尺寸。
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

// 基础类名：所有按钮共用的布局、圆角、过渡和交互态样式
const base = 'relative inline-flex items-center justify-center gap-2 font-medium rounded-xl transition-all duration-200 select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/60 disabled:opacity-40 disabled:cursor-not-allowed active:scale-[0.97]'

// 变体样式
const variants = {
  // Primary：实色强调填充，悬停时微提升
  primary: 'text-[var(--on-accent)] shadow-[0_4px_14px_-4px_var(--accent-soft)] bg-[var(--accent)] hover:brightness-110 hover:shadow-[0_6px_18px_-4px_var(--accent-soft)]',
  // Glass：半透明面板按钮
  glass: 'glass text-[var(--text)] hover:bg-[var(--glass-bg-strong)]',
  // Ghost：无背景，悬停高亮
  ghost: 'text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-[var(--accent-soft)]',
  // Danger：危险操作
  danger: 'text-[#c8554f] glass hover:bg-[rgba(200,85,79,0.12)]',
}

// 尺寸
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
