<script setup lang="ts">
import { SelectPortal, SelectContent, SelectViewport, SelectScrollUpButton, SelectScrollDownButton } from "reka-ui"
import { ChevronUp, ChevronDown } from "lucide-vue-next"
import { cn } from "@/lib/utils"

const props = defineProps<{ class?: string; position?: "item-aligned" | "popper" }>()
</script>

<template>
  <SelectPortal>
    <SelectContent
      :position="position ?? 'popper'"
      :class="
        cn(
          'relative z-50 max-h-72 min-w-[8rem] overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
          props.class,
        )
      "
    >
      <SelectScrollUpButton class="flex cursor-default items-center justify-center py-1">
        <ChevronUp class="h-4 w-4" />
      </SelectScrollUpButton>
      <SelectViewport
        :class="cn('p-1', position === 'popper' && 'h-[var(--reka-select-trigger-height)] w-full min-w-[var(--reka-select-trigger-width)]')"
      >
        <slot />
      </SelectViewport>
      <SelectScrollDownButton class="flex cursor-default items-center justify-center py-1">
        <ChevronDown class="h-4 w-4" />
      </SelectScrollDownButton>
    </SelectContent>
  </SelectPortal>
</template>
