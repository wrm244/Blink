<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import type { Settings } from '../../bindings/pocketmind/internal/config/models'
import type { State } from '../../bindings/pocketmind/internal/breakengine/models'
import { Eye, Play, Pause, Clock, Bell, Languages, Check, Sun, Moon, Monitor, Sparkles, Keyboard, Info, Timer, Coffee, Settings as Settings2, RotateCcw } from 'lucide-vue-next'
import GButton from '@/components/GButton.vue'
import GlassPanel from '@/components/GlassPanel.vue'
import GSlider from '@/components/GSlider.vue'
import GToggle from '@/components/GToggle.vue'
import { setLocale, type Locale } from '@/i18n'
import { applyTheme, type Theme } from '@/theme'

const { t, locale } = useI18n()

const s = reactive({} as Settings)
const state = ref<State>({} as State)
const dirty = ref(false)
const saved = ref(false)
const tab = ref('timing')
let off: (() => void) | undefined

onMounted(async () => {
  Object.assign(s, await BreakService.GetSettings())
  applyTheme((s.theme as Theme) || 'system')
  if (s.language === 'zh-CN' || s.language === 'en') {
    setLocale(s.language)
    locale.value = s.language
  }
  state.value = await BreakService.GetState()
  off = Events.On('pm:tick', (ev: { data: State }) => { state.value = ev.data })
})
onUnmounted(() => off?.())

const onboarded = computed(() => s.onboarded === true)
const isPaused = computed(() => state.value.paused || state.value.phase === 'idle')
const isBreak = computed(() => state.value.phase === 'shortbreak' || state.value.phase === 'longbreak')

function touch() { dirty.value = true; saved.value = false }
function changeLanguage(v: string) { setLocale(v as Locale); locale.value = v; s.language = v; touch() }
function changeTheme(v: string) { s.theme = v; applyTheme(v as Theme); touch() }

async function save() {
  await BreakService.SaveSettings({ ...s })
  dirty.value = false
  saved.value = true
}
async function completeOnboarding() {
  s.onboarded = true
  await BreakService.SaveSettings({ ...s })
  await BreakService.CompleteOnboarding()
  dirty.value = false
}
function startBreak() { BreakService.StartBreakNow() }
function togglePause() { isPaused.value ? BreakService.Resume() : BreakService.Pause() }
function reset() { BreakService.Reset() }

function setNum(field: keyof Settings, v: number) { (s as any)[field] = v; touch() }

const progress = computed(() => {
  const total = state.value.totalSec
  const rem = state.value.remainingSec
  if (!total || total <= 0) return 0
  return Math.max(0, Math.min(1, 1 - rem / total))
})

const heroTitle = computed(() => {
  switch (state.value.phase) {
    case 'focusing': return t('hero.focusing')
    case 'shortbreak': case 'longbreak': return t('hero.resting')
    case 'paused': case 'idle': return t('hero.paused')
    case 'prebreak': return t('hero.resting')
    default: return t('hero.startFocus')
  }
})
const heroSub = computed(() => {
  switch (state.value.phase) {
    case 'focusing': case 'prebreak': return t('status.breakIn', { sec: state.value.remainingSec })
    case 'shortbreak': case 'longbreak': return t('status.remaining') + ' ' + fmt(state.value.remainingSec)
    case 'paused': case 'idle': return t('status.paused')
    default: return t('onboarding.intro2')
  }
})
const phaseColor = computed(() => {
  switch (state.value.phase) {
    case 'shortbreak': case 'longbreak': return '#5fd8a4'
    case 'prebreak': return '#ffb454'
    case 'paused': case 'idle': return '#9aa2b1'
    default: return '#6b8dff'
  }
})
const heroTime = computed(() => fmt(state.value.remainingSec))
function fmt(sec: number): string {
  const m = Math.floor(sec / 60)
  const ss = sec % 60
  return `${m}:${String(ss).padStart(2, '0')}`
}

// Drawer nav grouped into sections for better organisation.
const navGeneral = computed(() => [
  { value: 'timing', label: t('nav.timing'), icon: Timer },
  { value: 'options', label: t('nav.options'), icon: Sparkles },
  { value: 'shortcuts', label: t('nav.shortcuts'), icon: Keyboard },
  { value: 'about', label: t('nav.about'), icon: Info },
])

// Cycle stats shown in the drawer footer.
const shortDone = computed(() => state.value.shortBreakCount ?? 0)
const breaksUntilLong = computed(() => state.value.breaksUntilLong ?? 0)
const cycleRunning = computed(() => !!state.value.phase && state.value.phase !== 'idle')

const themeOptions = [
  { value: 'system', label: '', icon: Monitor },
  { value: 'light', label: '', icon: Sun },
  { value: 'dark', label: '', icon: Moon },
]
const langOptions = [
  { value: 'zh-CN', label: '中' },
  { value: 'en', label: 'EN' },
]
function themeLabel(v: string) { return t('options.theme' + (v === 'system' ? 'System' : v === 'light' ? 'Light' : 'Dark')) }

const timingRows = computed(() => [
  { key: 'focusDurationMin' as const, label: t('timing.focusDuration'), desc: t('timing.focusDesc'), value: s.focusDurationMin, unit: t('timing.minutes'), min: 5, max: 60, step: 1 },
  { key: 'shortBreakDurationSec' as const, label: t('timing.shortBreak'), desc: t('timing.shortDesc'), value: s.shortBreakDurationSec, unit: t('timing.seconds'), min: 5, max: 120, step: 5 },
  { key: 'longBreakDurationMin' as const, label: t('timing.longBreak'), desc: t('timing.longDesc'), value: s.longBreakDurationMin, unit: t('timing.minutes'), min: 1, max: 20, step: 1 },
  { key: 'longBreakInterval' as const, label: t('timing.longBreakEvery'), desc: t('timing.everyDesc'), value: s.longBreakInterval, unit: t('timing.breaks'), min: 1, max: 10, step: 1 },
  { key: 'preBreakWarningSec' as const, label: t('timing.preBreakWarning'), desc: t('timing.preDesc'), value: s.preBreakWarningSec ?? 0, unit: t('timing.seconds'), min: 0, max: 60, step: 5 },
  { key: 'idleThresholdMin' as const, label: t('timing.idlePause'), desc: t('timing.idleDesc'), value: s.idleThresholdMin, unit: t('timing.minutes'), min: 1, max: 30, step: 1 },
])

const shortcutRows = computed(() => [
  { key: 'shortcutStartBreak' as const, label: t('shortcuts.startBreak'), icon: Clock, value: s.shortcutStartBreak, placeholder: 'Cmd+Shift+B' },
  { key: 'shortcutSkipBreak' as const, label: t('shortcuts.skipBreak'), icon: Play, value: s.shortcutSkipBreak, placeholder: 'Cmd+Shift+S' },
  { key: 'shortcutPostponeBreak' as const, label: t('shortcuts.postponeBreak'), icon: Clock, value: s.shortcutPostponeBreak, placeholder: 'Cmd+Shift+P' },
  { key: 'shortcutPreferences' as const, label: t('shortcuts.preferences'), icon: Settings2, value: s.shortcutPreferences, placeholder: 'Cmd+Shift+,' },
])
</script>

<template>
  <div class="root">
    <div class="backdrop" aria-hidden="true">
      <div class="glow glow-a" />
      <div class="glow glow-b" />
    </div>

    <div class="layout">
      <!-- MAIN (left) -->
      <main class="main">
        <!-- HERO -->
        <GlassPanel strong class="hero" :class="{ 'hero--break': isBreak }">
          <div class="hero__top">
            <div class="hero__brand">
              <div class="hero__logo"><Eye class="size-5" /></div>
              <div>
                <h1 class="hero__title">{{ t('app.name') }}</h1>
                <p class="hero__sub">
                  <span class="hero__dot" :style="{ background: phaseColor, boxShadow: `0 0 8px ${phaseColor}` }" />
                  <span class="hero__label">{{ heroTitle }}</span>
                </p>
              </div>
            </div>
            <div v-if="onboarded && state.phase" class="hero__clock">
              <span class="hero__time tabular-nums">{{ heroTime }}</span>
            </div>
          </div>

          <div v-if="onboarded && state.phase && !isPaused" class="hero__progress">
            <div class="hero__progress-bar" :style="{ width: (progress * 100) + '%', background: `linear-gradient(90deg, ${phaseColor}, var(--accent-2))` }" />
          </div>

          <div class="hero__bar">
            <span class="hero__hint">{{ heroSub }}</span>
            <div v-if="onboarded" class="hero__actions">
              <GButton variant="glass" size="sm" @click="togglePause">
                <component :is="isPaused ? Play : Pause" class="size-3.5" />
                {{ isPaused ? t('actions.resume') : t('actions.pause') }}
              </GButton>
              <GButton variant="primary" size="sm" @click="startBreak">
                <Clock class="size-3.5" />
                {{ t('actions.breakNow') }}
              </GButton>
            </div>
          </div>

          <!-- Onboarding (first run): inline in hero area -->
          <div v-if="!onboarded" class="onboard">
            <div class="onboard__intro">{{ t('onboarding.intro') }}</div>
            <div class="onboard__row">
              <div class="onboard__field">
                <span class="onboard__label"><Languages class="size-3.5" />{{ t('onboarding.language') }}</span>
                <div class="seg-row">
                  <button v-for="o in langOptions" :key="o.value" class="seg" :class="{ 'seg--on': locale === o.value }" @click="changeLanguage(o.value)">{{ o.label }}</button>
                </div>
              </div>
              <div class="onboard__field">
                <span class="onboard__label"><Sun class="size-3.5" />{{ t('options.theme') }}</span>
                <div class="seg-row">
                  <button v-for="o in themeOptions" :key="o.value" class="seg seg--icon" :class="{ 'seg--on': (s.theme || 'system') === o.value }" @click="changeTheme(o.value)" :title="themeLabel(o.value)">
                    <component :is="o.icon" class="size-3.5" />
                  </button>
                </div>
              </div>
            </div>
            <GButton variant="primary" size="lg" class="onboard__start" @click="completeOnboarding">
              <Play class="size-4" />
              {{ t('onboarding.startButton') }}
            </GButton>
          </div>
        </GlassPanel>

        <!-- PANEL CONTENT (onboarded only) -->
        <div v-if="onboarded" class="panels">
          <!-- TIMING -->
          <section v-show="tab === 'timing'" class="panel-stack">
            <GlassPanel v-for="row in timingRows" :key="row.key" class="timing-row">
              <div class="timing-row__head">
                <div>
                  <div class="timing-row__label">{{ row.label }}</div>
                  <div class="timing-row__desc">{{ row.desc }}</div>
                </div>
                <div class="timing-row__value tabular-nums">{{ row.value }}<span class="timing-row__unit">{{ row.unit }}</span></div>
              </div>
              <GSlider :model-value="row.value" :min="row.min" :max="row.max" :step="row.step" @update:model-value="(v: number) => setNum(row.key, v)" />
            </GlassPanel>
          </section>

          <!-- OPTIONS -->
          <section v-show="tab === 'options'" class="panel-stack">
            <GlassPanel class="opt-row">
              <div class="opt-row__left">
                <span class="opt-row__label"><Coffee class="size-4" />{{ t('options.longBreaks') }}</span>
                <span class="opt-row__desc">{{ t('options.longBreaksDesc') }}</span>
              </div>
              <GToggle :model-value="s.enableLongBreaks" @update:model-value="(v: boolean) => { s.enableLongBreaks = v; touch() }" />
            </GlassPanel>
            <GlassPanel class="opt-row">
              <div class="opt-row__left">
                <span class="opt-row__label"><Bell class="size-4" />{{ t('options.sound') }}</span>
                <span class="opt-row__desc">{{ t('options.soundDesc') }}</span>
              </div>
              <GToggle :model-value="s.soundEnabled" @update:model-value="(v: boolean) => { s.soundEnabled = v; touch() }" />
            </GlassPanel>
            <GlassPanel class="opt-row">
              <div class="opt-row__left">
                <span class="opt-row__label"><Play class="size-4" />{{ t('options.autoStart') }}</span>
                <span class="opt-row__desc">{{ t('options.autoStartDesc') }}</span>
              </div>
              <GToggle :model-value="s.autoStart" @update:model-value="(v: boolean) => { s.autoStart = v; touch() }" />
            </GlassPanel>
          </section>

          <!-- SHORTCUTS -->
          <section v-show="tab === 'shortcuts'" class="panel-stack">
            <GlassPanel v-for="row in shortcutRows" :key="row.key" class="shortcut-row">
              <span class="shortcut-row__label"><component :is="row.icon" class="size-4" />{{ row.label }}</span>
              <input
                class="keycap"
                :value="row.value"
                :placeholder="row.placeholder"
                @input="(e) => { (s as any)[row.key] = (e.target as HTMLInputElement).value; touch() }"
              />
            </GlassPanel>
            <p class="hint">{{ t('shortcuts.hint', { code: 'Cmd+Shift+B' }) }}</p>
          </section>

          <!-- ABOUT -->
          <section v-show="tab === 'about'" class="panel-stack">
            <GlassPanel class="about">
              <div class="about__head">
                <div class="about__logo"><Eye class="size-6" /></div>
                <div>
                  <div class="about__name">{{ t('app.name') }}</div>
                  <div class="about__ver">{{ t('about.version') }} 0.1.0</div>
                </div>
              </div>
              <p class="about__desc">{{ t('about.description') }}</p>
              <div class="about__rule">
                <div class="about__rule-title"><Sparkles class="size-4" />{{ t('about.rule') }}</div>
                <p class="about__rule-desc">{{ t('about.ruleDesc') }}</p>
              </div>
            </GlassPanel>
          </section>
        </div>
      </main>

      <!-- DRAWER (right) -->
      <aside class="drawer">
        <GlassPanel strong class="drawer__panel">
          <!-- Nav -->
          <nav class="nav">
            <div class="nav__section">{{ t('nav.sectionGeneral') }}</div>
            <button v-for="item in navGeneral" :key="item.value" class="nav__item" :class="{ 'nav__item--on': tab === item.value }" @click="tab = item.value">
              <component :is="item.icon" class="size-4" />
              <span>{{ item.label }}</span>
            </button>
          </nav>

          <!-- Cycle stats -->
          <div v-if="onboarded" class="cycle">
            <div class="nav__section">{{ t('nav.sectionCycle') }}</div>
            <div class="cycle__grid">
              <div class="cycle__cell">
                <div class="cycle__value tabular-nums">{{ shortDone }}</div>
                <div class="cycle__label">{{ t('stats.shortBreaksDone') }}</div>
              </div>
              <div class="cycle__cell">
                <div class="cycle__value tabular-nums">{{ s.enableLongBreaks ? breaksUntilLong : '—' }}</div>
                <div class="cycle__label">{{ t('stats.nextLong') }}</div>
              </div>
            </div>
            <div class="cycle__status">
              <span class="cycle__dot" :style="{ background: cycleRunning ? phaseColor : '#9aa2b1' }" />
              <span>{{ cycleRunning ? t('hero.focusing') : t('stats.notRunning') }}</span>
            </div>
            <button class="cycle__reset" @click="reset">
              <RotateCcw class="size-3.5" />{{ t('actions.reset') }}
            </button>
          </div>

          <div class="drawer__spacer" />

          <!-- Appearance quick settings -->
          <div class="quick">
            <div class="quick__row">
              <span class="quick__label"><Sun class="size-3.5" />{{ t('options.theme') }}</span>
              <div class="seg-row">
                <button v-for="o in themeOptions" :key="o.value" class="seg seg--icon" :class="{ 'seg--on': (s.theme || 'system') === o.value }" @click="changeTheme(o.value)" :title="themeLabel(o.value)">
                  <component :is="o.icon" class="size-3.5" />
                </button>
              </div>
            </div>
            <div class="quick__row">
              <span class="quick__label"><Languages class="size-3.5" />{{ t('onboarding.language') }}</span>
              <div class="seg-row">
                <button v-for="o in langOptions" :key="o.value" class="seg" :class="{ 'seg--on': locale === o.value }" @click="changeLanguage(o.value)">{{ o.label }}</button>
              </div>
            </div>
          </div>

          <!-- Save -->
          <div v-if="onboarded" class="drawer__foot">
            <span class="saved" :class="{ 'is-on': saved }"><Check class="size-3.5" />{{ t('actions.saved') }}</span>
            <GButton variant="primary" size="md" :disabled="!dirty" @click="save">{{ t('actions.save') }}</GButton>
          </div>
        </GlassPanel>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.root {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
}
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 0;
  background: linear-gradient(160deg, var(--bg-from), var(--bg-to));
}
.glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
}
.glow-a {
  width: 40vw;
  height: 40vw;
  top: -12vw;
  right: -8vw;
  background: var(--glow-a);
}
.glow-b {
  width: 32vw;
  height: 32vw;
  bottom: -10vw;
  left: -8vw;
  background: var(--glow-b);
}

/* ---- two-column layout ---- */
.layout {
  position: relative;
  z-index: 1;
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 16px;
  padding: 44px 28px 22px;
  box-sizing: border-box;
}
.main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

/* ---- hero ---- */
.hero { padding: 22px 24px; animation: pm-fade-up 0.5s ease both; }
.hero--break :deep(.hero__logo) { background: rgba(95, 216, 164, 0.16) !important; }
.hero__top { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.hero__brand { display: flex; gap: 12px; align-items: center; }
.hero__logo {
  width: 42px; height: 42px;
  display: grid; place-items: center;
  border-radius: 13px;
  background: var(--accent-soft);
  color: var(--accent);
}
.hero__title { margin: 0; font-size: 19px; font-weight: 600; letter-spacing: -0.01em; }
.hero__sub { margin: 3px 0 0; display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-muted); }
.hero__dot { width: 7px; height: 7px; border-radius: 50%; }
.hero__label { font-weight: 500; }
.hero__clock { text-align: right; }
.hero__time { font-size: 34px; font-weight: 250; line-height: 1; color: var(--text); }
.hero__progress { margin-top: 16px; height: 5px; border-radius: 9999px; background: var(--track); overflow: hidden; }
.hero__progress-bar { height: 100%; border-radius: 9999px; transition: width 1s linear; }
.hero__bar { margin-top: 14px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.hero__hint { font-size: 13px; color: var(--text-muted); }
.hero__actions { display: flex; gap: 8px; }

/* ---- onboarding ---- */
.onboard {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.onboard__intro { font-size: 13px; color: var(--text-muted); line-height: 1.6; }
.onboard__row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.onboard__field { display: flex; flex-direction: column; gap: 8px; }
.onboard__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.onboard__start { align-self: flex-end; min-width: 150px; }

/* ---- panels ---- */
.panels { flex: 1; min-height: 0; overflow-y: auto; }
.panel-stack { display: flex; flex-direction: column; gap: 10px; animation: pm-fade-up 0.4s ease both; }

.timing-row { padding: 16px 20px; }
.timing-row__head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 12px; }
.timing-row__label { font-size: 14px; font-weight: 500; color: var(--text); }
.timing-row__desc { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.timing-row__value { font-size: 24px; font-weight: 300; color: var(--accent); }
.timing-row__unit { font-size: 12px; color: var(--text-faint); margin-left: 4px; font-weight: 400; }

.opt-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 15px 20px; }
.opt-row__left { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.opt-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.opt-row__desc { font-size: 12px; color: var(--text-faint); }

.shortcut-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 13px 20px; }
.shortcut-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.keycap {
  width: 160px;
  text-align: center;
  padding: 8px 12px;
  font-size: 13px;
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  color: var(--text);
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  outline: none;
  transition: border-color 0.15s;
}
.keycap:focus { border-color: var(--accent); }
.hint { font-size: 12px; color: var(--text-faint); padding: 4px 6px; }

.about { padding: 24px; }
.about__head { display: flex; gap: 14px; align-items: center; margin-bottom: 16px; }
.about__logo { width: 46px; height: 46px; display: grid; place-items: center; border-radius: 13px; background: var(--accent-soft); color: var(--accent); }
.about__name { font-size: 18px; font-weight: 600; }
.about__ver { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.about__desc { font-size: 13px; color: var(--text-muted); line-height: 1.6; margin: 0 0 16px; }
.about__rule { padding: 16px; border-radius: 12px; background: var(--accent-soft); }
.about__rule-title { font-size: 13px; font-weight: 600; display: flex; align-items: center; gap: 8px; color: var(--accent); margin-bottom: 6px; }
.about__rule-desc { font-size: 13px; color: var(--text-muted); margin: 0; line-height: 1.5; }

/* ---- drawer ---- */
.drawer { min-width: 0; }
.drawer__panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px 14px;
  animation: pm-fade-up 0.5s 0.05s ease both;
}

.nav__section {
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-faint);
  padding: 8px 10px 4px;
}
.nav__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  font-size: 13.5px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 9px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.nav__item:hover { background: var(--accent-soft); color: var(--text); }
.nav__item--on {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  color: var(--on-accent);
  box-shadow: 0 6px 16px -6px var(--accent-soft);
}

/* cycle stats */
.cycle { margin-top: 6px; }
.cycle__grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; padding: 4px 4px 8px; }
.cycle__cell {
  padding: 12px 10px;
  border-radius: 11px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  text-align: center;
}
.cycle__value { font-size: 22px; font-weight: 300; color: var(--text); line-height: 1.1; }
.cycle__label { font-size: 10.5px; color: var(--text-faint); margin-top: 3px; }
.cycle__status {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  color: var(--text-muted);
  padding: 2px 10px 8px;
}
.cycle__dot { width: 7px; height: 7px; border-radius: 50%; }
.cycle__reset {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 2px 4px 0;
  padding: 6px 10px;
  font-size: 12px;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.cycle__reset:hover { background: var(--accent-soft); color: var(--text); }

.drawer__spacer { flex: 1; }

/* appearance quick settings */
.quick { display: flex; flex-direction: column; gap: 10px; padding-top: 12px; border-top: 1px solid var(--glass-border); }
.quick__row { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 0 6px; }
.quick__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }

/* segmented pill row (self-contained, no GSegmented dependency here) */
.seg-row { display: inline-flex; gap: 3px; padding: 3px; border-radius: 9px; background: var(--glass-bg); border: 1px solid var(--glass-border); }
.seg {
  min-width: 30px;
  padding: 5px 10px;
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.seg:hover { color: var(--text); }
.seg--on {
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  color: var(--on-accent);
}
.seg--icon { display: grid; place-items: center; padding: 5px 8px; }

/* drawer footer */
.drawer__foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--glass-border);
}
.saved {
  font-size: 11.5px;
  font-weight: 500;
  color: #5fd8a4;
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.25s;
}
.saved.is-on { opacity: 1; }
</style>
