<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import type { Settings } from '../../bindings/pocketmind/internal/config/models'
import type { State } from '../../bindings/pocketmind/internal/breakengine/models'
import { Eye, Play, Pause, Clock, Bell, Languages, Check, Sun, Moon, Monitor, Sparkles, Keyboard, Info, Timer, Coffee, Settings as Settings2 } from 'lucide-vue-next'
import GButton from '@/components/GButton.vue'
import GlassPanel from '@/components/GlassPanel.vue'
import GSlider from '@/components/GSlider.vue'
import GToggle from '@/components/GToggle.vue'
import GSegmented from '@/components/GSegmented.vue'
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
const isPreBreak = computed(() => state.value.phase === 'prebreak')

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

function setNum(field: keyof Settings, v: number) { (s as any)[field] = v; touch() }

// Hero progress: fraction of the current phase elapsed (grows as it advances).
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
    case 'focusing': return t('status.breakIn', { sec: state.value.remainingSec })
    case 'prebreak': return t('status.breakIn', { sec: state.value.remainingSec })
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

const themeOptions = [
  { value: 'system', label: '', icon: Monitor },
  { value: 'light', label: '', icon: Sun },
  { value: 'dark', label: '', icon: Moon },
]
const langOptions = [
  { value: 'zh-CN', label: '中' },
  { value: 'en', label: 'EN' },
]
const tabs = computed(() => [
  { value: 'timing', label: t('nav.timing'), icon: Timer },
  { value: 'options', label: t('nav.options'), icon: Sparkles },
  { value: 'shortcuts', label: t('nav.shortcuts'), icon: Keyboard },
  { value: 'about', label: t('nav.about'), icon: Info },
])

// Timing rows config drives the template loop.
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
    <!-- Ambient backdrop: gradient + soft glows behind the glass. -->
    <div class="backdrop" aria-hidden="true">
      <div class="glow glow-a" />
      <div class="glow glow-b" />
    </div>

    <div class="content">
      <!-- HERO STATUS CARD -->
      <GlassPanel strong class="hero" :class="{ 'hero--break': isBreak, 'hero--pre': isPreBreak }">
        <div class="hero__top">
          <div class="hero__brand">
            <div class="hero__logo">
              <Eye class="size-5" />
            </div>
            <div>
              <h1 class="hero__title">{{ t('app.name') }}</h1>
              <p class="hero__sub">
                <span class="hero__dot" :style="{ background: phaseColor, boxShadow: `0 0 8px ${phaseColor}` }" />
                <span class="hero__label">{{ heroTitle }}</span>
              </p>
            </div>
          </div>

          <div v-if="onboarded" class="hero__clock">
            <span class="hero__time tabular-nums" v-if="state.phase">{{ heroTime }}</span>
          </div>
        </div>

        <!-- Progress bar (when running). -->
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

        <!-- Onboarding controls (first run only). -->
        <div v-if="!onboarded" class="onboard">
          <div class="onboard__row">
            <div class="onboard__field">
              <span class="onboard__label"><Languages class="size-3.5" />{{ t('onboarding.language') }}</span>
              <GSegmented :options="langOptions" :model-value="locale" @update:model-value="changeLanguage" />
            </div>
            <div class="onboard__field">
              <span class="onboard__label"><Sun class="size-3.5" />{{ t('options.theme') }}</span>
              <GSegmented :options="themeOptions.map(o => ({ value: o.value, label: t('options.theme' + (o.value === 'system' ? 'System' : o.value === 'light' ? 'Light' : 'Dark')), icon: o.icon }))" :model-value="s.theme || 'system'" @update:model-value="changeTheme" />
            </div>
          </div>
          <GButton variant="primary" size="lg" class="onboard__start" @click="completeOnboarding">
            <Play class="size-4" />
            {{ t('onboarding.startButton') }}
          </GButton>
        </div>
      </GlassPanel>

      <!-- REGULAR SETTINGS (onboarded) -->
      <template v-if="onboarded">
        <GSegmented class="nav" :options="tabs.map(t2 => ({ value: t2.value, label: t2.label, icon: t2.icon }))" :model-value="tab" @update:model-value="(v) => tab = v" />

        <div class="panels">
          <!-- TIMING -->
          <div v-show="tab === 'timing'" class="panel-stack">
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
          </div>

          <!-- OPTIONS -->
          <div v-show="tab === 'options'" class="panel-stack">
            <GlassPanel class="opt-row">
              <div class="opt-row__left">
                <span class="opt-row__label"><Sun class="size-4" />{{ t('options.theme') }}</span>
                <span class="opt-row__desc">{{ t('options.themeDesc') }}</span>
              </div>
              <GSegmented :options="themeOptions.map(o => ({ value: o.value, label: t('options.theme' + (o.value === 'system' ? 'System' : o.value === 'light' ? 'Light' : 'Dark')), icon: o.icon }))" :model-value="s.theme || 'system'" @update:model-value="changeTheme" />
            </GlassPanel>
            <GlassPanel class="opt-row">
              <div class="opt-row__left">
                <span class="opt-row__label"><Languages class="size-4" />{{ t('onboarding.language') }}</span>
                <span class="opt-row__desc">{{ locale === 'zh-CN' ? '简体中文' : 'English' }}</span>
              </div>
              <GSegmented :options="langOptions" :model-value="locale" @update:model-value="changeLanguage" />
            </GlassPanel>
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
          </div>

          <!-- SHORTCUTS -->
          <div v-show="tab === 'shortcuts'" class="panel-stack">
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
          </div>

          <!-- ABOUT -->
          <div v-show="tab === 'about'" class="panel-stack">
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
          </div>
        </div>

        <!-- FOOTER -->
        <div class="footer">
          <span class="footer__saved" :class="{ 'is-on': saved }">
            <Check class="size-3.5" />{{ t('actions.saved') }}
          </span>
          <GButton variant="primary" size="md" :disabled="!dirty" @click="save">{{ t('actions.save') }}</GButton>
        </div>
      </template>
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
  filter: blur(80px);
}
.glow-a {
  width: 50vw;
  height: 50vw;
  top: -15vw;
  right: -10vw;
  background: var(--glow-a);
}
.glow-b {
  width: 40vw;
  height: 40vw;
  bottom: -10vw;
  left: -10vw;
  background: var(--glow-b);
}
.content {
  position: relative;
  z-index: 1;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 40px 28px 20px;
  box-sizing: border-box;
}

/* ---- hero ---- */
.hero {
  padding: 20px;
  animation: pm-fade-up 0.5s ease both;
}
.hero--break :deep(.hero__logo) { background: rgba(95, 216, 164, 0.16) !important; }
.hero__top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.hero__brand { display: flex; gap: 12px; align-items: center; }
.hero__logo {
  width: 40px; height: 40px;
  display: grid; place-items: center;
  border-radius: 12px;
  background: var(--accent-soft);
  color: var(--accent);
}
.hero__title { margin: 0; font-size: 18px; font-weight: 600; letter-spacing: -0.01em; }
.hero__sub { margin: 3px 0 0; display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-muted); }
.hero__dot { width: 7px; height: 7px; border-radius: 50%; }
.hero__label { font-weight: 500; }
.hero__clock { text-align: right; }
.hero__time { font-size: 30px; font-weight: 260; line-height: 1; color: var(--text); }
.hero__progress {
  margin-top: 16px;
  height: 5px;
  border-radius: 9999px;
  background: var(--track);
  overflow: hidden;
}
.hero__progress-bar {
  height: 100%;
  border-radius: 9999px;
  transition: width 1s linear;
}
.hero__bar {
  margin-top: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.hero__hint { font-size: 13px; color: var(--text-muted); }
.hero__actions { display: flex; gap: 8px; }

/* ---- onboarding block ---- */
.onboard {
  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.onboard__row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.onboard__field { display: flex; flex-direction: column; gap: 8px; }
.onboard__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.onboard__start { align-self: flex-end; min-width: 150px; }

/* ---- nav ---- */
.nav { align-self: flex-start; animation: pm-fade-up 0.5s 0.05s ease both; }

/* ---- panels ---- */
.panels { flex: 1; min-height: 0; }
.panel-stack { display: flex; flex-direction: column; gap: 10px; animation: pm-fade-up 0.4s ease both; }

/* timing rows */
.timing-row { padding: 16px 18px; }
.timing-row__head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 12px; }
.timing-row__label { font-size: 14px; font-weight: 500; color: var(--text); }
.timing-row__desc { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.timing-row__value { font-size: 22px; font-weight: 300; color: var(--accent); }
.timing-row__unit { font-size: 12px; color: var(--text-faint); margin-left: 4px; font-weight: 400; }

/* option rows */
.opt-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 14px 18px; }
.opt-row__left { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.opt-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.opt-row__desc { font-size: 12px; color: var(--text-faint); }

/* shortcut rows */
.shortcut-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 12px 18px; }
.shortcut-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.keycap {
  width: 150px;
  text-align: center;
  padding: 7px 12px;
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
.hint { font-size: 12px; color: var(--text-faint); padding: 0 4px; }

/* about */
.about { padding: 22px; }
.about__head { display: flex; gap: 14px; align-items: center; margin-bottom: 16px; }
.about__logo { width: 44px; height: 44px; display: grid; place-items: center; border-radius: 12px; background: var(--accent-soft); color: var(--accent); }
.about__name { font-size: 17px; font-weight: 600; }
.about__ver { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.about__desc { font-size: 13px; color: var(--text-muted); line-height: 1.6; margin: 0 0 16px; }
.about__rule { padding: 14px; border-radius: 12px; background: var(--accent-soft); }
.about__rule-title { font-size: 13px; font-weight: 600; display: flex; align-items: center; gap: 8px; color: var(--accent); margin-bottom: 6px; }
.about__rule-desc { font-size: 13px; color: var(--text-muted); margin: 0; line-height: 1.5; }

/* footer */
.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 4px 4px;
}
.footer__saved {
  font-size: 12px;
  font-weight: 500;
  color: #5fd8a4;
  display: flex;
  align-items: center;
  gap: 5px;
  opacity: 0;
  transition: opacity 0.25s;
}
.footer__saved.is-on { opacity: 1; }
</style>
