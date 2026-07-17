<script setup lang="ts">
import { TabsRoot } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<{
  defaultValue?: string
  modelValue?: string
  class?: string
}>()
const emits = defineEmits<{ "update:modelValue": [value: string] }>()
</script>

<template>
  <TabsRoot
    :default-value="defaultValue"
    :model-value="modelValue"
    @update:model-value="(v: string) => emits('update:modelValue', v)"
    :class="cn('w-full', props.class)"
  >
    <slot />
  </TabsRoot>
</template>
