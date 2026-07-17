<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { BreakService } from '../../bindings/pocketmind'
import { Phase } from '../../bindings/pocketmind/internal/breakengine/models'

const phase = ref<string>('')
const remaining = ref(0)
const total = ref(0)
let off: (() => void) | undefined

onMounted(async () => {
  const st = await BreakService.GetState()
  phase.value = st.phase
  remaining.value = st.remainingSec
  total.value = st.totalSec
  off = Events.On('pm:tick', (ev: { data: { phase: string; remainingSec: number; totalSec: number } }) => {
    phase.value = ev.data.phase
    remaining.value = ev.data.remainingSec
    total.value = ev.data.totalSec
  })
})
onUnmounted(() => off?.())

const isLong = computed(() => phase.value === Phase.PhaseLongBreak)
const label = computed(() => (isLong.value ? 'Long break' : 'Eye break'))
const clock = computed(() => formatClock(remaining.value))
const ringDash = computed(() => {
  if (total.value <= 0) return 0
  return (remaining.value / total.value) * 100
})

function formatClock(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
function skip() {
  BreakService.SkipBreak()
}
</script>

<template>
  <div class="overlay">
    <div class="overlay__glow" aria-hidden="true" />
    <main class="overlay__content">
      <p class="overlay__label">{{ label }}</p>
      <div class="overlay__clock">{{ clock }}</div>
      <p class="overlay__hint">Look at something about 20 feet away and relax your eyes.</p>
      <button class="overlay__skip" @click="skip">Skip break</button>
    </main>
    <!-- Progress ring as a thin bar along the bottom. -->
    <div class="overlay__progress" aria-hidden="true">
      <div class="overlay__progress-bar" :style="{ width: ringDash + '%' }" />
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  /* Semi-opaque dim over the translucent (blurred) window backdrop. */
  background: rgba(8, 10, 20, 0.55);
  color: #f4f6fb;
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Inter', sans-serif;
  user-select: none;
}
.overlay__glow {
  position: absolute;
  width: 60vmin;
  height: 60vmin;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(86, 156, 214, 0.28), transparent 70%);
  filter: blur(40px);
}
.overlay__content {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  text-align: center;
}
.overlay__label {
  margin: 0;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  font-size: 14px;
  color: rgba(244, 246, 251, 0.62);
}
.overlay__clock {
  font-size: clamp(96px, 22vmin, 240px);
  font-weight: 260;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 4px 40px rgba(0, 0, 0, 0.4);
}
.overlay__hint {
  margin: 0;
  max-width: 32ch;
  font-size: 16px;
  color: rgba(244, 246, 251, 0.5);
}
.overlay__skip {
  margin-top: 10px;
  padding: 10px 22px;
  font-size: 14px;
  color: rgba(244, 246, 251, 0.8);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 10px;
  cursor: pointer;
  backdrop-filter: blur(10px);
  transition: background 0.15s ease;
}
.overlay__skip:hover {
  background: rgba(255, 255, 255, 0.16);
}
.overlay__progress {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 4px;
  background: rgba(255, 255, 255, 0.08);
}
.overlay__progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #5b9dff, #8b7dff);
  transition: width 1s linear;
}
</style>
