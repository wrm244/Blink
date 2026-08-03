<script setup lang="ts">
import { ChevronRight } from '@lucide/vue'
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/blink'
import { Phase } from '../../bindings/blink/internal/breakengine/models'

const { t, locale } = useI18n()

// ---- rest-screen content (quote + wallpaper) ----
// Fetched from the free xygeng.cn open APIs when a break overlay appears. All
// failures fall back silently to the plain slate backdrop so an offline app is
// never worse than before. The day key keeps the Bing image cached per-calendar
// day (a "today's picture" is pointless refetched a minute later), while the
// quote is refetched each break so it feels fresh.
const quote = ref('')
const quoteMeta = ref('')
const bgUrl = ref('')
const dayKey = new Date().toISOString().slice(0, 10)

function enrichRestScreen() {
  fetchOne()
  fetchBing()
}

async function fetchOne() {
  try {
    const res = await fetch('https://api.xygeng.cn/openapi/one')
    if (!res.ok) return
    const json = await res.json()
    const content = json?.data?.content
    if (typeof content !== 'string' || !content.trim()) return
    quote.value = content.trim()
    const origin = json.data.origin
    const name = json.data.name
    quoteMeta.value = [origin, name].filter(Boolean).join(' · ')
  } catch {
    /* offline — leave the plain backdrop */
  }
}

async function fetchBing() {
  try {
    const cached = sessionStorage.getItem('pm:bing')
    if (cached) {
      const parsed = JSON.parse(cached)
      if (parsed?.day === dayKey && typeof parsed.url === 'string') {
        bgUrl.value = parsed.url
        return
      }
    }
    const res = await fetch('https://api.xygeng.cn/openapi/bing', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ index: 0 }),
    })
    if (!res.ok) return
    const json = await res.json()
    const urls = json?.data?.urls
    if (!Array.isArray(urls) || urls.length === 0) return
    // urls is sorted highest-resolution first, so urls[0] is the 1920×1080
    // variant — the "today's picture" at its best quality.
    const url = urls[0]
    try {
      sessionStorage.setItem('pm:bing', JSON.stringify({ day: dayKey, url }))
    } catch { /* storage full — cache is best-effort */ }
    bgUrl.value = url
  } catch {
    /* offline — leave the plain backdrop */
  }
}

// ---- break engine state (countdown) ----
const phase = ref<string>('')
const remaining = ref(0)
const total = ref(0)
let off: (() => void) | undefined

// ---- wall clock (current date + time, precise to the second) ----
// `now` is refreshed at each whole-second boundary so the displayed seconds
// never jump or stutter. We recompute the timeout to the next boundary on
// every tick instead of a fixed 1 s interval, which would drift under load.
const now = ref(new Date())
let clockTimer: ReturnType<typeof setTimeout> | undefined

onMounted(async () => {
  // --- rest-screen content (quote + wallpaper) ---
  // Fire the network requests before the state binding so the break screen is
  // populated as early as possible (and so a slow binding never delays them).
  enrichRestScreen()

  // --- countdown ---
  const st = await BreakService.GetState()
  phase.value = st.phase
  remaining.value = st.remainingSec
  total.value = st.totalSec
  off = Events.On('blink:tick', (ev: { data: { phase: string; remainingSec: number; totalSec: number } }) => {
    phase.value = ev.data.phase
    remaining.value = ev.data.remainingSec
    total.value = ev.data.totalSec
  })

  // --- wall clock: align ticks to second boundaries ---
  now.value = new Date()
  scheduleClockTick()
})

onUnmounted(() => {
  off?.()
  if (clockTimer !== undefined) clearTimeout(clockTimer)
})

/**
 * Schedules the next wall-clock update at the start of the next second.
 * Using setTimeout (re-scheduled each tick) instead of setInterval keeps
 * the display pinned to real second boundaries — setInterval(1000) drifts
 * because the callback can fire a few ms late and accumulate.
 */
function scheduleClockTick() {
  const msToNextSecond = 1000 - (Date.now() % 1000)
  clockTimer = setTimeout(() => {
    now.value = new Date()
    scheduleClockTick()
  }, msToNextSecond)
}

// ---- derived values ----

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

// Break countdown formatted as M:SS.
const clock = computed(() => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
})

/**
 * Current date + time, localised and precise to the second.
 * Intl.DateTimeFormat is used so the weekday/month names follow the active
 * locale (zh-CN / en). hour12:false gives a 24-hour clock in both locales,
 * which reads cleanly on a rest screen. The formatter is memoized on locale
 * (constructing one does locale negotiation) rather than rebuilt every tick.
 */
const dateFmt = computed(() => {
  const loc = locale.value === 'zh-CN' ? 'zh-CN' : 'en-US'
  return new Intl.DateTimeFormat(loc, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
})
const datetime = computed(() => dateFmt.value.format(now.value))

// The quote block only takes space once the quote has loaded, so the hero
// countdown never jumps around as the request resolves.
const hasQuote = computed(() => quote.value.length > 0)
// date-style quotes (e.g. "腊月廿四：小年") are short and carry a leading
// "：" — long aphorisms with an early colon must not be misdetected.
const isDate = computed(() => {
  const idx = quote.value.indexOf('：')
  return idx > 0 && idx < 12 && quote.value.length < 40
})

function skip() { BreakService.SkipBreak() }
</script>

<template>
  <div class="overlay" :style="{ '--a': accent }">
    <img v-if="bgUrl" :src="bgUrl" class="wallpaper" alt="" aria-hidden="true" />
    <div class="scrim" aria-hidden="true" />
    <div class="vignette" aria-hidden="true" />

    <main class="content">
      <!-- Current date + time: secondary context at the top, very muted. -->
      <p class="datetime tabular-nums">{{ datetime }}</p>

      <!-- One-quote: quiet inspiration, only rendered once loaded. -->
      <div v-if="hasQuote" class="quote">
        <p class="quote__text" :class="{ 'quote__text--date': isDate }">{{ quote }}</p>
        <p v-if="quoteMeta" class="quote__meta">{{ quoteMeta }}</p>
      </div>

      <p class="label">{{ label }}</p>

      <!-- Break countdown: the hero element. -->
      <div class="clock tabular-nums">{{ clock }}</div>

      <div class="progress" aria-hidden="true">
        <div class="progress__fill" :style="{ transform: `scaleX(${fraction})` }" />
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
 * The break countdown is the hero; the wall clock sits above it as quiet
 * context. No SVG ring (avoids any square viewport artifact), no ambient
 * blobs, no gradients on accents. The window is created at ScreenSaver
 * window level (see windows.go) so it covers the Dock and menu bar too.
 */
.overlay {
  position: fixed;
  inset: 0;
  z-index: 2147483647;
  display: flex;
  align-items: center;
  justify-content: center;
  /* Slate gradient fallback — replaced by the wallpaper photo when it loads. */
  background: linear-gradient(160deg, #181a1d 0%, #1f2227 50%, #15171a 100%);
  color: #e8e6e1;
  user-select: none;
  -webkit-user-select: none;
  -webkit-touch-callout: none;
  cursor: default;
  overflow: hidden;
}
/* Belt-and-braces: never show a selection highlight over the photo. */
.overlay ::selection {
  background: transparent;
}

/* Bing "picture of the day" wallpaper: full-bleed cover behind everything. */
.wallpaper {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  z-index: 0;
  animation: pm-wallpaper-in 0.9s ease both;
}
@keyframes pm-wallpaper-in {
  from { opacity: 0; transform: scale(1.03); }
  to   { opacity: 1; transform: scale(1); }
}

/* Dark scrim over the photo so the countdown text stays readable in any
   lighting. Solid gradient, no blur (cheap, and the photo is already soft). */
.scrim {
  position: absolute;
  inset: 0;
  z-index: 1;
  background: linear-gradient(180deg, rgba(8, 10, 12, 0.66) 0%, rgba(8, 10, 12, 0.45) 42%, rgba(8, 10, 12, 0.72) 100%);
  pointer-events: none;
}

/* Single soft vignette for visual focus on the center. */
.vignette {
  position: absolute;
  inset: 0;
  z-index: 2;
  background: radial-gradient(ellipse at center, transparent 58%, rgba(0,0,0,0.35) 100%);
  pointer-events: none;
}

.content {
  position: relative;
  z-index: 3;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 26px;
  text-align: center;
  animation: pm-rise 0.6s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes pm-rise {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* Wall clock: smallest text — quiet context, but bumped above the faintest
   level since it now sits on a photo. */
.datetime {
  margin: 0 0 -12px;
  font-size: 15px;
  font-weight: 500;
  letter-spacing: 0.02em;
  color: rgba(240, 238, 233, 0.6);
  text-shadow: 0 1px 6px rgba(0, 0, 0, 0.5);
}

/* Quote: max two lines, soft drop shadow so it stays legible over the photo.
   date-style quotes (e.g. "腊月廿四：小年") get a slightly larger, calmer
   treatment and drop the attribution line. */
.quote {
  max-width: min(680px, 82vw);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.quote__text {
  margin: 0;
  font-size: 17px;
  font-weight: 500;
  line-height: 1.6;
  letter-spacing: 0.01em;
  color: rgba(250, 248, 244, 0.94);
  text-shadow: 0 1px 10px rgba(0, 0, 0, 0.65), 0 2px 20px rgba(0, 0, 0, 0.4);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.quote__text--date {
  font-size: 22px;
  font-weight: 500;
}
.quote__meta {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.06em;
  color: rgba(240, 238, 233, 0.62);
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.5);
}

.label {
  margin: 0;
  letter-spacing: 0.42em;
  text-transform: uppercase;
  font-size: 14px;
  font-weight: 600;
  color: rgba(240, 238, 233, 0.72);
  text-shadow: 0 1px 8px rgba(0, 0, 0, 0.5);
  padding-left: 0.42em; /* offset for letter-spacing visual centering */
}

/* Hero clock — the dominant element. Medium weight + soft shadow so the thin
   numerals stay crisp over a photo; tabular-nums keeps digits from shifting. */
.clock {
  font-size: clamp(140px, 22vmin, 210px);
  font-weight: 400;
  line-height: 0.9;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  color: #f5f3ef;
  text-shadow: 0 2px 18px rgba(0, 0, 0, 0.55), 0 8px 40px rgba(0, 0, 0, 0.3);
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
  width: 100%;
  border-radius: 9999px;
  background: var(--a);
  transform-origin: left center;
  transition: transform 1s linear;
  will-change: transform;
}

.hint {
  margin: 0;
  max-width: 34ch;
  font-size: 15px;
  font-weight: 500;
  line-height: 1.55;
  color: rgba(240, 238, 233, 0.66);
  text-shadow: 0 1px 6px rgba(0, 0, 0, 0.5);
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
  cursor: pointer;
  user-select: none;
  -webkit-user-select: none;
  transition: background 0.2s ease, color 0.2s ease;
}
.skip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(232, 230, 225, 0.92);
}
.skip :deep(svg) { transition: transform 0.2s ease; }
.skip:hover :deep(svg) { transform: translateX(2px); }
</style>
