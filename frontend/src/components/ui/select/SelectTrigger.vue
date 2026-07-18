<script setup lang="ts">
import { SelectTrigger, SelectIcon } from "reka-ui"
import { ChevronDown } from "lucide-vue-next"
import { cn } from "@/lib/utils"

const props = defineProps<{ class?: string; id?: string }>()
</script>

<template>
  <SelectTrigger
    :id="id"
    :class="
      cn(
        'flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-black/30 px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
        props.class,
      )
    "
  >
    <slot />
    <SelectIcon as-child>
      <ChevronDown class="h-4 w-4 opacity-60" />
    </SelectIcon>
  </SelectTrigger>
</template>
