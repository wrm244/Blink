<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { BreakService } from '../../bindings/pocketmind'

const remaining = ref(0)
let off: (() => void) | undefined

onMounted(async () => {
  const st = await BreakService.GetState()
  remaining.value = st.remainingSec
  off = Events.On('pm:tick', (ev: { data: { remainingSec: number } }) => {
    remaining.value = ev.data.remainingSec
  })
})
onUnmounted(() => off?.())

function postpone() {
  BreakService.PostponeBreak()
}
</script>

<template>
  <div class="notice">
    <div class="notice__dot" aria-hidden="true" />
    <div class="notice__body">
      <div class="notice__title">Break in {{ remaining }}s</div>
      <div class="notice__sub">Wrap up your task — your screen will rest soon.</div>
    </div>
    <button class="notice__btn" @click="postpone">Postpone</button>
  </div>
</template>

<style scoped>
.notice {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  height: 100%;
  padding: 0 18px;
  box-sizing: border-box;
  background: rgba(20, 22, 34, 0.72);
  border-radius: 18px;
  color: #f4f6fb;
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Inter', sans-serif;
  user-select: none;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
}
.notice__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ffb454;
  box-shadow: 0 0 12px rgba(255, 180, 84, 0.8);
  flex-shrink: 0;
}
.notice__body {
  flex: 1;
  min-width: 0;
}
.notice__title {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.01em;
}
.notice__sub {
  font-size: 12px;
  color: rgba(244, 246, 251, 0.55);
  margin-top: 2px;
}
.notice__btn {
  padding: 7px 14px;
  font-size: 13px;
  color: rgba(244, 246, 251, 0.85);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 8px;
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s ease;
}
.notice__btn:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
