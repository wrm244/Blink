<script setup lang="ts">
import { computed } from 'vue'
import type { ClassValue } from 'clsx'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  class?: ClassValue
  strong?: boolean
}>(), { strong: false })

const classes = computed(() => cn(props.strong ? 'glass-strong' : 'glass', 'rounded-2xl', props.class))
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>
