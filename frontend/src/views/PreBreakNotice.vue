<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/blink'

const { t } = useI18n()
const remaining = ref(0)
let off: (() => void) | undefined

onMounted(async () => {
  const st = await BreakService.GetState()
  remaining.value = st.remainingSec
  off = Events.On('blink:tick', (ev: { data: { remainingSec: number } }) => {
    remaining.value = ev.data.remainingSec
  })
})
onUnmounted(() => off?.())

function postpone() { BreakService.PostponeBreak() }
</script>

<template>
  <div class="notice-wrap">
    <div class="notice">
      <!-- Left accent bar: visual anchor in the macOS notification spirit,
           tinted with the prebreak phase colour. Replaces the old pulsing
           dot + box-shadow which read as a heavy "card" element. -->
      <div class="accent" aria-hidden="true" />

      <!-- Countdown is the hero: large thin numerals (matching the break
           overlay's typography), unit tucked alongside as quiet context. -->
      <div class="countdown tabular-nums">
        <span class="num">{{ remaining }}</span>
        <span class="unit">{{ t('notice.secondsShort') }}</span>
      </div>

      <!-- Heading + sub: plain text, no card-like backing. -->
      <div class="body">
        <div class="title">{{ t('notice.title') }}</div>
        <div class="sub">{{ t('notice.sub') }}</div>
      </div>

      <!-- Postpone: ghost-style text button. No glass card, no border —
           just a quiet affordance that warms on hover. -->
      <button class="postpone" @click="postpone">
        {{ t('notice.postpone') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
/*
 * Pre-break notice — a slim heads-up that slides in at the top of the primary
 * display ~10s before a break begins. The window itself is transparent with
 * macOS native translucency (see windows.go noticeOptions), so the .notice
 * shell only adds a faint tint + hairline border to seat the content on top
 * of the system's vibrancy. No backdrop-filter here: layering CSS blur on top
 * of the window's native blur is what made the old version feel like a heavy
 * card pasted onto the desktop.
 */
.notice-wrap {
  width: 100%;
  height: 100%;
  padding: 6px;
  box-sizing: border-box;
}
.notice {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  height: 100%;
  padding: 0 14px 0 0;
  box-sizing: border-box;
  border-radius: 14px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  color: var(--text);
  user-select: none;
  overflow: hidden;
  animation: pm-fade-up 0.35s ease both;
}

/* Accent bar: 3px vertical strip hugging the left edge. Uses the prebreak
   phase colour so the notice reads as a warning without resorting to a loud
   icon or animated dot. */
.accent {
  width: 3px;
  height: 100%;
  background: var(--phase-prebreak);
  flex-shrink: 0;
}

/* Countdown cluster: number + unit, baseline-aligned. The number matches the
   break overlay's thin-weight aesthetic so the two screens feel related. */
.countdown {
  display: flex;
  align-items: baseline;
  gap: 3px;
  flex-shrink: 0;
  margin-left: 14px;
  color: var(--phase-prebreak);
}
.num {
  font-size: 34px;
  font-weight: 200;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.unit {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-faint);
}

.body {
  flex: 1;
  min-width: 0;
}
.title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.005em;
}
.sub {
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Postpone button: ghost text button. No glass card, no border — just a
   quiet affordance that warms on hover. Matches the restrained pill on the
   break overlay rather than the old GButton glass variant. */
.postpone {
  flex-shrink: 0;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.postpone:hover {
  background: var(--accent-soft);
  color: var(--text);
}
.postpone:active {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
