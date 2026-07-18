<script setup lang="ts">
import GButton from '@/components/GButton.vue'
import GlassPanel from '@/components/GlassPanel.vue'
import GTimeField from '@/components/GTimeField.vue'
import GToggle from '@/components/GToggle.vue'
import { setLocale, type Locale } from '@/i18n'
import { applyTheme, type Theme } from '@/theme'
import { Events } from '@wailsio/runtime'
import { Bell, Check, Clock, Coffee, Eye, Info, Keyboard, Languages, Monitor, Moon, PanelLeftClose, PanelLeftOpen, Pause, Play, RotateCcw, Settings as Settings2, Sparkles, Sun, Timer } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import type { State } from '../../bindings/pocketmind/internal/breakengine/models'
import type { Settings } from '../../bindings/pocketmind/internal/config/models'

const { t, locale } = useI18n()

const s = reactive({} as Settings)
const state = ref<State>({} as State)
const dirty = ref(false)
const saved = ref(false)
const tab = ref('timing')
const collapsed = ref(false)
let off: (() => void) | undefined

// Snapshot of last-saved settings, so Discard can restore them.
let savedSnapshot: Settings | null = null

onMounted(async () => {
  Object.assign(s, await BreakService.GetSettings())
  savedSnapshot = { ...s }
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

// A cycle is "paused" if either the user paused it or the engine went idle.
const isPaused = computed(() => !!state.value.paused || state.value.phase === 'idle')
const isBreak = computed(() => state.value.phase === 'shortbreak' || state.value.phase === 'longbreak')
const cycleRunning = computed(() => !!state.value.phase && state.value.phase !== 'idle' && !state.value.paused)

function touch() { dirty.value = true; saved.value = false }
function changeLanguage(v: string) { setLocale(v as Locale); locale.value = v; s.language = v; touch() }
function changeTheme(v: string) { s.theme = v; applyTheme(v as Theme); touch() }

async function save() {
  await BreakService.SaveSettings({ ...s })
  savedSnapshot = { ...s }
  dirty.value = false
  saved.value = true
}
function discard() {
  if (savedSnapshot) {
    Object.assign(s, savedSnapshot)
    applyTheme((s.theme as Theme) || 'system')
    if (s.language === 'zh-CN' || s.language === 'en') {
      setLocale(s.language)
      locale.value = s.language
    }
  }
  dirty.value = false
  saved.value = false
}
async function completeOnboarding() {
  s.onboarded = true
  await BreakService.SaveSettings({ ...s })
  savedSnapshot = { ...s }
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
  // User-paused or idle: countdown frozen.
  if (isPaused.value) return t('hero.paused')
  switch (state.value.phase) {
    case 'focusing': return t('hero.focusing')
    case 'shortbreak': case 'longbreak': return t('hero.resting')
    case 'prebreak': return t('hero.resting')
    default: return t('hero.startFocus')
  }
})
const heroSub = computed(() => {
  if (isPaused.value) {
    if (state.value.phase === 'idle') return t('status.idle')
    return t('status.paused')
  }
  switch (state.value.phase) {
    case 'focusing': case 'prebreak': return t('status.breakIn', { sec: state.value.remainingSec })
    case 'shortbreak': case 'longbreak': return t('status.remaining') + ' ' + fmt(state.value.remainingSec)
    default: return t('onboarding.intro2')
  }
})
const phaseColor = computed(() => {
  if (isPaused.value) return 'var(--phase-pause)'
  switch (state.value.phase) {
    case 'shortbreak': return 'var(--phase-break)'
    case 'longbreak': return 'var(--phase-long)'
    case 'prebreak': return 'var(--phase-prebreak)'
    default: return 'var(--phase-focus)'
  }
})
const heroTime = computed(() => fmt(state.value.remainingSec))
function fmt(sec: number): string {
  const m = Math.floor(sec / 60)
  const ss = sec % 60
  return `${m}:${String(ss).padStart(2, '0')}`
}

const navGeneral = computed(() => [
  { value: 'timing', label: t('nav.timing'), icon: Timer },
  { value: 'options', label: t('nav.options'), icon: Sparkles },
  { value: 'shortcuts', label: t('nav.shortcuts'), icon: Keyboard },
  { value: 'about', label: t('nav.about'), icon: Info },
])

const shortDone = computed(() => state.value.shortBreakCount ?? 0)
const breaksUntilLong = computed(() => state.value.breaksUntilLong ?? 0)

// Drawer status line: reflects focus / break / paused / idle accurately.
const cycleStatusText = computed(() => {
  if (isPaused.value) {
    if (state.value.phase === 'idle') return t('status.idle')
    return t('hero.paused')
  }
  switch (state.value.phase) {
    case 'focusing': return t('hero.focusing')
    case 'shortbreak': case 'longbreak': return t('hero.resting')
    case 'prebreak': return t('hero.resting')
    default: return t('stats.notRunning')
  }
})

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

// ---- Timing tab ----
// Time fields with steppers + preset chips. Each field is a compact GTimeField
// laid out in a 2-column grid for scannability and quick editing.

interface TimingField {
  key: keyof Settings
  label: string
  desc: string
  unit?: string
  min: number
  max: number
  step: number
  presets: number[]
  // For seconds-based fields, render chips as "1m", "5m" etc., and the value
  // display as "5 分" so users don't have to do mental arithmetic on "300 秒".
  formatChip?: (v: number) => string
  formatValue?: (v: number) => string
}

const focusPresets = [15, 20, 25, 30, 45, 50]
const shortBreakPresets = [20, 60, 180, 300, 600]
const longBreakPresets = [5, 10, 15, 20]
const intervalPresets = [2, 3, 4, 5]
const preWarnPresets = [0, 10, 15, 30, 60]
const idlePresets = [1, 3, 5, 10, 15]

// Friendly chip/value formatters for seconds-based fields.
// The value display returns a full label (e.g. "20 秒", "5 分") and we drop the
// separate unit prop, so users never have to read "300 秒".
function fmtSecs(v: number): string {
  if (v <= 0) return '0'
  if (v < 60) return `${v} 秒`
  const m = Math.floor(v / 60)
  const s = v % 60
  return s === 0 ? `${m} 分` : `${m} 分 ${s} 秒`
}
function fmtSecsChip(v: number): string {
  if (v < 60) return `${v}s`
  const m = Math.floor(v / 60)
  const s = v % 60
  return s === 0 ? `${m}m` : `${m}m${s}s`
}

const timingFields = computed<TimingField[]>(() => [
  {
    key: 'focusDurationMin',
    label: t('timing.focusDuration'),
    desc: t('timing.focusDesc'),
    unit: t('timing.minutes'),
    min: 5, max: 60, step: 1,
    presets: focusPresets,
  },
  {
    key: 'shortBreakDurationSec',
    label: t('timing.shortBreak'),
    desc: t('timing.shortDesc'),
    min: 5, max: 600, step: 5,
    presets: shortBreakPresets,
    formatChip: fmtSecsChip,
    formatValue: fmtSecs,
  },
  {
    key: 'longBreakDurationMin',
    label: t('timing.longBreak'),
    desc: t('timing.longDesc'),
    unit: t('timing.minutes'),
    min: 1, max: 20, step: 1,
    presets: longBreakPresets,
  },
  {
    key: 'longBreakInterval',
    label: t('timing.longBreakEvery'),
    desc: t('timing.everyDesc'),
    unit: t('timing.breaks'),
    min: 1, max: 10, step: 1,
    presets: intervalPresets,
  },
  {
    key: 'preBreakWarningSec',
    label: t('timing.preBreakWarning'),
    desc: t('timing.preDesc'),
    min: 0, max: 60, step: 5,
    presets: preWarnPresets,
    formatChip: fmtSecsChip,
    formatValue: fmtSecs,
  },
  {
    key: 'idleThresholdMin',
    label: t('timing.idlePause'),
    desc: t('timing.idleDesc'),
    unit: t('timing.minutes'),
    min: 1, max: 30, step: 1,
    presets: idlePresets,
  },
])

// ---- Preset profiles ----
// One-click applies a coherent set of timing values; selecting "Custom" just
// labels the current combination (it's auto-selected when values diverge).

interface PresetProfile {
  id: 'eye' | 'pomo' | '52' | 'custom'
  name: string
  desc: string
  values?: Partial<Settings>
}
const presetProfiles = computed<PresetProfile[]>(() => [
  {
    id: 'eye',
    name: t('timing.presetEye'),
    desc: t('timing.presetEyeDesc'),
    values: { focusDurationMin: 20, shortBreakDurationSec: 20, longBreakDurationMin: 5, longBreakInterval: 4, preBreakWarningSec: 10, idleThresholdMin: 5 },
  },
  {
    id: 'pomo',
    name: t('timing.presetPomo'),
    desc: t('timing.presetPomoDesc'),
    values: { focusDurationMin: 25, shortBreakDurationSec: 300, longBreakDurationMin: 15, longBreakInterval: 4, preBreakWarningSec: 30, idleThresholdMin: 5 },
  },
  {
    id: '52',
    name: t('timing.preset52'),
    desc: t('timing.preset52Desc'),
    values: { focusDurationMin: 50, shortBreakDurationSec: 600, longBreakDurationMin: 20, longBreakInterval: 3, preBreakWarningSec: 60, idleThresholdMin: 10 },
  },
  { id: 'custom', name: t('timing.presetCustom'), desc: '' },
])

const activePreset = computed<PresetProfile['id']>(() => {
  const eye = presetProfiles.value.find(p => p.id === 'eye')?.values
  const pomo = presetProfiles.value.find(p => p.id === 'pomo')?.values
  const p52 = presetProfiles.value.find(p => p.id === '52')?.values
  const match = (v?: Partial<Settings>) => v && Object.entries(v).every(([k, val]) => (s as any)[k] === val)
  if (match(eye)) return 'eye'
  if (match(pomo)) return 'pomo'
  if (match(p52)) return '52'
  return 'custom'
})
function applyPreset(p: PresetProfile) {
  if (!p.values) return
  Object.entries(p.values).forEach(([k, v]) => { (s as any)[k] = v })
  touch()
}

const shortcutRows = computed(() => [
  { key: 'shortcutStartBreak' as const, label: t('shortcuts.startBreak'), icon: Clock, value: s.shortcutStartBreak, placeholder: 'Cmd+Shift+B' },
  { key: 'shortcutSkipBreak' as const, label: t('shortcuts.skipBreak'), icon: Play, value: s.shortcutSkipBreak, placeholder: 'Cmd+Shift+S' },
  { key: 'shortcutPostponeBreak' as const, label: t('shortcuts.postponeBreak'), icon: Clock, value: s.shortcutPostponeBreak, placeholder: 'Cmd+Shift+P' },
  { key: 'shortcutPreferences' as const, label: t('shortcuts.preferences'), icon: Settings2, value: s.shortcutPreferences, placeholder: 'Cmd+Shift+,' },
])

// Save bar is relevant only on tabs that actually edit settings.
const editTabs = ['timing', 'options', 'shortcuts']
const showSaveBar = computed(() => onboarded.value && editTabs.includes(tab.value))
</script>

<template>
  <div class="root">
    <div class="backdrop" aria-hidden="true" />

    <div class="layout" :class="{ 'layout--collapsed': collapsed }">
      <!-- DRAWER (left, collapsible) -->
      <aside class="drawer">
        <GlassPanel strong class="drawer__panel">
          <!-- collapse toggle -->
          <button class="collapse-btn" @click="collapsed = !collapsed" :title="collapsed ? '' : t('nav.collapse')">
            <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="size-4" />
            <span v-if="!collapsed" class="collapse-btn__txt">{{ t('nav.menu') }}</span>
          </button>

          <!-- Collapsed: icon rail -->
          <nav v-if="collapsed" class="nav nav--rail">
            <button v-for="item in navGeneral" :key="item.value" class="nav__item nav__item--rail" :class="{ 'nav__item--on': tab === item.value }" @click="tab = item.value" :title="item.label">
              <component :is="item.icon" class="size-4" />
            </button>
          </nav>

          <!-- Expanded: full nav -->
          <template v-else>
            <nav class="nav">
              <div class="nav__section">{{ t('nav.sectionGeneral') }}</div>
              <button v-for="item in navGeneral" :key="item.value" class="nav__item" :class="{ 'nav__item--on': tab === item.value }" @click="tab = item.value">
                <component :is="item.icon" class="size-4" />
                <span>{{ item.label }}</span>
              </button>
            </nav>

            <div v-if="onboarded" class="cycle">
              <div class="nav__section">{{ t('nav.sectionCycle') }}</div>
              <div class="cycle__grid">
                <div class="cycle__cell">
                  <div class="cycle__value tabular-nums">{{ shortDone }}</div>
                  <div class="cycle__label">{{ t('stats.shortBreaksDone') }}</div>
                </div>
                <div class="cycle__cell">
                  <div class="cycle__value tabular-nums">{{ s.enableLongBreaks ? breaksUntilLong : '-' }}</div>
                  <div class="cycle__label">{{ t('stats.nextLong') }}</div>
                </div>
              </div>
              <div class="cycle__status">
                <span class="cycle__dot" :class="{ 'cycle__dot--paused': isPaused }" :style="{ background: cycleRunning ? phaseColor : 'var(--phase-pause)' }" />
                <span>{{ cycleStatusText }}</span>
              </div>
              <button class="cycle__reset" @click="reset">
                <RotateCcw class="size-3.5" />{{ t('actions.reset') }}
              </button>
            </div>

            <div class="drawer__spacer" />

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
          </template>
        </GlassPanel>
      </aside>

      <!-- MAIN (right) -->
      <main class="main">
        <!-- HERO (fixed header) -->
        <GlassPanel strong class="hero" :class="{ 'hero--break': isBreak, 'hero--paused': isPaused }">
          <div class="hero__top">
            <div class="hero__brand">
              <div class="hero__logo"><Eye class="size-5" /></div>
              <div>
                <h1 class="hero__title">{{ t('app.name') }}</h1>
                <p class="hero__sub">
                  <span class="hero__dot" :style="{ background: phaseColor }" />
                  <span class="hero__label">{{ heroTitle }}</span>
                </p>
              </div>
            </div>
            <div v-if="onboarded && state.phase" class="hero__clock">
              <span class="hero__time tabular-nums">{{ heroTime }}</span>
            </div>
          </div>

          <div v-if="onboarded && state.phase && !isPaused" class="hero__progress">
            <div class="hero__progress-bar" :style="{ width: (progress * 100) + '%', background: phaseColor }" />
          </div>

          <div class="hero__bar">
            <span class="hero__hint">{{ heroSub }}</span>
            <div v-if="onboarded" class="hero__actions">
              <GButton variant="glass" size="sm" @click="togglePause">
                <component :is="isPaused ? Play : Pause" class="size-3.5" />
                {{ isPaused ? t('actions.resume') : t('actions.pause') }}
              </GButton>
              <GButton v-if="!isBreak" variant="primary" size="sm" @click="startBreak">
                <Clock class="size-3.5" />
                {{ t('actions.breakNow') }}
              </GButton>
            </div>
          </div>

          <!-- Onboarding (first run) -->
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

        <!-- SCROLLABLE PANEL CONTENT -->
        <div v-if="onboarded" class="panels-wrap">
          <div class="panels">
            <!-- TIMING -->
            <section v-show="tab === 'timing'" class="panel-stack">
              <!-- Preset profiles -->
              <GlassPanel class="presets">
                <div class="presets__head">
                  <div class="presets__title">{{ t('timing.presets') }}</div>
                </div>
                <div class="presets__grid">
                  <button
                    v-for="p in presetProfiles"
                    :key="p.id"
                    class="preset"
                    :class="{ 'preset--on': activePreset === p.id, 'preset--custom': p.id === 'custom' }"
                    @click="applyPreset(p)"
                  >
                    <div class="preset__name">{{ p.name }}</div>
                    <div v-if="p.desc" class="preset__desc">{{ p.desc }}</div>
                    <Check v-if="activePreset === p.id" class="preset__check size-3.5" />
                  </button>
                </div>
              </GlassPanel>

              <!-- Timing fields grid -->
              <GlassPanel class="timing-grid">
                <GTimeField
                  v-for="f in timingFields"
                  :key="f.key"
                  :label="f.label"
                  :desc="f.desc"
                  :unit="f.unit"
                  :min="f.min"
                  :max="f.max"
                  :step="f.step"
                  :presets="f.presets"
                  :model-value="(s as any)[f.key] ?? 0"
                  :chip-label="f.formatChip"
                  :display-value="f.formatValue"
                  @update:model-value="(v: number) => setNum(f.key, v)"
                />
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

          <!-- Sticky save bar: lives in the main column, NOT the drawer.
               Hidden when there's nothing to edit (e.g. About tab). -->
          <transition name="pm-slide-up">
            <div v-if="showSaveBar" class="savebar" :class="{ 'savebar--dirty': dirty }">
              <div class="savebar__left">
                <span v-if="dirty" class="savebar__dot" />
                <span v-else-if="saved" class="savebar__saved"><Check class="size-3.5" />{{ t('actions.saved') }}</span>
                <span v-else class="savebar__idle">{{ t('actions.saved') }}</span>
              </div>
              <div class="savebar__right">
                <GButton v-if="dirty" variant="ghost" size="sm" @click="discard">{{ t('actions.discard') }}</GButton>
                <GButton variant="primary" size="sm" :disabled="!dirty" @click="save">
                  <Check class="size-3.5" />
                  {{ t('actions.save') }}
                </GButton>
              </div>
            </div>
          </transition>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.root {
  position: relative;
  height: 100vh;
  overflow: hidden;
}
.backdrop {
  position: fixed;
  inset: 0;
  z-index: 0;
  background: linear-gradient(160deg, var(--bg-from), var(--bg-to));
}

/* ---- fixed-height two-column layout ---- */
.layout {
  position: relative;
  z-index: 1;
  height: 100%;
  display: grid;
  grid-template-columns: 268px 1fr;
  gap: 16px;
  padding: 44px 24px 20px;
  box-sizing: border-box;
  transition: grid-template-columns 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.layout--collapsed {
  grid-template-columns: 64px 1fr;
}

/* ---- drawer (left) ---- */
.drawer { min-width: 0; min-height: 0; }
.drawer__panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 12px;
  animation: pm-fade-up 0.5s ease both;
  overflow: hidden;
}
.collapse-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 8px;
  font-size: 12.5px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 9px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.collapse-btn:hover { background: var(--accent-soft); color: var(--text); }
.layout--collapsed .collapse-btn { justify-content: center; }
.collapse-btn__txt { letter-spacing: 0.02em; }

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
  background: var(--accent);
  color: var(--on-accent);
}
/* icon rail (collapsed) */
.nav--rail { display: flex; flex-direction: column; gap: 2px; align-items: center; }
.nav__item--rail { justify-content: center; width: 40px; padding: 9px; }

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
.cycle__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  transition: background 0.2s;
}
.cycle__dot--paused {
  animation: none;
}
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

.drawer__spacer { flex: 1; min-height: 8px; }

.quick { display: flex; flex-direction: column; gap: 10px; padding-top: 12px; border-top: 1px solid var(--glass-border); }
.quick__row { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 0 6px; }
.quick__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }

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
  background: var(--accent);
  color: var(--on-accent);
}
.seg--icon { display: grid; place-items: center; padding: 5px 8px; }

/* ---- main (right) ---- */
.main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  min-height: 0;
}

.hero { padding: 20px 24px; flex-shrink: 0; animation: pm-fade-up 0.5s ease both; }
.hero--break :deep(.hero__logo) { background: rgba(91, 138, 111, 0.16) !important; color: var(--phase-break) !important; }
.hero--paused :deep(.hero__logo) { background: rgba(154, 162, 177, 0.18) !important; color: var(--text-muted) !important; }
.hero__top { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.hero__brand { display: flex; gap: 12px; align-items: center; }
.hero__logo {
  width: 42px; height: 42px;
  display: grid; place-items: center;
  border-radius: 13px;
  background: var(--accent-soft);
  color: var(--accent);
  transition: background 0.2s, color 0.2s;
}
.hero__title { margin: 0; font-size: 19px; font-weight: 600; letter-spacing: -0.01em; }
.hero__sub { margin: 3px 0 0; display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-muted); }
.hero__dot { width: 7px; height: 7px; border-radius: 50%; transition: background 0.2s, box-shadow 0.2s; }
.hero__label { font-weight: 500; }
.hero__clock { text-align: right; }
.hero__time { font-size: 34px; font-weight: 250; line-height: 1; color: var(--text); }
.hero__progress { margin-top: 16px; height: 5px; border-radius: 9999px; background: var(--track); overflow: hidden; }
.hero__progress-bar { height: 100%; border-radius: 9999px; transition: width 1s linear; }
.hero__bar { margin-top: 14px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.hero__hint { font-size: 13px; color: var(--text-muted); }
.hero__actions { display: flex; gap: 8px; }

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

/* panels-wrap holds the scroll area + the sticky save bar */
.panels-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.panels {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 4px;
  display: flex;
  flex-direction: column;
}
.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: pm-fade-up 0.4s ease both;
}

/* ---- timing tab ---- */
.presets { padding: 16px 18px; }
.presets__head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.presets__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
}
.presets__grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
.preset {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 12px;
  text-align: left;
  font-family: inherit;
  color: var(--text-muted);
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.18s;
  overflow: hidden;
}
.preset:hover { color: var(--text); border-color: var(--accent); }
.preset--on {
  color: var(--on-accent);
  background: var(--accent);
  border-color: transparent;
}
.preset__name { font-size: 13px; font-weight: 600; }
.preset__desc { font-size: 11px; color: var(--text-faint); line-height: 1.35; }
.preset--on .preset__desc { color: var(--on-accent); opacity: 0.85; }
.preset__check {
  position: absolute;
  top: 8px;
  right: 8px;
  color: var(--on-accent);
}

.timing-grid {
  padding: 14px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
@media (max-width: 720px) {
  .timing-grid { grid-template-columns: 1fr; }
  .presets__grid { grid-template-columns: 1fr 1fr; }
}

/* ---- options ---- */
.opt-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 15px 20px; }
.opt-row__left { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.opt-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.opt-row__desc { font-size: 12px; color: var(--text-faint); }

/* ---- shortcuts ---- */
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

/* ---- about ---- */
.about { padding: 24px; }
.about__head { display: flex; gap: 14px; align-items: center; margin-bottom: 16px; }
.about__logo { width: 46px; height: 46px; display: grid; place-items: center; border-radius: 13px; background: var(--accent-soft); color: var(--accent); }
.about__name { font-size: 18px; font-weight: 600; }
.about__ver { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.about__desc { font-size: 13px; color: var(--text-muted); line-height: 1.6; margin: 0 0 16px; }
.about__rule { padding: 16px; border-radius: 12px; background: var(--accent-soft); }
.about__rule-title { font-size: 13px; font-weight: 600; display: flex; align-items: center; gap: 8px; color: var(--accent); margin-bottom: 6px; }
.about__rule-desc { font-size: 13px; color: var(--text-muted); margin: 0; line-height: 1.5; }

/* ---- sticky save bar ---- */
.savebar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-radius: 14px;
  background: var(--glass-bg-strong);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid var(--glass-border);
  box-shadow: var(--glass-shadow);
  transition: opacity 0.2s, transform 0.25s;
}
.savebar__left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--text-muted);
}
.savebar__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
  animation: pm-pulse 2.4s ease-in-out infinite;
}
.savebar__saved { color: var(--phase-break); display: flex; align-items: center; gap: 5px; }
.savebar__idle { color: var(--text-faint); }
.savebar__right { display: flex; gap: 8px; align-items: center; }

@keyframes pm-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Slide-up transition for the save bar */
.pm-slide-up-enter-active, .pm-slide-up-leave-active {
  transition: opacity 0.25s, transform 0.25s;
}
.pm-slide-up-enter-from, .pm-slide-up-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
