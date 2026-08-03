<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Settings from './views/Settings.vue'
import BreakOverlay from './views/BreakOverlay.vue'
import PreBreakNotice from './views/PreBreakNotice.vue'

// 每个窗口以固定 hash 加载 SPA，hash 选择对应视图。
// hash 加载后不再变化，一次性读取即可（无需 hashchange 监听）。
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
