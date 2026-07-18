<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import { Phase } from '../../bindings/pocketmind/internal/breakengine/models'
import { ChevronRight } from 'lucide-vue-next'

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

// Long breaks use a calmer teal/green accent; short breaks use the blue focus accent.
const accent = computed(() => (isLong.value
  ? { from: '#5fd8a4', to: '#4fc8c8', glow: 'rgba(95,216,164,0.22)', ring: 'rgba(95,216,164,0.7)' }
  : { from: '#6b8dff', to: '#b89cff', glow: 'rgba(107,141,255,0.2)', ring: 'rgba(107,141,255,0.65)' }))

// Ring: fraction elapsed grows as the break advances.
const fraction = computed(() => {
  if (total.value <= 0) return 0
  return Math.max(0, Math.min(1, 1 - remaining.value / total.value))
})
const R = 146
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
  <div class="overlay" :style="{ '--a-from': accent.from, '--a-to': accent.to, '--a-glow': accent.glow, '--a-ring': accent.ring }">
    <!-- Ambient depth: layered soft glows -->
    <div class="amb amb-a" aria-hidden="true" />
    <div class="amb amb-b" aria-hidden="true" />
    <div class="vignette" aria-hidden="true" />

    <main class="content">
      <p class="label">{{ label }}</p>

      <div class="ring-wrap">
        <svg class="ring" viewBox="0 0 340 340">
          <defs>
            <linearGradient :id="'g'" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" :stop-color="accent.from" />
              <stop offset="100%" :stop-color="accent.to" />
            </linearGradient>
          </defs>
          <circle class="ring-track" cx="170" cy="170" :r="R" />
          <circle
            class="ring-progress"
            cx="170" cy="170" :r="R"
            :stroke-dasharray="CIRC"
            :stroke-dashoffset="dashOffset"
            :stroke="`url(#g)`"
          />
        </svg>
        <div class="clock-wrap">
          <div class="clock tabular-nums">{{ clock }}</div>
          <div class="breathe" aria-hidden="true">
            <span class="breathe__dot" />
          </div>
        </div>
      </div>

      <p class="hint">{{ t('break.hint') }}</p>

      <button class="skip" @click="skip">
        <span>{{ t('break.skip') }}</span>
        <ChevronRight class="size-4" />
      </button>
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
  background: linear-gradient(160deg, #0d1018 0%, #14171f 48%, #0a0c12 100%);
  color: #eef1f6;
  user-select: none;
  overflow: hidden;
}

/* Ambient glows for depth */
.amb {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  pointer-events: none;
}
.amb-a {
  width: 62vmin;
  height: 62vmin;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -52%);
  background: radial-gradient(circle, var(--a-glow), transparent 68%);
  animation: pm-drift 14s ease-in-out infinite alternate;
}
.amb-b {
  width: 44vmin;
  height: 44vmin;
  top: 50%;
  left: 50%;
  transform: translate(-32%, -64%);
  background: radial-gradient(circle, rgba(184, 156, 255, 0.1), transparent 70%);
  animation: pm-drift 18s ease-in-out infinite alternate-reverse;
}
@keyframes pm-drift {
  from { transform: translate(-50%, -52%) scale(1); opacity: 0.9; }
  to   { transform: translate(-46%, -56%) scale(1.08); opacity: 1; }
}
.vignette {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at center, transparent 55%, rgba(0,0,0,0.45) 100%);
  pointer-events: none;
}

.content {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 30px;
  text-align: center;
  animation: pm-rise 0.7s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes pm-rise {
  from { opacity: 0; transform: translateY(12px) scale(0.97); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

.label {
  margin: 0;
  letter-spacing: 0.34em;
  text-transform: uppercase;
  font-size: 12px;
  font-weight: 500;
  color: rgba(238, 241, 246, 0.5);
  padding-left: 0.34em; /* offset for letter-spacing visual centering */
}

.ring-wrap {
  position: relative;
  width: 300px;
  height: 300px;
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
  stroke: rgba(255, 255, 255, 0.06);
  stroke-width: 3;
}
.ring-progress {
  fill: none;
  stroke-width: 4;
  stroke-linecap: round;
  transition: stroke-dashoffset 1s linear;
  filter: drop-shadow(0 0 8px var(--a-ring));
}

.clock-wrap {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
}
.clock {
  font-size: clamp(56px, 11vmin, 92px);
  font-weight: 200;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
  text-shadow: 0 4px 30px rgba(0, 0, 0, 0.45);
}

/* Breathing guide: a dot that expands/contracts on a ~4s breath cycle */
.breathe {
  display: grid;
  place-items: center;
  width: 14px;
  height: 14px;
}
.breathe__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--a-from);
  box-shadow: 0 0 12px var(--a-ring);
  animation: pm-breath 4s ease-in-out infinite;
}
@keyframes pm-breath {
  0%, 100% { transform: scale(0.7); opacity: 0.5; }
  50%      { transform: scale(1.5); opacity: 1; }
}

.hint {
  margin: 0;
  max-width: 32ch;
  font-size: 14px;
  line-height: 1.5;
  color: rgba(238, 241, 246, 0.42);
}

/* Refined glass skip button */
.skip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 18px 9px 20px;
  font-size: 13.5px;
  font-weight: 500;
  font-family: inherit;
  color: rgba(238, 241, 246, 0.78);
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 9999px;
  backdrop-filter: blur(12px);
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}
.skip:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.18);
  color: rgba(238, 241, 246, 0.95);
}
.skip :deep(svg) { transition: transform 0.2s ease; }
.skip:hover :deep(svg) { transform: translateX(2px); }
</style>
