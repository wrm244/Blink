<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import type { Settings } from '../../bindings/pocketmind/internal/config/models'
import type { State } from '../../bindings/pocketmind/internal/breakengine/models'
import { Eye, Play, Pause, Clock, Bell, Languages, Check, Sun, Moon, Monitor } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Slider } from '@/components/ui/slider'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { setLocale, type Locale } from '@/i18n'
import { applyTheme, type Theme } from '@/theme'

const { t, locale } = useI18n()

const s = reactive({} as Settings)
const state = ref<State>({} as State)
const dirty = ref(false)
const saved = ref(false)
let off: (() => void) | undefined

onMounted(async () => {
  Object.assign(s, await BreakService.GetSettings())
  // Apply the persisted theme + locale from settings.
  applyTheme((s.theme as Theme) || 'system')
  if (s.language === 'zh-CN' || s.language === 'en') {
    setLocale(s.language)
    locale.value = s.language
  }
  state.value = await BreakService.GetState()
  off = Events.On('pm:tick', (ev: { data: State }) => {
    state.value = ev.data
  })
})
onUnmounted(() => off?.())

const onboarded = computed(() => s.onboarded === true)
// "Paused" covers both an explicit user pause and the idle (inactivity) pause.
const isPaused = computed(() => state.value.paused || state.value.phase === 'idle')

function touch() {
  dirty.value = true
  saved.value = false
}

function changeLanguage(v: string) {
  setLocale(v as Locale)
  locale.value = v
  s.language = v
  touch()
}

function changeTheme(v: string) {
  s.theme = v
  applyTheme(v as Theme)
  touch()
}

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
function togglePause() {
  if (isPaused.value) BreakService.Resume()
  else BreakService.Pause()
}

// Slider values are bound as number[] (reka-ui Slider contract); convert.
function num(v: number[] | undefined, fallback: number): number[] {
  return v && v.length ? v : [fallback]
}
function setNum(field: keyof Settings, v: number[] | undefined) {
  ;(s as any)[field] = v?.[0] ?? 0
  touch()
}

const phaseLabel = computed(() => {
  switch (state.value.phase) {
    case 'focusing': return t('status.focusing', { time: fmt(state.value.remainingSec) })
    case 'prebreak': return t('status.prebreak', { sec: state.value.remainingSec })
    case 'shortbreak': return t('status.shortbreak', { time: fmt(state.value.remainingSec) })
    case 'longbreak': return t('status.longbreak', { time: fmt(state.value.remainingSec) })
    case 'paused': return t('status.paused')
    case 'idle': return t('status.idle')
    default: return t('status.waiting')
  }
})
const phaseColor = computed(() => {
  switch (state.value.phase) {
    case 'shortbreak': case 'longbreak': return 'oklch(0.72 0.17 145)'
    case 'prebreak': return 'oklch(0.78 0.16 70)'
    case 'paused': case 'idle': return 'oklch(0.66 0.01 264)'
    default: return 'oklch(0.62 0.17 255)'
  }
})
function fmt(sec: number): string {
  const m = Math.floor(sec / 60)
  const ss = sec % 60
  return `${m}:${String(ss).padStart(2, '0')}`
}
</script>

<template>
  <div class="prefs-root">
    <!-- Header -->
    <header class="flex items-center justify-between gap-3 px-7 pt-12 pb-2">
      <div class="flex items-center gap-3">
        <div class="grid size-10 place-items-center rounded-xl bg-primary/15 text-primary ring-1 ring-primary/25">
          <Eye class="size-5" />
        </div>
        <div>
          <h1 class="text-[17px] font-semibold tracking-tight leading-tight">{{ t('app.name') }}</h1>
          <p class="flex items-center gap-1.5 text-[12px] text-muted-foreground mt-0.5">
            <span class="size-1.5 rounded-full" :style="{ background: phaseColor, boxShadow: `0 0 6px ${phaseColor}` }" />
            <span class="tabular-nums">{{ phaseLabel }}</span>
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2" v-if="onboarded">
        <Button variant="ghost" size="sm" @click="togglePause">
          <component :is="isPaused ? Play : Pause" class="size-3.5" />
          {{ isPaused ? t('actions.resume') : t('actions.pause') }}
        </Button>
        <Button variant="secondary" size="sm" @click="startBreak">
          <Clock class="size-3.5" />
          {{ t('actions.breakNow') }}
        </Button>
      </div>
    </header>

    <!-- Onboarding -->
    <section v-if="!onboarded" class="px-7 pt-2 pb-6 flex-1 flex flex-col">
      <Card class="border-primary/20 bg-primary/[0.04]">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Languages class="size-4 text-primary" />
            {{ t('onboarding.welcome') }}
          </CardTitle>
          <CardDescription class="leading-relaxed">
            {{ t('onboarding.intro') }}<br />
            <span class="text-muted-foreground/70">{{ t('onboarding.intro2') }}</span>
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-5">
          <!-- language + appearance, side by side -->
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label class="flex items-center gap-1.5"><Languages class="size-3.5" />{{ t('onboarding.language') }}</Label>
              <Select :model-value="locale" @update:model-value="changeLanguage">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="zh-CN">简体中文</SelectItem>
                  <SelectItem value="en">English</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label class="flex items-center gap-1.5"><Sun class="size-3.5" />{{ t('options.theme') }}</Label>
              <Select :model-value="s.theme || 'system'" @update:model-value="changeTheme">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="system">{{ t('options.themeSystem') }}</SelectItem>
                  <SelectItem value="light">{{ t('options.themeLight') }}</SelectItem>
                  <SelectItem value="dark">{{ t('options.themeDark') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <Separator />

          <!-- quick defaults preview -->
          <div class="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">{{ t('timing.focusDuration') }}</span>
              <span class="font-medium tabular-nums">{{ s.focusDurationMin }} {{ t('timing.minutes') }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">{{ t('timing.shortBreak') }}</span>
              <span class="font-medium tabular-nums">{{ s.shortBreakDurationSec }} {{ t('timing.seconds') }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">{{ t('timing.longBreak') }}</span>
              <span class="font-medium tabular-nums">{{ s.longBreakDurationMin }} {{ t('timing.minutes') }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted-foreground">{{ t('timing.longBreakEvery') }}</span>
              <span class="font-medium tabular-nums">{{ s.longBreakInterval }} {{ t('timing.breaks') }}</span>
            </div>
          </div>

          <p class="text-xs text-muted-foreground/70">
            {{ t('about.ruleDesc') }}
          </p>
        </CardContent>
      </Card>

      <div class="mt-auto pt-6 flex justify-end">
        <Button size="lg" @click="completeOnboarding" class="min-w-[140px]">
          <Play class="size-4" />
          {{ t('onboarding.startButton') }}
        </Button>
      </div>
    </section>

    <!-- Regular settings -->
    <section v-else class="px-7 pb-4 flex-1 min-h-0">
      <Tabs default-value="timing" class="flex h-full flex-col">
        <TabsList class="self-start">
          <TabsTrigger value="timing"><Clock class="size-3.5 mr-1" />{{ t('nav.timing') }}</TabsTrigger>
          <TabsTrigger value="options">{{ t('nav.options') }}</TabsTrigger>
          <TabsTrigger value="shortcuts">{{ t('nav.shortcuts') }}</TabsTrigger>
          <TabsTrigger value="about">{{ t('nav.about') }}</TabsTrigger>
        </TabsList>

        <!-- Timing -->
        <TabsContent value="timing" class="flex-1 min-h-0 overflow-y-auto pr-2 -mr-2">
          <Card>
            <CardHeader>
              <CardTitle>{{ t('timing.title') }}</CardTitle>
            </CardHeader>
            <CardContent class="space-y-7">
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.focusDuration') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.focusDurationMin }} {{ t('timing.minutes') }}</span>
                </div>
                <Slider :model-value="num(s.focusDurationMin ? [s.focusDurationMin] : undefined, 20)" :min="5" :max="60" :step="1" @update:model-value="(v:any) => setNum('focusDurationMin', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.focusDesc') }}</p>
              </div>
              <Separator />
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.shortBreak') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.shortBreakDurationSec }} {{ t('timing.seconds') }}</span>
                </div>
                <Slider :model-value="num(s.shortBreakDurationSec ? [s.shortBreakDurationSec] : undefined, 20)" :min="5" :max="120" :step="5" @update:model-value="(v:any) => setNum('shortBreakDurationSec', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.shortDesc') }}</p>
              </div>
              <Separator />
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.longBreak') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.longBreakDurationMin }} {{ t('timing.minutes') }}</span>
                </div>
                <Slider :model-value="num(s.longBreakDurationMin ? [s.longBreakDurationMin] : undefined, 5)" :min="1" :max="20" :step="1" @update:model-value="(v:any) => setNum('longBreakDurationMin', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.longDesc') }}</p>
              </div>
              <Separator />
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.longBreakEvery') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.longBreakInterval }} {{ t('timing.breaks') }}</span>
                </div>
                <Slider :model-value="num(s.longBreakInterval ? [s.longBreakInterval] : undefined, 4)" :min="1" :max="10" :step="1" @update:model-value="(v:any) => setNum('longBreakInterval', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.everyDesc') }}</p>
              </div>
              <Separator />
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.preBreakWarning') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.preBreakWarningSec }} {{ t('timing.seconds') }}</span>
                </div>
                <Slider :model-value="num(s.preBreakWarningSec !== undefined ? [s.preBreakWarningSec] : undefined, 10)" :min="0" :max="60" :step="5" @update:model-value="(v:any) => setNum('preBreakWarningSec', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.preDesc') }}</p>
              </div>
              <Separator />
              <div class="space-y-2">
                <div class="flex items-baseline justify-between">
                  <Label>{{ t('timing.idlePause') }}</Label>
                  <span class="text-sm tabular-nums text-primary font-medium">{{ s.idleThresholdMin }} {{ t('timing.minutes') }}</span>
                </div>
                <Slider :model-value="num(s.idleThresholdMin ? [s.idleThresholdMin] : undefined, 5)" :min="1" :max="30" :step="1" @update:model-value="(v:any) => setNum('idleThresholdMin', v)" />
                <p class="text-xs text-muted-foreground">{{ t('timing.idleDesc') }}</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <!-- Options -->
        <TabsContent value="options" class="flex-1 min-h-0 overflow-y-auto pr-2 -mr-2">
          <Card>
            <CardHeader><CardTitle>{{ t('options.title') }}</CardTitle></CardHeader>
            <CardContent class="space-y-1">
              <!-- appearance -->
              <div class="flex items-center justify-between gap-4 py-3">
                <div class="space-y-0.5">
                  <Label class="flex items-center gap-1.5"><Sun class="size-3.5" />{{ t('options.theme') }}</Label>
                  <p class="text-xs text-muted-foreground">{{ t('options.themeDesc') }}</p>
                </div>
                <Select :model-value="s.theme || 'system'" @update:model-value="changeTheme">
                  <SelectTrigger class="w-[150px]"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="system"><span class="flex items-center gap-2"><Monitor class="size-3.5" />{{ t('options.themeSystem') }}</span></SelectItem>
                    <SelectItem value="light"><span class="flex items-center gap-2"><Sun class="size-3.5" />{{ t('options.themeLight') }}</span></SelectItem>
                    <SelectItem value="dark"><span class="flex items-center gap-2"><Moon class="size-3.5" />{{ t('options.themeDark') }}</span></SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Separator />
              <!-- language -->
              <div class="flex items-center justify-between gap-4 py-3">
                <div class="space-y-0.5">
                  <Label class="flex items-center gap-1.5"><Languages class="size-3.5" />{{ t('onboarding.language') }}</Label>
                  <p class="text-xs text-muted-foreground">{{ locale === 'zh-CN' ? '简体中文' : 'English' }}</p>
                </div>
                <Select :model-value="locale" @update:model-value="changeLanguage">
                  <SelectTrigger class="w-[150px]"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="zh-CN">简体中文</SelectItem>
                    <SelectItem value="en">English</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Separator />
              <div class="flex items-start justify-between gap-4 py-3">
                <div class="space-y-0.5">
                  <Label>{{ t('options.longBreaks') }}</Label>
                  <p class="text-xs text-muted-foreground">{{ t('options.longBreaksDesc') }}</p>
                </div>
                <Switch :model-value="s.enableLongBreaks" @update:model-value="(v:any) => { s.enableLongBreaks = v; touch() }" />
              </div>
              <Separator />
              <div class="flex items-start justify-between gap-4 py-3">
                <div class="space-y-0.5">
                  <Label class="flex items-center gap-1.5"><Bell class="size-3.5" />{{ t('options.sound') }}</Label>
                  <p class="text-xs text-muted-foreground">{{ t('options.soundDesc') }}</p>
                </div>
                <Switch :model-value="s.soundEnabled" @update:model-value="(v:any) => { s.soundEnabled = v; touch() }" />
              </div>
              <Separator />
              <div class="flex items-start justify-between gap-4 py-3">
                <div class="space-y-0.5">
                  <Label>{{ t('options.autoStart') }}</Label>
                  <p class="text-xs text-muted-foreground">{{ t('options.autoStartDesc') }}</p>
                </div>
                <Switch :model-value="s.autoStart" @update:model-value="(v:any) => { s.autoStart = v; touch() }" />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <!-- Shortcuts -->
        <TabsContent value="shortcuts" class="flex-1 min-h-0 overflow-y-auto pr-2 -mr-2">
          <Card>
            <CardHeader><CardTitle>{{ t('shortcuts.title') }}</CardTitle></CardHeader>
            <CardContent class="space-y-4">
              <div class="grid grid-cols-2 gap-x-6 gap-y-4">
                <div class="space-y-1.5">
                  <Label>{{ t('shortcuts.startBreak') }}</Label>
                  <Input :model-value="s.shortcutStartBreak" @update:model-value="(v:any) => { s.shortcutStartBreak = v; touch() }" placeholder="Cmd+Shift+B" />
                </div>
                <div class="space-y-1.5">
                  <Label>{{ t('shortcuts.skipBreak') }}</Label>
                  <Input :model-value="s.shortcutSkipBreak" @update:model-value="(v:any) => { s.shortcutSkipBreak = v; touch() }" placeholder="Cmd+Shift+S" />
                </div>
                <div class="space-y-1.5">
                  <Label>{{ t('shortcuts.postponeBreak') }}</Label>
                  <Input :model-value="s.shortcutPostponeBreak" @update:model-value="(v:any) => { s.shortcutPostponeBreak = v; touch() }" placeholder="Cmd+Shift+P" />
                </div>
                <div class="space-y-1.5">
                  <Label>{{ t('shortcuts.preferences') }}</Label>
                  <Input :model-value="s.shortcutPreferences" @update:model-value="(v:any) => { s.shortcutPreferences = v; touch() }" placeholder="Cmd+Shift+," />
                </div>
              </div>
              <p class="text-xs text-muted-foreground">{{ t('shortcuts.hint', { code: 'Cmd+Shift+B' }) }}</p>
            </CardContent>
          </Card>
        </TabsContent>

        <!-- About -->
        <TabsContent value="about" class="flex-1 min-h-0 overflow-y-auto pr-2 -mr-2">
          <Card>
            <CardHeader>
              <CardTitle class="flex items-center gap-2"><Eye class="size-4 text-primary" />{{ t('app.name') }}</CardTitle>
              <CardDescription>{{ t('about.version') }} 0.1.0</CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <p class="text-sm text-muted-foreground leading-relaxed">{{ t('about.description') }}</p>
              <Separator />
              <div class="space-y-1">
                <Label class="text-foreground">{{ t('about.rule') }}</Label>
                <p class="text-sm text-muted-foreground">{{ t('about.ruleDesc') }}</p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </section>

    <!-- Footer: settings save only. Day-to-day controls (skip / postpone /
         reset) live in the tray menu and shortcuts, so the settings panel
         stays focused on configuration. -->
    <footer v-if="onboarded" class="flex items-center justify-end gap-2 px-7 py-4 border-t border-border/60">
      <span class="mr-auto text-xs font-medium text-emerald-500 transition-opacity" :class="saved ? 'opacity-100' : 'opacity-0'">
        <Check class="inline size-3 mr-1" />{{ t('actions.saved') }}
      </span>
      <Button size="sm" :disabled="!dirty" @click="save">{{ t('actions.save') }}</Button>
    </footer>
  </div>
</template>

<style scoped>
.prefs-root {
  /* Semi-opaque surface over the translucent window: keeps text readable while
     letting a hint of macOS vibrancy show through, and follows the theme. */
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--surface);
}
</style>
