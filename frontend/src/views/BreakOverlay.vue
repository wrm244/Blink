<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import { Phase } from '../../bindings/pocketmind/internal/breakengine/models'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
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
const label = computed(() => (isLong.value ? t('break.longLabel') : t('break.shortLabel')))
// Ring progress: fraction elapsed (grows as the break advances).
const ringFraction = computed(() => {
  if (total.value <= 0) return 0
  return 1 - remaining.value / total.value
})
const RADIUS = 150
const CIRC = 2 * Math.PI * RADIUS
const dashOffset = computed(() => CIRC * (1 - ringFraction.value))

const m = computed(() => Math.floor(remaining.value / 60))
const s = computed(() => remaining.value % 60)
const clock = computed(() => `${m.value}:${String(s.value).padStart(2, '0')}`)

function skip() { BreakService.SkipBreak() }
</script>

<template>
  <div class="overlay">
    <div class="glow" aria-hidden="true" />
    <main class="content">
      <p class="label">{{ label }}</p>

      <div class="ring-wrap">
        <svg class="ring" viewBox="0 0 340 340">
          <circle class="ring-track" cx="170" cy="170" :r="RADIUS" />
          <circle
            class="ring-progress"
            cx="170" cy="170" :r="RADIUS"
            :stroke-dasharray="CIRC"
            :stroke-dashoffset="dashOffset"
          />
        </svg>
        <div class="clock">{{ clock }}</div>
      </div>

      <p class="hint">{{ t('break.hint') }}</p>
      <Button variant="secondary" size="lg" class="backdrop-blur" @click="skip">{{ t('break.skip') }}</Button>
    </main>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 20, 0.55);
  color: #f4f6fb;
  user-select: none;
}
.glow {
  position: absolute;
  width: 70vmin;
  height: 70vmin;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(91, 157, 255, 0.22), transparent 70%);
  filter: blur(50px);
}
.content {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  text-align: center;
}
.label {
  margin: 0;
  letter-spacing: 0.32em;
  text-transform: uppercase;
  font-size: 13px;
  font-weight: 500;
  color: rgba(244, 246, 251, 0.6);
}
.ring-wrap {
  position: relative;
  width: 340px;
  height: 340px;
  display: grid;
  place-items: center;
}
.ring {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}
.ring-track {
  fill: none;
  stroke: rgba(255, 255, 255, 0.08);
  stroke-width: 6;
}
.ring-progress {
  fill: none;
  stroke: url(#grad);
  stroke: #5b9dff;
  stroke-width: 6;
  stroke-linecap: round;
  transition: stroke-dashoffset 1s linear;
  filter: drop-shadow(0 0 8px rgba(91, 157, 255, 0.5));
}
.clock {
  font-size: clamp(64px, 14vmin, 112px);
  font-weight: 250;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 4px 30px rgba(0, 0, 0, 0.4);
}
.hint {
  margin: 0;
  max-width: 34ch;
  font-size: 15px;
  color: rgba(244, 246, 251, 0.48);
}
</style>
