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

// Single desaturated accent per break type. Long breaks use a calm gray-green,
// short breaks use a neutral slate. No gradients.
const accent = computed(() => (isLong.value ? '#6b8f7a' : '#8a96a8'))

// Progress: fraction of the break that has elapsed (0 → 1).
const fraction = computed(() => {
  if (total.value <= 0) return 0
  return Math.max(0, Math.min(1, 1 - remaining.value / total.value))
})

const clock = computed(() => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
})
function skip() { BreakService.SkipBreak() }
</script>

<template>
  <div class="overlay" :style="{ '--a': accent }">
    <div class="vignette" aria-hidden="true" />

    <main class="content">
      <p class="label">{{ label }}</p>

      <div class="clock tabular-nums">{{ clock }}</div>

      <div class="progress" aria-hidden="true">
        <div class="progress__fill" :style="{ width: (fraction * 100) + '%' }" />
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
/*
 * Break overlay — full-screen, top-priority rest screen.
 * The number is the hero: huge, thin, centered on a calm dark slate field.
 * No SVG ring (avoids any square viewport artifact), no ambient blobs, no
 * gradients on accents. A single thin progress line beneath the clock is the
 * only chromatic accent. Window is already AlwaysOnTop + Status level +
 * CanJoinAllSpaces (see windows.go), so it sits above everything system-wide.
 */
.overlay {
  position: fixed;
  inset: 0;
  z-index: 2147483647;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(160deg, #181a1d 0%, #1f2227 50%, #15171a 100%);
  color: #e8e6e1;
  user-select: none;
  overflow: hidden;
}

/* Single soft vignette for visual focus on the center. */
.vignette {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at center, transparent 58%, rgba(0,0,0,0.35) 100%);
  pointer-events: none;
}

.content {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 36px;
  text-align: center;
  animation: pm-rise 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes pm-rise {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0); }
}

.label {
  margin: 0;
  letter-spacing: 0.42em;
  text-transform: uppercase;
  font-size: 13px;
  font-weight: 500;
  color: rgba(232, 230, 225, 0.42);
  padding-left: 0.42em; /* offset for letter-spacing visual centering */
}

/* Hero clock — the dominant element. Thin weight keeps it calm despite the
   size; tabular-nums prevents the digits from shifting each second. */
.clock {
  font-size: clamp(160px, 26vmin, 240px);
  font-weight: 200;
  line-height: 0.9;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  color: #f0eee9;
}

/* Thin progress line beneath the clock. Track is barely visible; the fill is
   the only chromatic accent on the screen. */
.progress {
  width: 260px;
  height: 2px;
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.07);
  overflow: hidden;
}
.progress__fill {
  height: 100%;
  border-radius: 9999px;
  background: var(--a);
  transition: width 1s linear;
}

.hint {
  margin: 0;
  max-width: 34ch;
  font-size: 14px;
  line-height: 1.55;
  color: rgba(232, 230, 225, 0.4);
}

/* Skip button: restrained glass pill, sits low and unobtrusive. */
.skip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 9px 18px 9px 20px;
  font-size: 13.5px;
  font-weight: 500;
  font-family: inherit;
  color: rgba(232, 230, 225, 0.72);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 9999px;
  backdrop-filter: blur(12px);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;
}
.skip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(232, 230, 225, 0.92);
}
.skip :deep(svg) { transition: transform 0.2s ease; }
.skip:hover :deep(svg) { transform: translateX(2px); }
</style>
