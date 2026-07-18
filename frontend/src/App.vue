<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Settings from './views/Settings.vue'
import BreakOverlay from './views/BreakOverlay.vue'
import PreBreakNotice from './views/PreBreakNotice.vue'

// Each window loads the SPA with a fixed hash that selects its view. The hash
// never changes after load, so a one-shot read is enough (no hashchange needed).
const view = ref<'settings' | 'break' | 'notice'>('settings')
onMounted(() => {
  const h = window.location.hash
  if (h === '#break') view.value = 'break'
  else if (h === '#notice') view.value = 'notice'
})
const current = computed(() => {
  switch (view.value) {
    case 'break': return BreakOverlay
    case 'notice': return PreBreakNotice
    default: return Settings
  }
})
</script>

<template>
  <component :is="current" />
</template>
