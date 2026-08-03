<script setup lang="ts">
import { ChevronRight } from '@lucide/vue'
import { Events } from '@wailsio/runtime'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BreakService } from '@bindings/blink'
import { Phase } from '@bindings/blink/internal/breakengine/models'
import { useRestScreen } from '@/composables/useRestScreen'

const { t, locale } = useI18n()

// ---- 休息屏幕内容（每日一句 + 壁纸） ----
const { quote, quoteMeta, bgUrl, enrichRestScreen } = useRestScreen()

// ---- 休息引擎状态（倒计时） ----
const phase = ref<string>('')
const remaining = ref(0)
const total = ref(0)
let off: (() => void) | undefined

// ---- 墙钟（当前日期+时间，精确到秒） ----
// `now` 在每个整秒边界刷新，确保显示的秒数不跳动。
// 每次重新计算到下一秒边界的超时，而非固定 1s 间隔（会漂移）。
const now = ref(new Date())
let clockTimer: ReturnType<typeof setTimeout> | undefined

onMounted(async () => {
  // --- 休息屏幕内容（每日一句 + 壁纸） ---
  // 在状态绑定之前发起网络请求，让休息屏幕尽早填充内容
  enrichRestScreen()

  // --- 倒计时 ---
  const st = await BreakService.GetState()
  phase.value = st.phase
  remaining.value = st.remainingSec
  total.value = st.totalSec
  off = Events.On('blink:tick', (ev: { data: { phase: string; remainingSec: number; totalSec: number } }) => {
    phase.value = ev.data.phase
    remaining.value = ev.data.remainingSec
    total.value = ev.data.totalSec
  })

  // --- 墙钟：对齐到秒边界 ---
  now.value = new Date()
  scheduleClockTick()
})

onUnmounted(() => {
  off?.()
  if (clockTimer !== undefined) clearTimeout(clockTimer)
})

/**
 * 将下一次墙钟更新调度到下一个整秒边界。
 * 使用 setTimeout（每次重新调度）而非 setInterval，
 * 让显示锁定在真实秒边界上——setInterval(1000) 因为回调可能
 * 延迟几毫秒而累积漂移。
 */
function scheduleClockTick() {
  const msToNextSecond = 1000 - (Date.now() % 1000)
  clockTimer = setTimeout(() => {
    now.value = new Date()
    scheduleClockTick()
  }, msToNextSecond)
}

// ---- 派生值 ----

const isLong = computed(() => phase.value === Phase.PhaseLongBreak)
const label = computed(() => (isLong.value ? t('break.longLabel') : t('break.shortLabel')))

// 每种休息类型使用单一去饱和强调色。长休息用柔和灰绿，
// 短休息用中性板岩色。无渐变。
const accent = computed(() => (isLong.value ? '#6b8f7a' : '#8a96a8'))

// 进度：已过时间占比 (0 -> 1)
const fraction = computed(() => {
  if (total.value <= 0) return 0
  return Math.max(0, Math.min(1, 1 - remaining.value / total.value))
})

// 休息倒计时格式化为 M:SS
const clock = computed(() => {
  const m = Math.floor(remaining.value / 60)
  const s = remaining.value % 60
  return `${m}:${String(s).padStart(2, '0')}`
})

/**
 * 当前日期+时间，本地化并精确到秒。
 * 使用 Intl.DateTimeFormat 让星期/月份名称跟随活动语言（zh-CN / en）。
 * hour12:false 在两种语言下都给出 24 小时制，休息屏幕上读起来更清晰。
 * formatter 按 locale 缓存（构造时进行语言协商）而非每次 tick 重建。
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

// 每日一句加载后才占空间，这样英雄区倒计时不会因请求完成而跳动
const hasQuote = computed(() => quote.value.length > 0)
// 日期式名句（如"腊月廿四：小年"）较短且带前导"：" -
// 带早期冒号的长名言不应被误判。
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
      <!-- 当前日期+时间：顶部次要信息，非常柔和 -->
      <p class="datetime tabular-nums">{{ datetime }}</p>

      <!-- 每日一句：安静灵感，加载后渲染 -->
      <div v-if="hasQuote" class="quote">
        <p class="quote__text" :class="{ 'quote__text--date': isDate }">{{ quote }}</p>
        <p v-if="quoteMeta" class="quote__meta">{{ quoteMeta }}</p>
      </div>

      <p class="label">{{ label }}</p>

      <!-- 休息倒计时：核心元素 -->
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
 * 休息遮罩 - 全屏、最高优先级的休息屏幕。
 * 休息倒计时是核心；墙钟在其上方作为安静的上下文。
 * 无 SVG 环（避免方形视口伪影），无环境光斑，无强调色渐变。
 * 窗口创建在 ScreenSaver 层级（见 windows.go），覆盖 Dock 和菜单栏。
 */
.overlay {
  position: fixed;
  inset: 0;
  z-index: 2147483647;
  display: flex;
  align-items: center;
  justify-content: center;
  /* 板岩色渐变兜底 - 壁纸加载后替换 */
  background: linear-gradient(160deg, #181a1d 0%, #1f2227 50%, #15171a 100%);
  color: #e8e6e1;
  user-select: none;
  -webkit-user-select: none;
  -webkit-touch-callout: none;
  cursor: default;
  overflow: hidden;
}
/* 双保险：永不在照片上显示选中高亮 */
.overlay ::selection {
  background: transparent;
}

/* Bing "每日图片"壁纸：全出血覆盖在一切之后 */
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

/* 照片上的暗色遮罩，确保倒计时文字在任何光线下都清晰可读。
   实色渐变，无模糊（开销低，且照片本身已足够柔和） */
.scrim {
  position: absolute;
  inset: 0;
  z-index: 1;
  background: linear-gradient(180deg, rgba(8, 10, 12, 0.66) 0%, rgba(8, 10, 12, 0.45) 42%, rgba(8, 10, 12, 0.72) 100%);
  pointer-events: none;
}

/* 单一柔和暗角，将视觉焦点引向中心 */
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

/* 墙钟：最小文字 - 安静上下文，但比最淡层级稍亮
   因为现在叠在照片上 */
.datetime {
  margin: 0 0 -12px;
  font-size: 15px;
  font-weight: 500;
  letter-spacing: 0.02em;
  color: rgba(240, 238, 233, 0.6);
  text-shadow: 0 1px 6px rgba(0, 0, 0, 0.5);
}

/* 每日一句：最多两行，柔和投影确保叠在照片上可读。
   日期式名句（如"腊月廿四：小年"）使用稍大、更平静的处理，
   并省略出处行。 */
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
  padding-left: 0.42em; /* 偏移以补偿字间距的视觉居中 */
}

/* 核心倒计时 - 主导元素。中等字重 + 柔和投影让细数字在照片上保持清晰；
   tabular-nums 防止数字跳动 */
.clock {
  font-size: clamp(140px, 22vmin, 210px);
  font-weight: 400;
  line-height: 0.9;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  color: #f5f3ef;
  text-shadow: 0 2px 18px rgba(0, 0, 0, 0.55), 0 8px 40px rgba(0, 0, 0, 0.3);
}

/* 倒计时下方的细进度线。轨道几乎不可见；
   填充是屏幕上唯一的色彩强调 */
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

/* 跳过按钮：克制的玻璃药丸，位置低且不显眼 */
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
