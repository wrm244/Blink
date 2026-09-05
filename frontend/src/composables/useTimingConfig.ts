// 计时设置字段与预设方案的组合式函数。
// 定义各计时字段的元数据（标签、范围、预设值）和一键应用的预设方案。

import type { Settings } from '@bindings/blink/internal/config/models'
import type { Reactive } from 'vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export interface TimingField {
  key: keyof Settings
  label: string
  desc: string
  unit?: string
  min: number
  max: number
  step: number
  presets: number[]
  // 秒为单位的字段可自定义 chip 和显示格式
  formatChip?: (v: number) => string
  formatValue?: (v: number) => string
}

export interface PresetProfile {
  id: 'eye' | 'pomo' | '52' | 'custom'
  name: string
  desc: string
  values?: Partial<Settings>
}

// 预设值列表（52 与 1020s=17m 对应 52/17 预设）
const focusPresets = [15, 20, 25, 30, 45, 50, 52]
const shortBreakPresets = [20, 60, 180, 300, 600, 1020]
const longBreakPresets = [5, 10, 15, 20]
const intervalPresets = [2, 3, 4, 5]
const preWarnPresets = [0, 10, 15, 30, 60]
const idlePresets = [1, 3, 5, 10, 15]

/** 将秒数格式化为友好的中文标签 */
function fmtSecs(v: number): string {
  if (v <= 0) return '0'
  if (v < 60) return `${v} 秒`
  const m = Math.floor(v / 60)
  const s = v % 60
  return s === 0 ? `${m} 分` : `${m} 分 ${s} 秒`
}

/** 将秒数格式化为简短的 chip 标签 */
function fmtSecsChip(v: number): string {
  if (v < 60) return `${v}s`
  const m = Math.floor(v / 60)
  const s = v % 60
  return s === 0 ? `${m}m` : `${m}m${s}s`
}

export function useTimingConfig(s: Reactive<Settings>, touch: () => void) {
  const { t } = useI18n()

  /** 整数 + 单位（分 / 次）的完整显示串 */
  const fmtUnit = (unit: string) => (v: number) => `${v} ${unit}`
  const fmtMinutes = fmtUnit(t('timing.minutes'))
  const fmtBreaks = fmtUnit(t('timing.breaks'))
  /** 休息前提醒的 chip：0 显示"关闭" */
  const chipWarn = (v: number) => (v === 0 ? t('timing.off') : fmtSecsChip(v))
  /** 休息前提醒的完整显示：0 显示"关闭" */
  const fmtWarn = (v: number) => (v === 0 ? t('timing.off') : fmtSecs(v))

  // 计时字段定义
  const timingFields = computed<TimingField[]>(() => [
    {
      key: 'focusDurationMin',
      label: t('timing.focusDuration'),
      desc: t('timing.focusDesc'),
      unit: t('timing.minutes'),
      min: 5, max: 60, step: 1,
      presets: focusPresets,
      formatValue: fmtMinutes,
    },
    {
      key: 'shortBreakDurationSec',
      label: t('timing.shortBreak'),
      desc: t('timing.shortDesc'),
      min: 5, max: 1200, step: 5,
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
      formatValue: fmtMinutes,
    },
    {
      key: 'longBreakInterval',
      label: t('timing.longBreakEvery'),
      desc: t('timing.everyDesc'),
      unit: t('timing.breaks'),
      min: 1, max: 10, step: 1,
      presets: intervalPresets,
      formatValue: fmtBreaks,
    },
    {
      key: 'preBreakWarningSec',
      label: t('timing.preBreakWarning'),
      desc: t('timing.preDesc'),
      min: 0, max: 60, step: 5,
      presets: preWarnPresets,
      formatChip: chipWarn,
      formatValue: fmtWarn,
    },
    {
      key: 'idleThresholdMin',
      label: t('timing.idlePause'),
      desc: t('timing.idleDesc'),
      unit: t('timing.minutes'),
      min: 1, max: 30, step: 1,
      presets: idlePresets,
      formatValue: fmtMinutes,
    },
  ])

  // 预设方案
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
      // 52/17 节奏：52 分钟专注 + 17 分钟（1020 秒）短休息
      values: { focusDurationMin: 52, shortBreakDurationSec: 1020, longBreakDurationMin: 20, longBreakInterval: 3, preBreakWarningSec: 60, idleThresholdMin: 10 },
    },
    { id: 'custom', name: t('timing.presetCustom'), desc: '' },
  ])

  // 当前激活的预设
  const activePreset = computed<PresetProfile['id']>(() => {
    const match = (v?: Partial<Settings>) => v && Object.entries(v).every(([k, val]) => (s as any)[k] === val)
    for (const p of presetProfiles.value) {
      if (p.values && match(p.values)) return p.id
    }
    return 'custom'
  })

  /** 应用预设方案 */
  function applyPreset(p: PresetProfile) {
    if (!p.values) return
    Object.entries(p.values).forEach(([k, v]) => { (s as any)[k] = v })
    touch()
  }

  /** 设置数值字段 */
  function setNum(field: keyof Settings, v: number) { (s as any)[field] = v; touch() }

  return { timingFields, presetProfiles, activePreset, applyPreset, setNum }
}
