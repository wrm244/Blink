<script setup lang="ts">
import { computed } from "vue"
import { SelectRoot } from "reka-ui"

const props = defineProps<{
  modelValue?: string
  defaultValue?: string
  disabled?: boolean
}>()
const emits = defineEmits<{ "update:modelValue": [value: string] }>()

const value = computed({
  get: () => props.modelValue,
  set: (v) => emits("update:modelValue", v as string),
})
</script>

<template>
  <SelectRoot v-model="value" :default-value="defaultValue" :disabled="disabled">
    <slot />
  </SelectRoot>
</template>
