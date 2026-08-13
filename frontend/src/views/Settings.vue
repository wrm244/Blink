<script setup lang="ts">
// Settings - 主设置视图。组合各 composable 和子组件。
import CloseConfirmDialog from '@/components/CloseConfirmDialog.vue'
import GButton from '@/components/GButton.vue'
import GlassPanel from '@/components/GlassPanel.vue'
import GToggle from '@/components/GToggle.vue'
import TimingTab from '@/components/SettingsTabs/TimingTab.vue'
import OptionsTab from '@/components/SettingsTabs/OptionsTab.vue'
import ShortcutsTab from '@/components/SettingsTabs/ShortcutsTab.vue'
import StatsTab from '@/components/SettingsTabs/StatsTab.vue'
import AboutTab from '@/components/SettingsTabs/AboutTab.vue'
import { setLocale, type Locale } from '@/i18n'
import { applyTheme, type Theme } from '@/theme'
import { Check, Clock, Coffee, Info, Keyboard, Languages, Monitor, Moon, PanelLeftClose, PanelLeftOpen, Pause, Play, RotateCcw, Sparkles, Sun, Timer, TrendingUp } from '@lucide/vue'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events } from '@wailsio/runtime'
import { BreakService } from '@bindings/blink'
import { useEngineState } from '@/composables/useEngineState'
import { useCycleStatus } from '@/composables/useCycleStatus'
import { useTimingConfig } from '@/composables/useTimingConfig'
import { useShortcutsConfig } from '@/composables/useShortcutsConfig'
import type { Settings } from '@bindings/blink/internal/config/models'

const { t, locale } = useI18n()
const tab = ref('timing')
const collapsed = ref(false)

// 引擎状态与控制
const { s, state, dirty, saved, ready, touch, save, discard, completeOnboarding, applyExternal } = useEngineState()

// ---- 关闭确认弹窗（Windows）----
// Go 端拦下窗口关闭后发来 blink:close-request，由这里弹窗询问；
// 用户的选择通过 ResolveClose 回传，Go 端据此隐藏窗口或退出应用。
// macOS 不安装关闭钩子，因此永远收不到该事件，弹窗也就不会出现。
const closeAsk = ref(false)
let offCloseAsk: (() => void) | undefined
let offNav: (() => void) | undefined
onMounted(() => {
  offCloseAsk = Events.On('blink:close-request', () => {
    closeAsk.value = true
    // 立刻回执，停掉后端的兜底定时器——弹窗已经在用户眼前了，
    // 接下来等多久都属于他的思考时间，不该被自动收场打断。
    BreakService.AckClose()
  })
  // 托盘菜单的"统计"项在窗口已存在时通过 blink:nav 事件请求切 tab
  // （窗口首次创建时走 GetPendingNav，见 tray.go showStats）。
  offNav = Events.On('blink:nav', (ev: { data: string }) => {
    if (ev.data) tab.value = ev.data
  })
})
onUnmounted(() => {
  offCloseAsk?.()
  offNav?.()
})

function resolveClose(action: 'background' | 'quit' | 'cancel', remember: boolean) {
  closeAsk.value = false
  // 后端在 remember 时会自己落盘，这里同步本地副本（含保存快照），
  // 否则用户下次改别的设置一保存，就会把刚记住的选择覆盖回旧值。
  if (remember && action !== 'cancel') applyExternal({ closeAction: action })
  BreakService.ResolveClose(action, remember)
}

// 数据加载完成、界面渲染就绪后通知 Go 端显示窗口，避免白屏
watch(ready, async (v) => {
  if (v) {
    // 托盘菜单的"统计"项会设置 pending nav，ready 后读取并切换。
    const nav = await BreakService.GetPendingNav()
    if (nav) tab.value = nav
    BreakService.ShowWindow()
  }
})

// 周期状态派生
const { onboarded, isPaused, isBreak, cycleRunning, progress, heroTitle, heroSub, phaseColor, heroTime, shortDone, breaksUntilLong, cycleStatusText } = useCycleStatus(state, s)

// 计时设置
const { timingFields, presetProfiles, activePreset, applyPreset, setNum } = useTimingConfig(s, touch)

// 快捷键配置
const { shortcutRows, conflictFor } = useShortcutsConfig(s, touch)

// ---- 主题与语言切换 ----
function changeLanguage(v: string) { setLocale(v as Locale); locale.value = v; s.language = v; touch() }
function changeTheme(v: string) { s.theme = v; applyTheme(v as Theme); touch() }

function startBreak() { BreakService.StartBreakNow() }
function togglePause() { isPaused.value ? BreakService.Resume() : BreakService.Pause() }
function reset() { BreakService.Reset() }

// 设置字段（布尔或字符串通用）
function setBool(field: keyof Settings, v: boolean | string) { (s as any)[field] = v; touch() }

// 主题选项
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

// 导航项
const navGeneral = computed(() => [
  { value: 'timing', label: t('nav.timing'), icon: Timer },
  { value: 'options', label: t('nav.options'), icon: Sparkles },
  { value: 'shortcuts', label: t('nav.shortcuts'), icon: Keyboard },
  { value: 'stats', label: t('nav.stats'), icon: TrendingUp },
  { value: 'about', label: t('nav.about'), icon: Info },
])

// 保存栏仅在编辑相关标签页显示
const editTabs = ['timing', 'options', 'shortcuts']
const showSaveBar = computed(() => onboarded.value && editTabs.includes(tab.value))
</script>

<template>
  <div class="root">
    <div class="backdrop" aria-hidden="true" />

    <div class="layout" :class="{ 'layout--collapsed': collapsed }">
      <!-- 侧栏（左，可折叠） -->
      <aside class="drawer">
        <GlassPanel strong class="drawer__panel">
          <button class="collapse-btn" @click="collapsed = !collapsed" :title="collapsed ? '' : t('nav.collapse')">
            <component :is="collapsed ? PanelLeftOpen : PanelLeftClose" class="size-4" />
            <span v-if="!collapsed" class="collapse-btn__txt">{{ t('nav.menu') }}</span>
          </button>

          <nav v-if="collapsed" class="nav nav--rail">
            <button v-for="item in navGeneral" :key="item.value" class="nav__item nav__item--rail" :class="{ 'nav__item--on': tab === item.value }" @click="tab = item.value" :title="item.label">
              <component :is="item.icon" class="size-4" />
            </button>
          </nav>

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

      <!-- 主区域（右） -->
      <main class="main">
        <GlassPanel strong class="hero">
          <div class="hero__top">
            <div class="hero__brand">
              <img class="hero__logo-img" src="/logo.png" alt="Blink" />
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
            <div class="hero__progress-bar" :style="{ transform: `scaleX(${progress})`, background: phaseColor }" />
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

          <!-- 引导（首次运行） -->
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

        <!-- 可滚动面板内容 -->
        <div v-if="onboarded" class="panels-wrap">
          <div class="panels">
            <TimingTab v-show="tab === 'timing'"
              :preset-profiles="presetProfiles" :active-preset="activePreset"
              :timing-fields="timingFields" :s="s"
              @apply-preset="applyPreset" @set-num="(f, v) => setNum(f as keyof Settings, v)"
            />
            <OptionsTab v-show="tab === 'options'" :s="s" @update="(k, v) => setBool(k as keyof Settings, v)" />
            <ShortcutsTab v-show="tab === 'shortcuts'"
              :shortcut-rows="shortcutRows" :conflict-for="conflictFor"
              @set-shortcut="(k, v) => { (s as any)[k] = v; touch() }"
            />
            <StatsTab v-show="tab === 'stats'" />
            <AboutTab v-show="tab === 'about'" />
          </div>

          <!-- 粘性保存栏 -->
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

    <!-- 关闭确认（Windows：Go 端拦截关闭后唤起） -->
    <CloseConfirmDialog :open="closeAsk" @resolve="resolveClose" />
  </div>
</template>

<style scoped>
.root { position: relative; height: 100vh; overflow: hidden; }
.backdrop { position: fixed; inset: 0; z-index: 0; background: linear-gradient(160deg, var(--bg-from), var(--bg-to)); }

/* ---- 固定高度双列布局 ---- */
.layout {
  position: relative; z-index: 1; height: 100%;
  display: grid; grid-template-columns: 268px 1fr; gap: 16px;
  padding: 44px 24px 20px; box-sizing: border-box;
  transition: grid-template-columns 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}
.layout--collapsed { grid-template-columns: 64px 1fr; }

/* ---- 侧栏 ---- */
.drawer { min-width: 0; min-height: 0; }
.drawer__panel { height: 100%; display: flex; flex-direction: column; gap: 6px; padding: 12px; animation: pm-fade-up 0.5s ease both; overflow: hidden; }
.collapse-btn { display: flex; align-items: center; gap: 8px; width: 100%; padding: 8px; font-size: 12.5px; font-weight: 500; font-family: inherit; color: var(--text-muted); background: transparent; border: 1px solid transparent; border-radius: 9px; cursor: pointer; transition: background 0.15s, color 0.15s; }
.collapse-btn:hover { background: var(--accent-soft); color: var(--text); }
.layout--collapsed .collapse-btn { justify-content: center; }
.collapse-btn__txt { letter-spacing: 0.02em; }

.nav__section { font-size: 10.5px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-faint); padding: 8px 10px 4px; }
.nav__item { display: flex; align-items: center; gap: 10px; width: 100%; padding: 9px 10px; font-size: 13.5px; font-weight: 500; font-family: inherit; color: var(--text-muted); background: transparent; border: none; border-radius: 9px; cursor: pointer; transition: background 0.15s, color 0.15s; }
.nav__item:hover { background: var(--accent-soft); color: var(--text); }
.nav__item--on { background: var(--accent); color: var(--on-accent); }
.nav--rail { display: flex; flex-direction: column; gap: 2px; align-items: center; }
.nav__item--rail { justify-content: center; width: 40px; padding: 9px; }

/* 周期统计 */
.cycle { margin-top: 6px; }
.cycle__grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; padding: 4px 4px 8px; }
.cycle__cell { padding: 12px 10px; border-radius: 11px; background: var(--glass-bg); border: 1px solid var(--glass-border); text-align: center; }
.cycle__value { font-size: 22px; font-weight: 300; color: var(--text); line-height: 1.1; }
.cycle__label { font-size: 10.5px; color: var(--text-faint); margin-top: 3px; }
.cycle__status { display: flex; align-items: center; gap: 7px; font-size: 12px; color: var(--text-muted); padding: 2px 10px 8px; }
.cycle__dot { width: 7px; height: 7px; border-radius: 50%; transition: background 0.2s; }
.cycle__dot--paused { animation: none; }
.cycle__reset { display: inline-flex; align-items: center; gap: 6px; margin: 2px 4px 0; padding: 6px 10px; font-size: 12px; font-family: inherit; color: var(--text-muted); background: transparent; border: 1px solid var(--glass-border); border-radius: 8px; cursor: pointer; transition: background 0.15s, color 0.15s; }
.cycle__reset:hover { background: var(--accent-soft); color: var(--text); }
.drawer__spacer { flex: 1; min-height: 8px; }
.quick { display: flex; flex-direction: column; gap: 10px; padding-top: 12px; border-top: 1px solid var(--glass-border); }
.quick__row { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 0 6px; }
.quick__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.seg-row { display: inline-flex; gap: 3px; padding: 3px; border-radius: 9px; background: var(--glass-bg); border: 1px solid var(--glass-border); }
.seg { min-width: 30px; padding: 5px 10px; font-size: 12px; font-weight: 500; font-family: inherit; color: var(--text-muted); background: transparent; border: none; border-radius: 6px; cursor: pointer; transition: background 0.15s, color 0.15s; }
.seg:hover { color: var(--text); }
.seg--on { background: var(--accent); color: var(--on-accent); }
.seg--icon { display: grid; place-items: center; padding: 5px 8px; }

/* ---- 主区域 ---- */
.main { display: flex; flex-direction: column; gap: 12px; min-width: 0; min-height: 0; }
.hero { padding: 14px 20px; flex-shrink: 0; animation: pm-fade-up 0.5s ease both; }
.hero__top { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.hero__brand { display: flex; gap: 10px; align-items: center; }
.hero__logo-img { width: 32px; height: 32px; object-fit: contain; }
.hero__title { margin: 0; font-size: 16px; font-weight: 600; letter-spacing: -0.01em; line-height: 1.2; }
.hero__sub { margin: 2px 0 0; display: flex; align-items: center; gap: 6px; font-size: 11.5px; color: var(--text-muted); }
.hero__dot { width: 7px; height: 7px; border-radius: 50%; transition: background 0.2s, box-shadow 0.2s; }
.hero__label { font-weight: 500; }
.hero__clock { text-align: right; }
.hero__time { font-size: 26px; font-weight: 250; line-height: 1; color: var(--text); }
.hero__progress { margin-top: 10px; height: 4px; border-radius: 9999px; background: var(--track); overflow: hidden; }
.hero__progress-bar { height: 100%; width: 100%; border-radius: 9999px; transform-origin: left center; transition: transform 1s linear; will-change: transform; }
.hero__bar { margin-top: 10px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.hero__hint { font-size: 12.5px; color: var(--text-muted); }
.hero__actions { display: flex; gap: 8px; }

/* 引导 */
.onboard { margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--glass-border); display: flex; flex-direction: column; gap: 14px; }
.onboard__intro { font-size: 13px; color: var(--text-muted); line-height: 1.6; }
.onboard__row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.onboard__field { display: flex; flex-direction: column; gap: 8px; }
.onboard__label { font-size: 12px; color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.onboard__start { align-self: flex-end; min-width: 150px; }

/* 面板区域 */
.panels-wrap { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 12px; }
.panels {
  flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain;
  padding: 16px; display: flex; flex-direction: column; gap: 12px;
  background: var(--glass-bg-strong);
  border: 1px solid var(--glass-border);
  border-radius: 16px;
  box-shadow: var(--glass-shadow);
}

/* 保存栏 */
.savebar { flex-shrink: 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 16px; border-radius: 14px; background: var(--glass-bg-strong); border: 1px solid var(--glass-border); box-shadow: var(--glass-shadow); transition: opacity 0.2s, transform 0.25s; }
.savebar__left { display: flex; align-items: center; gap: 8px; font-size: 12.5px; color: var(--text-muted); }
.savebar__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 3px var(--accent-soft); animation: pm-pulse 2.4s ease-in-out infinite; }
.savebar__saved { color: var(--phase-break); display: flex; align-items: center; gap: 5px; }
.savebar__idle { color: var(--text-faint); }
.savebar__right { display: flex; gap: 8px; align-items: center; }

@keyframes pm-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.pm-slide-up-enter-active, .pm-slide-up-leave-active { transition: opacity 0.25s, transform 0.25s; }
.pm-slide-up-enter-from, .pm-slide-up-leave-to { opacity: 0; transform: translateY(8px); }
</style>
