<script setup lang="ts">
// StatsTab - 统计标签页：日历热力图 + 当月汇总 + 选中日详情列表。
import GlassPanel from '@/components/GlassPanel.vue'
import { CalendarDays, ChevronLeft, ChevronRight, Clock, Coffee, Activity, CalendarCheck } from '@lucide/vue'
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events } from '@wailsio/runtime'
import { BreakService } from '@bindings/blink'
import { useStatsData, fmtDuration, fmtMinutes, intensity } from '@/composables/useStatsData'
import type { State } from '@bindings/blink/internal/breakengine/models'

const { t, locale } = useI18n()
// 时长格式化按界面语言输出（fmtDuration 不依赖 vue-i18n，需手动传）。
const durLocale = computed(() => (locale.value === 'zh-CN' ? 'zh' : 'en'))
const { year, month, dayMap, loading, selectedDay, monthSummary, selectedStats, reload, prevMonth, nextMonth, prevYear, nextYear, goToday } = useStatsData()

// 周标题：周一开头（中文习惯）。en locale 用周日开头。
const weekStartsOnMonday = computed(() => locale.value === 'zh-CN')
const weekHeaders = computed(() => {
  if (weekStartsOnMonday.value) {
    return [t('stats.week.mon'), t('stats.week.tue'), t('stats.week.wed'), t('stats.week.thu'), t('stats.week.fri'), t('stats.week.sat'), t('stats.week.sun')]
  }
  return [t('stats.week.sun'), t('stats.week.mon'), t('stats.week.tue'), t('stats.week.wed'), t('stats.week.thu'), t('stats.week.fri'), t('stats.week.sat')]
})

// 月份名。
const monthNames = computed(() => {
  const months = weekStartsOnMonday.value
    ? ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月']
    : ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  return months
})

// 生成日历网格：按周分行，填充前导空格和尾部空格，使总单元数为 7 的倍数。
interface Cell {
  day: number       // 日期号（1-31），0 表示占位空格
  hasData: boolean
  focusSec: number
  breakSec: number
  shortBreaks: number
  longBreaks: number
  isToday: boolean
  isSelected: boolean
}

const calendarCells = computed<Cell[]>(() => {
  const firstDay = new Date(year.value, month.value - 1, 1)
  const daysInMonth = new Date(year.value, month.value, 0).getDate()
  // 首日是星期几（JS: 0=周日）。若周一开头，偏移量调整。
  let startDow = firstDay.getDay()
  if (weekStartsOnMonday.value) {
    startDow = startDow === 0 ? 6 : startDow - 1
  }
  const today = new Date()
  const cells: Cell[] = []
  // 前导空格
  for (let i = 0; i < startDow; i++) {
    cells.push({ day: 0, hasData: false, focusSec: 0, breakSec: 0, shortBreaks: 0, longBreaks: 0, isToday: false, isSelected: false })
  }
  for (let d = 1; d <= daysInMonth; d++) {
    const stats = dayMap.value[d]
    const isToday = today.getFullYear() === year.value && today.getMonth() + 1 === month.value && today.getDate() === d
    cells.push({
      day: d,
      hasData: !!stats && (stats.focusSec > 0 || stats.breakSec > 0),
      focusSec: stats?.focusSec || 0,
      breakSec: stats?.breakSec || 0,
      shortBreaks: stats?.shortBreaks || 0,
      longBreaks: stats?.longBreaks || 0,
      isToday,
      isSelected: d === selectedDay.value,
    })
  }
  // 尾部补齐到 7 的倍数
  while (cells.length % 7 !== 0) {
    cells.push({ day: 0, hasData: false, focusSec: 0, breakSec: 0, shortBreaks: 0, longBreaks: 0, isToday: false, isSelected: false })
  }
  return cells
})

function selectDay(d: number) {
  if (d === 0) return
  selectedDay.value = d
}

const monthLabel = computed(() => monthNames.value[month.value - 1])

// 监听引擎 tick：休息结束（阶段从 break 切回 focus）时自动刷新统计，
// 避免用户手动点"今天"才看到更新。只在阶段实际变化时 reload，不每秒刷新。
let lastPhase = ''
let offTick: (() => void) | undefined

onMounted(() => {
  reload()
  offTick = Events.On('blink:tick', (ev: { data: State }) => {
    const phase = ev.data?.phase
    if (!phase) return
    // 休息结束：上一刻是 break，现在不是 break（切回 focus/prebreak/idle）。
    const wasBreak = lastPhase === 'shortbreak' || lastPhase === 'longbreak'
    const isBreak = phase === 'shortbreak' || phase === 'longbreak'
    lastPhase = phase
    if (wasBreak && !isBreak) {
      reload()
    }
  })
})

onUnmounted(() => offTick?.())
</script>

<template>
  <section v-show="true" class="panel-stack">
    <!-- 当月汇总 -->
    <GlassPanel class="summary">
      <div class="summary__head">
        <CalendarDays class="size-4" />
        <span class="summary__title">{{ t('stats.monthTitle') }}</span>
        <span class="summary__month tabular-nums">{{ year }} · {{ monthLabel }}</span>
      </div>
      <div class="summary__grid">
        <div class="summary__cell">
          <div class="summary__value tabular-nums">{{ fmtDuration(monthSummary.focusSec, durLocale) }}</div>
          <div class="summary__label"><Clock class="size-3" />{{ t('stats.focusTotal') }}</div>
        </div>
        <div class="summary__cell">
          <div class="summary__value tabular-nums">{{ fmtDuration(monthSummary.breakSec, durLocale) }}</div>
          <div class="summary__label"><Coffee class="size-3" />{{ t('stats.breakTotal') }}</div>
        </div>
        <div class="summary__cell">
          <div class="summary__value tabular-nums">{{ monthSummary.shortBreaks + monthSummary.longBreaks }}</div>
          <div class="summary__label"><Activity class="size-3" />{{ t('stats.breakCount') }}</div>
        </div>
        <div class="summary__cell">
          <div class="summary__value tabular-nums">{{ monthSummary.activeDays }}</div>
          <div class="summary__label"><CalendarCheck class="size-3" />{{ t('stats.activeDays') }}</div>
        </div>
      </div>
    </GlassPanel>

    <!-- 日历 -->
    <GlassPanel class="calendar">
      <div class="cal__nav">
        <div class="cal__group">
          <button class="cal__btn" @click="prevYear" :title="t('stats.prevYear')"><ChevronLeft class="size-3" /><ChevronLeft class="size-3 cal__btn--double" /></button>
          <span class="cal__year tabular-nums">{{ year }}</span>
          <button class="cal__btn" @click="nextYear" :title="t('stats.nextYear')"><ChevronRight class="size-3" /><ChevronRight class="size-3 cal__btn--double" /></button>
        </div>
        <div class="cal__group">
          <button class="cal__btn" @click="prevMonth" :title="t('stats.prevMonth')"><ChevronLeft class="size-4" /></button>
          <span class="cal__title">{{ monthLabel }}</span>
          <button class="cal__btn" @click="nextMonth" :title="t('stats.nextMonth')"><ChevronRight class="size-4" /></button>
        </div>
        <button class="cal__today" @click="goToday">{{ t('stats.today') }}</button>
      </div>
      <div class="cal__weekdays">
        <span v-for="(w, i) in weekHeaders" :key="i" class="cal__wd">{{ w }}</span>
      </div>
      <div class="cal__grid">
        <button
          v-for="(cell, i) in calendarCells"
          :key="i"
          class="cal__cell"
          :class="{
            'cal__cell--empty': cell.day === 0,
            'cal__cell--today': cell.isToday,
            'cal__cell--selected': cell.isSelected,
            [`cal__cell--i${intensity(cell.focusSec)}`]: cell.day !== 0,
          }"
          :disabled="cell.day === 0"
          @click="selectDay(cell.day)"
        >
          <span class="cal__day">{{ cell.day || '' }}</span>
          <span v-if="cell.hasData" class="cal__mins">{{ fmtMinutes(cell.focusSec) }}</span>
        </button>
      </div>
      <!-- 图例 -->
      <div class="cal__legend">
        <span class="cal__legend-text">{{ t('stats.less') }}</span>
        <span class="cal__legend-box cal__cell--i0" />
        <span class="cal__legend-box cal__cell--i1" />
        <span class="cal__legend-box cal__cell--i2" />
        <span class="cal__legend-box cal__cell--i3" />
        <span class="cal__legend-box cal__cell--i4" />
        <span class="cal__legend-text">{{ t('stats.more') }}</span>
      </div>
    </GlassPanel>

    <!-- 选中日详情 -->
    <GlassPanel class="detail">
      <div class="detail__head">
        <span class="detail__title">
          {{ t('stats.dayDetail') }}
          <span v-if="selectedDay" class="detail__date">{{ year }} {{ monthLabel }} {{ selectedDay }}{{ t('stats.daySuffix') }}</span>
        </span>
      </div>
      <div v-if="selectedStats" class="detail__grid">
        <div class="detail__row">
          <Clock class="size-4 detail__icon" />
          <span class="detail__label">{{ t('stats.focusTime') }}</span>
          <span class="detail__value tabular-nums">{{ fmtDuration(selectedStats.focusSec, durLocale) }}</span>
        </div>
        <div class="detail__row">
          <Coffee class="size-4 detail__icon" />
          <span class="detail__label">{{ t('stats.breakTime') }}</span>
          <span class="detail__value tabular-nums">{{ fmtDuration(selectedStats.breakSec, durLocale) }}</span>
        </div>
        <div class="detail__row">
          <span class="detail__label">{{ t('stats.shortBreaks') }}</span>
          <span class="detail__value tabular-nums">{{ selectedStats.shortBreaks }} {{ t('stats.times') }}</span>
        </div>
        <div class="detail__row">
          <span class="detail__label">{{ t('stats.longBreaks') }}</span>
          <span class="detail__value tabular-nums">{{ selectedStats.longBreaks }} {{ t('stats.times') }}</span>
        </div>
      </div>
      <div v-else class="detail__empty">
        {{ selectedDay ? t('stats.noData') : t('stats.selectDay') }}
      </div>
    </GlassPanel>
  </section>
</template>

<style scoped>
.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: pm-fade-up 0.4s ease both;
}

/* ---- 当月汇总 ---- */
.summary { padding: 18px 20px; }
.summary__head { display: flex; align-items: center; gap: 8px; margin-bottom: 14px; }
.summary__title { font-size: 14px; font-weight: 600; color: var(--text); }
.summary__month { font-size: 13px; color: var(--text-muted); margin-left: auto; }
.summary__grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.summary__cell { text-align: center; padding: 10px 6px; border-radius: 10px; background: var(--glass-bg); border: 1px solid var(--glass-border); }
.summary__value { font-size: 18px; font-weight: 300; color: var(--text); line-height: 1.2; }
.summary__label { font-size: 11px; color: var(--text-faint); margin-top: 4px; display: flex; align-items: center; justify-content: center; gap: 4px; }

/* ---- 日历 ---- */
.calendar { padding: 16px 18px; }
.cal__nav { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.cal__group { display: flex; align-items: center; gap: 4px; }
.cal__btn { display: grid; place-items: center; width: 26px; height: 26px; font-family: inherit; color: var(--text-muted); background: var(--glass-bg); border: 1px solid var(--glass-border); border-radius: 7px; cursor: pointer; transition: background 0.15s, color 0.15s; position: relative; }
.cal__btn:hover { background: var(--accent-soft); color: var(--text); }
/* 双箭头：第二个图标叠加偏移，形成 «/» 效果 */
.cal__btn--double { position: absolute; left: 7px; top: 50%; transform: translateY(-50%); }
.cal__year { font-size: 14px; font-weight: 600; color: var(--text); min-width: 42px; text-align: center; }
.cal__title { font-size: 14px; font-weight: 600; color: var(--text); min-width: 56px; text-align: center; }
.cal__today { margin-left: auto; padding: 5px 12px; font-size: 12px; font-weight: 500; font-family: inherit; color: var(--text-muted); background: var(--glass-bg); border: 1px solid var(--glass-border); border-radius: 7px; cursor: pointer; transition: background 0.15s, color 0.15s; }
.cal__today:hover { background: var(--accent-soft); color: var(--text); }

.cal__weekdays { display: grid; grid-template-columns: repeat(7, 1fr); gap: 4px; margin-bottom: 6px; }
.cal__wd { font-size: 11px; font-weight: 500; color: var(--text-faint); text-align: center; padding: 4px 0; }

.cal__grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 4px; }
.cal__cell { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1px; height: 34px; font-family: inherit; border: 1px solid transparent; border-radius: 7px; cursor: pointer; transition: background 0.15s, border-color 0.15s; background: var(--track); }
.cal__cell:hover { border-color: var(--accent); }
.cal__cell--empty { background: transparent; border: none; cursor: default; }
.cal__cell--empty:hover { border: none; }
.cal__cell--today { border-color: var(--accent); }
.cal__cell--selected { border-color: var(--accent); box-shadow: 0 0 0 2px var(--accent-soft); }
.cal__day { font-size: 11px; font-weight: 500; color: var(--text-muted); line-height: 1; }
.cal__cell--i1 .cal__day, .cal__cell--i2 .cal__day, .cal__cell--i3 .cal__day, .cal__cell--i4 .cal__day { color: var(--text); }
.cal__mins { font-size: 9px; color: var(--text-faint); line-height: 1; }

/* 热力强度背景色 — GitHub 贡献图绿色配色 */
.cal__cell--i0 { background: var(--track); }
.cal__cell--i1 { background: #9be9a8; }
.cal__cell--i2 { background: #40c463; }
.cal__cell--i3 { background: #30a14e; }
.cal__cell--i4 { background: #216e39; }
/* 深色主题：GitHub 深色模式绿色 */
:is(.dark) .cal__cell--i1 { background: #0e4429; }
:is(.dark) .cal__cell--i2 { background: #006d32; }
:is(.dark) .cal__cell--i3 { background: #26a641; }
:is(.dark) .cal__cell--i4 { background: #39d353; }
/* 深色高强度格文字用白色保证对比度 */
:is(.dark) .cal__cell--i3 .cal__day, :is(.dark) .cal__cell--i4 .cal__day { color: #fff; }
:is(.dark) .cal__cell--i3 .cal__mins, :is(.dark) .cal__cell--i4 .cal__mins { color: rgba(255,255,255,0.75); }
/* 浅色高强度格文字用白色 */
.cal__cell--i3 .cal__day, .cal__cell--i4 .cal__day { color: #fff; }
.cal__cell--i3 .cal__mins, .cal__cell--i4 .cal__mins { color: rgba(255,255,255,0.8); }

.cal__legend { display: flex; align-items: center; gap: 5px; margin-top: 12px; justify-content: flex-end; }
.cal__legend-text { font-size: 10.5px; color: var(--text-faint); }
.cal__legend-box { width: 14px; height: 14px; border-radius: 4px; }

/* ---- 选中日详情 ---- */
.detail { padding: 16px 20px; }
.detail__head { margin-bottom: 12px; }
.detail__title { font-size: 14px; font-weight: 600; color: var(--text); display: flex; align-items: baseline; gap: 8px; }
.detail__date { font-size: 13px; font-weight: 400; color: var(--text-muted); }
.detail__grid { display: flex; flex-direction: column; gap: 10px; }
.detail__row { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border-radius: 9px; background: var(--glass-bg); border: 1px solid var(--glass-border); }
.detail__icon { color: var(--text-muted); }
.detail__label { font-size: 13px; color: var(--text-muted); }
.detail__value { font-size: 14px; font-weight: 500; color: var(--text); margin-left: auto; }
.detail__empty { font-size: 13px; color: var(--text-faint); padding: 16px 10px; text-align: center; }
</style>
