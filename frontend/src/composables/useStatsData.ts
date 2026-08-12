// 统计数据获取的组合式函数。
// 负责按月份加载日历数据、按日期加载详情，以及格式化时长。

import { ref, computed } from 'vue'
import { BreakService } from '@bindings/blink'
import type { DayStats } from '@bindings/blink/internal/stats/models'

// DayMap 是日历数据：key 为日期号（1-31），value 为该日统计。
type DayMap = Record<number, DayStats>

export function useStatsData() {
  const year = ref(new Date().getFullYear())
  const month = ref(new Date().getMonth() + 1) // 1-12
  const dayMap = ref<DayMap>({})
  const loading = ref(false)

  // 选中日期（默认今天），用于详情列表。
  const selectedDay = ref(new Date().getDate())

  async function loadMonth(y: number, m: number) {
    loading.value = true
    try {
      const raw = await BreakService.GetMonthlyStats(y, m)
      // 绑定返回的是以字符串 key（"1".."31"）的可选映射对象，转为数字 key。
      const out: DayMap = {}
      for (const [k, v] of Object.entries(raw || {})) {
        if (v) out[Number(k)] = v
      }
      dayMap.value = out
    } finally {
      loading.value = false
    }
  }

  async function reload() {
    await loadMonth(year.value, month.value)
  }

  function prevMonth() {
    let m = month.value - 1
    let y = year.value
    if (m < 1) { m = 12; y-- }
    year.value = y
    month.value = m
    selectedDay.value = 0 // 切月后清除选中
    reload()
  }

  function nextMonth() {
    let m = month.value + 1
    let y = year.value
    if (m > 12) { m = 1; y++ }
    year.value = y
    month.value = m
    selectedDay.value = 0
    reload()
  }

  function prevYear() {
    year.value -= 1
    selectedDay.value = 0
    reload()
  }

  function nextYear() {
    year.value += 1
    selectedDay.value = 0
    reload()
  }

  function goToday() {
    const now = new Date()
    year.value = now.getFullYear()
    month.value = now.getMonth() + 1
    selectedDay.value = now.getDate()
    reload()
  }

  // 当月汇总：累加所有有数据日期的统计。
  const monthSummary = computed(() => {
    let focus = 0, brk = 0, shorts = 0, longs = 0, activeDays = 0
    for (const d of Object.values(dayMap.value)) {
      focus += d.focusSec || 0
      brk += d.breakSec || 0
      shorts += d.shortBreaks || 0
      longs += d.longBreaks || 0
      if ((d.focusSec || 0) > 0) activeDays++
    }
    return { focusSec: focus, breakSec: brk, shortBreaks: shorts, longBreaks: longs, activeDays }
  })

  // 当前选中日的统计。
  const selectedStats = computed(() => dayMap.value[selectedDay.value] || null)

  return {
    year, month, dayMap, loading, selectedDay,
    monthSummary, selectedStats,
    reload, prevMonth, nextMonth, prevYear, nextYear, goToday,
  }
}

// fmtDuration 将秒数格式化为人类可读时长，如 "1时23分" / "20分" / "45秒"。
export function fmtDuration(sec: number): string {
  if (sec <= 0) return '0'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (h > 0) return `${h}时${m}分`
  if (m > 0) return `${m}分${s > 0 ? s + '秒' : ''}`
  return `${s}秒`
}

// fmtMinutes 将秒数格式化为分钟（向下取整），用于日历格内的紧凑显示。
export function fmtMinutes(sec: number): string {
  if (sec <= 0) return ''
  const m = Math.floor(sec / 60)
  if (m > 0) return `${m}m`
  return `${sec}s`
}

// intensity 根据专注秒数返回热力强度等级（0-4），用于日历格背景色深浅。
export function intensity(focusSec: number): number {
  if (focusSec <= 0) return 0
  if (focusSec < 600) return 1   // <10min
  if (focusSec < 1800) return 2  // <30min
  if (focusSec < 3600) return 3  // <1h
  return 4                       // >=1h
}
