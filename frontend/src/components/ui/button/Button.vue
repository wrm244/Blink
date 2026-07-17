<script setup lang="ts">
import { computed } from "vue"
import { cn } from "@/lib/utils"
import { buttonVariants, type ButtonVariants } from "."

interface Props {
  variant?: ButtonVariants["variant"]
  size?: ButtonVariants["size"]
  class?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: "default",
  size: "default",
})

const classes = computed(() =>
  cn(buttonVariants({ variant: props.variant, size: props.size }), props.class),
)
</script>

<template>
  <button :class="classes" :disabled="disabled">
    <slot />
  </button>
</template>
