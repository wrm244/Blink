<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import { Phase } from '../../bindings/pocketmind/internal/breakengine/models'
import { SkipForward } from 'lucide-vue-next'
import GButton from '@/components/GButton.vue'

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

// Ring: fraction elapsed grows as the break advances.
const fraction = computed(() => {
  if (total.value <= 0) return 0
  return Math.max(0, Math.min(1, 1 - remaining.value / total.value))
})
const R = 150
const CIRC = 2 * Math.PI * R
const dashOffset = computed(() => CIRC * (1 - fraction.value))

const clock = computed(() => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
})
function skip() { BreakService.SkipBreak() }
</script>

<template>
  <div class="overlay">
    <div class="glow" aria-hidden="true" />
    <div class="glow glow--2" aria-hidden="true" />
    <main class="content">
      <p class="label">{{ label }}</p>

      <div class="ring-wrap">
        <svg class="ring" viewBox="0 0 340 340">
          <defs>
            <linearGradient id="ringGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#6b8dff" />
              <stop offset="100%" stop-color="#9b7dff" />
            </linearGradient>
          </defs>
          <circle class="ring-track" cx="170" cy="170" :r="R" />
          <circle
            class="ring-progress"
            cx="170" cy="170" :r="R"
            :stroke-dasharray="CIRC"
            :stroke-dashoffset="dashOffset"
            stroke="url(#ringGrad)"
          />
        </svg>
        <div class="clock-wrap">
          <div class="clock tabular-nums">{{ clock }}</div>
          <div class="breathe" aria-hidden="true" />
        </div>
      </div>

      <p class="hint">{{ t('break.hint') }}</p>
      <GButton variant="glass" size="lg" @click="skip">
        <SkipForward class="size-4" />
        {{ t('break.skip') }}
      </GButton>
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
  background: radial-gradient(circle at 50% 40%, #1a1d28 0%, #0c0e14 100%);
  color: #f0f2f6;
  user-select: none;
  overflow: hidden;
}
.glow {
  position: absolute;
  width: 70vmin;
  height: 70vmin;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(107, 141, 255, 0.2), transparent 70%);
  filter: blur(60px);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}
.glow--2 {
  width: 50vmin;
  height: 50vmin;
  background: radial-gradient(circle, rgba(155, 125, 255, 0.14), transparent 70%);
  transform: translate(-30%, -60%);
}
.content {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 26px;
  text-align: center;
  animation: pm-scale-in 0.6s ease both;
}
.label {
  margin: 0;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  font-size: 13px;
  font-weight: 500;
  color: rgba(240, 242, 246, 0.55);
}
.ring-wrap {
  position: relative;
  width: 320px;
  height: 320px;
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
  stroke: rgba(255, 255, 255, 0.07);
  stroke-width: 5;
}
.ring-progress {
  fill: none;
  stroke-width: 6;
  stroke-linecap: round;
  transition: stroke-dashoffset 1s linear;
  filter: drop-shadow(0 0 10px rgba(107, 141, 255, 0.6));
}
.clock-wrap {
  position: relative;
  display: grid;
  place-items: center;
}
.clock {
  font-size: clamp(64px, 13vmin, 104px);
  font-weight: 250;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 4px 30px rgba(0, 0, 0, 0.5);
}
.breathe {
  position: absolute;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(107, 141, 255, 0.8);
  bottom: -28px;
  animation: pm-breathe 2.4s ease-in-out infinite;
}
.hint {
  margin: 0;
  max-width: 34ch;
  font-size: 15px;
  color: rgba(240, 242, 246, 0.45);
}
</style>
