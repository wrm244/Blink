<script setup lang="ts">
import { computed } from "vue"
import { Separator } from "reka-ui"
import { cn } from "@/lib/utils"

const props = withDefaults(
  defineProps<{ class?: string; orientation?: "horizontal" | "vertical" }>(),
  { orientation: "horizontal" },
)

const classes = computed(() =>
  cn("shrink-0 bg-border", props.orientation === "vertical" ? "h-full w-px" : "h-px w-full", props.class),
)
</script>

<template>
  <Separator :orientation="orientation" :class="classes" />
</template>
