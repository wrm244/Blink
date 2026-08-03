// 专注周期状态派生的组合式函数。
// 从引擎 State 计算是否暂停、是否休息中、进度比例、英雄区标题等。

import { computed } from 'vue'
import type { Ref, Reactive } from 'vue'
import type { State } from '@bindings/blink/internal/breakengine/models'
import type { Settings } from '@bindings/blink/internal/config/models'
import { useI18n } from 'vue-i18n'

export function useCycleStatus(
  state: Ref<State>,
  s: Reactive<Settings>,
) {
  const { t } = useI18n()

  // 已完成引导
  const onboarded = computed(() => s.onboarded === true)

  // 周期是否"暂停"（用户暂停或引擎空闲）
  const isPaused = computed(() => !!state.value.paused || state.value.phase === 'idle')
  // 是否在休息中
  const isBreak = computed(() => state.value.phase === 'shortbreak' || state.value.phase === 'longbreak')
  // 周期是否运行中
  const cycleRunning = computed(() => !!state.value.phase && state.value.phase !== 'idle' && !state.value.paused)

  // 进度比例 (0 ~ 1)
  const progress = computed(() => {
    const total = state.value.totalSec
    const rem = state.value.remainingSec
    if (!total || total <= 0) return 0
    return Math.max(0, Math.min(1, 1 - rem / total))
  })

  // 英雄区标题
  const heroTitle = computed(() => {
    if (isPaused.value) return t('hero.paused')
    switch (state.value.phase) {
      case 'focusing': return t('hero.focusing')
      case 'shortbreak': case 'longbreak': return t('hero.resting')
      case 'prebreak': return t('hero.resting')
      default: return t('hero.startFocus')
    }
  })

  // 英雄区副标题
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

  // 阶段颜色
  const phaseColor = computed(() => {
    if (isPaused.value) return 'var(--phase-pause)'
    switch (state.value.phase) {
      case 'shortbreak': return 'var(--phase-break)'
      case 'longbreak': return 'var(--phase-long)'
      case 'prebreak': return 'var(--phase-prebreak)'
      default: return 'var(--phase-focus)'
    }
  })

  // 英雄区倒计时文本
  const heroTime = computed(() => fmt(state.value.remainingSec))

  // 短休息完成次数
  const shortDone = computed(() => state.value.shortBreakCount ?? 0)
  // 距下次长休息次数
  const breaksUntilLong = computed(() => state.value.breaksUntilLong ?? 0)

  // 侧栏状态文本
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

  return {
    onboarded, isPaused, isBreak, cycleRunning,
    progress, heroTitle, heroSub, phaseColor, heroTime,
    shortDone, breaksUntilLong, cycleStatusText,
  }
}

/** 将秒数格式化为 M:SS */
function fmt(sec: number): string {
  const m = Math.floor(sec / 60)
  const ss = sec % 60
  return `${m}:${String(ss).padStart(2, '0')}`
}
