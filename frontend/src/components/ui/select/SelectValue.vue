<script setup lang="ts">
import { SelectValue } from "reka-ui"
const props = defineProps<{ placeholder?: string; class?: string }>()
</script>

<template>
  <SelectValue :placeholder="placeholder" :class="props.class" />
</template>
